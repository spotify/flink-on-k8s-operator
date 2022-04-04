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
	"reflect"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGetDesiredClusterState(t *testing.T) {
	var controller = true
	var blockOwnerDeletion = false
	var parallelism int32 = 2
	var jmRPCPort int32 = 6123
	var jmBlobPort int32 = 6124
	var jmQueryPort int32 = 6125
	var jmUIPort int32 = 8081
	var useTLS = true
	var tmDataPort int32 = 6121
	var tmRPCPort int32 = 6122
	var tmQueryPort int32 = 6125
	var replicas int32 = 42
	var tolerationSeconds int64 = 30
	var restartPolicy = v1beta1.JobRestartPolicyFromSavepointOnFailure
	var className = "org.apache.flink.examples.java.wordcount.WordCount"
	var serviceAccount = "default"
	var jarFile = "/cache/my-job.jar"
	var hostFormat = "{{$clusterName}}.example.com"
	var memoryOffHeapRatio int32 = 25
	var memoryOffHeapMin = resource.MustParse("600M")
	var memoryProcessRatio int32 = 80
	var jobMode v1beta1.JobMode = v1beta1.JobModeDetached
	var jobBackoffLimit int32 = 0
	var jmReadinessProbe = corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(jmRPCPort)),
			},
		},
		TimeoutSeconds:      10,
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    60,
	}
	var jmLivenessProbe = corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(jmRPCPort)),
			},
		},
		TimeoutSeconds:      10,
		InitialDelaySeconds: 5,
		PeriodSeconds:       60,
		FailureThreshold:    5,
	}
	var tmReadinessProbe = corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(tmRPCPort)),
			},
		},
		TimeoutSeconds:      10,
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    60,
	}
	var tmLivenessProbe = corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(tmRPCPort)),
			},
		},
		TimeoutSeconds:      10,
		InitialDelaySeconds: 5,
		PeriodSeconds:       60,
		FailureThreshold:    5,
	}
	var tolerations = []corev1.Toleration{
		{
			Key:               "toleration-key",
			Effect:            "toleration-effect",
			Operator:          "toleration-operator",
			TolerationSeconds: &tolerationSeconds,
			Value:             "toleration-value",
		},
		{
			Key:               "toleration-key2",
			Effect:            "toleration-effect2",
			Operator:          "toleration-operator2",
			TolerationSeconds: &tolerationSeconds,
			Value:             "toleration-value2",
		},
	}
	var userAndGroupId int64 = 9999
	var securityContext = corev1.PodSecurityContext{
		RunAsUser:  &userAndGroupId,
		RunAsGroup: &userAndGroupId,
	}
	var ingressPathType = networkingv1.PathTypePrefix

	// Setup.
	storageClassName := "default-class"
	var observed = &ObservedClusterState{
		cluster: &v1beta1.FlinkCluster{
			TypeMeta: metav1.TypeMeta{
				Kind:       "FlinkCluster",
				APIVersion: "flinkoperator.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flinkjobcluster-sample",
				Namespace: "default",
			},
			Spec: v1beta1.FlinkClusterSpec{
				Image:              v1beta1.ImageSpec{Name: "flink:1.8.1"},
				ServiceAccountName: &serviceAccount,
				Job: &v1beta1.JobSpec{
					Args:        []string{"--input", "./README.txt"},
					ClassName:   &className,
					JarFile:     &jarFile,
					Parallelism: &parallelism,
					Resources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
					Mode:          &jobMode,
					RestartPolicy: &restartPolicy,
					Volumes: []corev1.Volume{
						{
							Name: "cache-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "cache-volume", MountPath: "/cache"},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "gcs-downloader",
							Image:   "google/cloud-sdk",
							Command: []string{"gsutil"},
							Args: []string{
								"cp", "gs://my-bucket/my-job.jar", "/cache/my-job.jar",
							},
						},
					},
					PodAnnotations: map[string]string{
						"example.com": "example",
					},
					SecurityContext: &securityContext,
				},
				JobManager: &v1beta1.JobManagerSpec{
					AccessScope: v1beta1.AccessScopeVPC,
					Ingress: &v1beta1.JobManagerIngressSpec{
						HostFormat: &hostFormat,
						Annotations: map[string]string{
							"kubernetes.io/ingress.class":                "nginx",
							"certmanager.k8s.io/cluster-issuer":          "letsencrypt-stg",
							"nginx.ingress.kubernetes.io/rewrite-target": "/",
						},
						UseTLS: &useTLS,
					},
					Ports: v1beta1.JobManagerPorts{
						RPC:   &jmRPCPort,
						Blob:  &jmBlobPort,
						Query: &jmQueryPort,
						UI:    &jmUIPort,
					},
					LivenessProbe:  &jmLivenessProbe,
					ReadinessProbe: &jmReadinessProbe,
					Resources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
					Tolerations:        tolerations,
					MemoryOffHeapRatio: &memoryOffHeapRatio,
					MemoryOffHeapMin:   memoryOffHeapMin,
					MemoryProcessRatio: &memoryProcessRatio,
					PodAnnotations: map[string]string{
						"example.com": "example",
					},
					SecurityContext: &securityContext,
				},
				TaskManager: &v1beta1.TaskManagerSpec{
					Replicas: &replicas,
					Ports: v1beta1.TaskManagerPorts{
						Data:  &tmDataPort,
						RPC:   &tmRPCPort,
						Query: &tmQueryPort,
					},
					Resources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
					LivenessProbe:      &tmLivenessProbe,
					ReadinessProbe:     &tmReadinessProbe,
					MemoryOffHeapRatio: &memoryOffHeapRatio,
					MemoryOffHeapMin:   memoryOffHeapMin,
					MemoryProcessRatio: &memoryProcessRatio,
					Sidecars:           []corev1.Container{{Name: "sidecar", Image: "alpine"}},
					Volumes: []corev1.Volume{
						{
							Name: "cache-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "cache-volume", MountPath: "/cache"},
					},
					VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-test",
								OwnerReferences: []metav1.OwnerReference{
									{
										APIVersion:         "flinkoperator.k8s.io/v1beta1",
										Kind:               "FlinkCluster",
										Name:               "flinkjobcluster-sample",
										Controller:         &controller,
										BlockOwnerDeletion: &blockOwnerDeletion,
									},
								},
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
								Resources: corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										corev1.ResourceStorage: resource.MustParse("100Gi"),
									},
								},
								StorageClassName: &storageClassName,
							},
						},
					},
					Tolerations: tolerations,
					PodAnnotations: map[string]string{
						"example.com": "example",
					},
					SecurityContext: &securityContext,
				},
				EnvVars: []corev1.EnvVar{{Name: "FOO", Value: "abc"}},
				EnvFrom: []corev1.EnvFromSource{{ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "FOOMAP",
					}}}},
				HadoopConfig: &v1beta1.HadoopConfig{
					ConfigMapName: "hadoop-configmap",
					MountPath:     "/etc/hadoop/conf",
				},
				GCPConfig: &v1beta1.GCPConfig{
					ServiceAccount: &v1beta1.GCPServiceAccount{
						SecretName: "gcp-service-account-secret",
						KeyFile:    "gcp_service_account_key.json",
						MountPath:  "/etc/gcp_service_account/",
					},
				},
				LogConfig: map[string]string{
					"extra-file.txt":           "hello!",
					"log4j-console.properties": "foo",
					"logback-console.xml":      "bar",
					"log4j-cli.properties":     "baz",
				},
			},
			Status: v1beta1.FlinkClusterStatus{
				Revision: v1beta1.RevisionStatus{NextRevision: "flinkjobcluster-sample-85dc8f749-1"},
			},
		},
	}

	// Run.
	var desiredState = getDesiredClusterState(observed)

	// Verify.

	// JmStatefulSet
	var expectedDesiredJmStatefulSet = appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flinkjobcluster-sample-jobmanager",
			Namespace: "default",
			Labels: map[string]string{
				"app":             "flink",
				"cluster":         "flinkjobcluster-sample",
				"component":       "jobmanager",
				RevisionNameLabel: "flinkjobcluster-sample-85dc8f749",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "flinkoperator.k8s.io/v1beta1",
					Kind:               "FlinkCluster",
					Name:               "flinkjobcluster-sample",
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "flink",
					"cluster":   "flinkjobcluster-sample",
					"component": "jobmanager",
				},
			},
			ServiceName: "flinkjobcluster-sample-jobmanager",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "flink",
						"cluster":   "flinkjobcluster-sample",
						"component": "jobmanager",
					},
					Annotations: map[string]string{
						"example.com": "example",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: make([]corev1.Container, 0),
					Containers: []corev1.Container{
						{
							Name:  "jobmanager",
							Image: "flink:1.8.1",
							Args:  []string{"jobmanager"},
							Ports: []corev1.ContainerPort{
								{Name: "rpc", ContainerPort: jmRPCPort},
								{Name: "blob", ContainerPort: jmBlobPort},
								{Name: "query", ContainerPort: jmQueryPort},
								{Name: "ui", ContainerPort: jmUIPort},
							},
							LivenessProbe:  &jmLivenessProbe,
							ReadinessProbe: &jmReadinessProbe,
							Env: []corev1.EnvVar{
								{
									Name:  "FOO",
									Value: "abc",
								},
								{Name: "HADOOP_CONF_DIR", Value: "/etc/hadoop/conf"},
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/etc/gcp_service_account/gcp_service_account_key.json",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "FOOMAP",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "flink-config-volume",
									MountPath: "/opt/flink/conf",
								},
								{
									Name:      "hadoop-config-volume",
									MountPath: "/etc/hadoop/conf",
									ReadOnly:  true,
								},
								{
									Name:      "gcp-service-account-volume",
									MountPath: "/etc/gcp_service_account/",
									ReadOnly:  true,
								},
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"sleep", strconv.Itoa(preStopSleepSeconds)},
									},
								},
							},
						},
					},
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Tolerations:                   tolerations,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &userAndGroupId,
						RunAsGroup: &userAndGroupId,
					},
					ServiceAccountName: serviceAccount,
					Volumes: []corev1.Volume{
						{
							Name: "flink-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "flinkjobcluster-sample-configmap",
									},
								},
							},
						},
						{
							Name: "hadoop-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "hadoop-configmap",
									},
								},
							},
						},
						{
							Name: "gcp-service-account-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "gcp-service-account-secret",
								},
							},
						},
					},
				},
			},
		},
	}

	assert.Assert(t, desiredState.JmStatefulSet != nil)
	assert.DeepEqual(
		t,
		*desiredState.JmStatefulSet,
		expectedDesiredJmStatefulSet,
		cmpopts.IgnoreUnexported(resource.Quantity{}))

	// JmService
	var expectedDesiredJmService = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flinkjobcluster-sample-jobmanager",
			Namespace: "default",
			Labels: map[string]string{
				RevisionNameLabel: "flinkjobcluster-sample-85dc8f749",
				"app":             "flink",
				"cluster":         "flinkjobcluster-sample",
				"component":       "jobmanager",
			},
			Annotations: map[string]string{
				"networking.gke.io/load-balancer-type":                         "Internal",
				"networking.gke.io/internal-load-balancer-allow-global-access": "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "flinkoperator.k8s.io/v1beta1",
					Kind:               "FlinkCluster",
					Name:               "flinkjobcluster-sample",
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type: "LoadBalancer",
			Selector: map[string]string{
				"app":       "flink",
				"cluster":   "flinkjobcluster-sample",
				"component": "jobmanager",
			},
			Ports: []corev1.ServicePort{
				{Name: "rpc", Port: 6123, TargetPort: intstr.FromString("rpc")},
				{Name: "blob", Port: 6124, TargetPort: intstr.FromString("blob")},
				{Name: "query", Port: 6125, TargetPort: intstr.FromString("query")},
				{Name: "ui", Port: 8081, TargetPort: intstr.FromString("ui")},
			},
		},
	}
	assert.Assert(t, desiredState.JmService != nil)
	assert.DeepEqual(
		t,
		*desiredState.JmService,
		expectedDesiredJmService)

	// JmIngress
	var expectedDesiredJmIngress = networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flinkjobcluster-sample-jobmanager",
			Namespace: "default",
			Labels: map[string]string{
				"app":             "flink",
				"cluster":         "flinkjobcluster-sample",
				"component":       "jobmanager",
				RevisionNameLabel: "flinkjobcluster-sample-85dc8f749",
			},
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                "nginx",
				"certmanager.k8s.io/cluster-issuer":          "letsencrypt-stg",
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "flinkoperator.k8s.io/v1beta1",
					Kind:               "FlinkCluster",
					Name:               "flinkjobcluster-sample",
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "flinkjobcluster-sample.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &ingressPathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "flinkjobcluster-sample-jobmanager",
									Port: networkingv1.ServiceBackendPort{
										Name:   "flinkjobcluster-sample-jobmanager",
										Number: intstr.FromString("ui").IntVal,
									},
								},
							}},
						}},
				},
			}},
			TLS: []networkingv1.IngressTLS{{
				Hosts: []string{"flinkjobcluster-sample.example.com"},
			}},
		},
	}

	assert.Assert(t, desiredState.JmIngress != nil)
	assert.DeepEqual(
		t,
		*desiredState.JmIngress,
		expectedDesiredJmIngress)

	// TmStatefulSet
	var expectedDesiredTmStatefulSet = appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flinkjobcluster-sample-taskmanager",
			Namespace: "default",
			Labels: map[string]string{
				"app":             "flink",
				"cluster":         "flinkjobcluster-sample",
				"component":       "taskmanager",
				RevisionNameLabel: "flinkjobcluster-sample-85dc8f749",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "flinkoperator.k8s.io/v1beta1",
					Kind:               "FlinkCluster",
					Name:               "flinkjobcluster-sample",
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			ServiceName:         "flinkjobcluster-sample-taskmanager",
			PodManagementPolicy: "Parallel",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "flink",
					"cluster":   "flinkjobcluster-sample",
					"component": "taskmanager",
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pvc-test",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "flinkoperator.k8s.io/v1beta1",
								Kind:               "FlinkCluster",
								Name:               "flinkjobcluster-sample",
								Controller:         &controller,
								BlockOwnerDeletion: &blockOwnerDeletion,
							},
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: map[corev1.ResourceName]resource.Quantity{
								corev1.ResourceStorage: resource.MustParse("100Gi"),
							},
						},
						StorageClassName: &storageClassName,
					},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "flink",
						"cluster":   "flinkjobcluster-sample",
						"component": "taskmanager",
					},
					Annotations: map[string]string{
						"example.com": "example",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: make([]corev1.Container, 0),
					Containers: []corev1.Container{
						{
							Name:  "taskmanager",
							Image: "flink:1.8.1",
							Args:  []string{"taskmanager"},
							Ports: []corev1.ContainerPort{
								{Name: "data", ContainerPort: 6121},
								{Name: "rpc", ContainerPort: 6122},
								{Name: "query", ContainerPort: 6125},
							},
							LivenessProbe:  &tmLivenessProbe,
							ReadinessProbe: &tmReadinessProbe,
							Env: []corev1.EnvVar{
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
								{
									Name:  "FOO",
									Value: "abc",
								},
								{Name: "HADOOP_CONF_DIR", Value: "/etc/hadoop/conf"},
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/etc/gcp_service_account/gcp_service_account_key.json",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "FOOMAP",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "cache-volume", MountPath: "/cache"},
								{Name: "flink-config-volume", MountPath: "/opt/flink/conf"},
								{
									Name:      "hadoop-config-volume",
									MountPath: "/etc/hadoop/conf",
									ReadOnly:  true,
								},
								{
									Name:      "gcp-service-account-volume",
									MountPath: "/etc/gcp_service_account/",
									ReadOnly:  true,
								},
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"sleep", strconv.Itoa(preStopSleepSeconds)},
									},
								},
							},
						},
						{Name: "sidecar", Image: "alpine"},
					},
					Volumes: []corev1.Volume{
						{
							Name: "cache-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "flink-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "flinkjobcluster-sample-configmap",
									},
								},
							},
						},
						{
							Name: "hadoop-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "hadoop-configmap",
									},
								},
							},
						},
						{
							Name: "gcp-service-account-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "gcp-service-account-secret",
								},
							},
						},
					},
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Tolerations:                   tolerations,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &userAndGroupId,
						RunAsGroup: &userAndGroupId,
					},
					ServiceAccountName: serviceAccount,
				},
			},
		},
	}

	assert.Assert(t, desiredState.TmStatefulSet != nil)
	assert.DeepEqual(
		t,
		*desiredState.TmStatefulSet,
		expectedDesiredTmStatefulSet,
		cmpopts.IgnoreUnexported(resource.Quantity{}))

	// Job
	var expectedDesiredJob = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flinkjobcluster-sample-job-submitter",
			Namespace: "default",
			Labels: map[string]string{
				"app":             "flink",
				"cluster":         "flinkjobcluster-sample",
				RevisionNameLabel: "flinkjobcluster-sample-85dc8f749",
			},
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "flinkoperator.k8s.io/v1beta1",
					Kind:               "FlinkCluster",
					Name:               "flinkjobcluster-sample",
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                                "flink",
						"cluster":                            "flinkjobcluster-sample",
						"flinkoperator.k8s.io/revision-name": "flinkjobcluster-sample-85dc8f749",
					},
					Annotations: map[string]string{
						"example.com": "example",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:    "gcs-downloader",
							Image:   "google/cloud-sdk",
							Command: []string{"gsutil"},
							Args: []string{
								"cp", "gs://my-bucket/my-job.jar", "/cache/my-job.jar",
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "cache-volume", MountPath: "/cache"},
								{
									Name:      "flink-config-volume",
									MountPath: "/opt/flink-operator/submit-job.sh",
									SubPath:   "submit-job.sh",
								},
								{Name: "flink-config-volume", MountPath: "/opt/flink/conf"},
								{Name: "hadoop-config-volume", ReadOnly: true, MountPath: "/etc/hadoop/conf"},
								{Name: "gcp-service-account-volume", ReadOnly: true, MountPath: "/etc/gcp_service_account/"},
							},
							Env: []corev1.EnvVar{
								{Name: "FLINK_JM_ADDR", Value: "flinkjobcluster-sample-jobmanager:8081"},
								{Name: "FOO", Value: "abc"},
								{Name: "HADOOP_CONF_DIR", Value: "/etc/hadoop/conf"},
								{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/etc/gcp_service_account/gcp_service_account_key.json"},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "main",
							Image: "flink:1.8.1",
							Args: []string{
								"bash",
								"/opt/flink-operator/submit-job.sh",
								"--jobmanager",
								"flinkjobcluster-sample-jobmanager:8081",
								"--class",
								"org.apache.flink.examples.java.wordcount.WordCount",
								"--parallelism",
								"2",
								"--detached",
								"/cache/my-job.jar",
								"--input",
								"./README.txt",
							},
							Env: []corev1.EnvVar{
								{Name: "FLINK_JM_ADDR", Value: "flinkjobcluster-sample-jobmanager:8081"},
								{Name: "FOO", Value: "abc"},
								{Name: "HADOOP_CONF_DIR", Value: "/etc/hadoop/conf"},
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/etc/gcp_service_account/gcp_service_account_key.json",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "FOOMAP",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "cache-volume", MountPath: "/cache"},
								{
									Name:      "flink-config-volume",
									MountPath: "/opt/flink-operator/submit-job.sh",
									SubPath:   "submit-job.sh",
								},
								{
									Name:      "flink-config-volume",
									MountPath: "/opt/flink/conf",
								},
								{
									Name:      "hadoop-config-volume",
									MountPath: "/etc/hadoop/conf",
									ReadOnly:  true,
								},
								{
									Name:      "gcp-service-account-volume",
									MountPath: "/etc/gcp_service_account/",
									ReadOnly:  true,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: "cache-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "flink-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "flinkjobcluster-sample-configmap",
									},
								},
							},
						},
						{
							Name: "hadoop-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "hadoop-configmap",
									},
								},
							},
						},
						{
							Name: "gcp-service-account-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "gcp-service-account-secret",
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &userAndGroupId,
						RunAsGroup: &userAndGroupId,
					},
					ServiceAccountName: serviceAccount,
				},
			},
			BackoffLimit: &jobBackoffLimit,
		},
	}

	assert.Assert(t, desiredState.Job != nil)
	assert.DeepEqual(
		t,
		*desiredState.Job,
		expectedDesiredJob)

	// ConfigMap
	var flinkConfYaml = `blob.server.port: 6124
jobmanager.rpc.address: flinkjobcluster-sample-jobmanager
jobmanager.rpc.port: 6123
query.server.port: 6125
rest.port: 8081
taskmanager.heap.size: 452m
taskmanager.numberOfTaskSlots: 1
taskmanager.rpc.port: 6122
`
	var expectedConfigMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flinkjobcluster-sample-configmap",
			Namespace: "default",
			Labels: map[string]string{
				"app":             "flink",
				"cluster":         "flinkjobcluster-sample",
				RevisionNameLabel: "flinkjobcluster-sample-85dc8f749",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "flinkoperator.k8s.io/v1beta1",
					Kind:               "FlinkCluster",
					Name:               "flinkjobcluster-sample",
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Data: map[string]string{
			"flink-conf.yaml":          flinkConfYaml,
			"extra-file.txt":           "hello!",
			"log4j-console.properties": "foo",
			"log4j-cli.properties":     "baz",
			"logback-console.xml":      "bar",
			"submit-job.sh":            submitJobScript,
		},
	}
	assert.Assert(t, desiredState.ConfigMap != nil)
	assert.DeepEqual(
		t,
		*desiredState.ConfigMap,
		expectedConfigMap)
}

