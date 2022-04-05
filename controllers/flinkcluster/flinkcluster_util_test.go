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

package flinkcluster

import (
	"os"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/util"
	"gotest.tools/v3/assert"
)

func TestTimeConverter(t *testing.T) {
	var tc = &util.TimeConverter{}

	var str1 = "2019-10-23T05:10:36Z"
	var tm1 = tc.FromString(str1)
	var str2 = tc.ToString(tm1)
	assert.Assert(t, str1 == str2)

	var str3 = "2019-10-24T09:57:18+09:00"
	var tm2 = tc.FromString(str3)
	var str4 = tc.ToString(tm2)
	assert.Assert(t, str3 == str4)
}

func TestGetRetryCount(t *testing.T) {
	var data1 = map[string]string{}
	var result1, _ = getRetryCount(data1)
	assert.Equal(t, result1, "1")

	var data2 = map[string]string{"retries": "1"}
	var result2, _ = getRetryCount(data2)
	assert.Equal(t, result2, "2")
}

func TestNewRevision(t *testing.T) {
	var jmReplicas int32 = v1beta1.DefaultJobManagerReplicas
	var tmReplicas int32 = v1beta1.DefaultTaskManagerReplicas
	var rpcPort int32 = 8001
	var blobPort int32 = 8002
	var queryPort int32 = 8003
	var uiPort int32 = 8004
	var dataPort int32 = 8005
	var memoryOffHeapRatio int32 = 25
	var memoryOffHeapMin = resource.MustParse("600M")
	var jarFile = "gs://my-bucket/myjob.jar"
	var parallelism int32 = 2
	var savepointDir = "/savepoint_dir"
	var flinkCluster = v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mycluster",
			Namespace: "default",
		},
		Spec: v1beta1.FlinkClusterSpec{
			Image: v1beta1.ImageSpec{
				Name:       "flink:1.8.1",
				PullPolicy: corev1.PullPolicy("Always"),
			},
			JobManager: &v1beta1.JobManagerSpec{
				Replicas:    &jmReplicas,
				AccessScope: v1beta1.AccessScopeVPC,
				Ports: v1beta1.JobManagerPorts{
					RPC:   &rpcPort,
					Blob:  &blobPort,
					Query: &queryPort,
					UI:    &uiPort,
				},
				MemoryOffHeapRatio: &memoryOffHeapRatio,
				MemoryOffHeapMin:   memoryOffHeapMin,
			},
			TaskManager: &v1beta1.TaskManagerSpec{
				Replicas: &tmReplicas,
				Ports: v1beta1.TaskManagerPorts{
					RPC:   &rpcPort,
					Data:  &dataPort,
					Query: &queryPort,
				},
				MemoryOffHeapRatio: &memoryOffHeapRatio,
				MemoryOffHeapMin:   memoryOffHeapMin,
			},
			Job: &v1beta1.JobSpec{
				JarFile:       &jarFile,
				Parallelism:   &parallelism,
				SavepointsDir: &savepointDir,
			},
		},
	}
	var collisionCount int32 = 0
	var controller = true
	var blockOwnerDeletion = true
	var raw, _ = getPatch(&flinkCluster)
	var revision, _ = newRevision(&flinkCluster, 1, &collisionCount)
	var expectedRevision = appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mycluster-7bc87c954f",
			Namespace: "default",
			Labels: map[string]string{
				"flinkoperator.k8s.io/hash":       "7bc87c954f",
				"flinkoperator.k8s.io/managed-by": "mycluster",
			},
			Annotations: map[string]string{},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "flinkoperator.k8s.io/v1beta1",
				Kind:               "FlinkCluster",
				Name:               "mycluster",
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			}},
		},
		Revision: 1,
		Data:     runtime.RawExtension{Raw: raw},
	}
	assert.Assert(t, revision != nil)
	assert.DeepEqual(
		t,
		*revision,
		expectedRevision)
}

func TestCanTakeSavepoint(t *testing.T) {
	// session cluster
	var cluster = v1beta1.FlinkCluster{
		Spec: v1beta1.FlinkClusterSpec{},
	}
	var take = canTakeSavepoint(&cluster)
	assert.Equal(t, take, false)

	// no savepointDir and job status
	cluster = v1beta1.FlinkCluster{
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{},
		},
	}
	take = canTakeSavepoint(&cluster)
	assert.Equal(t, take, false)

	// no job status, job is to be started
	savepointDir := "/savepoints"
	cluster = v1beta1.FlinkCluster{
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{SavepointsDir: &savepointDir},
		},
	}
	take = canTakeSavepoint(&cluster)
	assert.Equal(t, take, true)

	// running job and no progressing savepoint
	savepointDir = "/savepoints"
	cluster = v1beta1.FlinkCluster{
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{SavepointsDir: &savepointDir},
		},
		Status: v1beta1.FlinkClusterStatus{Components: v1beta1.FlinkClusterComponentsStatus{
			Job: &v1beta1.JobStatus{State: "Running"},
		}},
	}
	take = canTakeSavepoint(&cluster)
	assert.Equal(t, take, true)

	// progressing savepoint
	savepointDir = "/savepoints"
	cluster = v1beta1.FlinkCluster{
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{SavepointsDir: &savepointDir},
		},
		Status: v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				Job: &v1beta1.JobStatus{State: "Running"},
			},
			Savepoint: &v1beta1.SavepointStatus{State: v1beta1.SavepointStateInProgress},
		},
	}
	take = canTakeSavepoint(&cluster)
	assert.Equal(t, take, false)
}

