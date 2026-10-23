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
	"context"
	"testing"
	"time"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/flink"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetStatefulSetStateNotReady(t *testing.T) {
	var replicas int32 = 3
	var statefulSet = appsv1.StatefulSet{
		Spec:   appsv1.StatefulSetSpec{Replicas: &replicas},
		Status: appsv1.StatefulSetStatus{ReadyReplicas: 2},
	}
	var state = getStatefulSetState(&statefulSet)
	assert.Assert(
		t, state == v1beta1.ComponentStateNotReady)
}

func TestClusterStatus(t *testing.T) {
	t.Run("not changed", func(t *testing.T) {
		var oldStatus = v1beta1.FlinkClusterStatus{}
		var newStatus = v1beta1.FlinkClusterStatus{}
		var updater = &ClusterStatusUpdater{}
		assert.Assert(t, updater.isStatusChanged(context.TODO(), oldStatus, newStatus) == false)
	})

	t.Run("changed", func(t *testing.T) {
		var oldStatus = v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				JobManager: &v1beta1.JobManagerStatus{
					Name:  "my-jobmanager",
					State: "NotReady",
				},
				TaskManager: &v1beta1.TaskManagerStatus{
					Name:  "my-taskmanager",
					State: "NotReady",
				},
				JobManagerService: v1beta1.JobManagerServiceStatus{
					Name:  "my-jobmanager",
					State: "NotReady",
				},
				JobManagerIngress: &v1beta1.JobManagerIngressStatus{
					Name:  "my-jobmanager",
					State: "NotReady",
				},
				Job: &v1beta1.JobStatus{
					SubmitterName: "my-job",
					State:         "Pending",
				},
			},
			State: "Creating"}
		var newStatus = v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				JobManager: &v1beta1.JobManagerStatus{
					Name:  "my-jobmanager",
					State: "Ready",
				},
				TaskManager: &v1beta1.TaskManagerStatus{
					Name:  "my-taskmanager",
					State: "Ready",
				},
				JobManagerService: v1beta1.JobManagerServiceStatus{
					Name:  "my-jobmanager",
					State: "Ready",
				},
				JobManagerIngress: &v1beta1.JobManagerIngressStatus{
					Name:  "my-jobmanager",
					State: "Ready",
					URLs:  []string{"http://my-jobmanager"},
				},
				Job: &v1beta1.JobStatus{
					SubmitterName: "my-job",
					State:         "Running",
				},
			},
			State: "Creating"}
		var updater = &ClusterStatusUpdater{}
		assert.Assert(t, updater.isStatusChanged(context.TODO(), oldStatus, newStatus))
	})

	t.Run("derive status", func(t *testing.T) {
		currentRevision := "1"
		nextRevision := "1-2"
		var oldStatus = v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				JobManager: &v1beta1.JobManagerStatus{
					Name:  "my-jobmanager",
					State: v1beta1.ComponentStateReady,
				},
				TaskManager: &v1beta1.TaskManagerStatus{
					Name:  "my-taskmanager",
					State: v1beta1.ComponentStateReady,
				},
				JobManagerService: v1beta1.JobManagerServiceStatus{
					Name:  "my-jobmanager",
					State: v1beta1.ComponentStateReady,
				},
				JobManagerIngress: &v1beta1.JobManagerIngressStatus{
					Name:  "my-jobmanager",
					State: v1beta1.ComponentStateNotReady,
				},
				Job: &v1beta1.JobStatus{
					SubmitterName: "my-job",
					State:         v1beta1.JobStatePending,
				},
			},
			State: v1beta1.ClusterStateCreating,
			Revision: v1beta1.RevisionStatus{
				CurrentRevision: currentRevision,
				NextRevision:    nextRevision,
			},
		}

		restart := v1beta1.JobRestartPolicyNever
		replicas := int32(1)
		var observed = ObservedClusterState{
			jmStatefulSet: &appsv1.StatefulSet{
				Spec: appsv1.StatefulSetSpec{
					Replicas: &replicas,
				},
				Status: appsv1.StatefulSetStatus{
					ReadyReplicas: 1,
				},
			},
			jmService: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type:      corev1.ServiceTypeClusterIP,
					ClusterIP: "127.0.0.1",
				},
			},
			tmStatefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tm-service",
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: &replicas,
				},
				Status: appsv1.StatefulSetStatus{
					ReadyReplicas: 1,
				},
			},
			revision: Revision{
				currentRevision: &appsv1.ControllerRevision{
					Revision: 1,
				},
				nextRevision: &appsv1.ControllerRevision{
					Revision: 2,
				},
			},
			cluster: &v1beta1.FlinkCluster{
				Spec: v1beta1.FlinkClusterSpec{
					Job: &v1beta1.JobSpec{
						RestartPolicy: &restart,
					},
				},
				Status: v1beta1.FlinkClusterStatus{
					Components: v1beta1.FlinkClusterComponentsStatus{
						ConfigMap: &v1beta1.ConfigMapStatus{
							Name:  "my-configmap",
							State: v1beta1.ComponentStateReady,
						},
						JobManager: &v1beta1.JobManagerStatus{
							Name:  "my-jobmanager",
							State: v1beta1.ComponentStateReady,
						},
						JobManagerService: v1beta1.JobManagerServiceStatus{
							Name:  "my-jobmanager",
							State: v1beta1.ComponentStateReady,
						},
						JobManagerIngress: &v1beta1.JobManagerIngressStatus{
							State: "NotReady",
						},
						TaskManager: &v1beta1.TaskManagerStatus{
							Name:  "my-taskamanger",
							State: v1beta1.ComponentStateReady,
						},
						Job: &v1beta1.JobStatus{
							State: v1beta1.JobStateSucceeded,
						},
					},
					Revision: v1beta1.RevisionStatus{
						CurrentRevision: currentRevision,
						NextRevision:    nextRevision,
					},
				},
			},
		}

		var updater = &ClusterStatusUpdater{observed: observed}
		cluster := v1beta1.FlinkCluster{Status: oldStatus, Spec: v1beta1.FlinkClusterSpec{TaskManager: &v1beta1.TaskManagerSpec{DeploymentType: v1beta1.DeploymentTypeStatefulSet}}}
		newStatus := updater.deriveClusterStatus(context.TODO(), &cluster, &observed)
		assert.Assert(t, updater.isStatusChanged(context.TODO(), oldStatus, newStatus))
		assert.Equal(t, newStatus.State, v1beta1.ClusterStateRunning)
	})
}

