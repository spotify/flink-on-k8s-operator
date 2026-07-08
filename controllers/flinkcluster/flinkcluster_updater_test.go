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

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	flink "github.com/spotify/flink-on-k8s-operator/internal/flink"
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

	t.Run("stopped recovers to running when job is active", func(t *testing.T) {
		restart := v1beta1.JobRestartPolicyNever
		replicas := int32(1)
		observed := ObservedClusterState{
			jmStatefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "rev"}},
				Spec:       appsv1.StatefulSetSpec{Replicas: &replicas},
				Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
			},
			jmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "rev"}},
				Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: "127.0.0.1"},
			},
			tmStatefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "tm",
					Labels: map[string]string{RevisionNameLabel: "rev"},
				},
				Spec:   appsv1.StatefulSetSpec{Replicas: &replicas},
				Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "rev"}},
			},
			tmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "rev"}},
			},
			revision: Revision{
				currentRevision: &appsv1.ControllerRevision{Revision: 1},
				nextRevision:    &appsv1.ControllerRevision{Revision: 1},
			},
			flinkJob: FlinkJob{
				status: &flink.Job{State: "RUNNING", Id: "abc123", Name: "my-job"},
			},
			cluster: &v1beta1.FlinkCluster{
				Spec: v1beta1.FlinkClusterSpec{
					Job:         &v1beta1.JobSpec{RestartPolicy: &restart},
					TaskManager: &v1beta1.TaskManagerSpec{DeploymentType: v1beta1.DeploymentTypeStatefulSet},
				},
				Status: v1beta1.FlinkClusterStatus{
					State: v1beta1.ClusterStateStopped,
					Components: v1beta1.FlinkClusterComponentsStatus{
						ConfigMap:         &v1beta1.ConfigMapStatus{State: v1beta1.ComponentStateReady},
						JobManager:        &v1beta1.JobManagerStatus{State: v1beta1.ComponentStateReady},
						JobManagerService: v1beta1.JobManagerServiceStatus{State: v1beta1.ComponentStateReady},
						TaskManager:       &v1beta1.TaskManagerStatus{State: v1beta1.ComponentStateReady},
						Job:               &v1beta1.JobStatus{State: v1beta1.JobStateCancelled},
					},
					Revision: v1beta1.RevisionStatus{
						CurrentRevision: "rev-1",
						NextRevision:    "rev-1",
					},
				},
			},
		}

		cluster := observed.cluster.DeepCopy()
		updater := &ClusterStatusUpdater{observed: observed}
		newStatus := updater.deriveClusterStatus(context.TODO(), cluster, &observed)

		assert.Equal(t, newStatus.State, v1beta1.ClusterStateRunning,
			"Stopped cluster should recover to Running when Flink job is active")
		assert.Equal(t, newStatus.Components.Job.State, v1beta1.JobStateRunning)
	})

	t.Run("stopped stays stopped when job is not active", func(t *testing.T) {
		restart := v1beta1.JobRestartPolicyNever
		replicas := int32(1)
		observed := ObservedClusterState{
			jmStatefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "rev"}},
				Spec:       appsv1.StatefulSetSpec{Replicas: &replicas},
				Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
			},
			jmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "rev"}},
				Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: "127.0.0.1"},
			},
			tmStatefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "tm",
					Labels: map[string]string{RevisionNameLabel: "rev"},
				},
				Spec:   appsv1.StatefulSetSpec{Replicas: &replicas},
				Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "rev"}},
			},
			tmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "rev"}},
			},
			revision: Revision{
				currentRevision: &appsv1.ControllerRevision{Revision: 1},
				nextRevision:    &appsv1.ControllerRevision{Revision: 1},
			},
			cluster: &v1beta1.FlinkCluster{
				Spec: v1beta1.FlinkClusterSpec{
					Job:         &v1beta1.JobSpec{RestartPolicy: &restart},
					TaskManager: &v1beta1.TaskManagerSpec{DeploymentType: v1beta1.DeploymentTypeStatefulSet},
				},
				Status: v1beta1.FlinkClusterStatus{
					State: v1beta1.ClusterStateStopped,
					Components: v1beta1.FlinkClusterComponentsStatus{
						ConfigMap:         &v1beta1.ConfigMapStatus{State: v1beta1.ComponentStateReady},
						JobManager:        &v1beta1.JobManagerStatus{State: v1beta1.ComponentStateReady},
						JobManagerService: v1beta1.JobManagerServiceStatus{State: v1beta1.ComponentStateReady},
						TaskManager:       &v1beta1.TaskManagerStatus{State: v1beta1.ComponentStateReady},
						Job:               &v1beta1.JobStatus{State: v1beta1.JobStateSucceeded},
					},
					Revision: v1beta1.RevisionStatus{
						CurrentRevision: "rev-1",
						NextRevision:    "rev-1",
					},
				},
			},
		}

		cluster := observed.cluster.DeepCopy()
		updater := &ClusterStatusUpdater{observed: observed}
		newStatus := updater.deriveClusterStatus(context.TODO(), cluster, &observed)

		assert.Equal(t, newStatus.State, v1beta1.ClusterStateStopped,
			"Stopped cluster should stay Stopped when job is not active")
	})

}

