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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterState defines states for a cluster.
const (
	ClusterStateCreating         = "Creating"
	ClusterStateRunning          = "Running"
	ClusterStateReconciling      = "Reconciling"
	ClusterStateUpdating         = "Updating"
	ClusterStateStopping         = "Stopping"
	ClusterStatePartiallyStopped = "PartiallyStopped"
	ClusterStateStopped          = "Stopped"
)

// ComponentState defines states for a cluster component.
const (
	ComponentStateNotReady = "NotReady"
	ComponentStateReady    = "Ready"
	ComponentStateUpdating = "Updating"
	ComponentStateDeleted  = "Deleted"
)

// JobMode defines the running mode for the job.
const (
	JobModeBlocking    = "Blocking"
	JobModeApplication = "Application"
	JobModeDetached    = "Detached"
)

type JobMode string

// JobState defines states for a Flink job deployment.
const (
	JobStatePending      = "Pending"
	JobStateUpdating     = "Updating"
	JobStateRestarting   = "Restarting"
	JobStateDeploying    = "Deploying"
	JobStateDeployFailed = "DeployFailed"
	JobStateRunning      = "Running"
	JobStateSucceeded    = "Succeeded"
	JobStateCancelled    = "Cancelled"
	JobStateFailed       = "Failed"
	JobStateLost         = "Lost"
	JobStateUnknown      = "Unknown"
)

// AccessScope defines the access scope of JobManager service.
const (
	AccessScopeCluster  = "Cluster"
	AccessScopeVPC      = "VPC"
	AccessScopeExternal = "External"
	AccessScopeNodePort = "NodePort"
	AccessScopeHeadless = "Headless"
)

// JobRestartPolicy defines the restart policy when a job fails.
type JobRestartPolicy string

const (
	// JobRestartPolicyNever - never restarts a failed job.
	JobRestartPolicyNever JobRestartPolicy = "Never"

	// JobRestartPolicyFromSavepointOnFailure - restart the job from the latest
	// savepoint if available, otherwise do not restart.
	JobRestartPolicyFromSavepointOnFailure JobRestartPolicy = "FromSavepointOnFailure"
)

// User requested control
const (
	// control annotation key
	ControlAnnotation = "flinkclusters.flinkoperator.k8s.io/user-control"

	// control name
	ControlNameSavepoint = "savepoint"
	ControlNameJobCancel = "job-cancel"

	// control state
	ControlStateRequested  = "Requested"
	ControlStateInProgress = "InProgress"
	ControlStateSucceeded  = "Succeeded"
	ControlStateFailed     = "Failed"
)

// Savepoint status
type SavepointReason string

const (
	SavepointStateInProgress    = "InProgress"
	SavepointStateTriggerFailed = "TriggerFailed"
	SavepointStateFailed        = "Failed"
	SavepointStateSucceeded     = "Succeeded"

	SavepointReasonUserRequested SavepointReason = "user requested"
	SavepointReasonJobCancel     SavepointReason = "job cancel"
	SavepointReasonScheduled     SavepointReason = "scheduled"
	SavepointReasonUpdate        SavepointReason = "update"
)

