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

var v10, _ = version.NewVersion("1.10")

// Sets default values for unspecified FlinkCluster properties.
func _SetDefault(cluster *FlinkCluster) {
	if cluster.Spec.BatchSchedulerName != nil {
		cluster.Spec.BatchScheduler = &BatchSchedulerSpec{
			Name: *cluster.Spec.BatchSchedulerName,
		}
	}

	flinkVersion, _ := version.NewVersion(cluster.Spec.FlinkVersion)
	if cluster.Spec.JobManager == nil {
		cluster.Spec.JobManager = &JobManagerSpec{}
	}
	_SetJobManagerDefault(cluster.Spec.JobManager, flinkVersion)
	if cluster.Spec.TaskManager == nil {
		cluster.Spec.TaskManager = &TaskManagerSpec{}
	}
	_SetTaskManagerDefault(cluster.Spec.TaskManager, flinkVersion)
}

func _SetJobManagerDefault(jmSpec *JobManagerSpec, flinkVersion *version.Version) {
	if jmSpec == nil {
		return
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

	if jmSpec.Ports.RPC != nil {
		var livenessProbe = corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
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
			ProbeHandler: corev1.ProbeHandler{
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
}

func _SetTaskManagerDefault(tmSpec *TaskManagerSpec, flinkVersion *version.Version) {
	if tmSpec == nil {
		return
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

	if tmSpec.Ports.RPC != nil {
		var livenessProbe = corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
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
			ProbeHandler: corev1.ProbeHandler{
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
}
