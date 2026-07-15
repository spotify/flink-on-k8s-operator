# API Reference

## Packages
- [flinkoperator.k8s.io/v1beta1](#flinkoperatork8siov1beta1)


## flinkoperator.k8s.io/v1beta1

Package v1beta1 contains API Schema definitions for the flinkoperator v1beta1 API group

### Resource Types
- [FlinkCluster](#flinkcluster)



#### BatchSchedulerSpec







_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | BatchScheduler name. |  |  |
| `queue` _string_ | _(Optional)_ Queue defines the queue in which resources will be allocates; if queue is<br />not specified, resources will be allocated in the schedulers default queue. |  | Optional: \{\} <br /> |
| `priorityClassName` _string_ | _(Optional)_ If specified, indicates the PodGroup's priority. "system-node-critical" and<br />"system-cluster-critical" are two special keywords which indicate the<br />highest priorities with the former being the highest priority. Any other<br />name must be defined by creating a PriorityClass object with that name.<br />If not specified, the priority will be default or zero if there is no<br />default. |  | Optional: \{\} <br /> |


#### CleanupAction

_Underlying type:_ _string_

CleanupAction defines the action to take after job finishes.



_Appears in:_
- [CleanupPolicy](#cleanuppolicy)



#### CleanupPolicy



CleanupPolicy defines the action to take after job finishes.
Use one of `KeepCluster, DeleteCluster, DeleteTaskManager` for the below fields.



_Appears in:_
- [JobSpec](#jobspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `afterJobSucceeds` _[CleanupAction](#cleanupaction)_ | Action to take after job succeeds, default: `DeleteCluster`. | DeleteCluster | Enum: [KeepCluster DeleteCluster DeleteTaskManager] <br /> |
| `afterJobFails` _[CleanupAction](#cleanupaction)_ | Action to take after job fails, default: `KeepCluster`. | KeepCluster | Enum: [KeepCluster DeleteCluster DeleteTaskManager] <br /> |
| `afterJobCancelled` _[CleanupAction](#cleanupaction)_ | Action to take after job is cancelled, default: `DeleteCluster`. | DeleteCluster | Enum: [KeepCluster DeleteCluster DeleteTaskManager] <br /> |


#### ClusterState

_Underlying type:_ _string_





_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description |
| --- | --- |
| `Creating` |  |
| `Running` |  |
| `Reconciling` |  |
| `Updating` |  |
| `Stopping` |  |
| `PartiallyStopped` |  |
| `Stopped` |  |


#### ComponentState

_Underlying type:_ _string_





_Appears in:_
- [ConfigMapStatus](#configmapstatus)
- [JobManagerIngressStatus](#jobmanageringressstatus)
- [JobManagerServiceStatus](#jobmanagerservicestatus)
- [JobManagerStatus](#jobmanagerstatus)
- [TaskManagerStatus](#taskmanagerstatus)

| Field | Description |
| --- | --- |
| `NotReady` |  |
| `Ready` |  |
| `Updating` |  |
| `Deleted` |  |


#### ConfigMapStatus







_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The resource name of the component. |  |  |
| `state` _[ComponentState](#componentstate)_ | The state of the component. |  |  |


#### DeploymentType

_Underlying type:_ _string_

K8s workload API kind for TaskManager workers



_Appears in:_
- [TaskManagerSpec](#taskmanagerspec)



#### FlinkCluster



FlinkCluster is the Schema for the flinkclusters API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `flinkoperator.k8s.io/v1beta1` | | |
| `kind` _string_ | `FlinkCluster` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[FlinkClusterSpec](#flinkclusterspec)_ |  |  |  |


#### FlinkClusterComponentsStatus



FlinkClusterComponentsStatus defines the observed status of the
components of a FlinkCluster.



_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `configMap` _[ConfigMapStatus](#configmapstatus)_ | The state of configMap. |  |  |
| `jobManager` _[JobManagerStatus](#jobmanagerstatus)_ | The state of JobManager. |  |  |
| `jobManagerService` _[JobManagerServiceStatus](#jobmanagerservicestatus)_ | The state of JobManager service. |  |  |
| `jobManagerIngress` _[JobManagerIngressStatus](#jobmanageringressstatus)_ | The state of JobManager ingress. |  |  |
| `taskManager` _[TaskManagerStatus](#taskmanagerstatus)_ | The state of TaskManager. |  |  |
| `job` _[JobStatus](#jobstatus)_ | The status of the job, available only when JobSpec is provided. |  |  |


#### FlinkClusterControlStatus



Control state



_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Control name |  |  |
| `details` _object (keys:string, values:string)_ | Control data |  |  |
| `state` _string_ | State |  |  |
| `message` _string_ | Message |  |  |
| `updateTime` _string_ | State update time |  |  |


#### FlinkClusterSpec



FlinkClusterSpec defines the desired state of FlinkCluster



_Appears in:_
- [FlinkCluster](#flinkcluster)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `flinkVersion` _string_ | The version of Flink to be managed. This version must match the version in the image. |  |  |
| `image` _[ImageSpec](#imagespec)_ | Flink image for JobManager, TaskManager and job containers. |  |  |
| `serviceAccountName` _string_ | _(Optional)_ The service account assigned to JobManager, TaskManager and Job submitter Pods. If empty, the default service account in the namespace will be used. |  |  |
| `batchSchedulerName` _string_ | Deprecated: BatchSchedulerName specifies the batch scheduler name for JobManager, TaskManager.<br />If empty, no batch scheduling is enabled. |  |  |
| `batchScheduler` _[BatchSchedulerSpec](#batchschedulerspec)_ | _(Optional)_ BatchScheduler specifies the batch scheduler for JobManager, TaskManager.<br />If empty, no batch scheduling is enabled. |  |  |
| `podDisruptionBudget` _[PodDisruptionBudgetSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#poddisruptionbudgetspec-v1-policy)_ | _(Optional)_ Defines the PodDisruptionBudget for JobManager and TaskManager.<br />If empty, no PodDisruptionBudget is created. |  |  |
| `jobManager` _[JobManagerSpec](#jobmanagerspec)_ | _(Optional)_ Flink JobManager spec. | \{ replicas:1 \} |  |
| `taskManager` _[TaskManagerSpec](#taskmanagerspec)_ | _(Optional)_ Flink TaskManager spec. | \{ replicas:3 \} |  |
| `job` _[JobSpec](#jobspec)_ | _(Optional)_ Job spec. If specified, this cluster is an ephemeral Job<br />Cluster, which will be automatically terminated after the job finishes;<br />otherwise, it is a long-running Session Cluster. |  |  |
| `envVars` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvar-v1-core) array_ | _(Optional)_ Environment variables shared by all JobManager, TaskManager and job<br />containers.<br />[More info](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/) |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envfromsource-v1-core) array_ | _(Optional)_ Environment variables injected from a source, shared by all JobManager,<br />TaskManager and job containers.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables) |  |  |
| `flinkProperties` _object (keys:string, values:string)_ | _(Optional)_ Flink properties which are appended to `flink-conf.yaml` (Flink 1.X) or `config.yaml` (Flink 2.X). |  |  |
| `hadoopConfig` _[HadoopConfig](#hadoopconfig)_ | _(Optional)_ Config for Hadoop. |  |  |
| `gcpConfig` _[GCPConfig](#gcpconfig)_ | _(Optional)_ Config for GCP. |  |  |
| `logConfig` _object (keys:string, values:string)_ | _(Optional)_ The logging configuration, which should have keys 'log4j-console.properties' and 'logback-console.xml'.<br />These will end up in the 'flink-config-volume' ConfigMap, which gets mounted at /opt/flink/conf.<br />If not provided, defaults that log to console only will be used.<br /><br> - log4j-console.properties: The contents of the log4j properties file to use. If not provided, a default that logs only to stdout will be provided.<br /><br> - logback-console.xml: The contents of the logback XML file to use. If not provided, a default that logs only to stdout will be provided.<br /><br> - Other arbitrary keys are also allowed, and will become part of the ConfigMap. |  |  |
| `revisionHistoryLimit` _integer_ | The maximum number of revision history to keep, default: 10. |  |  |
| `recreateOnUpdate` _boolean_ | Recreate components when updating flinkcluster, default: true. | true |  |




#### GCPConfig



GCPConfig defines configs for GCP.



_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceAccount` _[GCPServiceAccount](#gcpserviceaccount)_ | GCP service account. |  |  |


#### GCPServiceAccount



GCPServiceAccount defines the config about GCP service account.



_Appears in:_
- [GCPConfig](#gcpconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ | The name of the Secret holding the GCP service account key file.<br />The Secret must be in the same namespace as the FlinkCluster. |  |  |
| `keyFile` _string_ | The name of the service account key file. |  |  |
| `mountPath` _string_ | The path where to mount the Volume of the Secret. |  |  |


#### HadoopConfig



HadoopConfig defines configs for Hadoop.



_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `configMapName` _string_ | The name of the ConfigMap which contains the Hadoop config files.<br />The ConfigMap must be in the same namespace as the FlinkCluster. |  | MinLength: 1 <br /> |
| `mountPath` _string_ | The path where to mount the Volume of the ConfigMap.<br />default: `/etc/hadoop/conf`. | /etc/hadoop/conf |  |


#### HorizontalPodAutoscalerSpec







_Appears in:_
- [TaskManagerSpec](#taskmanagerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `minReplicas` _integer_ | minReplicas is the lower limit for the number of replicas to which the autoscaler<br />can scale down.  It defaults to 1 pod.  minReplicas is allowed to be 0 if the<br />alpha feature gate HPAScaleToZero is enabled and at least one Object or External<br />metric is configured.  Scaling is active as long as at least one metric value is<br />available. |  |  |
| `maxReplicas` _integer_ | maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale up.<br />It cannot be less that minReplicas. |  |  |
| `metrics` _[MetricSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#metricspec-v2-autoscaling) array_ | metrics contains the specifications for which to use to calculate the<br />desired replica count (the maximum replica count across all metrics will<br />be used).  The desired replica count is calculated multiplying the<br />ratio between the target value and the current value by the current<br />number of pods.  Ergo, metrics used must decrease as the pod count is<br />increased, and vice-versa.  See the individual metric source types for<br />more information about how each type of metric must respond.<br />If not set, the default metric will be set to 80% average CPU utilization. |  |  |
| `behavior` _[HorizontalPodAutoscalerBehavior](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#horizontalpodautoscalerbehavior-v2-autoscaling)_ | behavior configures the scaling behavior of the target<br />in both Up and Down directions (scaleUp and scaleDown fields respectively).<br />If not set, the default HPAScalingRules for scale up and scale down are used. |  |  |


#### ImageSpec



ImageSpec defines Flink image of JobManager and TaskManager containers.



_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Flink image name. |  | MinLength: 1 <br /> |
| `pullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#pullpolicy-v1-core)_ | Image pull policy. One of `Always, Never, IfNotPresent`, default: `Always`.<br />if :latest tag is specified, or IfNotPresent otherwise.<br />[More info](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy) | Always | Enum: [Always Never IfNotPresent] <br /> |
| `pullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#localobjectreference-v1-core) array_ | _(Optional)_ Secrets for image pull.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret) |  |  |


#### JobManagerIngressSpec



JobManagerIngressSpec defines ingress of JobManager



_Appears in:_
- [JobManagerSpec](#jobmanagerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `hostFormat` _string_ | _(Optional)_ Ingress host format. ex) \{\{$clusterName\}\}.example.com |  |  |
| `annotations` _object (keys:string, values:string)_ | _(Optional)_Annotations for ingress configuration.<br />[More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |  |  |
| `useTls` _boolean_ | TLS use, default: `false`. | false |  |
| `tlsSecretName` _string_ | _(Optional)_TLS secret name. |  |  |


#### JobManagerIngressStatus



JobManagerIngressStatus defines the status of a JobManager ingress.



_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The name of the Kubernetes ingress resource. |  |  |
| `state` _[ComponentState](#componentstate)_ | The state of the component. |  |  |
| `urls` _string array_ | The URLs of ingress. |  |  |


#### JobManagerPorts



JobManagerPorts defines ports of JobManager.



_Appears in:_
- [JobManagerSpec](#jobmanagerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `rpc` _integer_ | RPC port, default: `6123`. | 6123 | Maximum: 65535 <br />Minimum: 1 <br /> |
| `blob` _integer_ | Blob port, default: `6124`. | 6124 | Maximum: 65535 <br />Minimum: 1 <br /> |
| `query` _integer_ | Query port, default: `6125`. | 6125 | Maximum: 65535 <br />Minimum: 1 <br /> |
| `ui` _integer_ | UI port, default: `8081`. | 8081 | Maximum: 65535 <br />Minimum: 1 <br /> |


#### JobManagerServiceStatus



JobManagerServiceStatus defines the observed state of FlinkCluster



_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The name of the Kubernetes jobManager service. |  |  |
| `state` _[ComponentState](#componentstate)_ | The state of the component. |  |  |
| `nodePort` _integer_ | (Optional) The node port, present when `accessScope` is `NodePort`. |  |  |
| `loadBalancerIngress` _[LoadBalancerIngress](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#loadbalanceringress-v1-core) array_ | (Optional) The load balancer ingress, present when `accessScope` is `VPC` or `External` |  |  |


#### JobManagerSpec



JobManagerSpec defines properties of JobManager.



_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ | The number of JobManager replicas, default: `1` | 1 | Maximum: 1 <br />Minimum: 1 <br /> |
| `accessScope` _string_ | Access scope, default: `Cluster`.<br />`Cluster`: accessible from within the same cluster.<br />`VPC`: accessible from within the same VPC.<br />`External`: accessible from the internet.<br />`NodePort`: accessible through node port.<br />`Headless`: pod IPs assumed to be routable and advertised directly with `clusterIP: None``.<br />Currently `VPC, External` are only available for GKE. | Cluster | Enum: [Cluster VPC External NodePort Headless] <br /> |
| `ServiceAnnotations` _object (keys:string, values:string)_ | _(Optional)_ Define JobManager Service annotations for configuration. |  |  |
| `ServiceLabels` _object (keys:string, values:string)_ | _(Optional)_ Define JobManager Service labels for configuration. |  |  |
| `ingress` _[JobManagerIngressSpec](#jobmanageringressspec)_ | _(Optional)_ Provide external access to JobManager UI/API. |  |  |
| `ports` _[JobManagerPorts](#jobmanagerports)_ | Ports that JobManager listening on. | \{ blob:6124 query:6125 rpc:6123 ui:8081 \} |  |
| `extraPorts` _[NamedPort](#namedport) array_ | _(Optional)_ Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on.<br />Each port number and name must be unique among ports and extraPorts. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Compute resources required by each JobManager container.<br />default: 2 CPUs with 2Gi Memory.<br />It Cannot be updated.<br />[More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/) | \{ limits:map[cpu:2 memory:2Gi] requests:map[cpu:200m memory:512Mi] \} |  |
| `memoryOffHeapRatio` _integer_ | Percentage of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `25` |  |  |
| `memoryOffHeapMin` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#quantity-resource-api)_ | Minimum amount of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `600M`<br />You can express this value like 600M, 572Mi and 600e6<br />[More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) |  |  |
| `memoryProcessRatio` _integer_ | For Flink 1.10+. Percentage of memory process, as a safety margin to avoid OOM kill, default: `80` |  |  |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | _(Optional)_ Volumes in the JobManager pod.<br />[More info](https://kubernetes.io/docs/concepts/storage/volumes/) |  |  |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | _(Optional)_ Volume mounts in the JobManager container.<br />[More info](https://kubernetes.io/docs/concepts/storage/volumes/) |  |  |
| `volumeClaimTemplates` _[PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#persistentvolumeclaim-v1-core) array_ | _(Optional)_ A template for persistent volume claim each requested and mounted to JobManager pod,<br />This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend).<br />[More info](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims) |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Init containers of the Job Manager pod.<br />[More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#affinity-v1-core)_ | _(Optional)_ Defines the affinity of the JobManager pod<br />[More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | _(Optional)_ Selector which must match a node's labels for the JobManager pod to be<br />scheduled on that node.<br />[More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/) |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | _(Optional)_ Defines the node affinity of the JobManager pod<br />[More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) |  |  |
| `sidecars` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Sidecar containers running alongside with the JobManager container in the pod.<br />[More info](https://kubernetes.io/docs/concepts/containers/) |  |  |
| `podAnnotations` _object (keys:string, values:string)_ | _(Optional)_ JobManager StatefulSet pod template annotations.<br />[More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |  |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | _(Optional)_ SecurityContext of the JobManager pod.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod) |  |  |
| `podLabels` _object (keys:string, values:string)_ | _(Optional)_ JobManager StatefulSet pod template labels.<br />[More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Container liveness probe<br />If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L113-L123) will be used.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Container readiness probe<br />If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L129-L139) will be used.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) |  |  |
| `hostAliases` _[HostAlias](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#hostalias-v1-core) array_ | _(Optional)_ Adding entries to JobManager pod /etc/hosts with HostAliases<br />[More info](https://kubernetes.io/docs/tasks/network/customize-hosts-file-for-pods/) |  |  |


#### JobManagerStatus







_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The resource name of the component. |  |  |
| `state` _[ComponentState](#componentstate)_ | The state of the component. |  |  |
| `replicas` _integer_ | replicas is the number of desired replicas. |  |  |
| `readyReplicas` _integer_ | readyReplicas is the number of created pods with a Ready Condition. |  |  |
| `ready` _string_ |  |  |  |


#### JobMode

_Underlying type:_ _string_

JobMode defines the running mode for the job.



_Appears in:_
- [JobSpec](#jobspec)

| Field | Description |
| --- | --- |
| `Blocking` |  |
| `Application` |  |
| `Detached` |  |


#### JobRestartPolicy

_Underlying type:_ _string_

JobRestartPolicy defines the restart policy when a job fails.



_Appears in:_
- [JobSpec](#jobspec)

| Field | Description |
| --- | --- |
| `Never` | JobRestartPolicyNever - never restarts a failed job.<br /> |
| `FromSavepointOnFailure` | JobRestartPolicyFromSavepointOnFailure - restart the job from the latest<br />savepoint if available, otherwise do not restart.<br /> |


#### JobSpec



JobSpec defines properties of a Flink job.



_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `classPath` _string array_ | _(Optional)_ Adds URLs to each user code classloader on all nodes in the cluster.<br />The paths must specify a protocol (e.g. file://) and be accessible on all nodes (e.g. by means of a NFS share).<br />The protocol must be supported by the \{@link java.net.URLClassLoader\}.<br />You may add support to more protocol by setting the `java.protocol.handler.pkgs` java option |  |  |
| `jarFile` _string_ | _(Optional)_ JAR file of the job. It could be a local file or remote URI,<br />depending on which protocols (e.g., `https://, gs://`) are supported by the Flink image. |  |  |
| `className` _string_ | _(Optional)_ Fully qualified Java class name of the job. |  |  |
| `pyFile` _string_ | _(Optional)_ Python file of the job. It could be a local file or remote URI (e.g.,`https://`, `gs://`). |  |  |
| `pyFiles` _string_ | _(Optional)_ Python files of the job. It could be a local file (with .py/.egg/.zip/.whl), directory or remote URI (e.g.,`https://`, `gs://`).<br />See the Flink argument `--pyFiles` for the detail. |  |  |
| `pyModule` _string_ | _(Optional)_ Python module path of the job entry point. Must use with pythonFiles. |  |  |
| `args` _string array_ | _(Optional)_ Command-line args of the job. |  |  |
| `fromSavepoint` _string_ | _(Optional)_ FromSavepoint where to restore the job from<br />Savepoint where to restore the job from (e.g., gs://my-savepoint/1234).<br />If flink job must be restored from the latest available savepoint when Flink job updating, this field must be unspecified. |  |  |
| `allowNonRestoredState` _boolean_ | Allow non-restored state, default: `false`. | false |  |
| `savepointsDir` _string_ | _(Optional)_ Savepoints dir where to store savepoints of the job. |  |  |
| `savepointFormatType` _[SavepointFormatType](#savepointformattype)_ | _(Optional)_ Savepoint format type, "CANONICAL" or "NATIVE", default: "CANONICAL". Requires Flink 1.15 or later. | CANONICAL | Enum: [CANONICAL NATIVE] <br /> |
| `takeSavepointOnUpdate` _boolean_ | _(Optional)_ Should take savepoint before updating job, default: `true`.<br />If this is set as false, maxStateAgeToRestoreSeconds must be provided to limit the savepoint age to restore. |  |  |
| `maxStateAgeToRestoreSeconds` _integer_ | _(Optional)_ Maximum age of the savepoint that allowed to restore state.<br />This is applied to auto restart on failure, update from stopped state and update without taking savepoint.<br />If nil, job can be restarted only when the latest savepoint is the final job state (created by "stop with savepoint")<br />- that is, only when job can be resumed from the suspended state. |  | Minimum: 0 <br /> |
| `autoSavepointSeconds` _integer_ | _(Optional)_ Automatically take a savepoint to the `savepointsDir` every n seconds. |  |  |
| `savepointGeneration` _integer_ | _(Optional)_ Update this field to `jobStatus.savepointGeneration + 1` for a running job<br />cluster to trigger a new savepoint to `savepointsDir` on demand. |  |  |
| `parallelism` _integer_ | _(Optional)_ Job parallelism; if not set parallelism will be #replicas * #slots. |  |  |
| `noLoggingToStdout` _boolean_ | No logging output to STDOUT, default: `false`. | false |  |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | _(Optional)_ Volumes in the Job pod.<br />[More info](https://kubernetes.io/docs/concepts/storage/volumes/) |  |  |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | _(Optional)_ Volume mounts in the Job container.<br />[More info](https://kubernetes.io/docs/concepts/storage/volumes/) |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Init containers of the Job pod. A typical use case could be using an init<br />container to download a remote job jar to a local path which is<br />referenced by the `jarFile` property.<br />[More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#affinity-v1-core)_ | _(Optional)_ Defines the affinity of the Job submitter pod<br />[More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | _(Optional)_ Selector which must match a node's labels for the Job submitter pod to be<br />scheduled on that node.<br />[More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/) |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | _(Optional)_ Defines the node affinity of the Job submitter pod<br />[More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) |  |  |
| `restartPolicy` _[JobRestartPolicy](#jobrestartpolicy)_ | Restart policy when the job fails, one of `Never, FromSavepointOnFailure`,<br />default: `Never`.<br />`Never` means the operator will never try to restart a failed job, manual<br />cleanup and restart is required.<br />`FromSavepointOnFailure` means the operator will try to restart the failed<br />job from the savepoint recorded in the job status if available; otherwise,<br />the job will stay in failed state. This option is usually used together<br />with `autoSavepointSeconds` and `savepointsDir`. | Never | Enum: [Never FromSavepointOnFailure] <br /> |
| `cleanupPolicy` _[CleanupPolicy](#cleanuppolicy)_ | The action to take after job finishes. | \{ afterJobCancelled:DeleteCluster afterJobFails:KeepCluster afterJobSucceeds:DeleteCluster \} |  |
| `cancelRequested` _boolean_ | Deprecated: _(Optional)_ Request the job to be cancelled. Only applies to running jobs. If<br />`savePointsDir` is provided, a savepoint will be taken before stopping the<br />job. |  |  |
| `podAnnotations` _object (keys:string, values:string)_ | _(Optional)_ Job pod template annotations.<br />[More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |  |  |
| `podLabels` _object (keys:string, values:string)_ | _(Optional)_ Job pod template labels.<br />[More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | _(Optional)_ Compute resources required by each Job container.<br />If omitted, a default value will be used.<br />It Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/ | \{ limits:map[cpu:2 memory:2Gi] requests:map[cpu:200m memory:512Mi] \} |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | _(Optional)_ SecurityContext of the Job pod.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod) |  |  |
| `hostAliases` _[HostAlias](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#hostalias-v1-core) array_ | _(Optional)_ Adding entries to Job pod /etc/hosts with HostAliases<br />[More info](https://kubernetes.io/docs/tasks/network/customize-hosts-file-for-pods/) |  |  |
| `mode` _[JobMode](#jobmode)_ | Job running mode, `"Blocking", "Detached"`, default: `"Detached"` | Detached | Enum: [Detached Blocking Application] <br /> |


#### JobState

_Underlying type:_ _string_

JobState defines states for a Flink job deployment.



_Appears in:_
- [JobStatus](#jobstatus)

| Field | Description |
| --- | --- |
| `Pending` |  |
| `Updating` |  |
| `Restarting` |  |
| `Deploying` |  |
| `DeployFailed` |  |
| `Running` |  |
| `Succeeded` |  |
| `Cancelled` |  |
| `Failed` |  |
| `Lost` |  |
| `Unknown` |  |


#### JobStatus



JobStatus defines the status of a job.



_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _string_ | The ID of the Flink job. |  |  |
| `name` _string_ | The Name of the Flink job. |  |  |
| `submitterName` _string_ | The name of the Kubernetes job resource. |  |  |
| `submitterExitCode` _integer_ | Exit code of the JubSubmitter job resource. |  |  |
| `state` _[JobState](#jobstate)_ | The state of the Flink job deployment. |  |  |
| `fromSavepoint` _string_ | The actual savepoint from which this job started.<br />In case of restart, it might be different from the savepoint in the job<br />spec. |  |  |
| `savepointGeneration` _integer_ | The generation of the savepoint in `savepointsDir` taken by the operator.<br />The value starts from 0 when there is no savepoint and increases by 1 for<br />each successful savepoint. |  |  |
| `savepointLocation` _string_ | Savepoint location. |  |  |
| `savepointTime` _string_ | Last successful savepoint completed timestamp. |  |  |
| `finalSavepoint` _boolean_ | The savepoint recorded in savepointLocation is the final state of the job. |  |  |
| `deployTime` _string_ | The timestamp of the Flink job deployment that creating job submitter. |  |  |
| `startTime` _string_ | The Flink job started timestamp. |  |  |
| `restartCount` _integer_ | The number of restarts. |  |  |
| `completionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#time-v1-meta)_ | Job completion time. Present when job is terminated regardless of its state. |  |  |
| `failureReasons` _string array_ | Reasons for the job failure. Present if job state is Failure |  |  |


#### NamedPort



NamedPort defines the container port properties.



_Appears in:_
- [JobManagerSpec](#jobmanagerspec)
- [TaskManagerSpec](#taskmanagerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | _(Optional)_ If specified, this must be an IANA_SVC_NAME and unique within the pod. Each<br />named port in a pod must have a unique name. Name for the port that can be<br />referred to by services. |  |  |
| `containerPort` _integer_ | Number of port to expose on the pod's IP address.<br />This must be a valid port number, 0 < x < 65536. |  | Maximum: 65535 <br />Minimum: 1 <br /> |
| `protocol` _string_ | Protocol for port. One of `UDP, TCP, or SCTP`, default: `TCP`. |  | Enum: [TCP UDP SCTP] <br /> |


#### RevisionStatus







_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `currentRevision` _string_ | CurrentRevision indicates the version of FlinkCluster. |  |  |
| `nextRevision` _string_ | NextRevision indicates the version of FlinkCluster updating. |  |  |
| `collisionCount` _integer_ | collisionCount is the count of hash collisions for the FlinkCluster. The controller<br />uses this field as a collision avoidance mechanism when it needs to create the name for the<br />newest ControllerRevision. |  |  |


#### SavepointFormatType

_Underlying type:_ _string_

SavepointFormatType specifies the binary format of a savepoint.



_Appears in:_
- [JobSpec](#jobspec)
- [SavepointStatus](#savepointstatus)

| Field | Description |
| --- | --- |
| `CANONICAL` |  |
| `NATIVE` |  |


#### SavepointReason

_Underlying type:_ _string_

Savepoint status



_Appears in:_
- [SavepointStatus](#savepointstatus)

| Field | Description |
| --- | --- |
| `user requested` |  |
| `job cancel` |  |
| `scheduled` |  |
| `update` |  |


#### SavepointStatus



SavepointStatus is the status of savepoint progress.



_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `jobID` _string_ | The ID of the Flink job. |  |  |
| `triggerID` _string_ | Savepoint trigger ID. |  |  |
| `triggerTime` _string_ | Savepoint triggered time. |  |  |
| `triggerReason` _[SavepointReason](#savepointreason)_ | Savepoint triggered reason. |  |  |
| `requestTime` _string_ | Savepoint status update time. |  |  |
| `state` _string_ | Savepoint state. |  |  |
| `formatType` _[SavepointFormatType](#savepointformattype)_ | Savepoint format type. |  |  |
| `message` _string_ | Savepoint message. |  |  |


#### TaskManagerPorts



TaskManagerPorts defines ports of TaskManager.



_Appears in:_
- [TaskManagerSpec](#taskmanagerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `data` _integer_ | Data port, default: `6121`. | 6121 | Maximum: 65535 <br />Minimum: 1 <br /> |
| `rpc` _integer_ | RPC port, default: `6122`. | 6122 | Maximum: 65535 <br />Minimum: 1 <br /> |
| `query` _integer_ | Query port, default: `6125`. | 6125 | Maximum: 65535 <br />Minimum: 1 <br /> |


#### TaskManagerSpec



TaskManagerSpec defines properties of TaskManager.



_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `deploymentType` _[DeploymentType](#deploymenttype)_ | _(Optional)_ Defines the replica workload's type: `StatefulSet` or `Deployment`. If not specified, the default value is `StatefulSet`. | StatefulSet |  |
| `replicas` _integer_ | The number of replicas. default: `3` | 3 | Minimum: 1 <br /> |
| `ports` _[TaskManagerPorts](#taskmanagerports)_ | Ports that TaskManager listening on. | \{ data:6121 query:6125 rpc:6122 \} |  |
| `extraPorts` _[NamedPort](#namedport) array_ | _(Optional)_ Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on. |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Compute resources required by each TaskManager container.<br />default: 2 CPUs with 2Gi Memory.<br />It Cannot be updated.<br />[More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/) | \{ limits:map[cpu:2 memory:2Gi] requests:map[cpu:200m memory:512Mi] \} |  |
| `memoryOffHeapRatio` _integer_ | Percentage of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `25` |  |  |
| `memoryOffHeapMin` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#quantity-resource-api)_ | Minimum amount of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `600M`<br />You can express this value like 600M, 572Mi and 600e6<br />[More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) |  |  |
| `memoryProcessRatio` _integer_ | For Flink 1.10+. Percentage of process memory, as a safety margin to avoid OOM kill, default: `20` |  |  |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | _(Optional)_ Volumes in the TaskManager pods.<br />[More info](https://kubernetes.io/docs/concepts/storage/volumes/) |  |  |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | _(Optional)_ Volume mounts in the TaskManager containers.<br />[More info](https://kubernetes.io/docs/concepts/storage/volumes/) |  |  |
| `volumeClaimTemplates` _[PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#persistentvolumeclaim-v1-core) array_ | _(Optional)_ A template for persistent volume claim each requested and mounted to TaskManager pod,<br />This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend).<br />[More info](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)<br />If deploymentType: StatefulSet is used, these templates will be added to the taskManager statefulset template,<br />hence mounting persistent-pvcs to the indexed statefulset pods.<br />If deploymentType: Deployment is used, these templates are appended to the Ephemeral Volumes in the PodSpec,<br />hence mounting ephemeral-pvcs to the replicaset pods. |  |  |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Init containers of the Task Manager pod.<br />[More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#affinity-v1-core)_ | _(Optional)_ Defines the affinity of the Task Manager pod<br />[More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity) |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | _(Optional)_ Selector which must match a node's labels for the Task Manager pod to be<br />scheduled on that node.<br />[More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/) |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | _(Optional)_ Defines the node affinity of the Task Manager pod<br />[More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) |  |  |
| `sidecars` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Sidecar containers running alongside with the TaskManager container in the pod.<br />[More info](https://kubernetes.io/docs/concepts/containers/) |  |  |
| `podAnnotations` _object (keys:string, values:string)_ | _(Optional)_ TaskManager StatefulSet pod template annotations.<br />[More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |  |  |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | _(Optional)_ SecurityContext of the TaskManager pod.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod) |  |  |
| `podLabels` _object (keys:string, values:string)_ | _(Optional)_ TaskManager StatefulSet pod template labels.<br />[More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Container liveness probe<br />If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L177-L187) will be used.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Container readiness probe<br />If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L193-L203) will be used.<br />[More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) |  |  |
| `hostAliases` _[HostAlias](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#hostalias-v1-core) array_ | _(Optional)_ Adding entries to TaskManager pod /etc/hosts with HostAliases<br />[More info](https://kubernetes.io/docs/tasks/network/customize-hosts-file-for-pods/) |  |  |
| `horizontalPodAutoscaler` _[HorizontalPodAutoscalerSpec](#horizontalpodautoscalerspec)_ | _(Optional)_ HorizontalPodAutoscaler for TaskManager.<br />[More info](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) |  |  |


#### TaskManagerStatus







_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | The resource name of the component. |  |  |
| `state` _[ComponentState](#componentstate)_ | The state of the component. |  |  |
| `replicas` _integer_ | replicas is the number of desired Pods. |  |  |
| `readyReplicas` _integer_ | readyReplicas is the number of created pods with a Ready Condition. |  |  |
| `ready` _string_ |  |  |  |
| `selector` _string_ |  |  |  |