func TestSecurityContext(t *testing.T) {
	var jmRPCPort int32 = 6123
	var jmBlobPort int32 = 6124
	var jmQueryPort int32 = 6125
	var jmUIPort int32 = 8081
	var tmDataPort int32 = 6121
	var tmRPCPort int32 = 6122
	var tmQueryPort int32 = 6125
	var tmReplicas int32 = v1beta1.DefaultTaskManagerReplicas

	var userAndGroupId int64 = 9999
	var securityContext = corev1.PodSecurityContext{
		RunAsUser:  &userAndGroupId,
		RunAsGroup: &userAndGroupId,
	}
	var jobMode v1beta1.JobMode = v1beta1.JobModeDetached

	// Provided security context
	var observed = &ObservedClusterState{
		cluster: &v1beta1.FlinkCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flinkjobcluster-sample",
				Namespace: "default",
			},
			Spec: v1beta1.FlinkClusterSpec{
				Job: &v1beta1.JobSpec{
					Mode:            &jobMode,
					SecurityContext: &securityContext,
				},
				JobManager: &v1beta1.JobManagerSpec{
					AccessScope: v1beta1.AccessScopeVPC,
					Ports: v1beta1.JobManagerPorts{
						RPC:   &jmRPCPort,
						Blob:  &jmBlobPort,
						Query: &jmQueryPort,
						UI:    &jmUIPort,
					},
					SecurityContext: &securityContext,
				},
				TaskManager: &v1beta1.TaskManagerSpec{
					Replicas: &tmReplicas,
					Ports: v1beta1.TaskManagerPorts{
						Data:  &tmDataPort,
						RPC:   &tmRPCPort,
						Query: &tmQueryPort,
					},
					SecurityContext: &securityContext,
				},
			},
			Status: v1beta1.FlinkClusterStatus{
				Revision: v1beta1.RevisionStatus{NextRevision: "flinkjobcluster-sample-85dc8f749-1"},
			},
		},
	}

	var desired = getDesiredClusterState(observed)

	assert.DeepEqual(t, desired.Job.Spec.Template.Spec.SecurityContext, &securityContext)
	assert.DeepEqual(t, desired.JmStatefulSet.Spec.Template.Spec.SecurityContext, &securityContext)
	assert.DeepEqual(t, desired.TmStatefulSet.Spec.Template.Spec.SecurityContext, &securityContext)

	// No security context
	var observed2 = &ObservedClusterState{
		cluster: &v1beta1.FlinkCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flinkjobcluster-sample",
				Namespace: "default",
			},
			Spec: v1beta1.FlinkClusterSpec{
				Job: &v1beta1.JobSpec{Mode: &jobMode},
				JobManager: &v1beta1.JobManagerSpec{
					AccessScope: v1beta1.AccessScopeVPC,
					Ports: v1beta1.JobManagerPorts{
						RPC:   &jmRPCPort,
						Blob:  &jmBlobPort,
						Query: &jmQueryPort,
						UI:    &jmUIPort,
					},
				},
				TaskManager: &v1beta1.TaskManagerSpec{
					Replicas: &tmReplicas,
					Ports: v1beta1.TaskManagerPorts{
						Data:  &tmDataPort,
						RPC:   &tmRPCPort,
						Query: &tmQueryPort,
					},
				},
			},
			Status: v1beta1.FlinkClusterStatus{
				Revision: v1beta1.RevisionStatus{NextRevision: "flinkjobcluster-sample-85dc8f749-1"},
			},
		},
	}

	var desired2 = getDesiredClusterState(observed2)

	assert.Assert(t, desired2.Job.Spec.Template.Spec.SecurityContext == nil)
	assert.Assert(t, desired2.JmStatefulSet.Spec.Template.Spec.SecurityContext == nil)
	assert.Assert(t, desired2.TmStatefulSet.Spec.Template.Spec.SecurityContext == nil)
}

