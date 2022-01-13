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

package controllers

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/api/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/controllers/model"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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
	flinkJobPath            = "/opt/flink/job"
)

var (
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
func getDesiredClusterState(observed *ObservedClusterState) model.DesiredClusterState {
	var cluster = observed.cluster

	// The cluster has been deleted, all resources should be cleaned up.
	if cluster == nil {
		return model.DesiredClusterState{}
	}
	return model.DesiredClusterState{
		ConfigMap:     getDesiredConfigMap(cluster),
		JmStatefulSet: getDesiredJobManagerStatefulSet(cluster),
		JmService:     getDesiredJobManagerService(cluster),
		JmIngress:     getDesiredJobManagerIngress(cluster),
		TmStatefulSet: getDesiredTaskManagerStatefulSet(cluster),
		Job:           getDesiredJob(observed),
	}
}

// Gets the desired JobManager StatefulSet spec from the FlinkCluster spec.
func getDesiredJobManagerStatefulSet(
	flinkCluster *v1beta1.FlinkCluster) *appsv1.StatefulSet {

	if shouldCleanup(flinkCluster, "JobManagerStatefulSet") {
		return nil
	}

	var clusterNamespace = flinkCluster.Namespace
	var clusterName = flinkCluster.Name
	var clusterSpec = flinkCluster.Spec
	var imageSpec = clusterSpec.Image
	var serviceAccount = clusterSpec.ServiceAccountName
	var jobManagerSpec = clusterSpec.JobManager
	var rpcPort = corev1.ContainerPort{Name: "rpc", ContainerPort: *jobManagerSpec.Ports.RPC}
	var blobPort = corev1.ContainerPort{Name: "blob", ContainerPort: *jobManagerSpec.Ports.Blob}
	var queryPort = corev1.ContainerPort{Name: "query", ContainerPort: *jobManagerSpec.Ports.Query}
	var uiPort = corev1.ContainerPort{Name: "ui", ContainerPort: *jobManagerSpec.Ports.UI}
	var ports = []corev1.ContainerPort{rpcPort, blobPort, queryPort, uiPort}
	for _, port := range jobManagerSpec.ExtraPorts {
		ports = append(ports, corev1.ContainerPort{Name: port.Name, ContainerPort: port.ContainerPort, Protocol: corev1.Protocol(port.Protocol)})
	}
	var jobManagerStatefulSetName = getJobManagerStatefulSetName(clusterName)
	var podLabels = getComponentLabels(*flinkCluster, "jobmanager")
	podLabels = mergeLabels(podLabels, jobManagerSpec.PodLabels)
	var statefulSetLabels = mergeLabels(podLabels, getRevisionHashLabels(&flinkCluster.Status.Revision))
	var securityContext = jobManagerSpec.SecurityContext
	// Make Volume, VolumeMount to use configMap data for flink-conf.yaml, if flinkProperties is provided.
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	var confVol *corev1.Volume
	var confMount *corev1.VolumeMount
	confVol, confMount = convertFlinkConfig(clusterName)
	volumes = append(jobManagerSpec.Volumes, *confVol)
	volumeMounts = append(jobManagerSpec.VolumeMounts, *confMount)
	var envVars []corev1.EnvVar

	// Hadoop config.
	var hcVolume, hcMount, hcEnv = convertHadoopConfig(clusterSpec.HadoopConfig)
	if hcVolume != nil {
		volumes = append(volumes, *hcVolume)
	}
	if hcMount != nil {
		volumeMounts = append(volumeMounts, *hcMount)
	}
	if hcEnv != nil {
		envVars = append(envVars, *hcEnv)
	}

	// GCP service account config.
	var saVolume, saMount, saEnv = convertGCPConfig(clusterSpec.GCPConfig)
	if saVolume != nil {
		volumes = append(volumes, *saVolume)
	}
	if saMount != nil {
		volumeMounts = append(volumeMounts, *saMount)
	}
	if saEnv != nil {
		envVars = append(envVars, *saEnv)
	}

	envVars = append(envVars, flinkCluster.Spec.EnvVars...)
	var containers = []corev1.Container{{
		Name:            "jobmanager",
		Image:           imageSpec.Name,
		ImagePullPolicy: imageSpec.PullPolicy,
		Args:            []string{"jobmanager"},
		Ports:           ports,
		LivenessProbe:   jobManagerSpec.LivenessProbe,
		ReadinessProbe:  jobManagerSpec.ReadinessProbe,
		Resources:       jobManagerSpec.Resources,
		Env:             envVars,
		EnvFrom:         flinkCluster.Spec.EnvFrom,
		VolumeMounts:    volumeMounts,
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"sleep", strconv.Itoa(preStopSleepSeconds)},
				},
			},
		},
	}}

	containers = append(containers, jobManagerSpec.Sidecars...)

	var podSpec = corev1.PodSpec{
		InitContainers:                convertJobManagerInitContainers(&jobManagerSpec, saMount, saEnv),
		Containers:                    containers,
		Volumes:                       volumes,
		NodeSelector:                  jobManagerSpec.NodeSelector,
		Tolerations:                   jobManagerSpec.Tolerations,
		ImagePullSecrets:              imageSpec.PullSecrets,
		SecurityContext:               securityContext,
		ServiceAccountName:            getServiceAccountName(serviceAccount),
		TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
	}

	var pvcs []corev1.PersistentVolumeClaim
	if jobManagerSpec.VolumeClaimTemplates != nil {
		pvcs = make([]corev1.PersistentVolumeClaim, len(jobManagerSpec.VolumeClaimTemplates))
		for i, pvc := range jobManagerSpec.VolumeClaimTemplates {
			pvc.OwnerReferences = []metav1.OwnerReference{ToOwnerReference(flinkCluster)}
			pvcs[i] = pvc
		}
	}

	var jobManagerStatefulSet = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       clusterNamespace,
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
				Spec: podSpec,
			},
		},
	}
	return jobManagerStatefulSet
}

