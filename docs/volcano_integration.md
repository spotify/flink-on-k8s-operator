# Integration with Volcano for Batch Scheduling

[Volcano](https://volcano.sh) is a batch system built on Kubernetes. It provides a suite of mechanisms
currently missing from Kubernetes that are commonly required by many classes
of batch & elastic workloads.
With the integration with Volcano, Flink job and task managers can be scheduled simultaneously, which is particularly suitable for
clusters with resource shortage.

## Prerequisites

## Install Volcano

- Install from provided demo

Run the following

```bash
kubectl apply --server-side -f https://raw.githubusercontent.com/volcano-sh/volcano/v1.15.2/installer/volcano-development.yaml
```

- Install with advanced settings

Please refer to [Volcano Official Guide](https://volcano.sh/en/docs/)

### Verify Volcano is up and running

```bash
$ kubectl get pod -n volcano-system
NAME                                   READY   STATUS      RESTARTS   AGE
volcano-admission-8568b4468d-lrwsq     1/1     Running     0          5m28s
volcano-admission-init-56zz2           0/1     Completed   0          5m28s
volcano-controllers-5f5b5bb6c4-4vxp4   1/1     Running     0          5m28s
volcano-scheduler-7859c66dbf-6b74x     1/1     Running     0          5m28s

```

## Install Flink Operator

Please refer to [Deploy the operator to a Kubernetes cluster](./user_guide.md#deploy-the-operator-to-a-kubernetes-cluster)

# Create a sample Flink job cluster with batch scheduling enabled

Create a sample Flink job cluster with:

```bash
$ kubectl apply -f config/samples/flinkoperator_v1beta1_flinkjobcluster_volcano.yaml
```

and verify the pod is up and running with

```bash
$ kubectl get pod,svc -l cluster=flinkjobcluster-volcano
NAME                                              READY   STATUS      RESTARTS   AGE
pod/flinkjobcluster-volcano-job-submitter-krg6j   0/1     Completed   0          12m
pod/flinkjobcluster-volcano-jobmanager-0          1/1     Running     0          12m
pod/flinkjobcluster-volcano-taskmanager-0         1/1     Running     0          12m
pod/flinkjobcluster-volcano-taskmanager-1         1/1     Running     0          12m

NAME                                          TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                               AGE
service/flinkjobcluster-volcano-jobmanager    ClusterIP   10.106.105.171   <none>        6123/TCP,6124/TCP,6125/TCP,8081/TCP   12m
service/flinkjobcluster-volcano-taskmanager   ClusterIP   None             <none>        6121/TCP,6122/TCP,6125/TCP            12m
```

verify `job manager` and `task manager` are scheduled by volcano

```bash
$ kubectl get podgroup flink-flinkjobcluster-volcano -o yaml
apiVersion: scheduling.volcano.sh/v1beta1
kind: PodGroup
metadata:
  creationTimestamp: "2026-09-10T14:42:39Z"
  generation: 6
  name: flink-flinkjobcluster-volcano
  namespace: default
  ownerReferences:
  - apiVersion: flinkoperator.k8s.io/v1beta1
    blockOwnerDeletion: false
    controller: true
    kind: FlinkCluster
    name: flinkjobcluster-volcano
    uid: 9691ce99-6477-4ce2-8c6d-f750491d5916
  resourceVersion: "67130"
  uid: ffa4bd0c-5a71-4fc2-bc07-db37eb4a1605
spec:
  minMember: 4
  minResources:
    cpu: 3500m
    limits.cpu: 3500m
    limits.memory: 8Gi
    memory: 8Gi
    requests.cpu: 3500m
    requests.memory: 8Gi
  queue: default
status:
  phase: Running
  running: 3
  succeeded: 1
```

As shown above, the podgroup has three running pods and one succeeded pod, and the min required number is 4. If the cluster has insufficient resources for the job manager, two task managers, and job submitter, they are not scheduled.

Also you can check the job manager and task manager's scheduler name is now set to `volcano`

```bash
$ kubectl get pod flinkjobcluster-volcano-jobmanager-0 -ojsonpath={'.spec.schedulerName'}
volcano

$ kubectl get pod flinkjobcluster-volcano-taskmanager-0 -ojsonpath={'.spec.schedulerName'}
volcano
```

**Note**: the job submitter's pod is included in the podgroup.

# Create a sample Flink session cluster with batch scheduling enabled

you can create a sample session cluster as well with
`kubectl apply -f config/samples/flinkoperator_v1beta1_flinksessioncluster_volcano.yaml`
