/*
Copyright 2020 Google LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by statelicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package volcano

import (
	"testing"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spotify/flink-on-k8s-operator/internal/model"
)

var (
	jmRep    = int32(1)
	replicas = int32(4)
)

func getDesiredState() *model.DesiredClusterState {
	return &model.DesiredClusterState{
		JmStatefulSet: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "flinkjobcluster-sample-jobmanager",
				Namespace: "default",
				Labels: map[string]string{
					"app":       "flink",
					"cluster":   "flinkjobcluster-sample",
					"component": "jobmanager",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas:    &jmRep,
				ServiceName: "flinkjobcluster-sample-jobmanager",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":       "flink",
						"cluster":   "flinkjobcluster-sample",
						"component": "jobmanager",
					},
				},
				Template: v1.PodTemplateSpec{
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
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Name:  "jobmanager",
								Image: "flink:1.8.1",
								Args:  []string{"jobmanager"},
								Env: []v1.EnvVar{
									{Name: "HADOOP_CONF_DIR", Value: "/etc/hadoop/conf"},
									{
										Name:  "GOOGLE_APPLICATION_CREDENTIALS",
										Value: "/etc/gcp_service_account/gcp_service_account_key.json",
									},
									{
										Name:  "FOO",
										Value: "abc",
									},
								},
								Resources: v1.ResourceRequirements{
									Requests: map[v1.ResourceName]resource.Quantity{
										v1.ResourceCPU:    resource.MustParse("100m"),
										v1.ResourceMemory: resource.MustParse("256Mi"),
									},
									Limits: map[v1.ResourceName]resource.Quantity{
										v1.ResourceCPU:    resource.MustParse("200m"),
										v1.ResourceMemory: resource.MustParse("512Mi"),
									},
								},
								VolumeMounts: []v1.VolumeMount{
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
						Volumes: []v1.Volume{
							{
								Name: "flink-config-volume",
								VolumeSource: v1.VolumeSource{
									ConfigMap: &v1.ConfigMapVolumeSource{
										LocalObjectReference: v1.LocalObjectReference{
											Name: "flinkjobcluster-sample-configmap",
										},
									},
								},
							},
							{
								Name: "hadoop-config-volume",
								VolumeSource: v1.VolumeSource{
									ConfigMap: &v1.ConfigMapVolumeSource{
										LocalObjectReference: v1.LocalObjectReference{
											Name: "hadoop-configmap",
										},
									},
								},
							},
							{
								Name: "gcp-service-account-volume",
								VolumeSource: v1.VolumeSource{
									Secret: &v1.SecretVolumeSource{
										SecretName: "gcp-service-account-secret",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestGetClusterResource(t *testing.T) {
	desiredState := getDesiredState()
	desiredState.TmStatefulSet = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flinkjobcluster-sample-taskmanager",
			Namespace: "default",
			Labels: map[string]string{
				"app":       "flink",
				"cluster":   "flinkjobcluster-sample",
				"component": "taskmanager",
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
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{{
				Spec: v1.PersistentVolumeClaimSpec{
					VolumeName: "tm-claim",
					Resources: v1.ResourceRequirements{
						Requests: map[v1.ResourceName]resource.Quantity{
							v1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			}},
			Template: v1.PodTemplateSpec{
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
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  "taskmanager",
							Image: "flink:1.8.1",
							Args:  []string{"taskmanager"},
							Ports: []v1.ContainerPort{
								{Name: "data", ContainerPort: 6121},
								{Name: "rpc", ContainerPort: 6122},
								{Name: "query", ContainerPort: 6125},
							},
							Env: []v1.EnvVar{
								{
									Name: "TASK_MANAGER_CPU_LIMIT",
									ValueFrom: &v1.EnvVarSource{
										ResourceFieldRef: &v1.ResourceFieldSelector{
											ContainerName: "taskmanager",
											Resource:      "limits.cpu",
											Divisor:       resource.MustParse("1m"),
										},
									},
								},
								{
									Name: "TASK_MANAGER_MEMORY_LIMIT",
									ValueFrom: &v1.EnvVarSource{
										ResourceFieldRef: &v1.ResourceFieldSelector{
											ContainerName: "taskmanager",
											Resource:      "limits.memory",
											Divisor:       resource.MustParse("1Mi"),
										},
									},
								},
								{Name: "HADOOP_CONF_DIR", Value: "/etc/hadoop/conf"},
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/etc/gcp_service_account/gcp_service_account_key.json",
								},
								{
									Name:  "FOO",
									Value: "abc",
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("200m"),
									v1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("500m"),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
							VolumeMounts: []v1.VolumeMount{
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
						},
						v1.Container{Name: "sidecar", Image: "alpine"},
					},
					Volumes: []v1.Volume{
						{
							Name: "cache-volume",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "flink-config-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "flinkjobcluster-sample-configmap",
									},
								},
							},
						},
						{
							Name: "hadoop-config-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "hadoop-configmap",
									},
								},
							},
						},
						{
							Name: "gcp-service-account-volume",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "gcp-service-account-secret",
								},
							},
						},
					},
				},
			},
		},
	}

	res, size := getClusterResourceList(desiredState)
	assert.Assert(t, size == 5)
	assert.Equal(t, res.Memory().String(), "4608Mi")
	assert.Equal(t, res.Cpu().MilliValue(), int64(2200))
	assert.Equal(t, res.Storage().String(), "400Gi")
}

func TestGetClusterResourceForDeployment(t *testing.T) {
	desiredState := getDesiredState()
	desiredState.TmDeployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flinkjobcluster-sample-taskmanager",
			Namespace: "default",
			Labels: map[string]string{
				"app":       "flink",
				"cluster":   "flinkjobcluster-sample",
				"component": "taskmanager",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":       "flink",
					"cluster":   "flinkjobcluster-sample",
					"component": "taskmanager",
				},
			},
			Template: v1.PodTemplateSpec{
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
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  "taskmanager",
							Image: "flink:1.8.1",
							Args:  []string{"taskmanager"},
							Ports: []v1.ContainerPort{
								{Name: "data", ContainerPort: 6121},
								{Name: "rpc", ContainerPort: 6122},
								{Name: "query", ContainerPort: 6125},
							},
							Env: []v1.EnvVar{
								{
									Name: "TASK_MANAGER_CPU_LIMIT",
									ValueFrom: &v1.EnvVarSource{
										ResourceFieldRef: &v1.ResourceFieldSelector{
											ContainerName: "taskmanager",
											Resource:      "limits.cpu",
											Divisor:       resource.MustParse("1m"),
										},
									},
								},
								{
									Name: "TASK_MANAGER_MEMORY_LIMIT",
									ValueFrom: &v1.EnvVarSource{
										ResourceFieldRef: &v1.ResourceFieldSelector{
											ContainerName: "taskmanager",
											Resource:      "limits.memory",
											Divisor:       resource.MustParse("1Mi"),
										},
									},
								},
								{Name: "HADOOP_CONF_DIR", Value: "/etc/hadoop/conf"},
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: "/etc/gcp_service_account/gcp_service_account_key.json",
								},
								{
									Name:  "FOO",
									Value: "abc",
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("200m"),
									v1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("500m"),
									v1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
							VolumeMounts: []v1.VolumeMount{
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
						},
						v1.Container{Name: "sidecar", Image: "alpine"},
					},
					Volumes: []v1.Volume{
						{
							Name: "cache-volume",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "flink-config-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "flinkjobcluster-sample-configmap",
									},
								},
							},
						},
						{
							Name: "hadoop-config-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "hadoop-configmap",
									},
								},
							},
						},
						{
							Name: "gcp-service-account-volume",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "gcp-service-account-secret",
								},
							},
						},
						{
							Name: "tm-claim",
							VolumeSource: v1.VolumeSource{
								Ephemeral: &v1.EphemeralVolumeSource{
									VolumeClaimTemplate: &v1.PersistentVolumeClaimTemplate{
										Spec: v1.PersistentVolumeClaimSpec{
											AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
											Resources: v1.ResourceRequirements{
												Requests: map[v1.ResourceName]resource.Quantity{
													v1.ResourceStorage: resource.MustParse("100Gi"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	res, size := getClusterResourceList(desiredState)
	assert.Assert(t, size == 5)
	assert.Equal(t, res.Memory().String(), "4608Mi")
	assert.Equal(t, res.Cpu().MilliValue(), int64(2200))
	assert.Equal(t, res.Storage().String(), "400Gi")
}

func TestPogGroupsMinResources(t *testing.T) {
	desiredState := getDesiredState()
	res, size := getClusterResourceList(desiredState)

	assert.Equal(t, size, int32(1))
	assert.Equal(t, res.Memory().String(), "512Mi")
	assert.Equal(t, res.Cpu().MilliValue(), int64(200))
	assert.Equal(t, res.Storage().String(), "0")

	rl := *buildMinResource(res)

	assert.Equal(t, rl.Cpu().String(), "200m", "CPU should be 200m")
	assert.Equal(t, rl.Name(corev1.ResourceRequestsCPU, resource.DecimalSI).String(), "200m", "CPU request should be 200m")
	assert.Equal(t, rl.Name(corev1.ResourceLimitsCPU, resource.DecimalSI).String(), "200m", "CPU limit should be 200m")

	assert.Equal(t, rl.Memory().String(), "512Mi", "Memory should be 512Mi")
	assert.Equal(t, rl.Name(corev1.ResourceRequestsMemory, resource.DecimalSI).String(), "512Mi", "Memory request should be 512Mi")
	assert.Equal(t, rl.Name(corev1.ResourceLimitsMemory, resource.DecimalSI).String(), "512Mi", "Memory limit should be 512Mi")

	assert.Equal(t, rl.Storage().String(), "0", "Storage should be 0")
	assert.Equal(t, rl.Name(corev1.ResourceRequestsStorage, resource.DecimalSI).String(), "0", "Storage request should be 0")
}