// ImageSpec defines Flink image of JobManager and TaskManager containers.
type ImageSpec struct {
	// Flink image name.
	Name string `json:"name"`

	// Image pull policy. One of `Always, Never, IfNotPresent`, default: `Always`.
	// if :latest tag is specified, or IfNotPresent otherwise.
	// [More info](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy)
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`

	// _(Optional)_ Secrets for image pull.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret)
	PullSecrets []corev1.LocalObjectReference `json:"pullSecrets,omitempty"`
}

// NamedPort defines the container port properties.
type NamedPort struct {
	// _(Optional)_ If specified, this must be an IANA_SVC_NAME and unique within the pod. Each
	// named port in a pod must have a unique name. Name for the port that can be
	// referred to by services.
	Name string `json:"name,omitempty"`

	// Number of port to expose on the pod's IP address.
	// This must be a valid port number, 0 < x < 65536.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ContainerPort int32 `json:"containerPort"`

	// Protocol for port. One of `UDP, TCP, or SCTP`, default: `TCP`.
	// +kubebuilder:validation:Enum=TCP;UDP;SCTP
	Protocol string `json:"protocol,omitempty"`
}

// JobManagerPorts defines ports of JobManager.
type JobManagerPorts struct {
	// RPC port, default: `6123`.
	RPC *int32 `json:"rpc,omitempty"`

	// Blob port, default: `6124`.
	Blob *int32 `json:"blob,omitempty"`

	// Query port, default: `6125`.
	Query *int32 `json:"query,omitempty"`

	// UI port, default: `8081`.
	UI *int32 `json:"ui,omitempty"`
}

// JobManagerIngressSpec defines ingress of JobManager
type JobManagerIngressSpec struct {
	// _(Optional)_ Ingress host format. ex) {{$clusterName}}.example.com
	HostFormat *string `json:"hostFormat,omitempty"`

	// _(Optional)_Annotations for ingress configuration.
	// [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
	Annotations map[string]string `json:"annotations,omitempty"`

	// TLS use, default: `false`.
	UseTLS *bool `json:"useTls,omitempty"`

	// _(Optional)_TLS secret name.
	TLSSecretName *string `json:"tlsSecretName,omitempty"`
}

// JobManagerSpec defines properties of JobManager.
type JobManagerSpec struct {
	// The number of JobManager replicas, default: `1`
	Replicas *int32 `json:"replicas,omitempty"`

	// Access scope, default: `Cluster`.
	// `Cluster`: accessible from within the same cluster.
	// `VPC`: accessible from within the same VPC.
	// `External`: accessible from the internet.
	// `NodePort`: accessible through node port.
	// `Headless`: pod IPs assumed to be routable and advertised directly with `clusterIP: None``.
	// Currently `VPC, External` are only available for GKE.
	AccessScope string `json:"accessScope,omitempty"`

	// _(Optional)_ Define JobManager Service annotations for configuration.
	ServiceAnnotations map[string]string `json:"ServiceAnnotations,omitempty"`

	// _(Optional)_ Define JobManager Service labels for configuration.
	ServiceLabels map[string]string `json:"ServiceLabels,omitempty"`

	// _(Optional)_ Provide external access to JobManager UI/API.
	Ingress *JobManagerIngressSpec `json:"ingress,omitempty"`

	// Ports that JobManager listening on.
	Ports JobManagerPorts `json:"ports,omitempty"`

	// _(Optional)_ Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on.
	// Each port number and name must be unique among ports and extraPorts.
	ExtraPorts []NamedPort `json:"extraPorts,omitempty"`

	// Compute resources required by each JobManager container.
	// default: 2 CPUs with 2Gi Memory.
	// It Cannot be updated.
	// [More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/)
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Percentage of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `25`
	MemoryOffHeapRatio *int32 `json:"memoryOffHeapRatio,omitempty"`

	// Minimum amount of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `600M`
	// You can express this value like 600M, 572Mi and 600e6
	// [More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory)
	MemoryOffHeapMin resource.Quantity `json:"memoryOffHeapMin,omitempty"`

	// For Flink 1.10+. Percentage of memory process, as a safety margin to avoid OOM kill, default: `80`
	MemoryProcessRatio *int32 `json:"memoryProcessRatio,omitempty"`

	// _(Optional)_ Volumes in the JobManager pod.
	// [More info](https://kubernetes.io/docs/concepts/storage/volumes/)
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// _(Optional)_ Volume mounts in the JobManager container.
	// [More info](https://kubernetes.io/docs/concepts/storage/volumes/)
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// _(Optional)_ A template for persistent volume claim each requested and mounted to JobManager pod,
	// This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend).
	// [More info](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// _(Optional)_ Init containers of the Job Manager pod.
	// [More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/)
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// _(Optional)_ Selector which must match a node's labels for the JobManager pod to be
	// scheduled on that node.
	// [More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// _(Optional)_ Defines the node affinity of the pod
	// [More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// _(Optional)_ Sidecar containers running alongside with the JobManager container in the pod.
	// [More info](https://kubernetes.io/docs/concepts/containers/)
	Sidecars []corev1.Container `json:"sidecars,omitempty"`

	// _(Optional)_ JobManager StatefulSet pod template annotations.
	// [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// _(Optional)_ SecurityContext of the JobManager pod.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod)
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// _(Optional)_ JobManager StatefulSet pod template labels.
	// [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
	PodLabels map[string]string `json:"podLabels,omitempty"`

	// Container liveness probe
	// If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L113-L123) will be used.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// Container readiness probe
	// If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L129-L139) will be used.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
}