// Gets the desired JobManager service spec from a cluster spec.
func getDesiredJobManagerService(
	flinkCluster *v1beta1.FlinkCluster) *corev1.Service {

	if shouldCleanup(flinkCluster, "JobManagerService") {
		return nil
	}

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
	var podLabels = getComponentLabels(*flinkCluster, "jobmanager")
	podLabels = mergeLabels(podLabels, jobManagerSpec.PodLabels)
	var serviceLabels = mergeLabels(podLabels, getRevisionHashLabels(&flinkCluster.Status.Revision))
	var jobManagerService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      jobManagerServiceName,
			OwnerReferences: []metav1.OwnerReference{
				ToOwnerReference(flinkCluster)},
			Labels: serviceLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: podLabels,
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
		jobManagerService.Annotations =
			map[string]string{
				"networking.gke.io/load-balancer-type":                         "Internal",
				"networking.gke.io/internal-load-balancer-allow-global-access": "true",
			}
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
func getDesiredJobManagerIngress(
	flinkCluster *v1beta1.FlinkCluster) *networkingv1.Ingress {
	var jobManagerIngressSpec = flinkCluster.Spec.JobManager.Ingress
	if jobManagerIngressSpec == nil {
		return nil
	}

	if shouldCleanup(flinkCluster, "JobManagerIngress") {
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
		getComponentLabels(*flinkCluster, "jobmanager"),
		getRevisionHashLabels(&flinkCluster.Status.Revision))
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
							Path: "/*",
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

// Gets the desired TaskManager StatefulSet spec from a cluster spec.
func getDesiredTaskManagerStatefulSet(
	flinkCluster *v1beta1.FlinkCluster) *appsv1.StatefulSet {

	if shouldCleanup(flinkCluster, "TaskManagerStatefulSet") {
		return nil
	}

	var clusterNamespace = flinkCluster.Namespace
	var clusterName = flinkCluster.Name
	var clusterSpec = flinkCluster.Spec
	var imageSpec = flinkCluster.Spec.Image
	var serviceAccount = clusterSpec.ServiceAccountName
	var taskManagerSpec = flinkCluster.Spec.TaskManager
	var dataPort = corev1.ContainerPort{Name: "data", ContainerPort: *taskManagerSpec.Ports.Data}
	var rpcPort = corev1.ContainerPort{Name: "rpc", ContainerPort: *taskManagerSpec.Ports.RPC}
	var queryPort = corev1.ContainerPort{Name: "query", ContainerPort: *taskManagerSpec.Ports.Query}
	var ports = []corev1.ContainerPort{dataPort, rpcPort, queryPort}
	for _, port := range taskManagerSpec.ExtraPorts {
		ports = append(ports, corev1.ContainerPort{Name: port.Name, ContainerPort: port.ContainerPort, Protocol: corev1.Protocol(port.Protocol)})
	}
	var taskManagerStatefulSetName = getTaskManagerStatefulSetName(clusterName)
	var podLabels = getComponentLabels(*flinkCluster, "taskmanager")
	podLabels = mergeLabels(podLabels, taskManagerSpec.PodLabels)
	var statefulSetLabels = mergeLabels(podLabels, getRevisionHashLabels(&flinkCluster.Status.Revision))

	var securityContext = taskManagerSpec.SecurityContext

	// Make Volume, VolumeMount to use configMap data for flink-conf.yaml
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount

	// Flink config.
	var confVol, confMount = convertFlinkConfig(clusterName)
	volumes = append(taskManagerSpec.Volumes, *confVol)
	volumeMounts = append(taskManagerSpec.VolumeMounts, *confMount)

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

	// Hadoop config.
	var hcVolume, hcMount, hcEnv = convertHadoopConfig(clusterSpec.HadoopConfig)
	if hcVolume != nil {
		volumes = append(volumes, *hcVolume)
	}
	if hcMount != nil {
		volumeMounts = append(volumeMounts, *hcMount)
	}
	if hcEnv != nil {
		envVars = append(envVars, *hcEnv)
	}

	// GCP service account config.
	var saVolume, saMount, saEnv = convertGCPConfig(clusterSpec.GCPConfig)
	if saVolume != nil {
		volumes = append(volumes, *saVolume)
	}
	if saMount != nil {
		volumeMounts = append(volumeMounts, *saMount)
	}
	if saEnv != nil {
		envVars = append(envVars, *saEnv)
	}
	envVars = append(envVars, flinkCluster.Spec.EnvVars...)

	var containers = []corev1.Container{{
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
		VolumeMounts:    volumeMounts,
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"sleep", strconv.Itoa(preStopSleepSeconds)},
				},
			},
		},
	}}
	containers = append(containers, taskManagerSpec.Sidecars...)
	var podSpec = corev1.PodSpec{
		InitContainers:                convertTaskManagerInitContainers(&taskManagerSpec, saMount, saEnv),
		Containers:                    containers,
		Volumes:                       volumes,
		NodeSelector:                  taskManagerSpec.NodeSelector,
		Tolerations:                   taskManagerSpec.Tolerations,
		ImagePullSecrets:              imageSpec.PullSecrets,
		SecurityContext:               securityContext,
		ServiceAccountName:            getServiceAccountName(serviceAccount),
		TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
	}

	var pvcs []corev1.PersistentVolumeClaim
	if taskManagerSpec.VolumeClaimTemplates != nil {
		pvcs = make([]corev1.PersistentVolumeClaim, len(taskManagerSpec.VolumeClaimTemplates))
		for i, pvc := range taskManagerSpec.VolumeClaimTemplates {
			pvc.OwnerReferences = []metav1.OwnerReference{ToOwnerReference(flinkCluster)}
			pvcs[i] = pvc
		}
	}

	var taskManagerStatefulSet = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      taskManagerStatefulSetName,
			OwnerReferences: []metav1.OwnerReference{
				ToOwnerReference(flinkCluster)},
			Labels: statefulSetLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             &taskManagerSpec.Replicas,
			Selector:             &metav1.LabelSelector{MatchLabels: podLabels},
			ServiceName:          taskManagerStatefulSetName,
			VolumeClaimTemplates: pvcs,
			PodManagementPolicy:  "Parallel",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: taskManagerSpec.PodAnnotations,
				},
				Spec: podSpec,
			},
		},
	}
	return taskManagerStatefulSet
}