func TestDeriveControlStatus_StopWithSavepointCancel(t *testing.T) {
	t.Run("job stopped via job-cancel annotation has state Cancelled", func(t *testing.T) {
		var recordedControl = &v1beta1.FlinkClusterControlStatus{
			Name:  v1beta1.ControlNameJobCancel,
			State: v1beta1.ControlStateInProgress,
		}
		var cluster = &v1beta1.FlinkCluster{
			Status: v1beta1.FlinkClusterStatus{Control: recordedControl},
		}
		var newSavepoint = &v1beta1.SavepointStatus{
			State:         v1beta1.SavepointStateSucceeded,
			TriggerReason: v1beta1.SavepointReasonJobCancel,
		}
		var newJob = &v1beta1.JobStatus{State: v1beta1.JobStateSucceeded}

		var c = deriveControlStatus(cluster, newSavepoint, newJob, recordedControl)

		assert.Equal(t, newJob.State, v1beta1.JobStateCancelled)
		assert.Equal(t, c.State, v1beta1.ControlStateSucceeded)
	})
}

func TestDeriveJobSubmitterExitCodeWithoutPod(t *testing.T) {
	// given: a Job whose submitter pod has not been observed
	updater := &ClusterStatusUpdater{}
	job := &batchv1.Job{}

	// when: the submitter exit code is derived
	exitCode, reason := updater.deriveJobSubmitterExitCodeAndReason(nil, job)

	// then: the Job is treated as still pending instead of dereferencing the missing pod
	assert.Equal(t, exitCode, int32(-1))
	assert.Equal(t, reason, "")
}