// TaskManagerPorts defines ports of TaskManager.
type TaskManagerPorts struct {
	// Data port, default: `6121`.
	Data *int32 `json:"data,omitempty"`

	// RPC port, default: `6122`.
	RPC *int32 `json:"rpc,omitempty"`

	// Query port, default: `6125`.
	Query *int32 `json:"query,omitempty"`
}

// TaskManagerSpec defines properties of TaskManager.
type TaskManagerSpec struct {
	// The number of replicas. default: `3`
	Replicas *int32 `json:"replicas,omitempty"`

	// Ports that TaskManager listening on.
	Ports TaskManagerPorts `json:"ports,omitempty"`

	// _(Optional)_ Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on.
	ExtraPorts []NamedPort `json:"extraPorts,omitempty"`

	// Compute resources required by each TaskManager container.
	// default: 2 CPUs with 2Gi Memory.
	// It Cannot be updated.
	// [More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/)
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// TODO: Memory calculation would be change. Let's watch the issue FLINK-13980.

	// Percentage of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `25`
	MemoryOffHeapRatio *int32 `json:"memoryOffHeapRatio,omitempty"`

	// Minimum amount of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `600M`
	// You can express this value like 600M, 572Mi and 600e6
	// [More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory)
	MemoryOffHeapMin resource.Quantity `json:"memoryOffHeapMin,omitempty"`

	// For Flink 1.10+. Percentage of process memory, as a safety margin to avoid OOM kill, default: `20`
	MemoryProcessRatio *int32 `json:"memoryProcessRatio,omitempty"`

	// _(Optional)_ Volumes in the TaskManager pods.
	// [More info](https://kubernetes.io/docs/concepts/storage/volumes/)
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// _(Optional)_ Volume mounts in the TaskManager containers.
	// [More info](https://kubernetes.io/docs/concepts/storage/volumes/)
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// _(Optional)_ A template for persistent volume claim each requested and mounted to JobManager pod,
	// This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend).
	// [More info](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// _(Optional)_ Init containers of the Task Manager pod.
	// [More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/)
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// _(Optional)_ Selector which must match a node's labels for the TaskManager pod to be
	// scheduled on that node.
	// [More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// _(Optional)_ Defines the node affinity of the pod
	// [More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// _(Optional)_ Sidecar containers running alongside with the TaskManager container in the pod.
	// [More info](https://kubernetes.io/docs/concepts/containers/)
	Sidecars []corev1.Container `json:"sidecars,omitempty"`

	// _(Optional)_ TaskManager StatefulSet pod template annotations.
	// [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// _(Optional)_ SecurityContext of the TaskManager pod.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod)
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// _(Optional)_ TaskManager StatefulSet pod template labels.
	// [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
	PodLabels map[string]string `json:"podLabels,omitempty"`

	// Container liveness probe
	// If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L177-L187) will be used.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	// Container readiness probe
	// If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L193-L203) will be used.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
}

// CleanupAction defines the action to take after job finishes.
type CleanupAction string

const (
	// CleanupActionKeepCluster - keep the entire cluster.
	CleanupActionKeepCluster = "KeepCluster"
	// CleanupActionDeleteCluster - delete the entire cluster.
	CleanupActionDeleteCluster = "DeleteCluster"
	// CleanupActionDeleteTaskManager - delete task manager, keep job manager.
	CleanupActionDeleteTaskManager = "DeleteTaskManager"
)

// CleanupPolicy defines the action to take after job finishes.
// Use one of `KeepCluster, DeleteCluster, DeleteTaskManager` for the below fields.
type CleanupPolicy struct {
	// Action to take after job succeeds, default: `DeleteCluster`.
	AfterJobSucceeds CleanupAction `json:"afterJobSucceeds,omitempty"`
	// Action to take after job fails, default: `KeepCluster`.
	AfterJobFails CleanupAction `json:"afterJobFails,omitempty"`
	// Action to take after job is cancelled, default: `DeleteCluster`.
	AfterJobCancelled CleanupAction `json:"afterJobCancelled,omitempty"`
}