// Gets the desired configMap.
func getDesiredConfigMap(
	flinkCluster *v1beta1.FlinkCluster) *corev1.ConfigMap {
	appVersion, _ := version.NewVersion(flinkCluster.Spec.FlinkVersion)

	if shouldCleanup(flinkCluster, "ConfigMap") {
		return nil
	}

	var clusterNamespace = flinkCluster.Namespace
	var clusterName = flinkCluster.Name
	var flinkProperties = flinkCluster.Spec.FlinkProperties
	var jmPorts = flinkCluster.Spec.JobManager.Ports
	var tmPorts = flinkCluster.Spec.TaskManager.Ports
	var configMapName = getConfigMapName(clusterName)
	var labels = mergeLabels(
		getClusterLabels(*flinkCluster),
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
			Namespace: clusterNamespace,
			Name:      configMapName,
			OwnerReferences: []metav1.OwnerReference{
				ToOwnerReference(flinkCluster)},
			Labels: labels,
		},
		Data: configData,
	}

	return configMap
}

// Gets the desired job spec from a cluster spec.
func getDesiredJob(observed *ObservedClusterState) *batchv1.Job {
	var flinkCluster = observed.cluster
	var recorded = flinkCluster.Status
	var jobSpec = flinkCluster.Spec.Job
	var jobStatus = recorded.Components.Job

	if jobSpec == nil {
		return nil
	}

	// When the job should be stopped, keep that state unless update is triggered or the job must to be restarted.
	if (shouldStopJob(flinkCluster) || jobStatus.IsStopped()) &&
		(!shouldUpdateJob(observed) && !jobStatus.ShouldRestart(jobSpec)) {
		return nil
	}

	var clusterSpec = flinkCluster.Spec
	var imageSpec = clusterSpec.Image
	var serviceAccount = clusterSpec.ServiceAccountName
	var jobManagerSpec = clusterSpec.JobManager
	var clusterNamespace = flinkCluster.Namespace
	var clusterName = flinkCluster.Name
	var jobName = getJobName(clusterName)
	var jobManagerServiceName = clusterName + "-jobmanager"
	var jobManagerAddress = fmt.Sprintf(
		"%s:%d", jobManagerServiceName, *jobManagerSpec.Ports.UI)
	var podLabels = getClusterLabels(*flinkCluster)
	podLabels = mergeLabels(podLabels, jobManagerSpec.PodLabels)
	var jobLabels = mergeLabels(podLabels, getRevisionHashLabels(&recorded.Revision))
	var jobArgs = []string{"bash", submitJobScriptPath}
	jobArgs = append(jobArgs, "--jobmanager", jobManagerAddress)
	if jobSpec.ClassName != nil {
		jobArgs = append(jobArgs, "--class", *jobSpec.ClassName)
	}

	var fromSavepoint = convertFromSavepoint(jobSpec, jobStatus, &recorded.Revision)
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

	if jobSpec.Mode != nil {
		switch *jobSpec.Mode {
		case v1beta1.JobModeBlocking:
		case v1beta1.JobModeDetached:
			jobArgs = append(jobArgs, "--detached")
		}
	}

	var securityContext = jobSpec.SecurityContext

	var envVars []corev1.EnvVar

	if jobSpec.JarFile != nil {
		jobArgs = append(jobArgs, getLocalPath(&envVars, "FLINK_JOB_JAR_URI", *jobSpec.JarFile))
	}

	if jobSpec.PyFile != nil {
		jobArgs = append(jobArgs, "--python", getLocalPath(&envVars, "FLINK_JOB_JAR_URI", *jobSpec.PyFile))
	}

	if jobSpec.PyFiles != nil {
		jobArgs = append(jobArgs, "--pyFiles", getLocalPath(&envVars, "FLINK_JOB_PYTHON_FILES_URI", *jobSpec.PyFiles))
	}

	if jobSpec.PyModule != nil {
		jobArgs = append(jobArgs, "--pyModule", *jobSpec.PyModule)
	}

	envVars = append(envVars,
		corev1.EnvVar{
			Name:  "FLINK_JM_ADDR",
			Value: jobManagerAddress,
		})

	jobArgs = append(jobArgs, jobSpec.Args...)

	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	volumes = append(volumes, jobSpec.Volumes...)
	volumeMounts = append(volumeMounts, jobSpec.VolumeMounts...)

	// Submit job script config.
	sbsVolume, sbsMount, confMount := convertSubmitJobScript(clusterName)
	volumes = append(volumes, *sbsVolume)
	volumeMounts = append(volumeMounts, *sbsMount, *confMount)

	// Hadoop config.
	var hcVolume, hcMount, hcEnv = convertHadoopConfig(clusterSpec.HadoopConfig)
	if hcVolume != nil {
		volumes = append(volumes, *hcVolume)
	}
	if hcMount != nil {
		volumeMounts = append(volumeMounts, *hcMount)
	}
	if hcEnv != nil {
		envVars = append(envVars, *hcEnv)
	}

	// GCP service account config.
	var saVolume, saMount, saEnv = convertGCPConfig(clusterSpec.GCPConfig)
	if saVolume != nil {
		volumes = append(volumes, *saVolume)
	}
	if saMount != nil {
		volumeMounts = append(volumeMounts, *saMount)
	}
	if saEnv != nil {
		envVars = append(envVars, *saEnv)
	}

	envVars = append(envVars, flinkCluster.Spec.EnvVars...)

	var podSpec = corev1.PodSpec{
		InitContainers: convertJobInitContainers(jobSpec, saMount, saEnv),
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
		SecurityContext:    securityContext,
		ServiceAccountName: getServiceAccountName(serviceAccount),
	}

	// Disable the retry mechanism of k8s Job, all retries should be initiated
	// by the operator based on the job restart policy. This is because Flink
	// jobs are stateful, if a job fails after running for 10 hours, we probably
	// don't want to start over from the beginning, instead we want to resume
	// the job from the latest savepoint which means strictly speaking it is no
	// longer the same job as the previous one because the `--fromSavepoint`
	// parameter has changed.
	var backoffLimit int32 = 0
	var job = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      jobName,
			OwnerReferences: []metav1.OwnerReference{
				ToOwnerReference(flinkCluster)},
			Labels: jobLabels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: jobSpec.PodAnnotations,
				},
				Spec: podSpec,
			},
			BackoffLimit: &backoffLimit,
		},
	}
	return job
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
	// Creating for the first time
	case jobStatus == nil:
		if !isBlank(jobSpec.FromSavepoint) {
			return jobSpec.FromSavepoint
		}
		return nil
	// Updating with FromSavepoint provided
	case revision.IsUpdateTriggered() && !isBlank(jobSpec.FromSavepoint):
		return jobSpec.FromSavepoint
	// Latest savepoint
	case jobStatus.SavepointLocation != "":
		return &jobStatus.SavepointLocation
	// The savepoint from which current job was restored
	case jobStatus.FromSavepoint != "":
		return &jobStatus.FromSavepoint
	}
	return nil
}

