//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BatchSchedulerSpec) DeepCopyInto(out *BatchSchedulerSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BatchSchedulerSpec.
func (in *BatchSchedulerSpec) DeepCopy() *BatchSchedulerSpec {
	if in == nil {
		return nil
	}
	out := new(BatchSchedulerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CleanupPolicy) DeepCopyInto(out *CleanupPolicy) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CleanupPolicy.
func (in *CleanupPolicy) DeepCopy() *CleanupPolicy {
	if in == nil {
		return nil
	}
	out := new(CleanupPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlinkCluster) DeepCopyInto(out *FlinkCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlinkCluster.
func (in *FlinkCluster) DeepCopy() *FlinkCluster {
	if in == nil {
		return nil
	}
	out := new(FlinkCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FlinkCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlinkClusterComponentState) DeepCopyInto(out *FlinkClusterComponentState) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlinkClusterComponentState.
func (in *FlinkClusterComponentState) DeepCopy() *FlinkClusterComponentState {
	if in == nil {
		return nil
	}
	out := new(FlinkClusterComponentState)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlinkClusterComponentsStatus) DeepCopyInto(out *FlinkClusterComponentsStatus) {
	*out = *in
	out.ConfigMap = in.ConfigMap
	out.JobManagerStatefulSet = in.JobManagerStatefulSet
	in.JobManagerService.DeepCopyInto(&out.JobManagerService)
	if in.JobManagerIngress != nil {
		in, out := &in.JobManagerIngress, &out.JobManagerIngress
		*out = new(JobManagerIngressStatus)
		(*in).DeepCopyInto(*out)
	}
	out.TaskManagerStatefulSet = in.TaskManagerStatefulSet
	out.TaskManagerDeployment = in.TaskManagerDeployment
	if in.Job != nil {
		in, out := &in.Job, &out.Job
		*out = new(JobStatus)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlinkClusterComponentsStatus.
func (in *FlinkClusterComponentsStatus) DeepCopy() *FlinkClusterComponentsStatus {
	if in == nil {
		return nil
	}
	out := new(FlinkClusterComponentsStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlinkClusterControlStatus) DeepCopyInto(out *FlinkClusterControlStatus) {
	*out = *in
	if in.Details != nil {
		in, out := &in.Details, &out.Details
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlinkClusterControlStatus.
func (in *FlinkClusterControlStatus) DeepCopy() *FlinkClusterControlStatus {
	if in == nil {
		return nil
	}
	out := new(FlinkClusterControlStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlinkClusterList) DeepCopyInto(out *FlinkClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]FlinkCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlinkClusterList.
func (in *FlinkClusterList) DeepCopy() *FlinkClusterList {
	if in == nil {
		return nil
	}
	out := new(FlinkClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FlinkClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlinkClusterSpec) DeepCopyInto(out *FlinkClusterSpec) {
	*out = *in
	in.Image.DeepCopyInto(&out.Image)
	if in.ServiceAccountName != nil {
		in, out := &in.ServiceAccountName, &out.ServiceAccountName
		*out = new(string)
		**out = **in
	}
	if in.BatchSchedulerName != nil {
		in, out := &in.BatchSchedulerName, &out.BatchSchedulerName
		*out = new(string)
		**out = **in
	}
	if in.BatchScheduler != nil {
		in, out := &in.BatchScheduler, &out.BatchScheduler
		*out = new(BatchSchedulerSpec)
		**out = **in
	}
	if in.JobManager != nil {
		in, out := &in.JobManager, &out.JobManager
		*out = new(JobManagerSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.TaskManager != nil {
		in, out := &in.TaskManager, &out.TaskManager
		*out = new(TaskManagerSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Job != nil {
		in, out := &in.Job, &out.Job
		*out = new(JobSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.EnvVars != nil {
		in, out := &in.EnvVars, &out.EnvVars
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.EnvFrom != nil {
		in, out := &in.EnvFrom, &out.EnvFrom
		*out = make([]v1.EnvFromSource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.FlinkProperties != nil {
		in, out := &in.FlinkProperties, &out.FlinkProperties
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.HadoopConfig != nil {
		in, out := &in.HadoopConfig, &out.HadoopConfig
		*out = new(HadoopConfig)
		**out = **in
	}
	if in.GCPConfig != nil {
		in, out := &in.GCPConfig, &out.GCPConfig
		*out = new(GCPConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.LogConfig != nil {
		in, out := &in.LogConfig, &out.LogConfig
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.RevisionHistoryLimit != nil {
		in, out := &in.RevisionHistoryLimit, &out.RevisionHistoryLimit
		*out = new(int32)
		**out = **in
	}
	if in.RecreateOnUpdate != nil {
		in, out := &in.RecreateOnUpdate, &out.RecreateOnUpdate
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlinkClusterSpec.
func (in *FlinkClusterSpec) DeepCopy() *FlinkClusterSpec {
	if in == nil {
		return nil
	}
	out := new(FlinkClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlinkClusterStatus) DeepCopyInto(out *FlinkClusterStatus) {
	*out = *in
	in.Components.DeepCopyInto(&out.Components)
	if in.Control != nil {
		in, out := &in.Control, &out.Control
		*out = new(FlinkClusterControlStatus)
		(*in).DeepCopyInto(*out)
	}
	if in.Savepoint != nil {
		in, out := &in.Savepoint, &out.Savepoint
		*out = new(SavepointStatus)
		**out = **in
	}
	in.Revision.DeepCopyInto(&out.Revision)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlinkClusterStatus.
func (in *FlinkClusterStatus) DeepCopy() *FlinkClusterStatus {
	if in == nil {
		return nil
	}
	out := new(FlinkClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPConfig) DeepCopyInto(out *GCPConfig) {
	*out = *in
	if in.ServiceAccount != nil {
		in, out := &in.ServiceAccount, &out.ServiceAccount
		*out = new(GCPServiceAccount)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPConfig.
func (in *GCPConfig) DeepCopy() *GCPConfig {
	if in == nil {
		return nil
	}
	out := new(GCPConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GCPServiceAccount) DeepCopyInto(out *GCPServiceAccount) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GCPServiceAccount.
func (in *GCPServiceAccount) DeepCopy() *GCPServiceAccount {
	if in == nil {
		return nil
	}
	out := new(GCPServiceAccount)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HadoopConfig) DeepCopyInto(out *HadoopConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HadoopConfig.
func (in *HadoopConfig) DeepCopy() *HadoopConfig {
	if in == nil {
		return nil
	}
	out := new(HadoopConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageSpec) DeepCopyInto(out *ImageSpec) {
	*out = *in
	if in.PullSecrets != nil {
		in, out := &in.PullSecrets, &out.PullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageSpec.
func (in *ImageSpec) DeepCopy() *ImageSpec {
	if in == nil {
		return nil
	}
	out := new(ImageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobManagerIngressSpec) DeepCopyInto(out *JobManagerIngressSpec) {
	*out = *in
	if in.HostFormat != nil {
		in, out := &in.HostFormat, &out.HostFormat
		*out = new(string)
		**out = **in
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.UseTLS != nil {
		in, out := &in.UseTLS, &out.UseTLS
		*out = new(bool)
		**out = **in
	}
	if in.TLSSecretName != nil {
		in, out := &in.TLSSecretName, &out.TLSSecretName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobManagerIngressSpec.
func (in *JobManagerIngressSpec) DeepCopy() *JobManagerIngressSpec {
	if in == nil {
		return nil
	}
	out := new(JobManagerIngressSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobManagerIngressStatus) DeepCopyInto(out *JobManagerIngressStatus) {
	*out = *in
	if in.URLs != nil {
		in, out := &in.URLs, &out.URLs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobManagerIngressStatus.
func (in *JobManagerIngressStatus) DeepCopy() *JobManagerIngressStatus {
	if in == nil {
		return nil
	}
	out := new(JobManagerIngressStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobManagerPorts) DeepCopyInto(out *JobManagerPorts) {
	*out = *in
	if in.RPC != nil {
		in, out := &in.RPC, &out.RPC
		*out = new(int32)
		**out = **in
	}
	if in.Blob != nil {
		in, out := &in.Blob, &out.Blob
		*out = new(int32)
		**out = **in
	}
	if in.Query != nil {
		in, out := &in.Query, &out.Query
		*out = new(int32)
		**out = **in
	}
	if in.UI != nil {
		in, out := &in.UI, &out.UI
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobManagerPorts.
func (in *JobManagerPorts) DeepCopy() *JobManagerPorts {
	if in == nil {
		return nil
	}
	out := new(JobManagerPorts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobManagerServiceStatus) DeepCopyInto(out *JobManagerServiceStatus) {
	*out = *in
	if in.LoadBalancerIngress != nil {
		in, out := &in.LoadBalancerIngress, &out.LoadBalancerIngress
		*out = make([]v1.LoadBalancerIngress, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobManagerServiceStatus.
func (in *JobManagerServiceStatus) DeepCopy() *JobManagerServiceStatus {
	if in == nil {
		return nil
	}
	out := new(JobManagerServiceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobManagerSpec) DeepCopyInto(out *JobManagerSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.ServiceAnnotations != nil {
		in, out := &in.ServiceAnnotations, &out.ServiceAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ServiceLabels != nil {
		in, out := &in.ServiceLabels, &out.ServiceLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Ingress != nil {
		in, out := &in.Ingress, &out.Ingress
		*out = new(JobManagerIngressSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Ports.DeepCopyInto(&out.Ports)
	if in.ExtraPorts != nil {
		in, out := &in.ExtraPorts, &out.ExtraPorts
		*out = make([]NamedPort, len(*in))
		copy(*out, *in)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.MemoryOffHeapRatio != nil {
		in, out := &in.MemoryOffHeapRatio, &out.MemoryOffHeapRatio
		*out = new(int32)
		**out = **in
	}
	out.MemoryOffHeapMin = in.MemoryOffHeapMin.DeepCopy()
	if in.MemoryProcessRatio != nil {
		in, out := &in.MemoryProcessRatio, &out.MemoryProcessRatio
		*out = new(int32)
		**out = **in
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeClaimTemplates != nil {
		in, out := &in.VolumeClaimTemplates, &out.VolumeClaimTemplates
		*out = make([]v1.PersistentVolumeClaim, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.PodLabels != nil {
		in, out := &in.PodLabels, &out.PodLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.LivenessProbe != nil {
		in, out := &in.LivenessProbe, &out.LivenessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.ReadinessProbe != nil {
		in, out := &in.ReadinessProbe, &out.ReadinessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobManagerSpec.
func (in *JobManagerSpec) DeepCopy() *JobManagerSpec {
	if in == nil {
		return nil
	}
	out := new(JobManagerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobSpec) DeepCopyInto(out *JobSpec) {
	*out = *in
	if in.ClassPath != nil {
		in, out := &in.ClassPath, &out.ClassPath
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.JarFile != nil {
		in, out := &in.JarFile, &out.JarFile
		*out = new(string)
		**out = **in
	}
	if in.ClassName != nil {
		in, out := &in.ClassName, &out.ClassName
		*out = new(string)
		**out = **in
	}
	if in.PyFile != nil {
		in, out := &in.PyFile, &out.PyFile
		*out = new(string)
		**out = **in
	}
	if in.PyFiles != nil {
		in, out := &in.PyFiles, &out.PyFiles
		*out = new(string)
		**out = **in
	}
	if in.PyModule != nil {
		in, out := &in.PyModule, &out.PyModule
		*out = new(string)
		**out = **in
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.FromSavepoint != nil {
		in, out := &in.FromSavepoint, &out.FromSavepoint
		*out = new(string)
		**out = **in
	}
	if in.AllowNonRestoredState != nil {
		in, out := &in.AllowNonRestoredState, &out.AllowNonRestoredState
		*out = new(bool)
		**out = **in
	}
	if in.SavepointsDir != nil {
		in, out := &in.SavepointsDir, &out.SavepointsDir
		*out = new(string)
		**out = **in
	}
	if in.TakeSavepointOnUpdate != nil {
		in, out := &in.TakeSavepointOnUpdate, &out.TakeSavepointOnUpdate
		*out = new(bool)
		**out = **in
	}
	if in.MaxStateAgeToRestoreSeconds != nil {
		in, out := &in.MaxStateAgeToRestoreSeconds, &out.MaxStateAgeToRestoreSeconds
		*out = new(int32)
		**out = **in
	}
	if in.AutoSavepointSeconds != nil {
		in, out := &in.AutoSavepointSeconds, &out.AutoSavepointSeconds
		*out = new(int32)
		**out = **in
	}
	if in.Parallelism != nil {
		in, out := &in.Parallelism, &out.Parallelism
		*out = new(int32)
		**out = **in
	}
	if in.NoLoggingToStdout != nil {
		in, out := &in.NoLoggingToStdout, &out.NoLoggingToStdout
		*out = new(bool)
		**out = **in
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.RestartPolicy != nil {
		in, out := &in.RestartPolicy, &out.RestartPolicy
		*out = new(JobRestartPolicy)
		**out = **in
	}
	if in.CleanupPolicy != nil {
		in, out := &in.CleanupPolicy, &out.CleanupPolicy
		*out = new(CleanupPolicy)
		**out = **in
	}
	if in.CancelRequested != nil {
		in, out := &in.CancelRequested, &out.CancelRequested
		*out = new(bool)
		**out = **in
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PodLabels != nil {
		in, out := &in.PodLabels, &out.PodLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.Mode != nil {
		in, out := &in.Mode, &out.Mode
		*out = new(JobMode)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobSpec.
func (in *JobSpec) DeepCopy() *JobSpec {
	if in == nil {
		return nil
	}
	out := new(JobSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JobStatus) DeepCopyInto(out *JobStatus) {
	*out = *in
	if in.CompletionTime != nil {
		in, out := &in.CompletionTime, &out.CompletionTime
		*out = (*in).DeepCopy()
	}
	if in.FailureReasons != nil {
		in, out := &in.FailureReasons, &out.FailureReasons
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JobStatus.
func (in *JobStatus) DeepCopy() *JobStatus {
	if in == nil {
		return nil
	}
	out := new(JobStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamedPort) DeepCopyInto(out *NamedPort) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamedPort.
func (in *NamedPort) DeepCopy() *NamedPort {
	if in == nil {
		return nil
	}
	out := new(NamedPort)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionStatus) DeepCopyInto(out *RevisionStatus) {
	*out = *in
	if in.CollisionCount != nil {
		in, out := &in.CollisionCount, &out.CollisionCount
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionStatus.
func (in *RevisionStatus) DeepCopy() *RevisionStatus {
	if in == nil {
		return nil
	}
	out := new(RevisionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SavepointStatus) DeepCopyInto(out *SavepointStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SavepointStatus.
func (in *SavepointStatus) DeepCopy() *SavepointStatus {
	if in == nil {
		return nil
	}
	out := new(SavepointStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TaskManagerPorts) DeepCopyInto(out *TaskManagerPorts) {
	*out = *in
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = new(int32)
		**out = **in
	}
	if in.RPC != nil {
		in, out := &in.RPC, &out.RPC
		*out = new(int32)
		**out = **in
	}
	if in.Query != nil {
		in, out := &in.Query, &out.Query
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TaskManagerPorts.
func (in *TaskManagerPorts) DeepCopy() *TaskManagerPorts {
	if in == nil {
		return nil
	}
	out := new(TaskManagerPorts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TaskManagerSpec) DeepCopyInto(out *TaskManagerSpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	in.Ports.DeepCopyInto(&out.Ports)
	if in.ExtraPorts != nil {
		in, out := &in.ExtraPorts, &out.ExtraPorts
		*out = make([]NamedPort, len(*in))
		copy(*out, *in)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.MemoryOffHeapRatio != nil {
		in, out := &in.MemoryOffHeapRatio, &out.MemoryOffHeapRatio
		*out = new(int32)
		**out = **in
	}
	out.MemoryOffHeapMin = in.MemoryOffHeapMin.DeepCopy()
	if in.MemoryProcessRatio != nil {
		in, out := &in.MemoryProcessRatio, &out.MemoryProcessRatio
		*out = new(int32)
		**out = **in
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeClaimTemplates != nil {
		in, out := &in.VolumeClaimTemplates, &out.VolumeClaimTemplates
		*out = make([]v1.PersistentVolumeClaim, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = make([]v1.Container, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PodAnnotations != nil {
		in, out := &in.PodAnnotations, &out.PodAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.PodLabels != nil {
		in, out := &in.PodLabels, &out.PodLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.LivenessProbe != nil {
		in, out := &in.LivenessProbe, &out.LivenessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.ReadinessProbe != nil {
		in, out := &in.ReadinessProbe, &out.ReadinessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TaskManagerSpec.
func (in *TaskManagerSpec) DeepCopy() *TaskManagerSpec {
	if in == nil {
		return nil
	}
	out := new(TaskManagerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TimeConverter) DeepCopyInto(out *TimeConverter) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TimeConverter.
func (in *TimeConverter) DeepCopy() *TimeConverter {
	if in == nil {
		return nil
	}
	out := new(TimeConverter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Validator) DeepCopyInto(out *Validator) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Validator.
func (in *Validator) DeepCopy() *Validator {
	if in == nil {
		return nil
	}
	out := new(Validator)
	in.DeepCopyInto(out)
	return out
}
