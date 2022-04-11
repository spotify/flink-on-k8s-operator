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

package v1beta1

import (
	"github.com/hashicorp/go-version"
	"github.com/imdario/mergo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	DefaultJobManagerReplicas  = 1
	DefaultTaskManagerReplicas = 3
)

var (
	v10, _           = version.NewVersion("1.10")
	DefaultResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}
)

// Sets default values for unspecified FlinkCluster properties.
func _SetDefault(cluster *FlinkCluster) {
	if cluster.Spec.RecreateOnUpdate == nil {
		cluster.Spec.RecreateOnUpdate = new(bool)
		*cluster.Spec.RecreateOnUpdate = true
	}

	if cluster.Spec.BatchSchedulerName != nil {
		cluster.Spec.BatchScheduler = &BatchSchedulerSpec{
			Name: *cluster.Spec.BatchSchedulerName,
		}
	}

	_SetImageDefault(&cluster.Spec.Image)
	flinkVersion, _ := version.NewVersion(cluster.Spec.FlinkVersion)
	if cluster.Spec.JobManager == nil {
		cluster.Spec.JobManager = &JobManagerSpec{}
	}
	_SetJobManagerDefault(cluster.Spec.JobManager, flinkVersion)
	if cluster.Spec.TaskManager == nil {
		cluster.Spec.TaskManager = &TaskManagerSpec{}
	}
	_SetTaskManagerDefault(cluster.Spec.TaskManager, flinkVersion)
	_SetJobDefault(cluster.Spec.Job)
	_SetHadoopConfigDefault(cluster.Spec.HadoopConfig)

}

func _SetImageDefault(imageSpec *ImageSpec) {
	if len(imageSpec.PullPolicy) == 0 {
		imageSpec.PullPolicy = corev1.PullAlways
	}
}

func _SetJobManagerDefault(jmSpec *JobManagerSpec, flinkVersion *version.Version) {
	if jmSpec == nil {
		return
	}

	if jmSpec.Replicas == nil {
		jmSpec.Replicas = new(int32)
		*jmSpec.Replicas = DefaultJobManagerReplicas
	}
	if len(jmSpec.AccessScope) == 0 {
		jmSpec.AccessScope = AccessScopeCluster
	}
	if jmSpec.Ingress != nil {
		if jmSpec.Ingress.UseTLS == nil {
			jmSpec.Ingress.UseTLS = new(bool)
			*jmSpec.Ingress.UseTLS = false
		}
	}
	if jmSpec.Ports.RPC == nil {
		jmSpec.Ports.RPC = new(int32)
		*jmSpec.Ports.RPC = 6123
	}
	if jmSpec.Ports.Blob == nil {
		jmSpec.Ports.Blob = new(int32)
		*jmSpec.Ports.Blob = 6124
	}
	if jmSpec.Ports.Query == nil {
		jmSpec.Ports.Query = new(int32)
		*jmSpec.Ports.Query = 6125
	}
	if jmSpec.Ports.UI == nil {
		jmSpec.Ports.UI = new(int32)
		*jmSpec.Ports.UI = 8081
	}

	if flinkVersion == nil || flinkVersion.LessThan(v10) {
		if jmSpec.MemoryOffHeapMin.Format == "" {
			jmSpec.MemoryOffHeapMin = *resource.NewScaledQuantity(600, 6) // 600MB
		}
		if jmSpec.MemoryOffHeapRatio == nil {
			jmSpec.MemoryOffHeapRatio = new(int32)
			*jmSpec.MemoryOffHeapRatio = 25
		}
	} else {
		if jmSpec.MemoryProcessRatio == nil {
			jmSpec.MemoryProcessRatio = new(int32)
			*jmSpec.MemoryProcessRatio = 80
		}
	}
	if jmSpec.Resources.Size() == 0 {
		jmSpec.Resources = DefaultResources
	}

	var livenessProbe = corev1.Probe{
		Handler: corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(*jmSpec.Ports.RPC)),
			},
		},
		TimeoutSeconds:      10,
		InitialDelaySeconds: 5,
		PeriodSeconds:       60,
		FailureThreshold:    5,
	}
	if jmSpec.LivenessProbe != nil {
		mergo.Merge(&livenessProbe, jmSpec.LivenessProbe, mergo.WithOverride)
	}
	jmSpec.LivenessProbe = &livenessProbe

	var readinessProbe = corev1.Probe{
		Handler: corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(*jmSpec.Ports.RPC)),
			},
		},
		TimeoutSeconds:      10,
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    60,
	}
	if jmSpec.ReadinessProbe != nil {
		mergo.Merge(&readinessProbe, jmSpec.ReadinessProbe, mergo.WithOverride)
	}
	jmSpec.ReadinessProbe = &readinessProbe
}

