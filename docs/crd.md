# FlinkCluster Custom Resource Definition

The Kubernetes Operator for Apache Flink uses [CustomResourceDefinition](https://kubernetes.io/docs/concepts/api-extension/custom-resources/)
named `FlinkCluster` for specifying a Flink job cluster ([sample](../config/samples/flinkoperator_v1beta1_flinkjobcluster.yaml))
or Flink session cluster ([sample](../config/samples/flinkoperator_v1beta1_flinksessioncluster.yaml)), depending on
whether the job spec is specified. Similarly to other kinds of Kubernetes resources, the custom resource consists of a
resource `Metadata`, a specification in a `Spec` field and a `Status` field. The definitions are organized in the
following structure. The v1beta1 version of the API definition is implemented [here](../api/v1beta1/flinkcluster_types.go).

---
## Flink job or session cluster spec.

| Key | Type | Optional | Description
|-----|------|----------|-------------
| `.spec.flinkVersion` | string | X | The version of Flink to be managed. This version must match the version in the image.
| `.spec.image` | object | X | Flink image for JobManager, TaskManager and job containers.
| `.spec.image.name` | string | X | Image name.
| `.spec.image.pullPolicy` | string | O | Image pull policy.
| `.spec.image.pullSecrets` | string array | O | Secrets for image pull.
| *(deprecated)* `.spec.batchSchedulerName` | string | O | BatchSchedulerName specifies the batch scheduler name for JobManager, TaskManager. If empty, no batch scheduling is enabled.
| `.spec.batchScheduler` | object | O | BatchScheduler specifies the batch scheduler properties for Job, JobManager, TaskManager. If empty, no batch scheduling is enabled.
| `.spec.batchScheduler.name` | string | X | Scheduler name.
| `.spec.batchScheduler.queue` | string | O | Queue name in which resources will be allocates.
| `.spec.batchScheduler.priorityClassName` | string | O | If specified, indicates the PodGroup's priority. "system-node-critical" and "system-cluster-critical" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the priority will be default or zero if there is no default.
| `.spec.serviceAccountName` | string | O | the name of the service account(which must already exist in the namespace). If empty, the default service account in the namespace will be used.
| `.spec.jobManager` | object | X | JobManager spec.
| `.spec.jobManager.replicas` | object | X | The number of JobManager replicas.
| `.spec.jobManager.accessScope` | string | X | Access scope of the JobManager service. `enum("Cluster", "VPC", "External", "NodePort", "Headless")`. <br> `Cluster`: accessible from within the same cluster <br> `VPC`: accessible from within the same VPC. <br> `External`: accessible from the internet. <br> `NodePort`: accessible through node port. <br> `Headless`: pod IPs assumed to be routable and advertised directly with `clusterIP: None`. <br> Currently `VPC` and `External` are only available for GKE.
| `.spec.jobManager.ports` | object | O | Ports that JobManager listening on.
| `.spec.jobManager.ports.rpc` | int | O | RPC port, default: 6123.
| `.spec.jobManager.ports.blob` | int | O | Blob port, default: 6124.
| `.spec.jobManager.ports.query` | int | O | Query port, default: 6125.
| `.spec.jobManager.ports.ui` | int | O | UI port, default: 8081.
| `.spec.jobManager.extraPorts` | object array | O | Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on. Each port number and name must be unique among ports and extraPorts. ContainerPort is required and name and protocol are optional.
| `.spec.jobManager.extraPorts.name` | string | O | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
| `.spec.jobManager.extraPorts.containerPort` | int | X | Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.
| `.spec.jobManager.extraPorts.protocol` | string | O | Protocol for port. Must be UDP, TCP, or SCTP. defaults: TCP.
| `.spec.jobManager.ingress` | object | O | Provide external access to JobManager UI/API.
| `.spec.jobManager.ingress.hostFormat` | string | O | Host format for generating URLs. ex) `{{$clusterName}}.example.com`
| `.spec.jobManager.ingress.annotations` | object | O | Annotations for ingress configuration.
| `.spec.jobManager.ingress.useTLS` | bool | O | TLS use, default: false.
| `.spec.jobManager.ingress.tlsSecretName` | string | O | Kubernetes secret resource name for TLS.
| `.spec.jobManager.resources` | object | O | Compute resources required by JobManager container. If omitted, a default value will be used. See [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/) about resources.
| `.spec.jobManager.memoryOffHeapRatio` | int | O | Percentage of off-heap memory in containers, as a safety margin, default: 25
| `.spec.jobManager.memoryOffHeapMin` | string | O | Minimum amount of off-heap memory in containers, as a safety margin, default: 600M. You can express this value like 600M, 572Mi and 600e6. See [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) about value expression.
| `.spec.jobManager.memoryProcessRatio` | int | O | For Flink 1.10+. Percentage of memory process, as a safety margin to avoid OOM kill, default: 80
| `.spec.jobManager.volumes` | object array | O | Volumes in the JobManager pod. See [more info](https://kubernetes.io/docs/concepts/storage/volumes/) about volumes.
| `.spec.jobManager.volumeMounts` | object array | O | Volume mounts in the JobManager container. See [more info](https://kubernetes.io/docs/concepts/storage/volumes/) volume mounts.
| `.spec.jobManager.volumeClaimTemplates` | object array | O | A template for persistent volume claim each requested and mounted to JobManager pod, This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend)
| `.spec.jobManager.initContainers` | object array | O | Init containers of the JobManager pod. See [more info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) about init containers.
| `.spec.jobManager.nodeSelector` | object | O | Selector which must match a node's labels for the JobManager pod to be scheduled on that node. See [more info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
| `.spec.jobManager.tolerations` | object array | O | Allows the JobManager pod to run on a tainted node in the cluster. See [more info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
| `.spec.jobManager.sidecars` | object array | O | Sidecar containers running alongside with the JobManager container in the pod. See [more info](https://kubernetes.io/docs/concepts/containers/) about containers.
| `.spec.jobManager.podAnnotations` | object | O | Pod template annotations for the JobManager StatefulSet. See [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) about annotations.
| `.spec.jobManager.podLabels` | object | O | Pod template labels for the JobManager StatefulSet.
| `.spec.jobManager.securityContext` | object | O | PodSecurityContext for the JobManager pod. See [more info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod).
| `.spec.jobManager.livenessProbe` | object | O | LivenessProbe for the JobManager pod. See [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
| `.spec.jobManager.readinessProbe` | object | O | ReadinessProbe for the JobManager pod. See [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
| `.spec.taskManager` | int | X | TaskManager spec.
| `.spec.taskManager.replicas` | int | X | The number of TaskManager replicas.
| `.spec.taskManager.ports` | object | O | Ports that TaskManager listening on.
| `.spec.taskManager.ports.data` | int | O | Data port.
| `.spec.taskManager.ports.rpc` | int | O | RPC port.
| `.spec.taskManager.ports.query` | int | O | Query port.
| `.spec.taskManager.extraPorts` | object array | O | Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on. Each port number and name must be unique among ports and extraPorts. ContainerPort is required and name and protocol are optional.
| `.spec.taskManager.extraPorts.name` | string | O | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
| `.spec.taskManager.extraPorts.containerPort` | int | X | Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.
| `.spec.taskManager.extraPorts.protocol` | string | O | Protocol for port. Must be UDP, TCP, or SCTP. defaults: TCP.
| `.spec.taskManager.resources` | object | O | Compute resources required by JobManager container. If omitted, a default value will be used. See [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/) about resources.
| `.spec.taskManager.memoryOffHeapRatio` | int | O | Percentage of off-heap memory in containers, as a safety margin, default: 25
| `.spec.taskManager.memoryOffHeapMin` | string | O | Minimum amount of off-heap memory in containers, as a safety margin, default: 600M. You can express this value like 600M, 572Mi and 600e6. See [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) about value expression.
| `.spec.taskManager.memoryProcessRatio` | int | O | For Flink 1.10+. Percentage of memory process, as a safety margin to avoid OOM kill, default: 80
| `.spec.taskManager.volumes` | object array | O | Volumes in the TaskManager pod. See [more info](https://kubernetes.io/docs/concepts/storage/volumes/) about volumes.
| `.spec.taskManager.volumeMounts` | object array | O | Volume mounts in the TaskManager containers. See [more info](https://kubernetes.io/docs/concepts/storage/volumes/) about volume mounts.
| `.spec.taskManager.volumeClaimTemplates` | object array | O | A template for persistent volume claim each requested and mounted to each TaskManager pod, This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend) See [more info](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) about VolumeClaimTemplates in StatefulSet.
| `.spec.taskManager.initContainers` | object array | O | Init containers of the TaskManager pod. See [more info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) about init containers.
| `.spec.taskManager.nodeSelector` | object | O | Selector which must match a node's labels for the TaskManager pod tobe scheduled on that node. See [more info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
| `.spec.taskManager.tolerations` | object array | O | Allows the TaskManager pod to run on a tainted node in the cluster. See [more info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
| `.spec.taskManager.sidecars` | object array | O | Sidecar containers running alongside with the TaskManager container in the pod. See [more info](https://kubernetes.io/docs/concepts/containers/) about containers.
| `.spec.taskManager.podAnnotations` | object | O | Pod template annotations for the TaskManager StatefulSet. See [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) about annotations.
| `.spec.taskManager.podLabels` | object | O | Pod template labels for the TaskManager StatefulSet.
| `.spec.taskManager.securityContext` | object | O | PodSecurityContext for the TaskManager pods. See [more info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod).
| `.spec.taskManager.livenessProbe` | object | O | LivenessProbe for the TaskManager pod. See [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
| `.spec.taskManager.readinessProbe` | object | O | ReadinessProbe for the TaskManager pod. See [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
| `.spec.job` | string | O | Job spec. If specified, the cluster is a Flink job cluster; otherwise, it is a Flink session cluster.
| `.spec.job.jarFile` | string | O | JAR file of the job. It could be a local file or remote URI, depending on which protocols (e.g., `https://`, `gs://`) are supported by the Flink image.
| `.spec.job.className` | string | O | Fully qualified Java class name of the job.
| `.spec.job.pythonFile` | string | O | Python file of the job. It should be a local file.
| `.spec.job.pythonFiles` | string | O | Python files of the job. It should be a local file (with .py/.egg/.zip/.whl) or directory. See the Flink argument `--pyFiles` for the detail.
| `.spec.job.pythonModule` | string | O | Python module path of the job entry point. Must use with **pythonFiles**.
| `.spec.job.args` | string array | O | Command-line args of the job.
| `.spec.job.fromSavepoint` | string | O | Savepoint where to restore the job from. If Flink job must be restored from the latest available savepoint when Flink job updating, this field must be unspecified.
| `.spec.job.allowNonRestoredState` | bool | O | Allow non-restored state, default: false.
| `.spec.job.takeSavepointOnUpdate` | bool | O | Should take savepoint before updating the job, default: true. If this is set as false, maxStateAgeToRestoreSeconds must be provided to limit the savepoint age to restore.
| `.spec.job.maxStateAgeToRestoreSeconds` | int | O | Maximum age of the savepoint that allowed to restore state. This is applied to auto restart on failure, update from stopped state and update without taking savepoint. If nil, job can be restarted only when the latest savepoint is the final job state (created by "stop with savepoint") - that is, only when job can be resumed from the suspended state.
| `.spec.job.autoSavepointSeconds` | int | O | Automatically take a savepoint to the `savepointsDir` every n seconds.
| `.spec.job.savepointsDir` | string | O | Savepoints dir where to store automatically taken savepoints.
| `.spec.job.savepointGeneration` | int | O | Update this field to `jobStatus.savepointGeneration + 1` for a running job cluster to trigger a new savepoint to `savepointsDir` on demand.
| `.spec.job.parallelism` | int | O | Parallelism of the job, default: 1.
| `.spec.job.noLoggingToStdout` | bool | O | No logging output to STDOUT, default: false.
| `.spec.job.volumes` | object array | O | Volumes in the Job pod. See [more info](https://kubernetes.io/docs/concepts/storage/volumes/) about volumes.
| `.spec.job.volumeMounts` | object array | O | Volume mounts in the Job containers. If there is no confilcts, these mounts will be automatically added to init containers; otherwise, the mounts defined in init containers will take precedence. See [more info](https://kubernetes.io/docs/concepts/storage/volumes/) about volume mounts.
| `.spec.job.initContainers` | object array | O | Init containers of the Job pod. See [more info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) about init containers.
| `.spec.job.restartPolicy` | string | O | Restart policy when the job fails, `enum("Never", "FromSavepointOnFailure")`, default: `"Never"`. `"Never"` means the operator will never try to restart a failed job, manual cleanup is required. `"FromSavepointOnFailure"` means the operator will try to restart the failed job from the savepoint recorded in the job status if available; otherwise, the job will stay in failed state. This option is usually used together with `autoSavepointSeconds` and `savepointsDir`.
| `.spec.job.cleanupPolicy` | object | O | The action to take after job finishes.
| `.spec.job.cleanupPolicy.afterJobSucceeds` | string | O | The action to take after job succeeds, `enum("KeepCluster", "DeleteCluster", "DeleteTaskManager")`, default `"DeleteCluster"`.
| `.spec.job.cleanupPolicy.afterJobFails` | string | O | The action to take after job fails, `enum("KeepCluster", "DeleteCluster", "DeleteTaskManager")`, default `"KeepCluster"`.
| `.spec.job.cleanupPolicy.afterJobCancelled` | string | O | The action to take after job cancelled, `enum("KeepCluster", "DeleteCluster", "DeleteTaskManager")`, default `"DeleteCluster"`.
| `.spec.job.cancelRequested` | bool | O | Request the job to be cancelled. Only applies to running jobs. If `savePointsDir` is provided, a savepoint will be taken before stopping the job.
| `.spec.job.podAnnotations` | object | O | Pod template annotations for the job. See [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) about annotations.
| `.spec.job.podLabels` | object | O | Pod template labels for the job.
| `.spec.job.securityContext` | object | O | PodSecurityContext for the Job pod. See [more info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod).
| `.spec.job.resources` | object | O | Compute resources required by each Job container. If omitted, a default value will be used. Cannot be updated. [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/)
| `.spec.job.mode` | string | O | Job running mode, `enum("Detached", "Blocking")`
| `.spec.envVars` | object array | O | Environment variables shared by all JobManager, TaskManager and job containers.
| `.spec.envFrom` | object array | O | Environment variables from ConfigMaps or Secrets shared by all JobManager, TaskManager and job containers.
| `.spec.flinkProperties` | object | O | Flink properties which are appened to flink-conf.yaml.
| `.spec.hadoopConfig` | object | O | Configs for Hadoop.
| `.spec.hadoopConfig.configMapName` | string | O | The name of the ConfigMap which holds the Hadoop config files. The ConfigMap must be in the same namespace as the FlinkCluster.
| `.spec.hadoopConfig.mountPath` | string | O | The path where to mount the Volume of the ConfigMap.
| `.spec.gcpConfig` | object | O | Configs for GCP.
| `.spec.gcpConfig.serviceAccount` | object | O | GCP service account.
| `.spec.gcpConfig.serviceAccount.secretName` | string | O | The name of the Secret holding the GCP service account key file. The Secret must be in the same namespace as the FlinkCluster.
| `.spec.gcpConfig.serviceAccount.keyFile` | string | O | The name of the service account key file.
| `.spec.gcpConfig.serviceAccount.mountPath` | string | O | The path where to mount the Volume of the Secret.
| `.spec.logConfig` | object | O | The logging configuration, a string-to-string map that becomes the ConfigMap mounted at `/opt/flink/conf` in launched Flink pods. See below for some possible keys to include: <br> - **log4j-console.properties**: The contents of the log4j properties file to use. If not provided, a default that logs only to stdout will be provided. <br> - **logback-console.xml**: The contents of the logback XML file to use. If not provided, a default that logs only to stdout will be provided. <br> - Other arbitrary keys are also allowed, and will become part of the ConfigMap.
| `.spec.revisionHistoryLimit` | int | O | The maximum number of revision history to keep, default: 10.
| `.spec.recreateOnUpdate` | bool | O | Recreate components when updating flinkcluster, default: true.
---

## Flink job or session cluster status.

| Key | Description 
|-----|-------------
| `.status.state` | The overall state of the Flink cluster.
| `.status.components` | The status of the components.
| `.status.components.configMap` | The status of the ConfigMap.
| `.status.components.configMap.name` | The resource name of the ConfigMap.
| `.status.components.configMap.state` | The state of the ConfigMap.
| `.status.components.jobManagerStatefulSet` | The status of the JobManager StatefulSet.
| `.status.components.jobManagerStatefulSet.name` | The resource name of the JobManager StatefulSet.
| `.status.components.jobManagerStatefulSet.state` | The state of the JobManager StatefulSet.
| `.status.components.jobManagerService` | The status of the JobManager service.
| `.status.components.jobManagerService.name` | The resource name of the JobManager service.
| `.status.components.jobManagerService.state` | The state of the JobManager service.
| `.status.components.jobManagerService.nodePort` | (optional) The node port, present when `accessScope` is `NodePort`.
| `.status.components.jobManagerService.loadBalancerIngress` | (optional) The load balancer ingress, present when `accessScope` is `VPC` or `External`.
| `.status.components.jobManagerIngress` | The status of the JobManager ingress.
| `.status.components.jobManagerIngress.name` | The resource name of the JobManager ingress.
| `.status.components.jobManagerIngress.state` | The state of the JobManager ingress.
| `.status.components.jobManagerIngress.urls` | The generated URLs for JobManager.
| `.status.components.taskManagerStatefulSet` | The status of the TaskManager StatefulSet.
| `.status.components.taskManagerStatefulSet.name` | The resource name of the TaskManager StatefulSet.
| `.status.components.taskManagerStatefulSet.state` | The state of the TaskManager StatefulSet.
| `.status.components.job` | The status of the job.
| `.status.components.job.id` | The ID of the Flink job.
| `.status.components.job.name` | The Name of the Flink job.
| `.status.components.job.submitterName` | The name of the Kubernetes job submitter resource.
| `.status.components.job.completionTime` | Job completion time. Present when job is terminated regardless of its state.
| `.status.components.job.state` | The state of the job.
| `.status.components.job.fromSavepoint` | The actual savepoint from which this job started. In case of restart, it might be different from the savepoint in the job spec.
| `.status.components.job.savepointGeneration` | The generation of the savepoint in `savepointsDir` taken by the operator. The value starts from 0 when there is no savepoint and increases by 1 for each successful savepoint.
| `.status.components.job.savepointLocation` | Last savepoint location.
| `.status.components.job.lastSavepointTriggerID` | Last savepoint trigger ID.
| `.status.components.job.savepointTime` | Last successful or failed savepoint operation timestamp.
| `.status.components.job.finalSavepoint` | The savepoint recorded in savepointLocation is the final state of the job.
| `.status.components.job.deployTime` | The timestamp of the Flink job deployment that creating job submitter.
| `.status.components.job.startTime` | The Flink job started timestamp.
| `.status.components.job.restartCount` | The number of restarts.
| `.status.components.job.failureReasons` | Reasons for the job failure. Present if job state is Failure.
| `.status.control` | The status of control requested by user.
| `.status.control.name` | Requested control name.
| `.status.control.details` | Control details.
| `.status.control.state` | Control state.
| `.status.control.message` | Control message.
| `.status.control.updateTime` | The updated time of control status.
| `.status.savepoint` | The status of savepoint progress.
| `.status.savepoint.jobID` | The ID of the Flink job.
| `.status.savepoint.triggerID` | Savepoint trigger ID.
| `.status.savepoint.triggerTime` | Savepoint triggered time.
| `.status.savepoint.triggerReason` | Savepoint triggered reason.
| `.status.savepoint.requestTime` | Savepoint status update time.
| `.status.savepoint.state` | Savepoint state.
| `.status.savepoint.message` | Savepoint message.
| `.status.revision` | The status of revision.
| `.status.revision.currentRevision` | CurrentRevision indicates the version of FlinkCluster.
| `.status.revision.nextRevision` | NextRevision indicates the version of FlinkCluster updating.
| `.status.revision.collisionCount` | collisionCount is the count of hash collisions for the FlinkCluster. The controller uses this field as a collision avoidance mechanism when it needs to create the name for the newest ControllerRevision.
| `.status.lastUpdateTime` | Last update timestamp of this status.