func TestCalFlinkHeapSize(t *testing.T) {
	var memoryOffHeapRatio int32 = 25
	var memoryOffHeapMin = resource.MustParse("600M")

	// Case 1: Heap sizes are computed from memoryOffHeapMin or memoryOffHeapRatio
	cluster := &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mycluster",
			Namespace: "default",
		},
		Spec: v1beta1.FlinkClusterSpec{
			JobManager: &v1beta1.JobManagerSpec{
				Resources: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
				MemoryOffHeapRatio: &memoryOffHeapRatio,
				MemoryOffHeapMin:   memoryOffHeapMin,
			},
			TaskManager: &v1beta1.TaskManagerSpec{
				Resources: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
				MemoryOffHeapRatio: &memoryOffHeapRatio,
				MemoryOffHeapMin:   memoryOffHeapMin,
			},
		},
	}

	flinkHeapSize := calFlinkHeapSize(cluster)
	expectedFlinkHeapSize := map[string]string{
		"jobmanager.heap.size":  "452m",  // get values calculated with limit - memoryOffHeapMin
		"taskmanager.heap.size": "3072m", // get values calculated with limit - limit * memoryOffHeapRatio / 100
	}
	assert.Assert(t, len(flinkHeapSize) == 2)
	assert.DeepEqual(
		t,
		flinkHeapSize,
		expectedFlinkHeapSize)

	// Case 2: No values when memory limits are missing or insufficient
	cluster = &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mycluster",
			Namespace: "default",
		},
		Spec: v1beta1.FlinkClusterSpec{
			JobManager: &v1beta1.JobManagerSpec{
				Resources: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceMemory: resource.MustParse("500Mi"),
					},
				},
				MemoryOffHeapRatio: &memoryOffHeapRatio,
				MemoryOffHeapMin:   memoryOffHeapMin,
			},
			TaskManager: &v1beta1.TaskManagerSpec{
				MemoryOffHeapRatio: &memoryOffHeapRatio,
				MemoryOffHeapMin:   memoryOffHeapMin,
			},
		},
	}

	flinkHeapSize = calFlinkHeapSize(cluster)
	assert.Assert(t, len(flinkHeapSize) == 0)
}