// JobSpec defines properties of a Flink job.
type JobSpec struct {
	// _(Optional)_ JAR file of the job. It could be a local file or remote URI,
	// depending on which protocols (e.g., `https://, gs://`) are supported by the Flink image.
	JarFile *string `json:"jarFile,omitempty"`

	// _(Optional)_ Fully qualified Java class name of the job.
	ClassName *string `json:"className,omitempty"`

	// _(Optional)_ Python file of the job. It could be a local file or remote URI (e.g.,`https://`, `gs://`).
	PyFile *string `json:"pyFile,omitempty"`

	// _(Optional)_ Python files of the job. It could be a local file (with .py/.egg/.zip/.whl), directory or remote URI (e.g.,`https://`, `gs://`).
	// See the Flink argument `--pyFiles` for the detail.
	PyFiles *string `json:"pyFiles,omitempty"`

	// _(Optional)_ Python module path of the job entry point. Must use with pythonFiles.
	PyModule *string `json:"pyModule,omitempty"`

	// _(Optional)_ Command-line args of the job.
	Args []string `json:"args,omitempty"`

	// _(Optional)_ FromSavepoint where to restore the job from
	// Savepoint where to restore the job from (e.g., gs://my-savepoint/1234).
	// If flink job must be restored from the latest available savepoint when Flink job updating, this field must be unspecified.
	FromSavepoint *string `json:"fromSavepoint,omitempty"`

	// Allow non-restored state, default: `false`.
	AllowNonRestoredState *bool `json:"allowNonRestoredState,omitempty"`

	// _(Optional)_ Savepoints dir where to store savepoints of the job.
	SavepointsDir *string `json:"savepointsDir,omitempty"`

	// _(Optional)_ Should take savepoint before updating job, default: `true`.
	// If this is set as false, maxStateAgeToRestoreSeconds must be provided to limit the savepoint age to restore.
	TakeSavepointOnUpdate *bool `json:"takeSavepointOnUpdate,omitempty"`

	// _(Optional)_ Maximum age of the savepoint that allowed to restore state.
	// This is applied to auto restart on failure, update from stopped state and update without taking savepoint.
	// If nil, job can be restarted only when the latest savepoint is the final job state (created by "stop with savepoint")
	// - that is, only when job can be resumed from the suspended state.
	// +kubebuilder:validation:Minimum=0
	MaxStateAgeToRestoreSeconds *int32 `json:"maxStateAgeToRestoreSeconds,omitempty"`

	// _(Optional)_ Automatically take a savepoint to the `savepointsDir` every n seconds.
	AutoSavepointSeconds *int32 `json:"autoSavepointSeconds,omitempty"`

	// _(Optional)_ Update this field to `jobStatus.savepointGeneration + 1` for a running job
	// cluster to trigger a new savepoint to `savepointsDir` on demand.
	SavepointGeneration int32 `json:"savepointGeneration,omitempty"`

	// Job parallelism, default: `1`.
	Parallelism *int32 `json:"parallelism,omitempty"`

	// No logging output to STDOUT, default: `false`.
	NoLoggingToStdout *bool `json:"noLoggingToStdout,omitempty"`

	// _(Optional)_ Volumes in the Job pod.
	// [More info](https://kubernetes.io/docs/concepts/storage/volumes/)
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// _(Optional)_ Volume mounts in the Job container.
	// [More info](https://kubernetes.io/docs/concepts/storage/volumes/)
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// _(Optional)_ Init containers of the Job pod. A typical use case could be using an init
	// container to download a remote job jar to a local path which is
	// referenced by the `jarFile` property.
	// [More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/)
	InitContainers []corev1.Container `json:"initContainers,omitempty"`

	// _(Optional)_ Selector which must match a node's labels for the Job submitter pod to be
	// scheduled on that node.
	// [More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// _(Optional)_ Defines the node affinity of the Job submitter pod
	// [More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Restart policy when the job fails, one of `Never, FromSavepointOnFailure`,
	// default: `Never`.
	// `Never` means the operator will never try to restart a failed job, manual
	// cleanup and restart is required.
	// `FromSavepointOnFailure` means the operator will try to restart the failed
	// job from the savepoint recorded in the job status if available; otherwise,
	// the job will stay in failed state. This option is usually used together
	// with `autoSavepointSeconds` and `savepointsDir`.
	RestartPolicy *JobRestartPolicy `json:"restartPolicy"`

	// The action to take after job finishes.
	CleanupPolicy *CleanupPolicy `json:"cleanupPolicy,omitempty"`

	// Deprecated: _(Optional)_ Request the job to be cancelled. Only applies to running jobs. If
	// `savePointsDir` is provided, a savepoint will be taken before stopping the
	// job.
	CancelRequested *bool `json:"cancelRequested,omitempty"`

	// _(Optional)_ Job pod template annotations.
	// [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// _(Optional)_ Job pod template labels.
	// [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
	PodLabels map[string]string `json:"podLabels,omitempty"`

	// _(Optional)_ Compute resources required by each Job container.
	// If omitted, a default value will be used.
	// It Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// _(Optional)_ SecurityContext of the Job pod.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod)
	SecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`

	// Job running mode, `"Blocking", "Detached"`, default: `"Detached"`
	Mode *JobMode `json:"mode,omitempty"`
}

type BatchSchedulerSpec struct {
	// BatchScheduler name.
	Name string `json:"name"`

	// _(Optional)_ Queue defines the queue in which resources will be allocates; if queue is
	// not specified, resources will be allocated in the schedulers default queue.
	// +optional
	Queue string `json:"queue,omitempty"`

	// _(Optional)_ If specified, indicates the PodGroup's priority. "system-node-critical" and
	// "system-cluster-critical" are two special keywords which indicate the
	// highest priorities with the former being the highest priority. Any other
	// name must be defined by creating a PriorityClass object with that name.
	// If not specified, the priority will be default or zero if there is no
	// default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// FlinkClusterSpec defines the desired state of FlinkCluster
type FlinkClusterSpec struct {
	// The version of Flink to be managed. This version must match the version in the image.
	FlinkVersion string `json:"flinkVersion"`

	// Flink image for JobManager, TaskManager and job containers.
	Image ImageSpec `json:"image"`

	// _(Optional)_ The service account assigned to JobManager, TaskManager and Job submitter Pods. If empty, the default service account in the namespace will be used.
	ServiceAccountName *string `json:"serviceAccountName,omitempty"`

	// Deprecated: BatchSchedulerName specifies the batch scheduler name for JobManager, TaskManager.
	// If empty, no batch scheduling is enabled.
	BatchSchedulerName *string `json:"batchSchedulerName,omitempty"`

	// _(Optional)_ BatchScheduler specifies the batch scheduler for JobManager, TaskManager.
	// If empty, no batch scheduling is enabled.
	BatchScheduler *BatchSchedulerSpec `json:"batchScheduler,omitempty"`

	// _(Optional)_ Flink JobManager spec.
	JobManager *JobManagerSpec `json:"jobManager,omitempty"`

	// _(Optional)_ Flink TaskManager spec.
	TaskManager *TaskManagerSpec `json:"taskManager,omitempty"`

	// _(Optional)_ Job spec. If specified, this cluster is an ephemeral Job
	// Cluster, which will be automatically terminated after the job finishes;
	// otherwise, it is a long-running Session Cluster.
	Job *JobSpec `json:"job,omitempty"`

	// _(Optional)_ Environment variables shared by all JobManager, TaskManager and job
	// containers.
	// [More info](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
	EnvVars []corev1.EnvVar `json:"envVars,omitempty"`

	// _(Optional)_ Environment variables injected from a source, shared by all JobManager,
	// TaskManager and job containers.
	// [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables)
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// _(Optional)_ Flink properties which are appened to flink-conf.yaml.
	FlinkProperties map[string]string `json:"flinkProperties,omitempty"`

	// _(Optional)_ Config for Hadoop.
	HadoopConfig *HadoopConfig `json:"hadoopConfig,omitempty"`

	// _(Optional)_ Config for GCP.
	GCPConfig *GCPConfig `json:"gcpConfig,omitempty"`

	// _(Optional)_ The logging configuration, which should have keys 'log4j-console.properties' and 'logback-console.xml'.
	// These will end up in the 'flink-config-volume' ConfigMap, which gets mounted at /opt/flink/conf.
	// If not provided, defaults that log to console only will be used.
	// <br> - log4j-console.properties: The contents of the log4j properties file to use. If not provided, a default that logs only to stdout will be provided.
	// <br> - logback-console.xml: The contents of the logback XML file to use. If not provided, a default that logs only to stdout will be provided.
	// <br> - Other arbitrary keys are also allowed, and will become part of the ConfigMap.
	LogConfig map[string]string `json:"logConfig,omitempty"`

	// The maximum number of revision history to keep, default: 10.
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit,omitempty"`

	// Recreate components when updating flinkcluster, default: true.
	RecreateOnUpdate *bool `json:"recreateOnUpdate,omitempty"`
}