func _SetTaskManagerDefault(tmSpec *TaskManagerSpec, flinkVersion *version.Version) {
	if tmSpec == nil {
		return
	}
	if tmSpec.Replicas == nil {
		tmSpec.Replicas = new(int32)
		*tmSpec.Replicas = DefaultTaskManagerReplicas
	}
	if tmSpec.Ports.Data == nil {
		tmSpec.Ports.Data = new(int32)
		*tmSpec.Ports.Data = 6121
	}
	if tmSpec.Ports.RPC == nil {
		tmSpec.Ports.RPC = new(int32)
		*tmSpec.Ports.RPC = 6122
	}
	if tmSpec.Ports.Query == nil {
		tmSpec.Ports.Query = new(int32)
		*tmSpec.Ports.Query = 6125
	}
	if flinkVersion == nil || flinkVersion.LessThan(v10) {
		if tmSpec.MemoryOffHeapMin.Format == "" {
			tmSpec.MemoryOffHeapMin = *resource.NewScaledQuantity(600, 6) // 600MB
		}
		if tmSpec.MemoryOffHeapRatio == nil {
			tmSpec.MemoryOffHeapRatio = new(int32)
			*tmSpec.MemoryOffHeapRatio = 25
		}
	} else {
		if tmSpec.MemoryProcessRatio == nil {
			tmSpec.MemoryProcessRatio = new(int32)
			*tmSpec.MemoryProcessRatio = 80
		}
	}
	if tmSpec.Resources.Size() == 0 {
		tmSpec.Resources = DefaultResources
	}

	var livenessProbe = corev1.Probe{
		Handler: corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(*tmSpec.Ports.RPC)),
			},
		},
		TimeoutSeconds:      10,
		InitialDelaySeconds: 5,
		PeriodSeconds:       60,
		FailureThreshold:    5,
	}
	if tmSpec.LivenessProbe != nil {
		mergo.Merge(&livenessProbe, tmSpec.LivenessProbe, mergo.WithOverride)
	}
	tmSpec.LivenessProbe = &livenessProbe

	var readinessProbe = corev1.Probe{
		Handler: corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(*tmSpec.Ports.RPC)),
			},
		},
		TimeoutSeconds:      10,
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		FailureThreshold:    60,
	}
	if tmSpec.ReadinessProbe != nil {
		mergo.Merge(&readinessProbe, tmSpec.ReadinessProbe, mergo.WithOverride)
	}
	tmSpec.ReadinessProbe = &readinessProbe
}

func _SetJobDefault(jobSpec *JobSpec) {
	if jobSpec == nil {
		return
	}
	if jobSpec.AllowNonRestoredState == nil {
		jobSpec.AllowNonRestoredState = new(bool)
		*jobSpec.AllowNonRestoredState = false
	}
	if jobSpec.NoLoggingToStdout == nil {
		jobSpec.NoLoggingToStdout = new(bool)
		*jobSpec.NoLoggingToStdout = false
	}
	if jobSpec.RestartPolicy == nil {
		jobSpec.RestartPolicy = new(JobRestartPolicy)
		*jobSpec.RestartPolicy = JobRestartPolicyNever
	}
	if jobSpec.CleanupPolicy == nil {
		jobSpec.CleanupPolicy = &CleanupPolicy{
			AfterJobSucceeds:  CleanupActionDeleteCluster,
			AfterJobFails:     CleanupActionKeepCluster,
			AfterJobCancelled: CleanupActionDeleteCluster,
		}
	}
	if jobSpec.Mode == nil {
		jobSpec.Mode = new(JobMode)
		*jobSpec.Mode = JobModeDetached
	}
	if jobSpec.Resources.Size() == 0 {
		jobSpec.Resources = DefaultResources
	}
}

func _SetHadoopConfigDefault(hadoopConfig *HadoopConfig) {
	if hadoopConfig == nil {
		return
	}
	if len(hadoopConfig.MountPath) == 0 {
		hadoopConfig.MountPath = "/etc/hadoop/conf"
	}
}