// Copy any non-duplicate volume mounts to the specified initContainers
func ensureVolumeMountsInitContainer(initContainers []corev1.Container, volumeMounts []corev1.VolumeMount) []corev1.Container {
	var updatedInitContainers = []corev1.Container{}
	for _, initContainer := range initContainers {
		for _, mounts := range volumeMounts {
			var conflict = false
			for _, mount := range initContainer.VolumeMounts {
				if mounts.MountPath == mount.MountPath {
					conflict = true
					break
				}
			}
			if !conflict {
				initContainer.VolumeMounts =
					append(initContainer.VolumeMounts, mounts)
			}
		}
		updatedInitContainers = append(updatedInitContainers, initContainer)
	}
	return updatedInitContainers
}

func setGSAEnv(initContainers []corev1.Container, saMount *corev1.VolumeMount, saEnv *corev1.EnvVar) []corev1.Container {
	updatedInitContainers := []corev1.Container{}
	for _, initContainer := range initContainers {
		if saEnv != nil {
			initContainer.Env = append(initContainer.Env, *saEnv)
		}
		if saMount != nil {
			initContainer.VolumeMounts = append(initContainer.VolumeMounts, *saMount)
		}
		updatedInitContainers = append(updatedInitContainers, initContainer)
	}
	return updatedInitContainers
}

