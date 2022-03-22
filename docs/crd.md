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

| Field | Description |
| --- | --- |
| `name` _string_ | BatchScheduler name. |
| `queue` _string_ | _(Optional)_ Queue defines the queue in which resources will be allocates; if queue is not specified, resources will be allocated in the schedulers default queue. |
| `priorityClassName` _string_ | _(Optional)_ If specified, indicates the PodGroup's priority. "system-node-critical" and "system-cluster-critical" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the priority will be default or zero if there is no default. |


#### CleanupPolicy



CleanupPolicy defines the action to take after job finishes. Use one of `KeepCluster, DeleteCluster, DeleteTaskManager` for the below fields.

_Appears in:_
- [JobSpec](#jobspec)

| Field | Description |
| --- | --- |
| `afterJobSucceeds` _CleanupAction_ | Action to take after job succeeds, default: `DeleteCluster`. |
| `afterJobFails` _CleanupAction_ | Action to take after job fails, default: `KeepCluster`. |
| `afterJobCancelled` _CleanupAction_ | Action to take after job is cancelled, default: `DeleteCluster`. |


#### FlinkCluster



FlinkCluster is the Schema for the flinkclusters API



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `flinkoperator.k8s.io/v1beta1`
| `kind` _string_ | `FlinkCluster`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[FlinkClusterSpec](#flinkclusterspec)_ |  |


#### FlinkClusterComponentState



FlinkClusterComponentState defines the observed state of a component of a FlinkCluster.

_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description |
| --- | --- |
| `name` _string_ | The resource name of the component. |
| `state` _string_ | The state of the component. |


#### FlinkClusterComponentsStatus



FlinkClusterComponentsStatus defines the observed status of the components of a FlinkCluster.

_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description |
| --- | --- |
| `configMap` _[FlinkClusterComponentState](#flinkclustercomponentstate)_ | The state of configMap. |
| `jobManagerStatefulSet` _[FlinkClusterComponentState](#flinkclustercomponentstate)_ | The state of JobManager StatefulSet. |
| `jobManagerService` _[JobManagerServiceStatus](#jobmanagerservicestatus)_ | The state of JobManager service. |
| `jobManagerIngress` _[JobManagerIngressStatus](#jobmanageringressstatus)_ | The state of JobManager ingress. |
| `taskManagerStatefulSet` _[FlinkClusterComponentState](#flinkclustercomponentstate)_ | The state of TaskManager StatefulSet. |
| `job` _[JobStatus](#jobstatus)_ | The status of the job, available only when JobSpec is provided. |


#### FlinkClusterControlStatus



Control state

_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description |
| --- | --- |
| `name` _string_ | Control name |
| `details` _object (keys:string, values:string)_ | Control data |
| `state` _string_ | State |
| `message` _string_ | Message |
| `updateTime` _string_ | State update time |


#### FlinkClusterSpec



FlinkClusterSpec defines the desired state of FlinkCluster

_Appears in:_
- [FlinkCluster](#flinkcluster)

| Field | Description |
| --- | --- |
| `flinkVersion` _string_ | The version of Flink to be managed. This version must match the version in the image. |
| `image` _[ImageSpec](#imagespec)_ | Flink image for JobManager, TaskManager and job containers. |
| `serviceAccountName` _string_ | _(Optional)_ The service account assigned to JobManager, TaskManager and Job submitter Pods. If empty, the default service account in the namespace will be used. |
| `batchSchedulerName` _string_ | Deprecated: BatchSchedulerName specifies the batch scheduler name for JobManager, TaskManager. If empty, no batch scheduling is enabled. |
| `batchScheduler` _[BatchSchedulerSpec](#batchschedulerspec)_ | _(Optional)_ BatchScheduler specifies the batch scheduler for JobManager, TaskManager. If empty, no batch scheduling is enabled. |
| `jobManager` _[JobManagerSpec](#jobmanagerspec)_ | _(Optional)_ Flink JobManager spec. |
| `taskManager` _[TaskManagerSpec](#taskmanagerspec)_ | _(Optional)_ Flink TaskManager spec. |
| `job` _[JobSpec](#jobspec)_ | _(Optional)_ Job spec. If specified, this cluster is an ephemeral Job Cluster, which will be automatically terminated after the job finishes; otherwise, it is a long-running Session Cluster. |
| `envVars` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envvar-v1-core) array_ | _(Optional)_ Environment variables shared by all JobManager, TaskManager and job containers. [More info](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/) |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#envfromsource-v1-core) array_ | _(Optional)_ Environment variables injected from a source, shared by all JobManager, TaskManager and job containers. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables) |
| `flinkProperties` _object (keys:string, values:string)_ | _(Optional)_ Flink properties which are appened to flink-conf.yaml. |
| `hadoopConfig` _[HadoopConfig](#hadoopconfig)_ | _(Optional)_ Config for Hadoop. |
| `gcpConfig` _[GCPConfig](#gcpconfig)_ | _(Optional)_ Config for GCP. |
| `logConfig` _object (keys:string, values:string)_ | _(Optional)_ The logging configuration, which should have keys 'log4j-console.properties' and 'logback-console.xml'. These will end up in the 'flink-config-volume' ConfigMap, which gets mounted at /opt/flink/conf. If not provided, defaults that log to console only will be used. <br> - log4j-console.properties: The contents of the log4j properties file to use. If not provided, a default that logs only to stdout will be provided. <br> - logback-console.xml: The contents of the logback XML file to use. If not provided, a default that logs only to stdout will be provided. <br> - Other arbitrary keys are also allowed, and will become part of the ConfigMap. |
| `revisionHistoryLimit` _integer_ | The maximum number of revision history to keep, default: 10. |
| `recreateOnUpdate` _boolean_ | Recreate components when updating flinkcluster, default: true. |




#### GCPConfig



GCPConfig defines configs for GCP.

_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description |
| --- | --- |
| `serviceAccount` _[GCPServiceAccount](#gcpserviceaccount)_ | GCP service account. |


#### GCPServiceAccount



GCPServiceAccount defines the config about GCP service account.

_Appears in:_
- [GCPConfig](#gcpconfig)

| Field | Description |
| --- | --- |
| `secretName` _string_ | The name of the Secret holding the GCP service account key file. The Secret must be in the same namespace as the FlinkCluster. |
| `keyFile` _string_ | The name of the service account key file. |
| `mountPath` _string_ | The path where to mount the Volume of the Secret. |


#### HadoopConfig



HadoopConfig defines configs for Hadoop.

_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description |
| --- | --- |
| `configMapName` _string_ | The name of the ConfigMap which contains the Hadoop config files. The ConfigMap must be in the same namespace as the FlinkCluster. |
| `mountPath` _string_ | The path where to mount the Volume of the ConfigMap. default: `/etc/hadoop/conf`. |


#### ImageSpec



ImageSpec defines Flink image of JobManager and TaskManager containers.

_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Flink image name. |
| `pullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#pullpolicy-v1-core)_ | Image pull policy. One of `Always, Never, IfNotPresent`, default: `Always`. if :latest tag is specified, or IfNotPresent otherwise. [More info](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy) |
| `pullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#localobjectreference-v1-core) array_ | _(Optional)_ Secrets for image pull. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret) |


#### JobManagerIngressSpec



JobManagerIngressSpec defines ingress of JobManager

_Appears in:_
- [JobManagerSpec](#jobmanagerspec)

| Field | Description |
| --- | --- |
| `hostFormat` _string_ | _(Optional)_ Ingress host format. ex) {{$clusterName}}.example.com |
| `annotations` _object (keys:string, values:string)_ | _(Optional)_Annotations for ingress configuration. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |
| `useTls` _boolean_ | TLS use, default: `false`. |
| `tlsSecretName` _string_ | _(Optional)_TLS secret name. |


#### JobManagerIngressStatus



JobManagerIngressStatus defines the status of a JobManager ingress.

_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description |
| --- | --- |
| `name` _string_ | The name of the Kubernetes ingress resource. |
| `state` _string_ | The state of the component. |
| `urls` _string array_ | The URLs of ingress. |


#### JobManagerPorts



JobManagerPorts defines ports of JobManager.

_Appears in:_
- [JobManagerSpec](#jobmanagerspec)

| Field | Description |
| --- | --- |
| `rpc` _integer_ | RPC port, default: `6123`. |
| `blob` _integer_ | Blob port, default: `6124`. |
| `query` _integer_ | Query port, default: `6125`. |
| `ui` _integer_ | UI port, default: `8081`. |


#### JobManagerServiceStatus



JobManagerServiceStatus defines the observed state of FlinkCluster

_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description |
| --- | --- |
| `name` _string_ | The name of the Kubernetes jobManager service. |
| `state` _string_ | The state of the component. |
| `nodePort` _integer_ | (Optional) The node port, present when `accessScope` is `NodePort`. |
| `loadBalancerIngress` _[LoadBalancerIngress](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#loadbalanceringress-v1-core) array_ | (Optional) The load balancer ingress, present when `accessScope` is `VPC` or `External` |


#### JobManagerSpec



JobManagerSpec defines properties of JobManager.

_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description |
| --- | --- |
| `replicas` _integer_ | The number of JobManager replicas, default: `1` |
| `accessScope` _string_ | Access scope, default: `Cluster`. `Cluster`: accessible from within the same cluster. `VPC`: accessible from within the same VPC. `External`: accessible from the internet. `NodePort`: accessible through node port. `Headless`: pod IPs assumed to be routable and advertised directly with `clusterIP: None``. Currently `VPC, External` are only available for GKE. |
| `ingress` _[JobManagerIngressSpec](#jobmanageringressspec)_ | _(Optional)_ Provide external access to JobManager UI/API. |
| `ports` _[JobManagerPorts](#jobmanagerports)_ | Ports that JobManager listening on. |
| `extraPorts` _[NamedPort](#namedport) array_ | _(Optional)_ Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on. Each port number and name must be unique among ports and extraPorts. |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Compute resources required by each JobManager container. default: 2 CPUs with 2Gi Memory. It Cannot be updated. [More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/) |
| `memoryOffHeapRatio` _integer_ | Percentage of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `25` |
| `memoryOffHeapMin` _Quantity_ | Minimum amount of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `600M` You can express this value like 600M, 572Mi and 600e6 [More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) |
| `memoryProcessRatio` _integer_ | For Flink 1.10+. Percentage of memory process, as a safety margin to avoid OOM kill, default: `80` |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | _(Optional)_ Volumes in the JobManager pod. [More info](https://kubernetes.io/docs/concepts/storage/volumes/) |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | _(Optional)_ Volume mounts in the JobManager container. [More info](https://kubernetes.io/docs/concepts/storage/volumes/) |
| `volumeClaimTemplates` _[PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#persistentvolumeclaim-v1-core) array_ | _(Optional)_ A template for persistent volume claim each requested and mounted to JobManager pod, This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend). [More info](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims) |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Init containers of the Job Manager pod. [More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) |
| `nodeSelector` _object (keys:string, values:string)_ | _(Optional)_ Selector which must match a node's labels for the JobManager pod to be scheduled on that node. [More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/) |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | _(Optional)_ Defines the node affinity of the pod [More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) |
| `sidecars` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Sidecar containers running alongside with the JobManager container in the pod. [More info](https://kubernetes.io/docs/concepts/containers/) |
| `podAnnotations` _object (keys:string, values:string)_ | _(Optional)_ JobManager StatefulSet pod template annotations. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | _(Optional)_ SecurityContext of the JobManager pod. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod) |
| `podLabels` _object (keys:string, values:string)_ | _(Optional)_ JobManager StatefulSet pod template labels. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Container liveness probe If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L113-L123) will be used. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Container readiness probe If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L129-L139) will be used. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) |
| `ServiceAnnotations` _object (keys:string, values:string)_ | _(Optional)_ JobManager Service annotations. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |
| `ServiceLabels` _object (keys:string, values:string)_ | _(Optional)_ JobManager Service labels. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |


#### JobSpec



JobSpec defines properties of a Flink job.

_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description |
| --- | --- |
| `jarFile` _string_ | _(Optional)_ JAR file of the job. It could be a local file or remote URI, depending on which protocols (e.g., `https://, gs://`) are supported by the Flink image. |
| `className` _string_ | _(Optional)_ Fully qualified Java class name of the job. |
| `pyFile` _string_ | _(Optional)_ Python file of the job. It could be a local file or remote URI (e.g.,`https://`, `gs://`). |
| `pyFiles` _string_ | _(Optional)_ Python files of the job. It could be a local file (with .py/.egg/.zip/.whl), directory or remote URI (e.g.,`https://`, `gs://`). See the Flink argument `--pyFiles` for the detail. |
| `pyModule` _string_ | _(Optional)_ Python module path of the job entry point. Must use with pythonFiles. |
| `args` _string array_ | _(Optional)_ Command-line args of the job. |
| `fromSavepoint` _string_ | _(Optional)_ FromSavepoint where to restore the job from Savepoint where to restore the job from (e.g., gs://my-savepoint/1234). If flink job must be restored from the latest available savepoint when Flink job updating, this field must be unspecified. |
| `allowNonRestoredState` _boolean_ | Allow non-restored state, default: `false`. |
| `savepointsDir` _string_ | _(Optional)_ Savepoints dir where to store savepoints of the job. |
| `takeSavepointOnUpdate` _boolean_ | _(Optional)_ Should take savepoint before updating job, default: `true`. If this is set as false, maxStateAgeToRestoreSeconds must be provided to limit the savepoint age to restore. |
| `maxStateAgeToRestoreSeconds` _integer_ | _(Optional)_ Maximum age of the savepoint that allowed to restore state. This is applied to auto restart on failure, update from stopped state and update without taking savepoint. If nil, job can be restarted only when the latest savepoint is the final job state (created by "stop with savepoint") - that is, only when job can be resumed from the suspended state. |
| `autoSavepointSeconds` _integer_ | _(Optional)_ Automatically take a savepoint to the `savepointsDir` every n seconds. |
| `savepointGeneration` _integer_ | _(Optional)_ Update this field to `jobStatus.savepointGeneration + 1` for a running job cluster to trigger a new savepoint to `savepointsDir` on demand. |
| `parallelism` _integer_ | Job parallelism, default: `1`. |
| `noLoggingToStdout` _boolean_ | No logging output to STDOUT, default: `false`. |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | _(Optional)_ Volumes in the Job pod. [More info](https://kubernetes.io/docs/concepts/storage/volumes/) |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | _(Optional)_ Volume mounts in the Job container. [More info](https://kubernetes.io/docs/concepts/storage/volumes/) |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Init containers of the Job pod. A typical use case could be using an init container to download a remote job jar to a local path which is referenced by the `jarFile` property. [More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) |
| `nodeSelector` _object (keys:string, values:string)_ | _(Optional)_ Selector which must match a node's labels for the Job submitter pod to be scheduled on that node. [More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/) |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | _(Optional)_ Defines the node affinity of the Job submitter pod [More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) |
| `restartPolicy` _JobRestartPolicy_ | Restart policy when the job fails, one of `Never, FromSavepointOnFailure`, default: `Never`. `Never` means the operator will never try to restart a failed job, manual cleanup and restart is required. `FromSavepointOnFailure` means the operator will try to restart the failed job from the savepoint recorded in the job status if available; otherwise, the job will stay in failed state. This option is usually used together with `autoSavepointSeconds` and `savepointsDir`. |
| `cleanupPolicy` _[CleanupPolicy](#cleanuppolicy)_ | The action to take after job finishes. |
| `cancelRequested` _boolean_ | Deprecated: _(Optional)_ Request the job to be cancelled. Only applies to running jobs. If `savePointsDir` is provided, a savepoint will be taken before stopping the job. |
| `podAnnotations` _object (keys:string, values:string)_ | _(Optional)_ Job pod template annotations. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |
| `podLabels` _object (keys:string, values:string)_ | _(Optional)_ Job pod template labels. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | _(Optional)_ Compute resources required by each Job container. If omitted, a default value will be used. It Cannot be updated. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/ |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | _(Optional)_ SecurityContext of the Job pod. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod) |
| `mode` _JobMode_ | Job running mode, `"Blocking", "Detached"`, default: `"Detached"` |


#### JobStatus



JobStatus defines the status of a job.

_Appears in:_
- [FlinkClusterComponentsStatus](#flinkclustercomponentsstatus)

| Field | Description |
| --- | --- |
| `id` _string_ | The ID of the Flink job. |
| `name` _string_ | The Name of the Flink job. |
| `submitterName` _string_ | The name of the Kubernetes job resource. |
| `state` _string_ | The state of the Flink job deployment. |
| `fromSavepoint` _string_ | The actual savepoint from which this job started. In case of restart, it might be different from the savepoint in the job spec. |
| `savepointGeneration` _integer_ | The generation of the savepoint in `savepointsDir` taken by the operator. The value starts from 0 when there is no savepoint and increases by 1 for each successful savepoint. |
| `savepointLocation` _string_ | Savepoint location. |
| `savepointTime` _string_ | Last successful savepoint completed timestamp. |
| `finalSavepoint` _boolean_ | The savepoint recorded in savepointLocation is the final state of the job. |
| `deployTime` _string_ | The timestamp of the Flink job deployment that creating job submitter. |
| `startTime` _string_ | The Flink job started timestamp. |
| `restartCount` _integer_ | The number of restarts. |
| `completionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#time-v1-meta)_ | Job completion time. Present when job is terminated regardless of its state. |
| `failureReasons` _string array_ | Reasons for the job failure. Present if job state is Failure |


#### NamedPort



NamedPort defines the container port properties.

_Appears in:_
- [JobManagerSpec](#jobmanagerspec)
- [TaskManagerSpec](#taskmanagerspec)

| Field | Description |
| --- | --- |
| `name` _string_ | _(Optional)_ If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services. |
| `containerPort` _integer_ | Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536. |
| `protocol` _string_ | Protocol for port. One of `UDP, TCP, or SCTP`, default: `TCP`. |


#### RevisionStatus





_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description |
| --- | --- |
| `currentRevision` _string_ | CurrentRevision indicates the version of FlinkCluster. |
| `nextRevision` _string_ | NextRevision indicates the version of FlinkCluster updating. |
| `collisionCount` _integer_ | collisionCount is the count of hash collisions for the FlinkCluster. The controller uses this field as a collision avoidance mechanism when it needs to create the name for the newest ControllerRevision. |


#### SavepointStatus



SavepointStatus is the status of savepoint progress.

_Appears in:_
- [FlinkClusterStatus](#flinkclusterstatus)

| Field | Description |
| --- | --- |
| `jobID` _string_ | The ID of the Flink job. |
| `triggerID` _string_ | Savepoint trigger ID. |
| `triggerTime` _string_ | Savepoint triggered time. |
| `triggerReason` _SavepointReason_ | Savepoint triggered reason. |
| `requestTime` _string_ | Savepoint status update time. |
| `state` _string_ | Savepoint state. |
| `message` _string_ | Savepoint message. |


#### TaskManagerPorts



TaskManagerPorts defines ports of TaskManager.

_Appears in:_
- [TaskManagerSpec](#taskmanagerspec)

| Field | Description |
| --- | --- |
| `data` _integer_ | Data port, default: `6121`. |
| `rpc` _integer_ | RPC port, default: `6122`. |
| `query` _integer_ | Query port, default: `6125`. |


#### TaskManagerSpec



TaskManagerSpec defines properties of TaskManager.

_Appears in:_
- [FlinkClusterSpec](#flinkclusterspec)

| Field | Description |
| --- | --- |
| `replicas` _integer_ | The number of replicas. default: `3` |
| `ports` _[TaskManagerPorts](#taskmanagerports)_ | Ports that TaskManager listening on. |
| `extraPorts` _[NamedPort](#namedport) array_ | _(Optional)_ Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on. |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Compute resources required by each TaskManager container. default: 2 CPUs with 2Gi Memory. It Cannot be updated. [More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/) |
| `memoryOffHeapRatio` _integer_ | Percentage of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `25` |
| `memoryOffHeapMin` _Quantity_ | Minimum amount of off-heap memory in containers, as a safety margin to avoid OOM kill, default: `600M` You can express this value like 600M, 572Mi and 600e6 [More info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) |
| `memoryProcessRatio` _integer_ | For Flink 1.10+. Percentage of process memory, as a safety margin to avoid OOM kill, default: `20` |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volume-v1-core) array_ | _(Optional)_ Volumes in the TaskManager pods. [More info](https://kubernetes.io/docs/concepts/storage/volumes/) |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#volumemount-v1-core) array_ | _(Optional)_ Volume mounts in the TaskManager containers. [More info](https://kubernetes.io/docs/concepts/storage/volumes/) |
| `volumeClaimTemplates` _[PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#persistentvolumeclaim-v1-core) array_ | _(Optional)_ A template for persistent volume claim each requested and mounted to JobManager pod, This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend). [More info](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims) |
| `initContainers` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Init containers of the Task Manager pod. [More info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) |
| `nodeSelector` _object (keys:string, values:string)_ | _(Optional)_ Selector which must match a node's labels for the TaskManager pod to be scheduled on that node. [More info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/) |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | _(Optional)_ Defines the node affinity of the pod [More info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) |
| `sidecars` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#container-v1-core) array_ | _(Optional)_ Sidecar containers running alongside with the TaskManager container in the pod. [More info](https://kubernetes.io/docs/concepts/containers/) |
| `podAnnotations` _object (keys:string, values:string)_ | _(Optional)_ TaskManager StatefulSet pod template annotations. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) |
| `securityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core)_ | _(Optional)_ SecurityContext of the TaskManager pod. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod) |
| `podLabels` _object (keys:string, values:string)_ | _(Optional)_ TaskManager StatefulSet pod template labels. [More info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Container liveness probe If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L177-L187) will be used. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#probe-v1-core)_ | Container readiness probe If omitted, a [default value](https://github.com/spotify/flink-on-k8s-operator/blob/a88ed2b/api/v1beta1/flinkcluster_default.go#L193-L203) will be used. [More info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) |
