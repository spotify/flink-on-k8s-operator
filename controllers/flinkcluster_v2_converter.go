/*
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
	"time"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/api/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/controllers/model"
)

// Converter which converts the FlinkCluster spec to the desired
// underlying Kubernetes resource specs.

// Gets the desired state of a cluster.
func (handler *FlinkClusterHandlerV2) getDesiredClusterState(observed ObservedClusterState, now time.Time) model.DesiredClusterState {
	var cluster = observed.cluster

	// The cluster has been deleted, all resources should be cleaned up.
	if observed.cluster == nil {
		return model.DesiredClusterState{}
	}

	return model.DesiredClusterState{
		ConfigMap:     getDesiredConfigMap(cluster, !handler.shouldCreateResource(observed, "ConfigMap")),
		JmStatefulSet: getDesiredJobManagerStatefulSet(observed.cluster, !handler.shouldCreateResource(observed, "JobManagerStatefulSet")),
		JmService:     getDesiredJobManagerService(observed.cluster, !handler.shouldCreateResource(observed, "JobManagerService")),
		JmIngress:     getDesiredJobManagerIngress(observed.cluster, !handler.shouldCreateResource(observed, "JobManagerIngress")),
		TmStatefulSet: getDesiredTaskManagerStatefulSet(observed.cluster, !handler.shouldCreateResource(observed, "TaskManagerStatefulSet")),
		Job:           getDesiredJob(observed.cluster, !handler.shouldCreateJob(observed)),
	}
}

// ======================================================================================

func (handler *FlinkClusterHandlerV2) shouldCreateJob(observed ObservedClusterState) bool {
	if observed.cluster.Spec.Job == nil {
		handler.log.Info("Is not job cluster")
		return false
	}

	if isJobCancelRequested(*observed.cluster) {
		return false
	}

	if observed.isNewRevision() {
		return true
	}

	if observed.cluster.Status.Components.Job != nil {
		switch observed.cluster.Status.Components.Job.State {
		case v1beta1.JobStateFailed, v1beta1.JobStateCancelled, v1beta1.JobStateSuspended, v1beta1.JobStateLost:
			if shouldRestartJobV2(observed.cluster.Spec.Job.RestartPolicy, observed.cluster.Status.Components.Job) {
				return true
			} else {
				return false
			}
		case v1beta1.JobStateSucceeded:
			handler.log.Info("Status succeeded")
			return false
		default:
			handler.log.Info("Other status")
		}
	}

	return true
}

func (handler *FlinkClusterHandlerV2) shouldCreateResource(observed ObservedClusterState, component string) bool {

	// Session cluster.
	if observed.cluster.Spec.Job == nil {
		return true
	}

	// Job not started
	if observed.cluster.Status.Components.Job == nil {
		return true
	}

	if observed.isNewRevision() {
		return true
	}

	var action v1beta1.CleanupAction
	switch observed.cluster.Status.Components.Job.State {
	case v1beta1.JobStateSucceeded:
		action = observed.cluster.Spec.Job.CleanupPolicy.AfterJobSucceeds
	case v1beta1.JobStateFailed, v1beta1.JobStateLost:
		action = observed.cluster.Spec.Job.CleanupPolicy.AfterJobFails
	case v1beta1.JobStateCancelled:
		action = observed.cluster.Spec.Job.CleanupPolicy.AfterJobCancelled
	default:
		return true
	}

	switch action {
	case v1beta1.CleanupActionDeleteCluster:
		return false
	case v1beta1.CleanupActionDeleteTaskManager:
		return component != "TaskManagerStatefulSet"
	default:
		return true
	}
}