// HadoopConfig defines configs for Hadoop.
type HadoopConfig struct {
	// The name of the ConfigMap which contains the Hadoop config files.
	// The ConfigMap must be in the same namespace as the FlinkCluster.
	ConfigMapName string `json:"configMapName,omitempty"`

	// The path where to mount the Volume of the ConfigMap.
	// default: `/etc/hadoop/conf`.
	MountPath string `json:"mountPath,omitempty"`
}

// GCPConfig defines configs for GCP.
type GCPConfig struct {
	// GCP service account.
	ServiceAccount *GCPServiceAccount `json:"serviceAccount,omitempty"`
}

// GCPServiceAccount defines the config about GCP service account.
type GCPServiceAccount struct {
	// The name of the Secret holding the GCP service account key file.
	// The Secret must be in the same namespace as the FlinkCluster.
	SecretName string `json:"secretName,omitempty"`

	// The name of the service account key file.
	KeyFile string `json:"keyFile,omitempty"`

	// The path where to mount the Volume of the Secret.
	MountPath string `json:"mountPath,omitempty"`
}

// FlinkClusterComponentState defines the observed state of a component
// of a FlinkCluster.
type FlinkClusterComponentState struct {
	// The resource name of the component.
	Name string `json:"name"`

	// The state of the component.
	State string `json:"state"`
}