func TestStoppedApplicationClusterRecoversFromActiveJob(t *testing.T) {
	var applicationMode = v1beta1.JobModeApplication
	var replicas int32 = 1

	for _, test := range []struct {
		name                 string
		readyReplicas        int32
		expectedClusterState v1beta1.ClusterState
	}{
		{name: "ready components", readyReplicas: 1, expectedClusterState: v1beta1.ClusterStateRunning},
		{name: "unready components", readyReplicas: 0, expectedClusterState: v1beta1.ClusterStateReconciling},
	} {
		t.Run(test.name, func(t *testing.T) {
			var cluster = &v1beta1.FlinkCluster{
				Spec: v1beta1.FlinkClusterSpec{
					TaskManager: &v1beta1.TaskManagerSpec{DeploymentType: v1beta1.DeploymentTypeDeployment},
					Job:         &v1beta1.JobSpec{Mode: &applicationMode},
				},
				Status: v1beta1.FlinkClusterStatus{
					State: v1beta1.ClusterStateStopped,
					Components: v1beta1.FlinkClusterComponentsStatus{
						Job: &v1beta1.JobStatus{State: v1beta1.JobStateSucceeded},
					},
					Revision: v1beta1.RevisionStatus{
						CurrentRevision: "cluster-current-1",
						NextRevision:    "cluster-current-1",
					},
				},
			}
			var observed = ObservedClusterState{
				cluster: cluster,
				flinkJobSubmitter: FlinkJobSubmitter{
					job: &batchv1.Job{Status: batchv1.JobStatus{Active: 1}},
					pod: &corev1.Pod{},
				},
				jmService: &corev1.Service{Spec: corev1.ServiceSpec{
					Type:      corev1.ServiceTypeClusterIP,
					ClusterIP: "127.0.0.1",
				}},
				tmDeployment: &appsv1.Deployment{
					Spec:   appsv1.DeploymentSpec{Replicas: &replicas},
					Status: appsv1.DeploymentStatus{ReadyReplicas: test.readyReplicas},
				},
				revision: Revision{
					currentRevision: &appsv1.ControllerRevision{ObjectMeta: metav1.ObjectMeta{Name: "cluster-current"}, Revision: 1},
					nextRevision:    &appsv1.ControllerRevision{ObjectMeta: metav1.ObjectMeta{Name: "cluster-current"}, Revision: 1},
				},
			}

			var updater = &ClusterStatusUpdater{observed: observed}
			var status = updater.deriveClusterStatus(context.Background(), cluster, &observed)

			assert.Equal(t, status.Components.Job.State, v1beta1.JobStateDeploying)
			assert.Equal(t, status.State, test.expectedClusterState)
		})
	}
}