func TestIsClusterUpdateToDate_applicationMode(t *testing.T) {
	appMode := v1beta1.JobModeApplication
	nextRevName := "my-cluster"

	t.Run("not up to date when submitter job missing", func(t *testing.T) {
		observed := &ObservedClusterState{
			cluster: &v1beta1.FlinkCluster{
				Spec: v1beta1.FlinkClusterSpec{
					Job:         &v1beta1.JobSpec{Mode: &appMode},
					TaskManager: &v1beta1.TaskManagerSpec{DeploymentType: v1beta1.DeploymentTypeStatefulSet},
				},
				Status: v1beta1.FlinkClusterStatus{
					Revision: v1beta1.RevisionStatus{
						CurrentRevision: "my-cluster-1",
						NextRevision:    "my-cluster-2",
					},
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			tmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			jmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			tmStatefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
		}

		assert.Assert(t, !isClusterUpdateToDate(observed),
			"should not be up to date when submitter Job is nil in application mode")
	})

	t.Run("up to date when submitter job has next revision", func(t *testing.T) {
		observed := &ObservedClusterState{
			cluster: &v1beta1.FlinkCluster{
				Spec: v1beta1.FlinkClusterSpec{
					Job:         &v1beta1.JobSpec{Mode: &appMode},
					TaskManager: &v1beta1.TaskManagerSpec{DeploymentType: v1beta1.DeploymentTypeStatefulSet},
				},
				Status: v1beta1.FlinkClusterStatus{
					Revision: v1beta1.RevisionStatus{
						CurrentRevision: "my-cluster-1",
						NextRevision:    "my-cluster-2",
					},
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			tmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			jmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			tmStatefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			flinkJobSubmitter: FlinkJobSubmitter{
				job: &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
				},
			},
		}

		assert.Assert(t, isClusterUpdateToDate(observed),
			"should be up to date when submitter Job has next revision label")
	})

	t.Run("not up to date when submitter job has old revision", func(t *testing.T) {
		observed := &ObservedClusterState{
			cluster: &v1beta1.FlinkCluster{
				Spec: v1beta1.FlinkClusterSpec{
					Job:         &v1beta1.JobSpec{Mode: &appMode},
					TaskManager: &v1beta1.TaskManagerSpec{DeploymentType: v1beta1.DeploymentTypeStatefulSet},
				},
				Status: v1beta1.FlinkClusterStatus{
					Revision: v1beta1.RevisionStatus{
						CurrentRevision: "my-cluster-1",
						NextRevision:    "my-cluster-2",
					},
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			tmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			jmService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			tmStatefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: nextRevName}},
			},
			flinkJobSubmitter: FlinkJobSubmitter{
				job: &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{RevisionNameLabel: "old-revision"}},
				},
			},
		}

		assert.Assert(t, !isClusterUpdateToDate(observed),
			"should not be up to date when submitter Job has old revision label")
	})
}
