# FlinkCluster Custom Resource Definition

The Kubernetes Operator for Apache Flink uses [CustomResourceDefinition](https://kubernetes.io/docs/concepts/api-extension/custom-resources/)
named `FlinkCluster` for specifying a Flink job cluster ([sample](../config/samples/flinkoperator_v1beta1_flinkjobcluster.yaml))
or Flink session cluster ([sample](../config/samples/flinkoperator_v1beta1_flinksessioncluster.yaml)), depending on
whether the job spec is specified. Similarly to other kinds of Kubernetes resources, the custom resource consists of a
resource `Metadata`, a specification in a `Spec` field and a `Status` field. The definitions are organized in the
following structure. The v1beta1 version of the API definition is implemented [here](../api/v1beta1/flinkcluster_types.go).

---
## Flink job or session cluster spec.

| Key | Type | Required | Description
|-----|------|----------|-------------
| `.spec.flinkVersion` | string | O | The version of Flink to be managed. This version must match the version in the image.
| `.spec.image` | object | O | Flink image for JobManager, TaskManager and job containers.
| `.spec.image.name` | string | O | Image name.
| `.spec.image.pullPolicy` | string | | Image pull policy. <br> [more info](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy)
| `.spec.image.pullSecrets` | string array | | Secrets for image pull.
| *(deprecated)* `.spec.batchSchedulerName` | string | | BatchSchedulerName specifies the batch scheduler name for JobManager, TaskManager. If empty, no batch scheduling is enabled.
| `.spec.batchScheduler` | object | | BatchScheduler specifies the batch scheduler properties for Job, JobManager, TaskManager. If empty, no batch scheduling is enabled.
| `.spec.batchScheduler.name` | string | O | Scheduler name.
| `.spec.batchScheduler.queue` | string | | Queue name in which resources will be allocates.
| `.spec.batchScheduler.priorityClassName` | string | | If specified, indicates the PodGroup's priority. "system-node-critical" and "system-cluster-critical" are two special keywords which indicate the highest priorities with the former being the highest priority. Any other name must be defined by creating a PriorityClass object with that name. If not specified, the priority will be default or zero if there is no default.
| `.spec.serviceAccountName` | string | | the name of the service account(which must already exist in the namespace). If empty, the default service account in the namespace will be used.
| `.spec.jobManager` | object | O | JobManager spec.
| `.spec.jobManager.replicas` | int | O | The number of JobManager replicas.
| `.spec.jobManager.accessScope` | string | O | Access scope of the JobManager service. <br> `Cluster`: accessible from within the same cluster <br> `VPC`: accessible from within the same VPC. <br> `External`: accessible from the internet. <br> `NodePort`: accessible through node port. <br> `Headless`: pod IPs assumed to be routable and advertised directly with `clusterIP: None`. <br> Currently `VPC` and `External` are only available for GKE.
| `.spec.jobManager.ports` | object | | Ports that JobManager listening on.
| `.spec.jobManager.ports.rpc` | int | | RPC port, default: 6123.
| `.spec.jobManager.ports.blob` | int | | Blob port, default: 6124.
| `.spec.jobManager.ports.query` | int | | Query port, default: 6125.
| `.spec.jobManager.ports.ui` | int | | UI port, default: 8081.
| `.spec.jobManager.extraPorts` | object array | | Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on. Each port number and name must be unique among ports and extraPorts. ContainerPort is required and name and protocol are optional.
| `.spec.jobManager.extraPorts.name` | string | | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
| `.spec.jobManager.extraPorts.containerPort` | int | O | Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.
| `.spec.jobManager.extraPorts.protocol` | string | | Protocol for port. Must be UDP, TCP, or SCTP. defaults: TCP.
| `.spec.jobManager.ingress` | object | | Provide external access to JobManager UI/API.
| `.spec.jobManager.ingress.hostFormat` | string | | Host format for generating URLs. ex) `{{$clusterName}}.example.com`
| `.spec.jobManager.ingress.annotations` | object | | Annotations for ingress configuration. <br> [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
| `.spec.jobManager.ingress.useTLS` | bool | | TLS use, default: false.
| `.spec.jobManager.ingress.tlsSecretName` | string | | Kubernetes secret resource name for TLS.
| `.spec.jobManager.resources` | object | | Compute resources required by JobManager container. If omitted, a default value will be used. <br> [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/)
| `.spec.jobManager.memoryOffHeapRatio` | int | | Percentage of off-heap memory in containers, as a safety margin, default: 25
| `.spec.jobManager.memoryOffHeapMin` | string | | Minimum amount of off-heap memory in containers, as a safety margin, default: 600M. You can express this value like 600M, 572Mi and 600e6. <br> [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) about value expression
| `.spec.jobManager.memoryProcessRatio` | int | | For Flink 1.10+. Percentage of memory process, as a safety margin to avoid OOM kill, default: 80
| `.spec.jobManager.volumes` | object array | | Volumes in the JobManager pod. <br> [more info](https://kubernetes.io/docs/concepts/storage/volumes/)
| `.spec.jobManager.volumeMounts` | object array | | Volume mounts in the JobManager container. <br> [more info](https://kubernetes.io/docs/concepts/storage/volumes/)
| `.spec.jobManager.volumeClaimTemplates` | object array | | A template for persistent volume claim each requested and mounted to JobManager pod, This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend). <br> [more info](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)
| `.spec.jobManager.initContainers` | object array | | Init containers of the JobManager pod. <br> [more info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/)
| `.spec.jobManager.nodeSelector` | object | | Selector which must match a node's labels for the JobManager pod to be scheduled on that node. <br> [more info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
| `.spec.jobManager.tolerations` | object array | | Allows the JobManager pod to run on a tainted node in the cluster. <br> [more info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
| `.spec.jobManager.sidecars` | object array | | Sidecar containers running alongside with the JobManager container in the pod. <br> [more info](https://kubernetes.io/docs/concepts/containers/)
| `.spec.jobManager.podAnnotations` | object | | Pod template annotations for the JobManager StatefulSet. <br> [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
| `.spec.jobManager.podLabels` | object | | Pod template labels for the JobManager StatefulSet. <br> [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
| `.spec.jobManager.securityContext` | object | | PodSecurityContext for the JobManager pod. <br> [more info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod)
| `.spec.jobManager.livenessProbe` | object | | LivenessProbe for the JobManager pod. <br> [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
| `.spec.jobManager.readinessProbe` | object | | ReadinessProbe for the JobManager pod. <br> [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
| `.spec.taskManager` | int | O | TaskManager spec.
| `.spec.taskManager.replicas` | int | O | The number of TaskManager replicas.
| `.spec.taskManager.ports` | object | | Ports that TaskManager listening on.
| `.spec.taskManager.ports.data` | int | | Data port.
| `.spec.taskManager.ports.rpc` | int | | RPC port.
| `.spec.taskManager.ports.query` | int | | Query port.
| `.spec.taskManager.extraPorts` | object array | | Extra ports to be exposed. For example, Flink metrics reporter ports: Prometheus, JMX and so on. Each port number and name must be unique among ports and extraPorts. ContainerPort is required and name and protocol are optional.
| `.spec.taskManager.extraPorts.name` | string | | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services.
| `.spec.taskManager.extraPorts.containerPort` | int | O | Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.
| `.spec.taskManager.extraPorts.protocol` | string | | Protocol for port. Must be UDP, TCP, or SCTP. defaults: TCP.
| `.spec.taskManager.resources` | object | | Compute resources required by JobManager container. If omitted, a default value will be used. <br> [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/)
| `.spec.taskManager.memoryOffHeapRatio` | int | | Percentage of off-heap memory in containers, as a safety margin, default: 25
| `.spec.taskManager.memoryOffHeapMin` | string | | Minimum amount of off-heap memory in containers, as a safety margin, default: 600M. You can express this value like 600M, 572Mi and 600e6. <br> [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#meaning-of-memory) about value expression
| `.spec.taskManager.memoryProcessRatio` | int | | For Flink 1.10+. Percentage of memory process, as a safety margin to avoid OOM kill, default: 80
| `.spec.taskManager.volumes` | object array | | Volumes in the TaskManager <br> [more info](https://kubernetes.io/docs/concepts/storage/volumes/)
| `.spec.taskManager.volumeMounts` | object array | | Volume mounts in the TaskManager containers. <br> [more info](https://kubernetes.io/docs/concepts/storage/volumes/)
| `.spec.taskManager.volumeClaimTemplates` | object array | | A template for persistent volume claim each requested and mounted to each TaskManager pod, This can be used to mount an external volume with a specific storageClass or larger captivity (for larger/faster state backend) <br> [more info](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) about VolumeClaimTemplates in StatefulSet
| `.spec.taskManager.initContainers` | object array | | Init containers of the TaskManager pod. <br> [more info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/)
| `.spec.taskManager.nodeSelector` | object | | Selector which must match a node's labels for the TaskManager pod tobe scheduled on that node. <br> [more info](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
| `.spec.taskManager.tolerations` | object array | | Allows the TaskManager pod to run on a tainted node in the cluster. <br> [more info](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
| `.spec.taskManager.sidecars` | object array | | Sidecar containers running alongside with the TaskManager container in the pod. <br> [more info](https://kubernetes.io/docs/concepts/containers/)
| `.spec.taskManager.podAnnotations` | object | | Pod template annotations for the TaskManager StatefulSet. <br> [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
| `.spec.taskManager.podLabels` | object | | Pod template labels for the TaskManager StatefulSet. <br> [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
| `.spec.taskManager.securityContext` | object | | PodSecurityContext for the TaskManager pods. <br> [more info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod)
| `.spec.taskManager.livenessProbe` | object | | LivenessProbe for the TaskManager pod. <br> [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
| `.spec.taskManager.readinessProbe` | object | | ReadinessProbe for the TaskManager pod. <br> [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
| `.spec.job` | string | | Job spec. If specified, the cluster is a Flink job cluster; otherwise, it is a Flink session cluster.
| `.spec.job.jarFile` | string | | JAR file of the job. It could be a local file or remote URI, depending on which protocols (e.g., `https://`, `gs://`) are supported by the Flink image.
| `.spec.job.className` | string | | Fully qualified Java class name of the job.
| `.spec.job.pythonFile` | string | | Python file of the job. It should be a local file.
| `.spec.job.pythonFiles` | string | | Python files of the job. It should be a local file (with .py/.egg/.zip/.whl) or directory. See the Flink argument `--pyFiles` for the detail.
| `.spec.job.pythonModule` | string | | Python module path of the job entry point. Must use with **pythonFiles**.
| `.spec.job.args` | string array | | Command-line args of the job.
| `.spec.job.fromSavepoint` | string | | Savepoint where to restore the job from. If Flink job must be restored from the latest available savepoint when Flink job updating, this field must be unspecified.
| `.spec.job.allowNonRestoredState` | bool | | Allow non-restored state, default: false.
| `.spec.job.takeSavepointOnUpdate` | bool | | Should take savepoint before updating the job, default: true. If this is set as false, maxStateAgeToRestoreSeconds must be provided to limit the savepoint age to restore.
| `.spec.job.maxStateAgeToRestoreSeconds` | int | | Maximum age of the savepoint that allowed to restore state. This is applied to auto restart on failure, update from stopped state and update without taking savepoint. If nil, job can be restarted only when the latest savepoint is the final job state (created by "stop with savepoint") - that is, only when job can be resumed from the suspended state.
| `.spec.job.autoSavepointSeconds` | int | | Automatically take a savepoint to the `savepointsDir` every n seconds.
| `.spec.job.savepointsDir` | string | | Savepoints dir where to store automatically taken savepoints.
| `.spec.job.savepointGeneration` | int | | Update this field to `jobStatus.savepointGeneration + 1` for a running job cluster to trigger a new savepoint to `savepointsDir` on demand.
| `.spec.job.parallelism` | int | | Parallelism of the job, default: 1.
| `.spec.job.noLoggingToStdout` | bool | | No logging output to STDOUT, default: false.
| `.spec.job.volumes` | object array | | Volumes in the Job pod. <br> [more info](https://kubernetes.io/docs/concepts/storage/volumes/)
| `.spec.job.volumeMounts` | object array | | Volume mounts in the Job containers. If there is no confilcts, these mounts will be automatically added to init containers; otherwise, the mounts defined in init containers will take precedence. <br> [more info](https://kubernetes.io/docs/concepts/storage/volumes/)
| `.spec.job.initContainers` | object array | | Init containers of the Job pod. <br> [more info](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/)
| `.spec.job.restartPolicy` | string | | Restart policy when the job fails, default: `"Never"`. <br> `"Never"`: the operator will never try to restart a failed job, manual cleanup is required. <br> `"FromSavepointOnFailure"`: the operator will try to restart the failed job from the savepoint recorded in the job status if available; otherwise, the job will stay in failed state. This option is usually used together with `autoSavepointSeconds` and `savepointsDir`.
| `.spec.job.cleanupPolicy` | object | | The action to take after job finishes.
| `.spec.job.cleanupPolicy.afterJobSucceeds` | string | | The action to take after job succeeds, `"KeepCluster", "DeleteCluster", "DeleteTaskManager"`, default `"DeleteCluster"`.
| `.spec.job.cleanupPolicy.afterJobFails` | string | | The action to take after job fails, `"KeepCluster", "DeleteCluster", "DeleteTaskManager"`, default `"KeepCluster"`.
| `.spec.job.cleanupPolicy.afterJobCancelled` | string | | The action to take after job cancelled, `"KeepCluster", "DeleteCluster", "DeleteTaskManager"`, default `"DeleteCluster"`.
| `.spec.job.cancelRequested` | bool | | Request the job to be cancelled. Only applies to running jobs. If `savePointsDir` is provided, a savepoint will be taken before stopping the job.
| `.spec.job.podAnnotations` | object | | Pod template annotations for the job. <br> [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
| `.spec.job.podLabels` | object | | Pod template labels for the job. <br> [more info](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
| `.spec.job.securityContext` | object | | PodSecurityContext for the Job pod. <br> [more info](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod)
| `.spec.job.resources` | object | | Compute resources required by each Job container. If omitted, a default value will be used. Cannot be updated. <br> [more info](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/)
| `.spec.job.mode` | string | | Job running mode, `"Detached", "Blocking"`, default: `"Detached"`.
| `.spec.envVars` | object array | | Environment variables shared by all JobManager, TaskManager and job containers. <br> [more info](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/)
| `.spec.envFrom` | object array | | Environment variables from ConfigMaps or Secrets shared by all JobManager, TaskManager and job containers. <br> [more info](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables)
| `.spec.flinkProperties` | object | | Flink properties consist of string-to-string map which are appened to flink-conf.yaml.
| `.spec.hadoopConfig` | object | | Configs for Hadoop.
| `.spec.hadoopConfig.configMapName` | string | | The name of the ConfigMap which holds the Hadoop config files. The ConfigMap must be in the same namespace as the FlinkCluster.
| `.spec.hadoopConfig.mountPath` | string | | The path where to mount the Volume of the ConfigMap.
| `.spec.gcpConfig` | object | | Configs for GCP.
| `.spec.gcpConfig.serviceAccount` | object | | GCP service account.
| `.spec.gcpConfig.serviceAccount.secretName` | string | | The name of the Secret holding the GCP service account key file. The Secret must be in the same namespace as the FlinkCluster.
| `.spec.gcpConfig.serviceAccount.keyFile` | string | | The name of the service account key file.
| `.spec.gcpConfig.serviceAccount.mountPath` | string | | The path where to mount the Volume of the Secret.
| `.spec.logConfig` | object | | The logging configuration, a string-to-string map that becomes the ConfigMap mounted at `/opt/flink/conf` in launched Flink pods. See below for some possible keys to include: <br> - **log4j-console.properties**: The contents of the log4j properties file to use. If not provided, a default that logs only to stdout will be provided. <br> - **logback-console.xml**: The contents of the logback XML file to use. If not provided, a default that logs only to stdout will be provided. <br> - Other arbitrary keys are also allowed, and will become part of the ConfigMap.
| `.spec.revisionHistoryLimit` | int | | The maximum number of revision history to keep, default: 10.
| `.spec.recreateOnUpdate` | bool | | Recreate components when updating flinkcluster, default: true.
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