func TestCalFlinkMemoryProcessSize(t *testing.T) {
	var memoryProcessRatio int32 = 80

	// Case 1: memory size is computed from memoryProcessRatio
	cluster := &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mycluster",
			Namespace: "default",
		},
		Spec: v1beta1.FlinkClusterSpec{
			JobManager: &v1beta1.JobManagerSpec{
				Resources: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
				MemoryProcessRatio: &memoryProcessRatio,
			},
			TaskManager: &v1beta1.TaskManagerSpec{
				Resources: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
				MemoryProcessRatio: &memoryProcessRatio,
			},
		},
	}

	flinkHeapSize := calFlinkMemoryProcessSize(cluster)
	expectedFlinkHeapSize := map[string]string{
		"jobmanager.memory.process.size":  "820m",
		"taskmanager.memory.process.size": "3277m",
	}
	assert.Assert(t, len(flinkHeapSize) == 2)
	assert.DeepEqual(
		t,
		flinkHeapSize,
		expectedFlinkHeapSize)

	// Case 2: No values when memory limits are missing or insufficient
	cluster = &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mycluster",
			Namespace: "default",
		},
		Spec: v1beta1.FlinkClusterSpec{
			JobManager: &v1beta1.JobManagerSpec{
				Resources: corev1.ResourceRequirements{
					Limits: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceMemory: resource.MustParse("500Mi"),
					},
				},
				MemoryProcessRatio: &memoryProcessRatio,
			},
			TaskManager: &v1beta1.TaskManagerSpec{
				MemoryProcessRatio: &memoryProcessRatio,
			},
		},
	}

	flinkHeapSize = calFlinkMemoryProcessSize(cluster)
	assert.Assert(t, len(flinkHeapSize) == 1)
}