func convertJobManagerInitContainers(jobManagerSpec *v1beta1.JobManagerSpec, saMount *corev1.VolumeMount, saEnv *corev1.EnvVar) []corev1.Container {
	updatedInitContainers := ensureVolumeMountsInitContainer(jobManagerSpec.InitContainers, jobManagerSpec.VolumeMounts)
	return setGSAEnv(updatedInitContainers, saMount, saEnv)
}

func convertTaskManagerInitContainers(taskSpec *v1beta1.TaskManagerSpec, saMount *corev1.VolumeMount, saEnv *corev1.EnvVar) []corev1.Container {
	updatedInitContainers := ensureVolumeMountsInitContainer(taskSpec.InitContainers, taskSpec.VolumeMounts)
	return setGSAEnv(updatedInitContainers, saMount, saEnv)
}

func convertJobInitContainers(jobSpec *v1beta1.JobSpec, saMount *corev1.VolumeMount, saEnv *corev1.EnvVar) []corev1.Container {
	updatedInitContainers := ensureVolumeMountsInitContainer(jobSpec.InitContainers, jobSpec.VolumeMounts)
	return setGSAEnv(updatedInitContainers, saMount, saEnv)
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
func shouldCleanup(
	cluster *v1beta1.FlinkCluster, component string) bool {
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

	parallelism := cluster.Spec.TaskManager.Replicas * value
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

func convertFlinkConfig(clusterName string) (*corev1.Volume, *corev1.VolumeMount) {
	var confVol *corev1.Volume
	var confMount *corev1.VolumeMount
	confVol = &corev1.Volume{
		Name: flinkConfigMapVolume,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: getConfigMapName(clusterName),
				},
			},
		},
	}
	confMount = &corev1.VolumeMount{
		Name:      flinkConfigMapVolume,
		MountPath: flinkConfigMapPath,
	}
	return confVol, confMount
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
		MountPath: path.Join(flinkConfigMapPath, "flink-conf.yaml"),
		SubPath:   "flink-conf.yaml",
	}
	return confVol, scriptMount, confMount
}