func TestGetNextRevisionNumber(t *testing.T) {
	var revisions []*appsv1.ControllerRevision
	var nextRevision = util.GetNextRevisionNumber(revisions)
	assert.Equal(t, nextRevision, int64(1))

	revisions = []*appsv1.ControllerRevision{{Revision: 1}, {Revision: 2}}
	nextRevision = util.GetNextRevisionNumber(revisions)
	assert.Equal(t, nextRevision, int64(3))
}

func TestIsComponentUpdated(t *testing.T) {
	var cluster = v1beta1.FlinkCluster{
		Status: v1beta1.FlinkClusterStatus{Revision: v1beta1.RevisionStatus{NextRevision: "cluster-85dc8f749-2"}},
	}
	var cluster2 = v1beta1.FlinkCluster{
		Spec: v1beta1.FlinkClusterSpec{
			JobManager: &v1beta1.JobManagerSpec{Ingress: &v1beta1.JobManagerIngressSpec{}},
			Job:        &v1beta1.JobSpec{},
		},
		Status: v1beta1.FlinkClusterStatus{Revision: v1beta1.RevisionStatus{NextRevision: "cluster-85dc8f749-2"}},
	}
	var deploy = &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
		RevisionNameLabel: "cluster-85dc8f749",
	}}}
	var update = isComponentUpdated(deploy, &cluster)
	assert.Equal(t, update, true)

	deploy = &appsv1.Deployment{}
	update = isComponentUpdated(deploy, &cluster)
	assert.Equal(t, update, false)

	deploy = nil
	update = isComponentUpdated(deploy, &cluster)
	assert.Equal(t, update, false)

	var job = &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
		RevisionNameLabel: "cluster-85dc8f749",
	}}}
	update = isComponentUpdated(job, &cluster2)
	assert.Equal(t, update, true)

	job = &batchv1.Job{}
	update = isComponentUpdated(job, &cluster2)
	assert.Equal(t, update, false)

	job = nil
	update = isComponentUpdated(job, &cluster2)
	assert.Equal(t, update, false)

	job = nil
	update = isComponentUpdated(job, &cluster)
	assert.Equal(t, update, true)
}

