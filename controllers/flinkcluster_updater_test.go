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

package controllers

import (
	"testing"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/api/v1beta1"
	"gotest.tools/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
		assert.Assert(t, updater.isStatusChanged(oldStatus, newStatus) == false)
	})

	t.Run("changed", func(t *testing.T) {
		var oldStatus = v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				JobManagerStatefulSet: v1beta1.FlinkClusterComponentState{
					Name:  "my-jobmanager",
					State: "NotReady",
				},
				TaskManagerStatefulSet: v1beta1.FlinkClusterComponentState{
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
				JobManagerStatefulSet: v1beta1.FlinkClusterComponentState{
					Name:  "my-jobmanager",
					State: "Ready",
				},
				TaskManagerStatefulSet: v1beta1.FlinkClusterComponentState{
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
		var updater = &ClusterStatusUpdater{log: log.Log}
		assert.Assert(t, updater.isStatusChanged(oldStatus, newStatus))
	})

	t.Run("derive status", func(t *testing.T) {
		currentRevision := "1"
		nextRevision := "1-2"
		var oldStatus = v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				JobManagerStatefulSet: v1beta1.FlinkClusterComponentState{
					Name:  "my-jobmanager",
					State: v1beta1.ComponentStateReady,
				},
				TaskManagerStatefulSet: v1beta1.FlinkClusterComponentState{
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
						ConfigMap: v1beta1.FlinkClusterComponentState{
							Name:  "my-configmap",
							State: v1beta1.ComponentStateReady,
						},
						JobManagerStatefulSet: v1beta1.FlinkClusterComponentState{
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
						TaskManagerStatefulSet: v1beta1.FlinkClusterComponentState{
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

		var updater = &ClusterStatusUpdater{log: log.Log, observed: observed}
		newStatus := updater.deriveClusterStatus(&oldStatus, &observed)
		assert.Assert(t, updater.isStatusChanged(oldStatus, newStatus))
		assert.Equal(t, newStatus.State, v1beta1.ClusterStateRunning)
	})

}