// FlinkClusterComponentsStatus defines the observed status of the
// components of a FlinkCluster.
type FlinkClusterComponentsStatus struct {
	// The state of configMap.
	ConfigMap FlinkClusterComponentState `json:"configMap"`

	// The state of JobManager StatefulSet.
	JobManagerStatefulSet FlinkClusterComponentState `json:"jobManagerStatefulSet"`

	// The state of JobManager service.
	JobManagerService JobManagerServiceStatus `json:"jobManagerService"`

	// The state of JobManager ingress.
	JobManagerIngress *JobManagerIngressStatus `json:"jobManagerIngress,omitempty"`

	// The state of TaskManager StatefulSet.
	TaskManagerStatefulSet FlinkClusterComponentState `json:"taskManagerStatefulSet"`

	// The status of the job, available only when JobSpec is provided.
	Job *JobStatus `json:"job,omitempty"`
}

// Control state
type FlinkClusterControlStatus struct {
	// Control name
	Name string `json:"name"`

	// Control data
	Details map[string]string `json:"details,omitempty"`

	// State
	State string `json:"state"`

	// Message
	Message string `json:"message,omitempty"`

	// State update time
	UpdateTime string `json:"updateTime"`
}

// JobStatus defines the status of a job.
type JobStatus struct {
	// The ID of the Flink job.
	ID string `json:"id,omitempty"`

	// The Name of the Flink job.
	Name string `json:"name,omitempty"`

	// The name of the Kubernetes job resource.
	SubmitterName string `json:"submitterName,omitempty"`

	// The state of the Flink job deployment.
	State string `json:"state"`

	// The actual savepoint from which this job started.
	// In case of restart, it might be different from the savepoint in the job
	// spec.
	FromSavepoint string `json:"fromSavepoint,omitempty"`

	// The generation of the savepoint in `savepointsDir` taken by the operator.
	// The value starts from 0 when there is no savepoint and increases by 1 for
	// each successful savepoint.
	SavepointGeneration int32 `json:"savepointGeneration,omitempty"`

	// Savepoint location.
	SavepointLocation string `json:"savepointLocation,omitempty"`

	// Last successful savepoint completed timestamp.
	SavepointTime string `json:"savepointTime,omitempty"`

	// The savepoint recorded in savepointLocation is the final state of the job.
	FinalSavepoint bool `json:"finalSavepoint,omitempty"`

	// The timestamp of the Flink job deployment that creating job submitter.
	DeployTime string `json:"deployTime,omitempty"`

	// The Flink job started timestamp.
	StartTime string `json:"startTime,omitempty"`

	// The number of restarts.
	RestartCount int32 `json:"restartCount,omitempty"`

	// Job completion time. Present when job is terminated regardless of its state.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Reasons for the job failure. Present if job state is Failure
	FailureReasons []string `json:"failureReasons,omitempty"`
}

