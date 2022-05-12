/*
Copyright 2019 Google LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flinkcluster

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/flink"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
	"github.com/spotify/flink-on-k8s-operator/internal/util"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/hashicorp/go-version"
)

// Converter which converts the FlinkCluster spec to the desired
// underlying Kubernetes resource specs.

const (
	preStopSleepSeconds     = 30
	flinkConfigMapPath      = "/opt/flink/conf"
	flinkConfigMapVolume    = "flink-config-volume"
	submitJobScriptPath     = "/opt/flink-operator/submit-job.sh"
	gcpServiceAccountVolume = "gcp-service-account-volume"
	hadoopConfigVolume      = "hadoop-config-volume"
	jobManagerAddrEnvVar    = "FLINK_JM_ADDR"
	jobJarUriEnvVar         = "FLINK_JOB_JAR_URI"
	jobPyFileUriEnvVar      = "FLINK_JOB_PY_FILE_URI"
	jobPyFilesUriEnvVar     = "FLINK_JOB_PY_FILES_URI"
	hadoopConfDirEnvVar     = "HADOOP_CONF_DIR"
	gacEnvVar               = "GOOGLE_APPLICATION_CREDENTIALS"
	maxUnavailableDefault   = "0%"
)

var (
	backoffLimit                  int32 = 0
	terminationGracePeriodSeconds int64 = 60
	flinkSysProps                       = map[string]struct{}{
		"jobmanager.rpc.address": {},
		"jobmanager.rpc.port":    {},
		"blob.server.port":       {},
		"query.server.port":      {},
		"rest.port":              {},
	}
	v10, _ = version.NewVersion("1.10")
)

// Gets the desired state of a cluster.
func getDesiredClusterState(observed *ObservedClusterState) *model.DesiredClusterState {
	state := &model.DesiredClusterState{}
	cluster := observed.cluster
	// The cluster has been deleted, all resources should be cleaned up.
	if cluster == nil {
		return state
	}

	jobSpec := cluster.Spec.Job
	applicationMode := IsApplicationModeCluster(cluster)

	if !shouldCleanup(cluster, "ConfigMap") {
		state.ConfigMap = newConfigMap(cluster)
	}
	if !shouldCleanup(cluster, "PodDisruptionBudget") {
		state.PodDisruptionBudget = newPodDisruptionBudget(cluster)
	}
	if !shouldCleanup(cluster, "JobManagerStatefulSet") && !applicationMode {
		state.JmStatefulSet = newJobManagerStatefulSet(cluster)
	}

	if !shouldCleanup(cluster, "TaskManagerStatefulSet") {
		state.TmStatefulSet = newTaskManagerStatefulSet(cluster)
	}

	if !shouldCleanup(cluster, "JobManagerService") {
		state.JmService = newJobManagerService(cluster)
	}

	if !shouldCleanup(cluster, "JobManagerIngress") {
		state.JmIngress = newJobManagerIngress(cluster)
	}

	if jobSpec != nil {
		jobStatus := cluster.Status.Components.Job

		keepJobState := (shouldStopJob(cluster) || jobStatus.IsStopped()) &&
			(!shouldUpdateJob(observed) && !jobStatus.ShouldRestart(jobSpec)) &&
			shouldCleanup(cluster, "Job")

		if !keepJobState {
			state.Job = newJob(cluster)
		}
	}

	return state
}

func newJobManagerContainer(flinkCluster *v1beta1.FlinkCluster) *corev1.Container {
	var clusterSpec = flinkCluster.Spec
	var imageSpec = clusterSpec.Image
	var jobManagerSpec = clusterSpec.JobManager
	var rpcPort = corev1.ContainerPort{Name: "rpc", ContainerPort: *jobManagerSpec.Ports.RPC}
	var blobPort = corev1.ContainerPort{Name: "blob", ContainerPort: *jobManagerSpec.Ports.Blob}
	var queryPort = corev1.ContainerPort{Name: "query", ContainerPort: *jobManagerSpec.Ports.Query}
	var uiPort = corev1.ContainerPort{Name: "ui", ContainerPort: *jobManagerSpec.Ports.UI}
	var ports = []corev1.ContainerPort{rpcPort, blobPort, queryPort, uiPort}
	for _, port := range jobManagerSpec.ExtraPorts {
		ports = append(ports, corev1.ContainerPort{Name: port.Name, ContainerPort: port.ContainerPort, Protocol: corev1.Protocol(port.Protocol)})
	}

	container := &corev1.Container{
		Name:            "jobmanager",
		Image:           imageSpec.Name,
		ImagePullPolicy: imageSpec.PullPolicy,
		Args:            []string{"jobmanager"},
		Ports:           ports,
		LivenessProbe:   jobManagerSpec.LivenessProbe,
		ReadinessProbe:  jobManagerSpec.ReadinessProbe,
		Resources:       jobManagerSpec.Resources,
		Env:             flinkCluster.Spec.EnvVars,
		EnvFrom:         flinkCluster.Spec.EnvFrom,
		VolumeMounts:    jobManagerSpec.VolumeMounts,
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"sleep", strconv.Itoa(preStopSleepSeconds)},
				},
			},
		},
	}

	if IsApplicationModeCluster(flinkCluster) {
		jobSpec := flinkCluster.Spec.Job
		status := flinkCluster.Status
		args := []string{"standalone-job"}
		if parallelism, err := calJobParallelism(flinkCluster); err == nil {
			args = append(args, fmt.Sprintf("-Dparallelism.default=%d", parallelism))
		}

		var fromSavepoint = convertFromSavepoint(jobSpec, status.Components.Job, &status.Revision)
		if fromSavepoint != nil {
			args = append(args, "--fromSavepoint", *fromSavepoint)
		}

		if jobSpec.AllowNonRestoredState != nil && *jobSpec.AllowNonRestoredState {
			args = append(args, "--allowNonRestoredState")
		}

		args = append(args,
			"--job-id", flink.GenJobId(flinkCluster.Namespace, flinkCluster.Name),
			"--job-classname", *jobSpec.ClassName,
		)

		args = append(args, jobSpec.Args...)
		container.Args = args
	}

	return container
}

func newJobManagerPodSpec(mainContainer *corev1.Container, flinkCluster *v1beta1.FlinkCluster) *corev1.PodSpec {
	var clusterSpec = flinkCluster.Spec
	var imageSpec = clusterSpec.Image
	var serviceAccount = clusterSpec.ServiceAccountName
	var jobManagerSpec = clusterSpec.JobManager

	var podSpec = &corev1.PodSpec{
		InitContainers:                convertContainers(jobManagerSpec.InitContainers, []corev1.VolumeMount{}, clusterSpec.EnvVars),
		Containers:                    []corev1.Container{*mainContainer},
		Volumes:                       jobManagerSpec.Volumes,
		NodeSelector:                  jobManagerSpec.NodeSelector,
		Tolerations:                   jobManagerSpec.Tolerations,
		ImagePullSecrets:              imageSpec.PullSecrets,
		SecurityContext:               jobManagerSpec.SecurityContext,
		ServiceAccountName:            getServiceAccountName(serviceAccount),
		TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
	}
	setFlinkConfig(getConfigMapName(flinkCluster.Name), podSpec)
	setHadoopConfig(flinkCluster.Spec.HadoopConfig, podSpec)
	setGCPConfig(flinkCluster.Spec.GCPConfig, podSpec)
	podSpec.Containers = append(podSpec.Containers, jobManagerSpec.Sidecars...)

	return podSpec
}

// Gets the desired JobManager StatefulSet spec from the FlinkCluster spec.
func newJobManagerStatefulSet(flinkCluster *v1beta1.FlinkCluster) *appsv1.StatefulSet {
	var jobManagerSpec = flinkCluster.Spec.JobManager
	var jobManagerStatefulSetName = getJobManagerStatefulSetName(flinkCluster.Name)
	var podLabels = getComponentLabels(flinkCluster, "jobmanager")
	podLabels = mergeLabels(podLabels, jobManagerSpec.PodLabels)
	var statefulSetLabels = mergeLabels(podLabels, getRevisionHashLabels(&flinkCluster.Status.Revision))

	mainContainer := newJobManagerContainer(flinkCluster)
	podSpec := newJobManagerPodSpec(mainContainer, flinkCluster)

	var pvcs []corev1.PersistentVolumeClaim
	if jobManagerSpec.VolumeClaimTemplates != nil {
		pvcs = make([]corev1.PersistentVolumeClaim, len(jobManagerSpec.VolumeClaimTemplates))
		for i, pvc := range jobManagerSpec.VolumeClaimTemplates {
			pvc.OwnerReferences = []metav1.OwnerReference{ToOwnerReference(flinkCluster)}
			pvcs[i] = pvc
		}
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       flinkCluster.Namespace,
			Name:            jobManagerStatefulSetName,
			OwnerReferences: []metav1.OwnerReference{ToOwnerReference(flinkCluster)},
			Labels:          statefulSetLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             jobManagerSpec.Replicas,
			Selector:             &metav1.LabelSelector{MatchLabels: podLabels},
			ServiceName:          jobManagerStatefulSetName,
			VolumeClaimTemplates: pvcs,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: jobManagerSpec.PodAnnotations,
				},
				Spec: *podSpec,
			},
		},
	}
}

// Gets the desired JobManager service spec from a cluster spec.
func newJobManagerService(flinkCluster *v1beta1.FlinkCluster) *corev1.Service {
	var clusterNamespace = flinkCluster.Namespace
	var clusterName = flinkCluster.Name
	var jobManagerSpec = flinkCluster.Spec.JobManager
	var rpcPort = corev1.ServicePort{
		Name:       "rpc",
		Port:       *jobManagerSpec.Ports.RPC,
		TargetPort: intstr.FromString("rpc")}
	var blobPort = corev1.ServicePort{
		Name:       "blob",
		Port:       *jobManagerSpec.Ports.Blob,
		TargetPort: intstr.FromString("blob")}
	var queryPort = corev1.ServicePort{
		Name:       "query",
		Port:       *jobManagerSpec.Ports.Query,
		TargetPort: intstr.FromString("query")}
	var uiPort = corev1.ServicePort{
		Name:       "ui",
		Port:       *jobManagerSpec.Ports.UI,
		TargetPort: intstr.FromString("ui")}
	var jobManagerServiceName = getJobManagerServiceName(clusterName)
	selectorLabels := getComponentLabels(flinkCluster, "jobmanager")
	serviceLabels := mergeLabels(selectorLabels, getRevisionHashLabels(&flinkCluster.Status.Revision))
	serviceLabels = mergeLabels(serviceLabels, jobManagerSpec.ServiceLabels)
	var serviceAnnotations = jobManagerSpec.ServiceAnnotations

	var jobManagerService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      jobManagerServiceName,
			OwnerReferences: []metav1.OwnerReference{
				ToOwnerReference(flinkCluster)},
			Labels:      serviceLabels,
			Annotations: serviceAnnotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels,
			Ports:    []corev1.ServicePort{rpcPort, blobPort, queryPort, uiPort},
		},
	}
	// This implementation is specific to GKE, see details at
	// https://cloud.google.com/kubernetes-engine/docs/how-to/exposing-apps
	// https://cloud.google.com/kubernetes-engine/docs/how-to/internal-load-balancing
	switch jobManagerSpec.AccessScope {
	case v1beta1.AccessScopeCluster:
		jobManagerService.Spec.Type = corev1.ServiceTypeClusterIP
	case v1beta1.AccessScopeVPC:
		jobManagerService.Spec.Type = corev1.ServiceTypeLoadBalancer
		jobManagerService.Annotations = mergeLabels(serviceAnnotations,
			map[string]string{
				"networking.gke.io/load-balancer-type":                         "Internal",
				"networking.gke.io/internal-load-balancer-allow-global-access": "true",
			})
	case v1beta1.AccessScopeExternal:
		jobManagerService.Spec.Type = corev1.ServiceTypeLoadBalancer
	case v1beta1.AccessScopeNodePort:
		jobManagerService.Spec.Type = corev1.ServiceTypeNodePort
	case v1beta1.AccessScopeHeadless:
		// Headless services do not allocate any sort of VIP or LoadBalancer, and merely
		// collect a set of Pod IPs that are assumed to be independently routable:
		jobManagerService.Spec.Type = corev1.ServiceTypeClusterIP
		jobManagerService.Spec.ClusterIP = "None"
	default:
		panic(fmt.Sprintf(
			"Unknown service access cope: %v", jobManagerSpec.AccessScope))
	}
	return jobManagerService
}

// Gets the desired JobManager ingress spec from a cluster spec.
func newJobManagerIngress(
	flinkCluster *v1beta1.FlinkCluster) *networkingv1.Ingress {
	var jobManagerIngressSpec = flinkCluster.Spec.JobManager.Ingress
	if jobManagerIngressSpec == nil {
		return nil
	}

	var clusterNamespace = flinkCluster.Namespace
	var clusterName = flinkCluster.Name
	var jobManagerServiceName = getJobManagerServiceName(clusterName)
	var jobManagerServiceUIPort = intstr.FromString("ui")
	var ingressName = getJobManagerIngressName(clusterName)
	var ingressAnnotations = jobManagerIngressSpec.Annotations
	var ingressHost string
	var ingressTLS []networkingv1.IngressTLS
	var labels = mergeLabels(
		getComponentLabels(flinkCluster, "jobmanager"),
		getRevisionHashLabels(&flinkCluster.Status.Revision))
	var pathType = networkingv1.PathTypePrefix
	if jobManagerIngressSpec.HostFormat != nil {
		ingressHost = getJobManagerIngressHost(*jobManagerIngressSpec.HostFormat, clusterName)
	}
	if jobManagerIngressSpec.UseTLS != nil && *jobManagerIngressSpec.UseTLS {
		var secretName string
		var hosts []string
		if ingressHost != "" {
			hosts = []string{ingressHost}
		}
		if jobManagerIngressSpec.TLSSecretName != nil {
			secretName = *jobManagerIngressSpec.TLSSecretName
		}
		if hosts != nil || secretName != "" {
			ingressTLS = []networkingv1.IngressTLS{{
				Hosts:      hosts,
				SecretName: secretName,
			}}
		}
	}
	var jobManagerIngress = &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      ingressName,
			OwnerReferences: []metav1.OwnerReference{
				ToOwnerReference(flinkCluster)},
			Labels:      labels,
			Annotations: ingressAnnotations,
		},
		Spec: networkingv1.IngressSpec{
			TLS: ingressTLS,
			Rules: []networkingv1.IngressRule{{
				Host: ingressHost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: jobManagerServiceName,
									Port: networkingv1.ServiceBackendPort{
										Name:   jobManagerServiceName,
										Number: jobManagerServiceUIPort.IntVal,
									},
								},
							},
						}},
					},
				},
			}},
		},
	}

	return jobManagerIngress
}

func newTaskMangerContainer(flinkCluster *v1beta1.FlinkCluster) *corev1.Container {
	var imageSpec = flinkCluster.Spec.Image
	var taskManagerSpec = flinkCluster.Spec.TaskManager
	var dataPort = corev1.ContainerPort{Name: "data", ContainerPort: *taskManagerSpec.Ports.Data}
	var rpcPort = corev1.ContainerPort{Name: "rpc", ContainerPort: *taskManagerSpec.Ports.RPC}
	var queryPort = corev1.ContainerPort{Name: "query", ContainerPort: *taskManagerSpec.Ports.Query}
	var ports = []corev1.ContainerPort{dataPort, rpcPort, queryPort}
	for _, port := range taskManagerSpec.ExtraPorts {
		ports = append(ports, corev1.ContainerPort{Name: port.Name, ContainerPort: port.ContainerPort, Protocol: corev1.Protocol(port.Protocol)})
	}

	var envVars = []corev1.EnvVar{
		{
			Name: "TASK_MANAGER_CPU_LIMIT",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: "taskmanager",
					Resource:      "limits.cpu",
					Divisor:       resource.MustParse("1m"),
				},
			},
		},
		{
			Name: "TASK_MANAGER_MEMORY_LIMIT",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					ContainerName: "taskmanager",
					Resource:      "limits.memory",
					Divisor:       resource.MustParse("1Mi"),
				},
			},
		},
	}

	envVars = append(envVars, flinkCluster.Spec.EnvVars...)

	return &corev1.Container{
		Name:            "taskmanager",
		Image:           imageSpec.Name,
		ImagePullPolicy: imageSpec.PullPolicy,
		Args:            []string{"taskmanager"},
		Ports:           ports,
		LivenessProbe:   taskManagerSpec.LivenessProbe,
		ReadinessProbe:  taskManagerSpec.ReadinessProbe,
		Resources:       taskManagerSpec.Resources,
		Env:             envVars,
		EnvFrom:         flinkCluster.Spec.EnvFrom,
		VolumeMounts:    taskManagerSpec.VolumeMounts,
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"sleep", strconv.Itoa(preStopSleepSeconds)},
				},
			},
		},
	}
}

func newTaskManagerPodSpec(mainContainer *corev1.Container, flinkCluster *v1beta1.FlinkCluster) *corev1.PodSpec {
	var clusterSpec = flinkCluster.Spec
	var imageSpec = flinkCluster.Spec.Image
	var serviceAccount = clusterSpec.ServiceAccountName
	var taskManagerSpec = flinkCluster.Spec.TaskManager

	var podSpec = &corev1.PodSpec{
		InitContainers:                convertContainers(taskManagerSpec.InitContainers, []corev1.VolumeMount{}, clusterSpec.EnvVars),
		Containers:                    []corev1.Container{*mainContainer},
		Volumes:                       taskManagerSpec.Volumes,
		NodeSelector:                  taskManagerSpec.NodeSelector,
		Tolerations:                   taskManagerSpec.Tolerations,
		ImagePullSecrets:              imageSpec.PullSecrets,
		SecurityContext:               taskManagerSpec.SecurityContext,
		ServiceAccountName:            getServiceAccountName(serviceAccount),
		TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
	}

	setFlinkConfig(getConfigMapName(flinkCluster.Name), podSpec)
	setHadoopConfig(flinkCluster.Spec.HadoopConfig, podSpec)
	setGCPConfig(flinkCluster.Spec.GCPConfig, podSpec)
	podSpec.Containers = append(podSpec.Containers, taskManagerSpec.Sidecars...)

	return podSpec
}

// Gets the desired TaskManager StatefulSet spec from a cluster spec.
func newTaskManagerStatefulSet(flinkCluster *v1beta1.FlinkCluster) *appsv1.StatefulSet {
	var taskManagerSpec = flinkCluster.Spec.TaskManager
	var taskManagerStatefulSetName = getTaskManagerStatefulSetName(flinkCluster.Name)
	var podLabels = getComponentLabels(flinkCluster, "taskmanager")
	podLabels = mergeLabels(podLabels, taskManagerSpec.PodLabels)
	var statefulSetLabels = mergeLabels(podLabels, getRevisionHashLabels(&flinkCluster.Status.Revision))

	mainContainer := newTaskMangerContainer(flinkCluster)
	podSpec := newTaskManagerPodSpec(mainContainer, flinkCluster)

	var pvcs []corev1.PersistentVolumeClaim
	if taskManagerSpec.VolumeClaimTemplates != nil {
		pvcs = make([]corev1.PersistentVolumeClaim, len(taskManagerSpec.VolumeClaimTemplates))
		for i, pvc := range taskManagerSpec.VolumeClaimTemplates {
			pvc.OwnerReferences = []metav1.OwnerReference{ToOwnerReference(flinkCluster)}
			pvcs[i] = pvc
		}
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       flinkCluster.Namespace,
			Name:            taskManagerStatefulSetName,
			OwnerReferences: []metav1.OwnerReference{ToOwnerReference(flinkCluster)},
			Labels:          statefulSetLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             taskManagerSpec.Replicas,
			Selector:             &metav1.LabelSelector{MatchLabels: podLabels},
			ServiceName:          taskManagerStatefulSetName,
			VolumeClaimTemplates: pvcs,
			PodManagementPolicy:  "Parallel",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: taskManagerSpec.PodAnnotations,
				},
				Spec: *podSpec,
			},
		},
	}
}

// Gets the desired PodDisruptionBudget.
func newPodDisruptionBudget(flinkCluster *v1beta1.FlinkCluster) *policyv1.PodDisruptionBudget {
	var jobSpec = flinkCluster.Spec.Job
	if jobSpec == nil {
		return nil
	}
	var clusterNamespace = flinkCluster.Namespace
	var clusterName = flinkCluster.Name
	var pdbName = getPodDisruptionBudgetName(clusterName)
	var labels = getClusterLabels(flinkCluster)

	var maxUnavailablePods = intstr.FromString(maxUnavailableDefault)

	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       clusterNamespace,
			Name:            pdbName,
			OwnerReferences: []metav1.OwnerReference{ToOwnerReference(flinkCluster)},
			Labels:          labels,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailablePods,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
	}
}

// Gets the desired configMap.
func newConfigMap(flinkCluster *v1beta1.FlinkCluster) *corev1.ConfigMap {
	appVersion, _ := version.NewVersion(flinkCluster.Spec.FlinkVersion)

	var clusterNamespace = flinkCluster.Namespace
	var clusterName = flinkCluster.Name
	var flinkProperties = flinkCluster.Spec.FlinkProperties
	var jmPorts = flinkCluster.Spec.JobManager.Ports
	var tmPorts = flinkCluster.Spec.TaskManager.Ports
	var configMapName = getConfigMapName(clusterName)
	var labels = mergeLabels(
		getClusterLabels(flinkCluster),
		getRevisionHashLabels(&flinkCluster.Status.Revision))
	// Properties which should be provided from real deployed environment.
	var flinkProps = map[string]string{
		"jobmanager.rpc.address": getJobManagerServiceName(clusterName),
		"jobmanager.rpc.port":    strconv.FormatInt(int64(*jmPorts.RPC), 10),
		"blob.server.port":       strconv.FormatInt(int64(*jmPorts.Blob), 10),
		"query.server.port":      strconv.FormatInt(int64(*jmPorts.Query), 10),
		"rest.port":              strconv.FormatInt(int64(*jmPorts.UI), 10),
		"taskmanager.rpc.port":   strconv.FormatInt(int64(*tmPorts.RPC), 10),
	}

	if appVersion == nil || appVersion.LessThan(v10) {
		var flinkHeapSize = calFlinkHeapSize(flinkCluster)
		if flinkHeapSize["jobmanager.heap.size"] != "" {
			flinkProps["jobmanager.heap.size"] = flinkHeapSize["jobmanager.heap.size"]
		}
		if flinkHeapSize["taskmanager.heap.size"] != "" {
			flinkProps["taskmanager.heap.size"] = flinkHeapSize["taskmanager.heap.size"]
		}
	} else {
		var flinkProcessMemorySize = calFlinkMemoryProcessSize(flinkCluster)
		if flinkProcessMemorySize["jobmanager.memory.process.size"] != "" {
			flinkProps["jobmanager.memory.process.size"] = flinkProcessMemorySize["jobmanager.memory.process.size"]
		}
		if flinkProcessMemorySize["taskmanager.memory.process.size"] != "" {
			flinkProps["taskmanager.memory.process.size"] = flinkProcessMemorySize["taskmanager.memory.process.size"]
		}
	}

	if taskSlots, err := calTaskManagerTaskSlots(flinkCluster); err == nil {
		flinkProps["taskmanager.numberOfTaskSlots"] = strconv.Itoa(int(taskSlots))
	}

	// Add custom Flink properties.
	for k, v := range flinkProperties {
		// Do not allow to override properties from real deployment.
		if _, ok := flinkSysProps[k]; ok {
			continue
		}
		flinkProps[k] = v
	}
	var configData = getLogConf(flinkCluster.Spec)
	configData["flink-conf.yaml"] = getFlinkProperties(flinkProps)
	configData["submit-job.sh"] = submitJobScript
	var configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       clusterNamespace,
			Name:            configMapName,
			OwnerReferences: []metav1.OwnerReference{ToOwnerReference(flinkCluster)},
			Labels:          labels,
		},
		Data: configData,
	}

	return configMap
}

func newJobSubmitterPodSpec(flinkCluster *v1beta1.FlinkCluster) *corev1.PodSpec {
	var jobSpec = flinkCluster.Spec.Job
	if jobSpec == nil {
		return nil
	}

	var status = flinkCluster.Status
	var clusterSpec = flinkCluster.Spec
	var imageSpec = clusterSpec.Image
	var serviceAccount = clusterSpec.ServiceAccountName
	var jobManagerSpec = clusterSpec.JobManager
	var clusterName = flinkCluster.Name
	var jobManagerServiceName = getJobManagerServiceName(clusterName)
	var jobManagerAddress = fmt.Sprintf(
		"%s:%d", jobManagerServiceName, *jobManagerSpec.Ports.UI)

	var jobArgs = []string{"bash", submitJobScriptPath}
	jobArgs = append(jobArgs, "--jobmanager", jobManagerAddress)
	if jobSpec.ClassName != nil {
		jobArgs = append(jobArgs, "--class", *jobSpec.ClassName)
	}

	var fromSavepoint = convertFromSavepoint(jobSpec, status.Components.Job, &status.Revision)
	if fromSavepoint != nil {
		jobArgs = append(jobArgs, "--fromSavepoint", *fromSavepoint)
	}

	if jobSpec.AllowNonRestoredState != nil &&
		*jobSpec.AllowNonRestoredState {
		jobArgs = append(jobArgs, "--allowNonRestoredState")
	}

	if parallelism, err := calJobParallelism(flinkCluster); err == nil {
		jobArgs = append(jobArgs, "--parallelism", fmt.Sprint(parallelism))
	}

	if jobSpec.NoLoggingToStdout != nil &&
		*jobSpec.NoLoggingToStdout {
		jobArgs = append(jobArgs, "--sysoutLogging")
	}

	if jobSpec.Mode != nil && *jobSpec.Mode == v1beta1.JobModeDetached {
		jobArgs = append(jobArgs, "--detached")
	}

	if jobSpec.ClassPath != nil && len(jobSpec.ClassPath) > 0 {
		for _, u := range jobSpec.ClassPath {
			jobArgs = append(jobArgs, "-C", u)
		}
	}

	envVars := []corev1.EnvVar{{
		Name:  jobManagerAddrEnvVar,
		Value: jobManagerAddress,
	}}
	envVars = append(envVars, flinkCluster.Spec.EnvVars...)

	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	volumes = append(volumes, jobSpec.Volumes...)
	volumeMounts = append(volumeMounts, jobSpec.VolumeMounts...)

	// Submit job script config.
	sbsVolume, sbsMount, confMount := convertSubmitJobScript(clusterName)
	volumes = append(volumes, *sbsVolume)
	volumeMounts = append(volumeMounts, *sbsMount, *confMount)

	if jobSpec.JarFile != nil {
		jobArgs = append(jobArgs, *jobSpec.JarFile)
	}

	if jobSpec.PyFile != nil {
		jobArgs = append(jobArgs, "--python", *jobSpec.PyFile)
	}

	if jobSpec.PyFiles != nil {
		jobArgs = append(jobArgs, "--pyFiles", *jobSpec.PyFiles)
	}

	if jobSpec.PyModule != nil {
		jobArgs = append(jobArgs, "--pyModule", *jobSpec.PyModule)
	}

	jobArgs = append(jobArgs, jobSpec.Args...)

	podSpec := &corev1.PodSpec{
		InitContainers: convertContainers(jobSpec.InitContainers, volumeMounts, envVars),
		Containers: []corev1.Container{
			{
				Name:            "main",
				Image:           imageSpec.Name,
				ImagePullPolicy: imageSpec.PullPolicy,
				Args:            jobArgs,
				Env:             envVars,
				EnvFrom:         flinkCluster.Spec.EnvFrom,
				VolumeMounts:    volumeMounts,
				Resources:       jobSpec.Resources,
			},
		},
		RestartPolicy:      corev1.RestartPolicyNever,
		Volumes:            volumes,
		ImagePullSecrets:   imageSpec.PullSecrets,
		SecurityContext:    jobSpec.SecurityContext,
		ServiceAccountName: getServiceAccountName(serviceAccount),
		NodeSelector:       jobSpec.NodeSelector,
		Tolerations:        jobSpec.Tolerations,
	}

	setFlinkConfig(getConfigMapName(flinkCluster.Name), podSpec)
	setHadoopConfig(flinkCluster.Spec.HadoopConfig, podSpec)
	setGCPConfig(flinkCluster.Spec.GCPConfig, podSpec)

	return podSpec
}

func newJob(flinkCluster *v1beta1.FlinkCluster) *batchv1.Job {
	jobSpec := flinkCluster.Spec.Job
	if jobSpec == nil {
		return nil
	}

	recorded := flinkCluster.Status
	jobManagerSpec := flinkCluster.Spec.JobManager
	labels := getClusterLabels(flinkCluster)
	labels = mergeLabels(labels, getRevisionHashLabels(&recorded.Revision))

	var jobName string
	var annotations map[string]string
	var podSpec *corev1.PodSpec

	if IsApplicationModeCluster(flinkCluster) {
		labels = mergeLabels(labels, getComponentLabels(flinkCluster, "jobmanager"))
		labels = mergeLabels(labels, jobManagerSpec.PodLabels)
		labels = mergeLabels(labels, map[string]string{
			"job-id": flink.GenJobId(flinkCluster.Namespace, flinkCluster.Name),
		})
		jobName = getJobManagerJobName(flinkCluster.Name)
		annotations = jobManagerSpec.PodAnnotations
		mainContainer := newJobManagerContainer(flinkCluster)
		podSpec = newJobManagerPodSpec(mainContainer, flinkCluster)
	} else {
		jobName = getSubmitterJobName(flinkCluster.Name)
		labels = mergeLabels(labels, jobSpec.PodLabels)
		annotations = jobSpec.PodAnnotations
		podSpec = newJobSubmitterPodSpec(flinkCluster)
	}

	// Disable the retry mechanism of k8s Job, all retries should be initiated
	// by the operator based on the job restart policy. This is because Flink
	// jobs are stateful, if a job fails after running for 10 hours, we probably
	// don't want to start over from the beginning, instead we want to resume
	// the job from the latest savepoint which means strictly speaking it is no
	// longer the same job as the previous one because the `--fromSavepoint`
	// parameter has changed.
	podSpec.RestartPolicy = corev1.RestartPolicyNever

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       flinkCluster.Namespace,
			Name:            jobName,
			OwnerReferences: []metav1.OwnerReference{ToOwnerReference(flinkCluster)},
			Labels:          labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: *podSpec,
			},
			BackoffLimit: &backoffLimit,
		},
	}
}

// Decide from which savepoint Flink job should be restored when the job created, updated or restarted
//
// case 1) Restore job from the user provided savepoint
// When FlinkCluster is created or updated, if spec.job.fromSavepoint is specified, Flink job will be restored from it.
//
// case 2) Restore Flink job from the latest savepoint.
// When FlinkCluster is updated with no spec.job.fromSavepoint, or job is restarted from the failed state,
// Flink job will be restored from the latest savepoint created by the operator.
//
// case 3) When latest created savepoint is unavailable, use the savepoint from which current job was restored.
func convertFromSavepoint(jobSpec *v1beta1.JobSpec, jobStatus *v1beta1.JobStatus, revision *v1beta1.RevisionStatus) *string {
	switch {
	// Updating with FromSavepoint provided
	case revision.IsUpdateTriggered() && !util.IsBlank(jobSpec.FromSavepoint):
		return jobSpec.FromSavepoint
	// Latest savepoint
	case jobStatus != nil && jobStatus.SavepointLocation != "":
		return &jobStatus.SavepointLocation
	// The savepoint from which current job was restored
	case jobStatus != nil && jobStatus.FromSavepoint != "":
		return &jobStatus.FromSavepoint
	}
	// Creating for the first time or other situation
	if !util.IsBlank(jobSpec.FromSavepoint) {
		return jobSpec.FromSavepoint
	}
	return nil
}

func appendVolumes(volumes []corev1.Volume, newVolumes ...corev1.Volume) []corev1.Volume {
	for _, mounts := range newVolumes {
		var conflict = false
		for _, mount := range volumes {
			if mounts.Name == mount.Name {
				conflict = true
				break
			}
		}
		if !conflict {
			volumes = append(volumes, mounts)
		}
	}

	return volumes
}

func appendVolumeMounts(volumeMounts []corev1.VolumeMount, newVolumeMounts ...corev1.VolumeMount) []corev1.VolumeMount {
	for _, mounts := range newVolumeMounts {
		var conflict = false
		for _, mount := range volumeMounts {
			if mounts.MountPath == mount.MountPath {
				conflict = true
				break
			}
		}
		if !conflict {
			volumeMounts = append(volumeMounts, mounts)
		}
	}

	return volumeMounts
}

func appendEnvVars(envVars []corev1.EnvVar, newEnvVars ...corev1.EnvVar) []corev1.EnvVar {
	for _, envVar := range newEnvVars {
		var conflict = false
		for _, env := range envVars {
			if envVar.Name == env.Name {
				conflict = true
				break
			}
		}
		if !conflict {
			envVars = append(envVars, envVar)
		}
	}

	return envVars
}

// Copy any non-duplicate volume mounts and env vars to each specified container
func convertContainer(container corev1.Container, volumeMounts []corev1.VolumeMount, envVars []corev1.EnvVar) corev1.Container {
	container.VolumeMounts = appendVolumeMounts(container.VolumeMounts, volumeMounts...)
	container.Env = appendEnvVars(container.Env, envVars...)

	return container
}

// Copy any non-duplicate volume mounts and env vars to the specified containers
func convertContainers(containers []corev1.Container, volumeMounts []corev1.VolumeMount, envVars []corev1.EnvVar) []corev1.Container {
	var updatedContainers = []corev1.Container{}
	for _, container := range containers {
		updatedContainer := convertContainer(container, volumeMounts, envVars)
		updatedContainers = append(updatedContainers, updatedContainer)
	}
	return updatedContainers
}

// Converts the FlinkCluster as owner reference for its child resources.
func ToOwnerReference(
	flinkCluster *v1beta1.FlinkCluster) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         flinkCluster.APIVersion,
		Kind:               flinkCluster.Kind,
		Name:               flinkCluster.Name,
		UID:                flinkCluster.UID,
		Controller:         &[]bool{true}[0],
		BlockOwnerDeletion: &[]bool{false}[0],
	}
}

// Gets Flink properties
func getFlinkProperties(properties map[string]string) string {
	var keys = make([]string, len(properties))
	i := 0
	for k := range properties {
		keys[i] = k
		i = i + 1
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("%s: %s\n", key, properties[key]))
	}
	return builder.String()
}

var jobManagerIngressHostRegex = regexp.MustCompile(`{{\s*[$]clusterName\s*}}`)

func getJobManagerIngressHost(ingressHostFormat string, clusterName string) string {
	// TODO: Validating webhook should verify hostFormat
	return jobManagerIngressHostRegex.ReplaceAllString(ingressHostFormat, clusterName)
}

// Checks whether the component should be deleted according to the cleanup
// policy. Always return false for session cluster.
func shouldCleanup(cluster *v1beta1.FlinkCluster, component string) bool {
	var jobStatus = cluster.Status.Components.Job
	// Session cluster.
	if jobStatus == nil {
		return false
	}

	if cluster.Status.Revision.IsUpdateTriggered() {
		return false
	}

	var action v1beta1.CleanupAction
	switch jobStatus.State {
	case v1beta1.JobStateSucceeded:
		action = cluster.Spec.Job.CleanupPolicy.AfterJobSucceeds
	case v1beta1.JobStateFailed, v1beta1.JobStateLost, v1beta1.JobStateDeployFailed:
		action = cluster.Spec.Job.CleanupPolicy.AfterJobFails
	case v1beta1.JobStateCancelled:
		action = cluster.Spec.Job.CleanupPolicy.AfterJobCancelled
	default:
		return false
	}

	switch action {
	case v1beta1.CleanupActionDeleteCluster:
		return true
	case v1beta1.CleanupActionDeleteTaskManager:
		return component == "TaskManagerStatefulSet"
	}

	return false
}

func calJobParallelism(cluster *v1beta1.FlinkCluster) (int32, error) {
	if cluster.Spec.Job.Parallelism != nil {
		return *cluster.Spec.Job.Parallelism, nil
	}

	value, err := calTaskManagerTaskSlots(cluster)
	if err != nil {
		return 0, err
	}

	parallelism := *cluster.Spec.TaskManager.Replicas * value
	return parallelism, nil
}

func calTaskManagerTaskSlots(cluster *v1beta1.FlinkCluster) (int32, error) {
	if ts, ok := cluster.Spec.FlinkProperties["taskmanager.numberOfTaskSlots"]; ok {
		parsed, err := strconv.ParseInt(ts, 10, 32)
		if err != nil {
			return 0, err
		}
		return int32(parsed), nil
	}

	slots := int32(cluster.Spec.TaskManager.Resources.Limits.Cpu().Value()) / 2
	if slots == 0 {
		return 1, nil
	}
	return slots, nil
}

func calFlinkHeapSize(cluster *v1beta1.FlinkCluster) map[string]string {
	jm := cluster.Spec.JobManager
	tm := cluster.Spec.TaskManager

	if jm.MemoryOffHeapRatio == nil || tm.MemoryOffHeapRatio == nil {
		return nil
	}

	var flinkHeapSize = make(map[string]string)

	jmHeapSizeMB := calHeapSize(
		jm.Resources.Limits.Memory().Value(),
		jm.MemoryOffHeapMin.Value(),
		int64(*jm.MemoryOffHeapRatio))
	if jmHeapSizeMB > 0 {
		flinkHeapSize["jobmanager.heap.size"] = strconv.FormatInt(jmHeapSizeMB, 10) + "m"
	}

	tmHeapSizeMB := calHeapSize(
		tm.Resources.Limits.Memory().Value(),
		tm.MemoryOffHeapMin.Value(),
		int64(*tm.MemoryOffHeapRatio))
	if tmHeapSizeMB > 0 {
		flinkHeapSize["taskmanager.heap.size"] = strconv.FormatInt(tmHeapSizeMB, 10) + "m"
	}

	return flinkHeapSize
}

// Converts memory value to the format of divisor and returns ceiling of the value.
func convertResourceMemoryToInt64(memory resource.Quantity, divisor resource.Quantity) int64 {
	return int64(math.Ceil(float64(memory.Value()) / float64(divisor.Value())))
}

// Calculate heap size in MB
func calHeapSize(memSize int64, offHeapMin int64, offHeapRatio int64) int64 {
	var heapSizeMB int64
	offHeapSize := int64(math.Ceil(float64(memSize*offHeapRatio) / 100))
	if offHeapSize < offHeapMin {
		offHeapSize = offHeapMin
	}
	heapSizeCalculated := memSize - offHeapSize
	if heapSizeCalculated > 0 {
		divisor := resource.MustParse("1Mi")
		heapSizeQuantity := resource.NewQuantity(heapSizeCalculated, resource.DecimalSI)
		heapSizeMB = convertResourceMemoryToInt64(*heapSizeQuantity, divisor)
	}
	return heapSizeMB
}

func calProcessMemorySize(memSize, ratio int64) int64 {
	size := int64(math.Ceil(float64((memSize * ratio)) / 100))
	divisor := resource.MustParse("1Mi")
	quantity := resource.NewQuantity(size, resource.DecimalSI)
	return convertResourceMemoryToInt64(*quantity, divisor)
}

// Calculate process memory size in MB
func calFlinkMemoryProcessSize(cluster *v1beta1.FlinkCluster) map[string]string {
	var flinkProcessMemory = make(map[string]string)
	jm := cluster.Spec.JobManager
	tm := cluster.Spec.TaskManager

	jmMemoryLimitByte := jm.Resources.Limits.Memory().Value()
	jmRatio := int64(*jm.MemoryProcessRatio)
	jmSizeMB := calProcessMemorySize(jmMemoryLimitByte, jmRatio)
	if jmSizeMB > 0 {
		flinkProcessMemory["jobmanager.memory.process.size"] = strconv.FormatInt(jmSizeMB, 10) + "m"
	}

	tmMemLimitByte := tm.Resources.Limits.Memory().Value()
	ratio := int64(*tm.MemoryProcessRatio)
	sizeMB := calProcessMemorySize(tmMemLimitByte, ratio)
	if sizeMB > 0 {
		flinkProcessMemory["taskmanager.memory.process.size"] = strconv.FormatInt(sizeMB, 10) + "m"
	}

	return flinkProcessMemory
}

func setFlinkConfig(name string, podSpec *corev1.PodSpec) bool {
	var envVars []corev1.EnvVar
	volumes := []corev1.Volume{{
		Name: flinkConfigMapVolume,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		},
	}}
	volumeMounts := []corev1.VolumeMount{{
		Name:      flinkConfigMapVolume,
		MountPath: flinkConfigMapPath,
	}}

	podSpec.Containers = convertContainers(podSpec.Containers, volumeMounts, envVars)
	podSpec.InitContainers = convertContainers(podSpec.InitContainers, volumeMounts, envVars)
	podSpec.Volumes = appendVolumes(podSpec.Volumes, volumes...)
	return true
}

func convertSubmitJobScript(clusterName string) (*corev1.Volume, *corev1.VolumeMount, *corev1.VolumeMount) {
	confVol := &corev1.Volume{
		Name: flinkConfigMapVolume,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: getConfigMapName(clusterName),
				},
			},
		},
	}
	scriptMount := &corev1.VolumeMount{
		Name:      flinkConfigMapVolume,
		MountPath: submitJobScriptPath,
		SubPath:   "submit-job.sh",
	}
	confMount := &corev1.VolumeMount{
		Name:      flinkConfigMapVolume,
		MountPath: flinkConfigMapPath,
	}
	return confVol, scriptMount, confMount
}

func setHadoopConfig(hadoopConfig *v1beta1.HadoopConfig, podSpec *corev1.PodSpec) bool {
	if hadoopConfig == nil {
		return false
	}

	var volumes = []corev1.Volume{{
		Name: hadoopConfigVolume,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: hadoopConfig.ConfigMapName,
				},
			},
		},
	}}
	var volumeMounts = []corev1.VolumeMount{{
		Name:      hadoopConfigVolume,
		MountPath: hadoopConfig.MountPath,
		ReadOnly:  true,
	}}
	var envVars = []corev1.EnvVar{{
		Name:  hadoopConfDirEnvVar,
		Value: hadoopConfig.MountPath,
	}}

	podSpec.Containers = convertContainers(podSpec.Containers, volumeMounts, envVars)
	podSpec.InitContainers = convertContainers(podSpec.InitContainers, volumeMounts, envVars)
	podSpec.Volumes = appendVolumes(podSpec.Volumes, volumes...)
	return true
}

func setGCPConfig(gcpConfig *v1beta1.GCPConfig, podSpec *corev1.PodSpec) bool {
	if gcpConfig == nil {
		return false
	}

	var saConfig = gcpConfig.ServiceAccount
	var saVolume = corev1.Volume{
		Name: gcpServiceAccountVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: gcpConfig.ServiceAccount.SecretName,
			},
		},
	}
	var saMount = corev1.VolumeMount{
		Name:      gcpServiceAccountVolume,
		MountPath: gcpConfig.ServiceAccount.MountPath,
		ReadOnly:  true,
	}
	if !strings.HasSuffix(saMount.MountPath, "/") {
		saMount.MountPath = saMount.MountPath + "/"
	}

	var saEnv = corev1.EnvVar{
		Name:  gacEnvVar,
		Value: saMount.MountPath + saConfig.KeyFile,
	}

	volumes := []corev1.Volume{saVolume}
	volumeMounts := []corev1.VolumeMount{saMount}
	envVars := []corev1.EnvVar{saEnv}
	podSpec.Containers = convertContainers(podSpec.Containers, volumeMounts, envVars)
	podSpec.InitContainers = convertContainers(podSpec.InitContainers, volumeMounts, envVars)
	podSpec.Volumes = appendVolumes(podSpec.Volumes, volumes...)
	return true
}

func getClusterLabels(cluster *v1beta1.FlinkCluster) map[string]string {
	return map[string]string{
		"cluster": cluster.Name,
		"app":     "flink",
	}
}

func getServiceAccountName(serviceAccount *string) string {
	if serviceAccount != nil {
		return *serviceAccount
	}

	return ""
}

func getComponentLabels(cluster *v1beta1.FlinkCluster, component string) map[string]string {
	return mergeLabels(getClusterLabels(cluster), map[string]string{
		"component": component,
	})
}

func getRevisionHashLabels(r *v1beta1.RevisionStatus) map[string]string {
	return map[string]string{
		RevisionNameLabel: getNextRevisionName(r),
	}
}

func mergeLabels(labels1 map[string]string, labels2 map[string]string) map[string]string {
	var mergedLabels = make(map[string]string)
	for k, v := range labels1 {
		mergedLabels[k] = v
	}
	for k, v := range labels2 {
		mergedLabels[k] = v
	}
	return mergedLabels
}

const (
	DefaultLog4jConfig = `log4j.rootLogger=INFO, console
log4j.logger.akka=INFO
log4j.logger.org.apache.kafka=INFO
log4j.logger.org.apache.hadoop=INFO
log4j.logger.org.apache.zookeeper=INFO
log4j.appender.console=org.apache.log4j.ConsoleAppender
log4j.appender.console.layout=org.apache.log4j.PatternLayout
log4j.appender.console.layout.ConversionPattern=%d{yyyy-MM-dd HH:mm:ss,SSS} %-5p %-60c %x - %m%n
log4j.logger.org.apache.flink.shaded.akka.org.jboss.netty.channel.DefaultChannelPipeline=ERROR, console`
	DefaultLogbackConfig = `<configuration>
    <appender name="console" class="ch.qos.logback.core.ConsoleAppender">
        <encoder>
            <pattern>%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{60} %X{sourceThread} - %msg%n</pattern>
        </encoder>
    </appender>
    <root level="INFO">
        <appender-ref ref="console"/>
    </root>
    <logger name="akka" level="INFO">
        <appender-ref ref="console"/>
    </logger>
    <logger name="org.apache.kafka" level="INFO">
        <appender-ref ref="console"/>
    </logger>
    <logger name="org.apache.hadoop" level="INFO">
        <appender-ref ref="console"/>
    </logger>
    <logger name="org.apache.zookeeper" level="INFO">
        <appender-ref ref="console"/>
    </logger>
    <logger name="org.apache.flink.shaded.akka.org.jboss.netty.channel.DefaultChannelPipeline" level="ERROR">
        <appender-ref ref="console"/>
    </logger>
</configuration>`
)

// TODO: Wouldn't it be better to create a file, put it in an operator image, and read from them?.
// Provide logging profiles
func getLogConf(spec v1beta1.FlinkClusterSpec) map[string]string {
	result := spec.LogConfig
	if result == nil {
		result = make(map[string]string, 2)
	}
	if _, isPresent := result["log4j-console.properties"]; !isPresent {
		result["log4j-console.properties"] = DefaultLog4jConfig
	}
	if _, isPresent := result["log4j-cli.properties"]; !isPresent {
		result["log4j-cli.properties"] = DefaultLog4jConfig
	}
	if _, isPresent := result["logback-console.xml"]; !isPresent {
		result["logback-console.xml"] = DefaultLogbackConfig
	}
	return result
}