func convertHadoopConfig(hadoopConfig *v1beta1.HadoopConfig) (
	*corev1.Volume, *corev1.VolumeMount, *corev1.EnvVar) {
	if hadoopConfig == nil {
		return nil, nil, nil
	}

	var volume = &corev1.Volume{
		Name: hadoopConfigVolume,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: hadoopConfig.ConfigMapName,
				},
			},
		},
	}
	var mount = &corev1.VolumeMount{
		Name:      hadoopConfigVolume,
		MountPath: hadoopConfig.MountPath,
		ReadOnly:  true,
	}
	var env = &corev1.EnvVar{
		Name:  "HADOOP_CONF_DIR",
		Value: hadoopConfig.MountPath,
	}
	return volume, mount, env
}

func convertGCPConfig(gcpConfig *v1beta1.GCPConfig) (*corev1.Volume, *corev1.VolumeMount, *corev1.EnvVar) {
	if gcpConfig == nil {
		return nil, nil, nil
	}

	var saConfig = gcpConfig.ServiceAccount
	var saVolume = &corev1.Volume{
		Name: gcpServiceAccountVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: gcpConfig.ServiceAccount.SecretName,
			},
		},
	}
	var saMount = &corev1.VolumeMount{
		Name:      gcpServiceAccountVolume,
		MountPath: gcpConfig.ServiceAccount.MountPath,
		ReadOnly:  true,
	}
	if !strings.HasSuffix(saMount.MountPath, "/") {
		saMount.MountPath = saMount.MountPath + "/"
	}
	var saEnv = &corev1.EnvVar{
		Name:  "GOOGLE_APPLICATION_CREDENTIALS",
		Value: saMount.MountPath + saConfig.KeyFile,
	}
	return saVolume, saMount, saEnv
}

func getClusterLabels(cluster v1beta1.FlinkCluster) map[string]string {
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

func getComponentLabels(cluster v1beta1.FlinkCluster, component string) map[string]string {
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

// getLocalPath puts the URI in the env variable and rewrite the path
// to a local path if the file is remote and returns the local path.
// The entrypoint script of the container will download it before submitting it to Flink.
func getLocalPath(envVars *[]corev1.EnvVar, envName string, filePath string) string {
	var localPath = filePath
	if strings.Contains(filePath, "://") {
		var parts = strings.Split(filePath, "/")
		localPath = path.Join(flinkJobPath, parts[len(parts)-1])
		*envVars = append(*envVars, corev1.EnvVar{
			Name:  envName,
			Value: filePath,
		})
	}

	return localPath
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
	if _, isPresent := result["logback-console.xml"]; !isPresent {
		result["logback-console.xml"] = DefaultLogbackConfig
	}
	return result
}