// SavepointStatus is the status of savepoint progress.
type SavepointStatus struct {
	// The ID of the Flink job.
	JobID string `json:"jobID,omitempty"`

	// Savepoint trigger ID.
	TriggerID string `json:"triggerID,omitempty"`

	// Savepoint triggered time.
	TriggerTime string `json:"triggerTime,omitempty"`

	// Savepoint triggered reason.
	TriggerReason SavepointReason `json:"triggerReason,omitempty"`

	// Savepoint status update time.
	UpdateTime string `json:"requestTime,omitempty"`

	// Savepoint state.
	State string `json:"state"`

	// Savepoint message.
	Message string `json:"message,omitempty"`
}

type RevisionStatus struct {
	// When the controller creates new ControllerRevision, it generates hash string from the FlinkCluster spec
	// which is to be stored in ControllerRevision and uses it to compose the ControllerRevision name.
	// Then the controller updates nextRevision to the ControllerRevision name.
	// When update process is completed, the controller updates currentRevision as nextRevision.
	// currentRevision and nextRevision is composed like this:
	// <FLINK_CLUSTER_NAME>-<FLINK_CLUSTER_SPEC_HASH>-<REVISION_NUMBER_IN_CONTROLLERREVISION>
	// e.g., myflinkcluster-c464ff7-5

	// CurrentRevision indicates the version of FlinkCluster.
	CurrentRevision string `json:"currentRevision,omitempty"`

	// NextRevision indicates the version of FlinkCluster updating.
	NextRevision string `json:"nextRevision,omitempty"`

	// collisionCount is the count of hash collisions for the FlinkCluster. The controller
	// uses this field as a collision avoidance mechanism when it needs to create the name for the
	// newest ControllerRevision.
	CollisionCount *int32 `json:"collisionCount,omitempty"`
}

// JobManagerIngressStatus defines the status of a JobManager ingress.
type JobManagerIngressStatus struct {
	// The name of the Kubernetes ingress resource.
	Name string `json:"name"`

	// The state of the component.
	State string `json:"state"`

	// The URLs of ingress.
	URLs []string `json:"urls,omitempty"`
}

// JobManagerServiceStatus defines the observed state of FlinkCluster
type JobManagerServiceStatus struct {
	// The name of the Kubernetes jobManager service.
	Name string `json:"name"`

	// The state of the component.
	State string `json:"state"`

	// (Optional) The node port, present when `accessScope` is `NodePort`.
	NodePort int32 `json:"nodePort,omitempty"`

	// (Optional) The load balancer ingress, present when `accessScope` is `VPC` or `External`
	LoadBalancerIngress []corev1.LoadBalancerIngress `json:"loadBalancerIngress,omitempty"`
}

// FlinkClusterStatus defines the observed state of FlinkCluster
type FlinkClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The overall state of the Flink cluster.
	State string `json:"state"`

	// The status of the components.
	Components FlinkClusterComponentsStatus `json:"components"`

	// The status of control requested by user.
	Control *FlinkClusterControlStatus `json:"control,omitempty"`

	// The status of savepoint progress.
	Savepoint *SavepointStatus `json:"savepoint,omitempty"`

	// The status of revision.
	Revision RevisionStatus `json:"revision,omitempty"`

	// Last update timestamp for this status.
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName="fc"
// +kubebuilder:printcolumn:name="version",type=string,JSONPath=`.spec.flinkVersion`
// +kubebuilder:printcolumn:name="status",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="age",type=date,JSONPath=`.metadata.creationTimestamp`

// FlinkCluster is the Schema for the flinkclusters API
type FlinkCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlinkClusterSpec   `json:"spec"`
	Status FlinkClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName="fc"

// FlinkClusterList contains a list of FlinkCluster
type FlinkClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlinkCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FlinkCluster{}, &FlinkClusterList{})
}