func Test_getLogConf(t *testing.T) {
	type args struct {
		spec v1beta1.FlinkClusterSpec
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "nil map uses defaults",
			args: args{v1beta1.FlinkClusterSpec{LogConfig: nil}},
			want: map[string]string{
				"log4j-console.properties": DefaultLog4jConfig,
				"log4j-cli.properties":     DefaultLog4jConfig,
				"logback-console.xml":      DefaultLogbackConfig,
			},
		},
		{
			name: "map missing log4j-console.properties uses default",
			args: args{v1beta1.FlinkClusterSpec{LogConfig: map[string]string{
				"logback-console.xml": "xyz",
			}}},
			want: map[string]string{
				"log4j-console.properties": DefaultLog4jConfig,
				"log4j-cli.properties":     DefaultLog4jConfig,
				"logback-console.xml":      "xyz",
			},
		},
		{
			name: "map missing logback-console.xml uses default",
			args: args{v1beta1.FlinkClusterSpec{LogConfig: map[string]string{
				"log4j-console.properties": "xyz",
			}}},
			want: map[string]string{
				"log4j-console.properties": "xyz",
				"log4j-cli.properties":     DefaultLog4jConfig,
				"logback-console.xml":      DefaultLogbackConfig,
			},
		},
		{
			name: "map missing log4j-cli.properties uses default",
			args: args{v1beta1.FlinkClusterSpec{LogConfig: map[string]string{
				"log4j-console.properties": "xyz",
				"logback-console.xml":      "xyz",
			}}},
			want: map[string]string{
				"log4j-console.properties": "xyz",
				"log4j-cli.properties":     DefaultLog4jConfig,
				"logback-console.xml":      "xyz",
			},
		},
		{
			name: "map with all three keys overrides defaults",
			args: args{v1beta1.FlinkClusterSpec{LogConfig: map[string]string{
				"log4j-console.properties": "hello",
				"log4j-cli.properties":     "xyz",
				"logback-console.xml":      "world",
			}}},
			want: map[string]string{
				"log4j-console.properties": "hello",
				"logback-console.xml":      "world",
				"log4j-cli.properties":     "xyz",
			},
		},
		{
			name: "extra keys preserved",
			args: args{v1beta1.FlinkClusterSpec{LogConfig: map[string]string{
				"log4j-console.properties": "abc",
				"file.txt":                 "def",
			}}},
			want: map[string]string{
				"log4j-console.properties": "abc",
				"log4j-cli.properties":     DefaultLog4jConfig,
				"logback-console.xml":      DefaultLogbackConfig,
				"file.txt":                 "def",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getLogConf(tt.args.spec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLogConf() = %v, want %v", got, tt.want)
			}
		})
	}
}