func TestDeriveJobStatusJobManagerDeployment(t *testing.T) {
	const jobID = "a1b2c3"
	var applicationMode = v1beta1.JobModeApplication
	var now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	var revision = v1beta1.RevisionStatus{CurrentRevision: "cluster-85dc8f749-1", NextRevision: "cluster-85dc8f749-1"}
	flinkJob := func(state string) FlinkJob {
		job := flink.Job{Id: jobID, State: state}
		return FlinkJob{status: &job, list: &flink.JobsOverview{Jobs: []flink.Job{job}}}
	}
	// A JobManager container waiting in CrashLoopBackOff after the given restarts and last run.
	crashingPod := func(restartCount int32, lastRun time.Duration) *corev1.Pod {
		finished := metav1.NewTime(now.Add(-time.Second))
		return &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name:         jobManagerContainerName,
			RestartCount: restartCount,
			State:        corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
				ExitCode:   1,
				Reason:     "Error",
				StartedAt:  metav1.NewTime(finished.Add(-lastRun)),
				FinishedAt: finished,
			}},
		}}}}
	}
	runningPod := func(startedAgo time.Duration) *corev1.Pod {
		return &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name:  jobManagerContainerName,
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(now.Add(-startedAgo))}},
		}}}}
	}
	deployment := func(replicas int32) *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: getNextRevisionName(&revision)}},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		}
	}
	completionTime := metav1.NewTime(now.Add(-time.Hour))

	for _, test := range []struct {
		name           string
		oldJob         v1beta1.JobStatus
		flinkJob       FlinkJob
		deployment     *appsv1.Deployment
		pod            *corev1.Pod
		expectedState  v1beta1.JobState
		expectedReason string
	}{
		{
			name:          "running job is kept while the Flink API is unavailable",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateRunning},
			deployment:    deployment(1),
			pod:           crashingPod(1, time.Second),
			expectedState: v1beta1.JobStateRunning,
		},
		{
			name:          "failed job is taken from the Flink API",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateRunning},
			flinkJob:      flinkJob("FAILED"),
			deployment:    deployment(1),
			expectedState: v1beta1.JobStateFailed,
		},
		{
			name:          "finished job is taken from the Flink API",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateRunning},
			flinkJob:      flinkJob("FINISHED"),
			deployment:    deployment(1),
			expectedState: v1beta1.JobStateSucceeded,
		},
		{
			name:          "running job unknown to the Flink API is lost",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateRunning},
			flinkJob:      FlinkJob{list: &flink.JobsOverview{}},
			deployment:    deployment(1),
			pod:           runningPod(jobManagerRecoveryGracePeriod),
			expectedState: v1beta1.JobStateLost,
		},
		{
			name:          "running job unknown to a recently started JobManager is kept while it recovers",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateRunning},
			flinkJob:      FlinkJob{list: &flink.JobsOverview{}},
			deployment:    deployment(1),
			pod:           runningPod(time.Minute),
			expectedState: v1beta1.JobStateRunning,
		},
		{
			name:          "lost job reported again by the Flink API is running",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateLost, CompletionTime: &completionTime},
			flinkJob:      flinkJob("RUNNING"),
			deployment:    deployment(1),
			expectedState: v1beta1.JobStateRunning,
		},
		{
			name:          "deploying job with a restarting JobManager keeps deploying",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateDeploying},
			deployment:    deployment(1),
			pod:           crashingPod(jobManagerCrashLoopRestartCount-1, time.Second),
			expectedState: v1beta1.JobStateDeploying,
		},
		{
			name:           "deploying job with a crash looping JobManager failed to deploy",
			oldJob:         v1beta1.JobStatus{State: v1beta1.JobStateDeploying},
			deployment:     deployment(1),
			pod:            crashingPod(jobManagerCrashLoopRestartCount, time.Second),
			expectedState:  v1beta1.JobStateDeployFailed,
			expectedReason: "JobManager container is crash looping after 3 restarts, last exit code 1: Error ",
		},
		{
			name:           "running job with a crash looping JobManager failed",
			oldJob:         v1beta1.JobStatus{State: v1beta1.JobStateRunning},
			deployment:     deployment(1),
			pod:            crashingPod(jobManagerCrashLoopRestartCount, time.Second),
			expectedState:  v1beta1.JobStateFailed,
			expectedReason: "JobManager container is crash looping after 3 restarts, last exit code 1: Error ",
		},
		{
			name:          "running job with a JobManager that restarted after long runs keeps running",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateRunning},
			deployment:    deployment(1),
			pod:           crashingPod(jobManagerCrashLoopRestartCount, time.Hour),
			expectedState: v1beta1.JobStateRunning,
		},
		{
			name:          "deploying job without the JobManager Deployment is deployed again",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateDeploying},
			expectedState: v1beta1.JobStatePending,
		},
		{
			name:          "running job without the JobManager Deployment is deployed again",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateRunning},
			expectedState: v1beta1.JobStatePending,
		},
		{
			name:          "stopped job with a redeployed JobManager Deployment is deploying",
			oldJob:        v1beta1.JobStatus{State: v1beta1.JobStateSucceeded, FinalSavepoint: true, DeployTime: "2026-01-01T11:59:00Z"},
			deployment:    deployment(1),
			expectedState: v1beta1.JobStateDeploying,
		},
		{
			name: "stopped job with a scaled down JobManager Deployment stays stopped",
			oldJob: v1beta1.JobStatus{
				State: v1beta1.JobStateSucceeded, DeployTime: "2026-01-01T11:00:00Z", CompletionTime: &completionTime,
			},
			deployment:    deployment(0),
			expectedState: v1beta1.JobStateSucceeded,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var oldJob = test.oldJob
			oldJob.ID = jobID
			var observed = ObservedClusterState{
				cluster: &v1beta1.FlinkCluster{
					Spec: v1beta1.FlinkClusterSpec{
						JobManager: &v1beta1.JobManagerSpec{DeploymentType: v1beta1.JobManagerDeploymentTypeDeployment},
						Job:        &v1beta1.JobSpec{Mode: &applicationMode},
					},
					Status: v1beta1.FlinkClusterStatus{
						Components: v1beta1.FlinkClusterComponentsStatus{Job: &oldJob},
						Revision:   revision,
					},
				},
				flinkJob:          test.flinkJob,
				flinkJobSubmitter: FlinkJobSubmitter{deployment: test.deployment, pod: test.pod},
				observeTime:       now,
			}

			var updater = &ClusterStatusUpdater{observed: observed}
			var jobStatus = updater.deriveJobStatus(context.Background())

			assert.Equal(t, jobStatus.State, test.expectedState)
			if test.expectedReason != "" {
				assert.DeepEqual(t, jobStatus.FailureReasons, []string{test.expectedReason})
			}
		})
	}
}