func TestGetUpdateState(t *testing.T) {
	var observed = ObservedClusterState{
		cluster: &v1beta1.FlinkCluster{
			Spec: v1beta1.FlinkClusterSpec{
				JobManager: &v1beta1.JobManagerSpec{Ingress: &v1beta1.JobManagerIngressSpec{}},
				Job:        &v1beta1.JobSpec{},
			},
			Status: v1beta1.FlinkClusterStatus{
				Components: v1beta1.FlinkClusterComponentsStatus{Job: &v1beta1.JobStatus{State: v1beta1.JobStateRunning}},
				Revision:   v1beta1.RevisionStatus{CurrentRevision: "cluster-85dc8f749-2", NextRevision: "cluster-aa5e3a87z-3"}},
		},
		flinkJobSubmitter:   FlinkJobSubmitter{job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-85dc8f749"}}}},
		configMap:           &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-85dc8f749"}}},
		podDisruptionBudget: &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-85dc8f749"}}},
		jmStatefulSet:       &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-85dc8f749"}}},
		tmStatefulSet:       &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-85dc8f749"}}},
		jmService:           &corev1.Service{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-85dc8f749"}}},
	}
	var state = getUpdateState(&observed)
	assert.Equal(t, state, UpdateStatePreparing)

	observed = ObservedClusterState{
		cluster: &v1beta1.FlinkCluster{
			Spec: v1beta1.FlinkClusterSpec{
				JobManager: &v1beta1.JobManagerSpec{Ingress: &v1beta1.JobManagerIngressSpec{}},
				Job:        &v1beta1.JobSpec{},
			},
			Status: v1beta1.FlinkClusterStatus{
				Revision: v1beta1.RevisionStatus{CurrentRevision: "cluster-85dc8f749-2", NextRevision: "cluster-aa5e3a87z-3"},
			},
		},
		jmStatefulSet: &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-aa5e3a87z"}}},
		tmStatefulSet: &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-85dc8f749"}}},
		jmService:     &corev1.Service{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-85dc8f749"}}},
	}
	state = getUpdateState(&observed)
	assert.Equal(t, state, UpdateStateInProgress)

	observed = ObservedClusterState{
		cluster: &v1beta1.FlinkCluster{
			Spec: v1beta1.FlinkClusterSpec{
				JobManager: &v1beta1.JobManagerSpec{Ingress: &v1beta1.JobManagerIngressSpec{}},
				Job:        &v1beta1.JobSpec{},
			},
			Status: v1beta1.FlinkClusterStatus{Revision: v1beta1.RevisionStatus{CurrentRevision: "cluster-85dc8f749-2", NextRevision: "cluster-aa5e3a87z-3"}},
		},
		flinkJobSubmitter:   FlinkJobSubmitter{job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-aa5e3a87z"}}}},
		configMap:           &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-aa5e3a87z"}}},
		podDisruptionBudget: &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-aa5e3a87z"}}},
		jmStatefulSet:       &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-aa5e3a87z"}}},
		tmStatefulSet:       &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-aa5e3a87z"}}},
		jmService:           &corev1.Service{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-aa5e3a87z"}}},
		jmIngress:           &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "cluster-aa5e3a87z"}}},
	}
	state = getUpdateState(&observed)
	assert.Equal(t, state, UpdateStateFinished)
}

func TestHasTimeElapsed(t *testing.T) {
	var tc = &util.TimeConverter{}
	var timeToCheckStr = "2020-01-01T00:00:00+00:00"
	var timeToCompare = tc.FromString("2020-01-01T00:00:20+00:00")
	var elapsed = hasTimeElapsed(timeToCheckStr, timeToCompare, 10)
	assert.Equal(t, elapsed, true)

	elapsed = hasTimeElapsed(timeToCheckStr, timeToCompare, 30)
	assert.Equal(t, elapsed, false)
}

func TestGetFlinkAPIBaseURL(t *testing.T) {
	var uiPort int32 = 8004
	var cluster = v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mycluster",
			Namespace: "default",
		},
		Spec: v1beta1.FlinkClusterSpec{
			JobManager: &v1beta1.JobManagerSpec{
				Ports: v1beta1.JobManagerPorts{
					UI: &uiPort,
				},
			},
		},
	}

	var apiBaseURL = getFlinkAPIBaseURL(&cluster)
	assert.Equal(t, apiBaseURL, "http://mycluster-jobmanager.default.svc.cluster.local:8004")

	os.Setenv("CLUSTER_DOMAIN", "my.domain")
	apiBaseURL = getFlinkAPIBaseURL(&cluster)
	assert.Equal(t, apiBaseURL, "http://mycluster-jobmanager.default.svc.my.domain:8004")
}

func TestGetNonLiveHistory(t *testing.T) {
	revison0 := appsv1.ControllerRevision{Revision: int64(0)}
	revison1 := appsv1.ControllerRevision{Revision: int64(1)}
	revisions := []*appsv1.ControllerRevision{&revison0, &revison1}

	historyLimit := 1
	nonLiveHistory := util.GetNonLiveHistory(revisions, historyLimit)
	assert.Equal(t, len(nonLiveHistory), 1)
	assert.Equal(t, nonLiveHistory[0].Revision, int64(0))

	historyLimit = 3
	nonLiveHistory = util.GetNonLiveHistory(revisions, historyLimit)
	assert.Equal(t, len(nonLiveHistory), 0)
}

func TestGetFlinkJobSubmitLog(t *testing.T) {
	var submit, expected *SubmitterLog

	// success
	log := `
  /opt/flink/bin/flink run --jobmanager flinkjobcluster-sample-jobmanager:8081 --class org.apache.flink.streaming.examples.wordcount.WordCount --parallelism 2 --detached ./examples/streaming/WordCount.jar --input ./README.txt
  Starting execution of program
  Printing result to stdout. Use --output to specify output path.
  Job has been submitted with JobID ec74209eb4e3db8ae72db00bd7a830aa
  Program execution finished
  Job with JobID ec74209eb4e3db8ae72db00bd7a830aa has finished.
  Job Runtime: 333688 ms
`
	expected = &SubmitterLog{
		jobID: "ec74209eb4e3db8ae72db00bd7a830aa",
		message: `
  /opt/flink/bin/flink run --jobmanager flinkjobcluster-sample-jobmanager:8081 --class org.apache.flink.streaming.examples.wordcount.WordCount --parallelism 2 --detached ./examples/streaming/WordCount.jar --input ./README.txt
  Starting execution of program
  Printing result to stdout. Use --output to specify output path.
  Job has been submitted with JobID ec74209eb4e3db8ae72db00bd7a830aa
  Program execution finished
  Job with JobID ec74209eb4e3db8ae72db00bd7a830aa has finished.
  Job Runtime: 333688 ms
`,
	}

	submit = getFlinkJobSubmitLogFromString(log)
	assert.Equal(t, submit.jobID, expected.jobID)
	assert.Equal(t, submit.message, expected.message)

	// job ID not found
	submit = getFlinkJobSubmitLogFromString("")
	assert.Equal(t, submit.jobID, "")
}
