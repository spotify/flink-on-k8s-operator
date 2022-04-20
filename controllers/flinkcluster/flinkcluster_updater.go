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

// Updater which updates the status of a cluster based on the status of its
// components.

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStatusUpdater updates the status of the FlinkCluster CR.
type ClusterStatusUpdater struct {
	k8sClient client.Client
	context   context.Context
	log       logr.Logger
	recorder  record.EventRecorder
	observed  ObservedClusterState
}

// Compares the current status recorded in the cluster's status field and the
// new status derived from the status of the components, updates the cluster
// status if it is changed, returns the new status.
func (updater *ClusterStatusUpdater) updateStatusIfChanged() (
	bool, error) {
	if updater.observed.cluster == nil {
		updater.log.Info("The cluster has been deleted, no status to update")
		return false, nil
	}

	// Current status recorded in the cluster's status field.
	var oldStatus = v1beta1.FlinkClusterStatus{}
	updater.observed.cluster.Status.DeepCopyInto(&oldStatus)
	oldStatus.LastUpdateTime = ""

	// New status derived from the cluster's components.
	var newStatus = updater.deriveClusterStatus(
		updater.observed.cluster, &updater.observed)

	// Compare
	var changed = updater.isStatusChanged(oldStatus, newStatus)

	// Update
	if changed {
		updater.log.Info(
			"Status changed",
			"old",
			updater.observed.cluster.Status,
			"new", newStatus)
		updater.createStatusChangeEvents(oldStatus, newStatus)
		var tc = &util.TimeConverter{}
		newStatus.LastUpdateTime = tc.ToString(time.Now())
		return true, updater.updateClusterStatus(newStatus)
	}

	updater.log.Info("No status change", "state", oldStatus.State)
	return false, nil
}

func (updater *ClusterStatusUpdater) createStatusChangeEvents(
	oldStatus v1beta1.FlinkClusterStatus,
	newStatus v1beta1.FlinkClusterStatus) {
	if oldStatus.Components.JobManagerStatefulSet.State !=
		newStatus.Components.JobManagerStatefulSet.State {
		updater.createStatusChangeEvent(
			"JobManager StatefulSet",
			oldStatus.Components.JobManagerStatefulSet.State,
			newStatus.Components.JobManagerStatefulSet.State)
	}

	// ConfigMap.
	if oldStatus.Components.ConfigMap.State !=
		newStatus.Components.ConfigMap.State {
		updater.createStatusChangeEvent(
			"ConfigMap",
			oldStatus.Components.ConfigMap.State,
			newStatus.Components.ConfigMap.State)
	}

	// JobManager service.
	if oldStatus.Components.JobManagerService.State !=
		newStatus.Components.JobManagerService.State {
		updater.createStatusChangeEvent(
			"JobManager service",
			oldStatus.Components.JobManagerService.State,
			newStatus.Components.JobManagerService.State)
	}

	// JobManager ingress.
	if oldStatus.Components.JobManagerIngress == nil && newStatus.Components.JobManagerIngress != nil {
		updater.createStatusChangeEvent(
			"JobManager ingress", "",
			newStatus.Components.JobManagerIngress.State)
	}
	if oldStatus.Components.JobManagerIngress != nil && newStatus.Components.JobManagerIngress != nil &&
		oldStatus.Components.JobManagerIngress.State != newStatus.Components.JobManagerIngress.State {
		updater.createStatusChangeEvent(
			"JobManager ingress",
			oldStatus.Components.JobManagerIngress.State,
			newStatus.Components.JobManagerIngress.State)
	}

	// TaskManager.
	if oldStatus.Components.TaskManagerStatefulSet.State !=
		newStatus.Components.TaskManagerStatefulSet.State {
		updater.createStatusChangeEvent(
			"TaskManager StatefulSet",
			oldStatus.Components.TaskManagerStatefulSet.State,
			newStatus.Components.TaskManagerStatefulSet.State)
	}

	// Job.
	if oldStatus.Components.Job == nil && newStatus.Components.Job != nil {
		updater.createStatusChangeEvent(
			"Job", "", newStatus.Components.Job.State)
	}
	if oldStatus.Components.Job != nil && newStatus.Components.Job != nil &&
		oldStatus.Components.Job.State != newStatus.Components.Job.State {
		updater.createStatusChangeEvent(
			"Job",
			oldStatus.Components.Job.State,
			newStatus.Components.Job.State)
	}

	// Cluster.
	if oldStatus.State != newStatus.State {
		updater.createStatusChangeEvent("Cluster", oldStatus.State, newStatus.State)
	}

	// Savepoint.
	if newStatus.Savepoint != nil && !reflect.DeepEqual(oldStatus.Savepoint, newStatus.Savepoint) {
		eventType, eventReason, eventMessage := getSavepointEvent(*newStatus.Savepoint)
		updater.recorder.Event(updater.observed.cluster, eventType, eventReason, eventMessage)
	}

	// Control.
	if newStatus.Control != nil && !reflect.DeepEqual(oldStatus.Control, newStatus.Control) {
		eventType, eventReason, eventMessage := getControlEvent(*newStatus.Control)
		updater.recorder.Event(updater.observed.cluster, eventType, eventReason, eventMessage)
	}
}

func (updater *ClusterStatusUpdater) createStatusChangeEvent(
	name string, oldStatus string, newStatus string) {
	if len(oldStatus) == 0 {
		updater.recorder.Event(
			updater.observed.cluster,
			"Normal",
			"StatusUpdate",
			fmt.Sprintf("%v status: %v", name, newStatus))
	} else {
		updater.recorder.Event(
			updater.observed.cluster,
			"Normal",
			"StatusUpdate",
			fmt.Sprintf(
				"%v status changed: %v -> %v", name, oldStatus, newStatus))
	}
}

func (updater *ClusterStatusUpdater) deriveClusterStatus(
	cluster *v1beta1.FlinkCluster,
	observed *ObservedClusterState) v1beta1.FlinkClusterStatus {
	var totalComponents int
	if IsApplicationModeCluster(cluster) {
		// jmService, tmStatefulSet.
		totalComponents = 2
	} else {
		// jmStatefulSet, jmService, tmStatefulSet.
		totalComponents = 3
	}

	var recorded = cluster.Status
	var status = v1beta1.FlinkClusterStatus{}
	var runningComponents = 0

	// ConfigMap.
	var observedConfigMap = observed.configMap
	if !isComponentUpdated(observedConfigMap, observed.cluster) && shouldUpdateCluster(observed) {
		recorded.Components.ConfigMap.DeepCopyInto(&status.Components.ConfigMap)
		status.Components.ConfigMap.State = v1beta1.ComponentStateUpdating
	} else if observedConfigMap != nil {
		status.Components.ConfigMap.Name = observedConfigMap.Name
		status.Components.ConfigMap.State = v1beta1.ComponentStateReady
	} else if recorded.Components.ConfigMap.Name != "" {
		status.Components.ConfigMap =
			v1beta1.FlinkClusterComponentState{
				Name:  recorded.Components.ConfigMap.Name,
				State: v1beta1.ComponentStateDeleted,
			}
	}

	// JobManager StatefulSet.
	var observedJmStatefulSet = observed.jmStatefulSet
	if !isComponentUpdated(observedJmStatefulSet, observed.cluster) && shouldUpdateCluster(observed) {
		recorded.Components.JobManagerStatefulSet.DeepCopyInto(&status.Components.JobManagerStatefulSet)
		status.Components.JobManagerStatefulSet.State = v1beta1.ComponentStateUpdating
	} else if observedJmStatefulSet != nil {
		status.Components.JobManagerStatefulSet.Name = observedJmStatefulSet.Name
		status.Components.JobManagerStatefulSet.State = getStatefulSetState(observedJmStatefulSet)
		if status.Components.JobManagerStatefulSet.State == v1beta1.ComponentStateReady {
			runningComponents++
		}
	} else if recorded.Components.JobManagerStatefulSet.Name != "" {
		status.Components.JobManagerStatefulSet =
			v1beta1.FlinkClusterComponentState{
				Name:  recorded.Components.JobManagerStatefulSet.Name,
				State: v1beta1.ComponentStateDeleted,
			}
	}

	// JobManager service.
	var observedJmService = observed.jmService
	if !isComponentUpdated(observedJmService, observed.cluster) && shouldUpdateCluster(observed) {
		recorded.Components.JobManagerService.DeepCopyInto(&status.Components.JobManagerService)
		status.Components.JobManagerService.State = v1beta1.ComponentStateUpdating
	} else if observedJmService != nil {
		var nodePort int32
		var loadBalancerIngress []corev1.LoadBalancerIngress
		state := v1beta1.ComponentStateNotReady

		switch observedJmService.Spec.Type {
		case corev1.ServiceTypeClusterIP:
			if observedJmService.Spec.ClusterIP != "" {
				state = v1beta1.ComponentStateReady
				runningComponents++
			}
		case corev1.ServiceTypeLoadBalancer:
			if len(observedJmService.Status.LoadBalancer.Ingress) > 0 {
				state = v1beta1.ComponentStateReady
				runningComponents++
				loadBalancerIngress = observedJmService.Status.LoadBalancer.Ingress
			}
		case corev1.ServiceTypeNodePort:
			if len(observedJmService.Spec.Ports) > 0 {
				state = v1beta1.ComponentStateReady
				runningComponents++
				for _, port := range observedJmService.Spec.Ports {
					if port.Name == "ui" {
						nodePort = port.NodePort
					}
				}
			}
		}

		status.Components.JobManagerService =
			v1beta1.JobManagerServiceStatus{
				Name:                observedJmService.Name,
				State:               state,
				NodePort:            nodePort,
				LoadBalancerIngress: loadBalancerIngress,
			}
	} else if recorded.Components.JobManagerService.Name != "" {
		status.Components.JobManagerService =
			v1beta1.JobManagerServiceStatus{
				Name:  recorded.Components.JobManagerService.Name,
				State: v1beta1.ComponentStateDeleted,
			}
	}

	// (Optional) JobManager ingress.
	var observedJmIngress = observed.jmIngress
	if !isComponentUpdated(observedJmIngress, observed.cluster) && shouldUpdateCluster(observed) {
		status.Components.JobManagerIngress = &v1beta1.JobManagerIngressStatus{}
		recorded.Components.JobManagerIngress.DeepCopyInto(status.Components.JobManagerIngress)
		status.Components.JobManagerIngress.State = v1beta1.ComponentStateUpdating
	} else if observedJmIngress != nil {
		var state string
		var urls []string
		var useTLS bool
		var useHost bool
		var loadbalancerReady bool

		if len(observedJmIngress.Spec.TLS) > 0 {
			useTLS = true
		}

		if useTLS {
			for _, tls := range observedJmIngress.Spec.TLS {
				for _, host := range tls.Hosts {
					if host != "" {
						urls = append(urls, "https://"+host)
					}
				}
			}
		} else {
			for _, rule := range observedJmIngress.Spec.Rules {
				if rule.Host != "" {
					urls = append(urls, "http://"+rule.Host)
				}
			}
		}
		if len(urls) > 0 {
			useHost = true
		}

		// Check loadbalancer is ready.
		if len(observedJmIngress.Status.LoadBalancer.Ingress) > 0 {
			var addr string
			for _, ingress := range observedJmIngress.Status.LoadBalancer.Ingress {
				// Get loadbalancer address.
				if ingress.Hostname != "" {
					addr = ingress.Hostname
				} else if ingress.IP != "" {
					addr = ingress.IP
				}
				// If ingress spec does not have host, get ip or hostname of loadbalancer.
				if !useHost && addr != "" {
					if useTLS {
						urls = append(urls, "https://"+addr)
					} else {
						urls = append(urls, "http://"+addr)
					}
				}
			}
			// If any ready LB found, state is ready.
			if addr != "" {
				loadbalancerReady = true
			}
		}

		// Jobmanager ingress state become ready when LB for ingress is specified.
		if loadbalancerReady {
			state = v1beta1.ComponentStateReady
		} else {
			state = v1beta1.ComponentStateNotReady
		}

		status.Components.JobManagerIngress =
			&v1beta1.JobManagerIngressStatus{
				Name:  observedJmIngress.Name,
				State: state,
				URLs:  urls,
			}
	} else if recorded.Components.JobManagerIngress != nil &&
		recorded.Components.JobManagerIngress.Name != "" {
		status.Components.JobManagerIngress =
			&v1beta1.JobManagerIngressStatus{
				Name:  recorded.Components.JobManagerIngress.Name,
				State: v1beta1.ComponentStateDeleted,
			}
	}

	// TaskManager StatefulSet.
	var observedTmStatefulSet = observed.tmStatefulSet
	if !isComponentUpdated(observedTmStatefulSet, observed.cluster) && shouldUpdateCluster(observed) {
		recorded.Components.TaskManagerStatefulSet.DeepCopyInto(&status.Components.TaskManagerStatefulSet)
		status.Components.TaskManagerStatefulSet.State = v1beta1.ComponentStateUpdating
	} else if observedTmStatefulSet != nil {
		status.Components.TaskManagerStatefulSet.Name =
			observedTmStatefulSet.Name
		status.Components.TaskManagerStatefulSet.State =
			getStatefulSetState(observedTmStatefulSet)
		if status.Components.TaskManagerStatefulSet.State ==
			v1beta1.ComponentStateReady {
			runningComponents++
		}
	} else if recorded.Components.TaskManagerStatefulSet.Name != "" {
		status.Components.TaskManagerStatefulSet =
			v1beta1.FlinkClusterComponentState{
				Name:  recorded.Components.TaskManagerStatefulSet.Name,
				State: v1beta1.ComponentStateDeleted,
			}
	}

	// Derive the new cluster state.
	var jobStatus = recorded.Components.Job
	switch recorded.State {
	case "", v1beta1.ClusterStateCreating:
		if runningComponents < totalComponents {
			status.State = v1beta1.ClusterStateCreating
			if jobStatus.IsStopped() {
				var policy = observed.cluster.Spec.Job.CleanupPolicy
				if jobStatus.State == v1beta1.JobStateSucceeded &&
					policy.AfterJobSucceeds != v1beta1.CleanupActionKeepCluster {
					status.State = v1beta1.ClusterStateStopping
				} else if jobStatus.IsFailed() &&
					policy.AfterJobFails != v1beta1.CleanupActionKeepCluster {
					status.State = v1beta1.ClusterStateStopping
				} else if jobStatus.State == v1beta1.JobStateCancelled &&
					policy.AfterJobCancelled != v1beta1.CleanupActionKeepCluster {
					status.State = v1beta1.ClusterStateStopping
				}
			}
		} else {
			status.State = v1beta1.ClusterStateRunning
		}
	case v1beta1.ClusterStateUpdating:
		if shouldUpdateCluster(observed) {
			status.State = v1beta1.ClusterStateUpdating
		} else if runningComponents < totalComponents {
			if recorded.Revision.IsUpdateTriggered() {
				status.State = v1beta1.ClusterStateUpdating
			} else {
				status.State = v1beta1.ClusterStateReconciling
			}
		} else {
			status.State = v1beta1.ClusterStateRunning
		}
	case v1beta1.ClusterStateRunning,
		v1beta1.ClusterStateReconciling:
		if shouldUpdateCluster(observed) {
			status.State = v1beta1.ClusterStateUpdating
		} else if !recorded.Revision.IsUpdateTriggered() && jobStatus.IsStopped() {
			var policy = observed.cluster.Spec.Job.CleanupPolicy
			if jobStatus.State == v1beta1.JobStateSucceeded &&
				policy.AfterJobSucceeds != v1beta1.CleanupActionKeepCluster {
				status.State = v1beta1.ClusterStateStopping
			} else if jobStatus.IsFailed() &&
				policy.AfterJobFails != v1beta1.CleanupActionKeepCluster {
				status.State = v1beta1.ClusterStateStopping
			} else if jobStatus.State == v1beta1.JobStateCancelled &&
				policy.AfterJobCancelled != v1beta1.CleanupActionKeepCluster {
				status.State = v1beta1.ClusterStateStopping
			} else {
				status.State = v1beta1.ClusterStateRunning
			}
		} else if runningComponents < totalComponents {
			status.State = v1beta1.ClusterStateReconciling
		} else {
			status.State = v1beta1.ClusterStateRunning
		}
	case v1beta1.ClusterStateStopping,
		v1beta1.ClusterStatePartiallyStopped:
		if shouldUpdateCluster(observed) {
			status.State = v1beta1.ClusterStateUpdating
		} else if runningComponents == 0 {
			status.State = v1beta1.ClusterStateStopped
		} else if runningComponents < totalComponents {
			status.State = v1beta1.ClusterStatePartiallyStopped
		} else {
			status.State = v1beta1.ClusterStateStopping
		}
	case v1beta1.ClusterStateStopped:
		if recorded.Revision.IsUpdateTriggered() {
			status.State = v1beta1.ClusterStateUpdating
		} else {
			status.State = v1beta1.ClusterStateStopped
		}
	default:
		panic(fmt.Sprintf("Unknown cluster state: %v", recorded.State))
	}

	// (Optional) Job.
	// Update job status.
	status.Components.Job = updater.deriveJobStatus()

	// (Optional) Savepoint.
	// Update savepoint status if it is in progress or requested.
	var newJobStatus = status.Components.Job
	status.Savepoint = updater.deriveSavepointStatus(
		&observed.savepoint,
		recorded.Savepoint,
		newJobStatus,
		updater.getFlinkJobID())

	// (Optional) Control.
	// Update user requested control status.
	status.Control = deriveControlStatus(
		observed.cluster,
		status.Savepoint,
		status.Components.Job,
		recorded.Control)

	// Update revision status.
	// When update completed, finish the process by marking CurrentRevision to NextRevision.
	status.Revision = deriveRevisionStatus(
		observed.updateState,
		&observed.revision,
		&recorded.Revision)

	return status
}

// Gets Flink job ID based on the observed state and the recorded state.
//
// It is possible that the recorded is not nil, but the observed is, due
// to transient error or being skiped as an optimization.
// If this returned nil, it is the state that job is not submitted or not identified yet.
func (updater *ClusterStatusUpdater) getFlinkJobID() *string {
	// Observed from active job manager.
	var observedFlinkJob = updater.observed.flinkJob.status
	if observedFlinkJob != nil && len(observedFlinkJob.Id) > 0 {
		return &observedFlinkJob.Id
	}

	var observedJobSubmitter = updater.observed.flinkJobSubmitter
	if observedJobSubmitter.pod != nil {
		if jobId, ok := observedJobSubmitter.pod.Labels["job-id"]; ok && jobId != "" {
			return &jobId
		}
	}

	// Observed from job submitter (when Flink API is not ready).
	if observedJobSubmitter.log != nil && observedJobSubmitter.log.jobID != "" {
		return &observedJobSubmitter.log.jobID
	}

	// Recorded.
	var recordedJobStatus = updater.observed.cluster.Status.Components.Job
	if recordedJobStatus != nil && len(recordedJobStatus.ID) > 0 {
		return &recordedJobStatus.ID
	}

	return nil
}

func (updater *ClusterStatusUpdater) deriveJobStatus() *v1beta1.JobStatus {
	var observed = updater.observed
	var observedCluster = observed.cluster
	var jobSpec = observedCluster.Spec.Job
	if jobSpec == nil {
		return nil
	}

	var observedSubmitter = observed.flinkJobSubmitter
	var observedFlinkJob = observed.flinkJob.status
	var observedSavepoint = observed.savepoint
	var recorded = observedCluster.Status
	var savepoint = recorded.Savepoint
	var oldJob = recorded.Components.Job
	var newJob *v1beta1.JobStatus

	// Derive new job state.
	if oldJob != nil {
		newJob = oldJob.DeepCopy()
	} else {
		newJob = new(v1beta1.JobStatus)
	}

	if observedSubmitter.job != nil {
		newJob.SubmitterName = observedSubmitter.job.Name
	}

	var newJobState string
	switch {
	case oldJob == nil:
		newJobState = v1beta1.JobStatePending
	case shouldUpdateJob(&observed):
		newJobState = v1beta1.JobStateUpdating
	case oldJob.ShouldRestart(jobSpec):
		newJobState = v1beta1.JobStateRestarting
	case oldJob.IsStopped():
		newJobState = oldJob.State
	case oldJob.IsPending() && oldJob.DeployTime != "":
		newJobState = v1beta1.JobStateDeploying
	// Derive the job state from the observed Flink job, if it exists.
	case observedFlinkJob != nil:
		newJobState = getFlinkJobDeploymentState(observedFlinkJob.State)
		// Unexpected Flink job state
		if newJobState == "" {
			panic(fmt.Sprintf("Unknown Flink job status: %s", observedFlinkJob.State))
		}
		newJob.ID = observedFlinkJob.Id
		newJob.Name = observedFlinkJob.Name
	case oldJob.IsActive() && observedSubmitter.job != nil && observedSubmitter.job.Status.Active == 0:
		if observedSubmitter.job.Status.Succeeded == 1 {
			newJobState = v1beta1.JobStateSucceeded
		} else if observedSubmitter.job.Status.Failed == 1 {
			newJobState = v1beta1.JobStateFailed
		} else {
			newJobState = oldJob.State
		}
	case shouldStopJob(observedCluster):
		newJobState = v1beta1.JobStateCancelled
	// When Flink job not found in JobManager or JobManager is unavailable
	case isFlinkAPIReady(observed.flinkJob.list):
		if oldJob.State == v1beta1.JobStateRunning {
			newJobState = v1beta1.JobStateLost
			break
		}
		fallthrough
	default:
		// Maintain the job state as recorded if job is not being deployed.
		if oldJob.State != v1beta1.JobStateDeploying {
			newJobState = oldJob.State
			break
		}
		// Job must be in deployment but the submitter not found or tracking failed.
		var jobDeployState = observedSubmitter.getState()
		if observedSubmitter.job == nil || jobDeployState == JobDeployStateUnknown {
			newJobState = v1beta1.JobStateLost
			break
		}
		// Case in which the job submission clearly fails even if it is not confirmed by JobManager
		// Job submitter is deployed but failed.
		if jobDeployState == JobDeployStateFailed {
			newJobState = v1beta1.JobStateDeployFailed
			break
		}

		newJobState = oldJob.State
	}
	// Update State
	newJob.State = newJobState

	// Derived new job status if the state is changed.
	if oldJob == nil || oldJob.State != newJob.State {
		// TODO: It would be ideal to set the times with the timestamp retrieved from the Flink API like /jobs/{job-id}.
		switch {
		case newJob.IsPending():
			newJob.DeployTime = ""
			switch newJob.State {
			case v1beta1.JobStateUpdating:
				newJob.RestartCount = 0
			case v1beta1.JobStateRestarting:
				newJob.RestartCount++
			}
		case newJob.State == v1beta1.JobStateRunning:
			util.SetTimestamp(&newJob.StartTime)
			newJob.CompletionTime = nil
			// When job started, the savepoint is not the final state of the job any more.
			if oldJob.FinalSavepoint {
				newJob.FinalSavepoint = false
			}
		case newJob.IsFailed():
			if len(newJob.FailureReasons) == 0 {
				newJob.FailureReasons = []string{}
				exceptions := observed.flinkJob.exceptions
				if exceptions != nil && len(exceptions.Exceptions) > 0 {
					for _, e := range exceptions.Exceptions {
						newJob.FailureReasons = append(newJob.FailureReasons, e.Exception)
					}
				}
				if observedSubmitter.log != nil {
					newJob.FailureReasons = append(newJob.FailureReasons, observedSubmitter.log.message)
				}
			}
			fallthrough
		case newJob.IsStopped():
			if newJob.CompletionTime.IsZero() {
				now := metav1.Now()
				newJob.CompletionTime = &now
			}
			// When tracking failed, we cannot guarantee if the savepoint is the final job state.
			if newJob.State == v1beta1.JobStateLost && oldJob.FinalSavepoint {
				newJob.FinalSavepoint = false
			}
		}
	}

	// Savepoint
	if observedSavepoint.status != nil && observedSavepoint.status.IsSuccessful() {
		newJob.SavepointGeneration++
		newJob.SavepointLocation = observedSavepoint.status.Location
		if finalSavepointRequested(newJob.ID, savepoint) {
			newJob.FinalSavepoint = true
		}
		// TODO: SavepointTime should be set with the timestamp generated in job manager.
		// Currently savepoint complete timestamp is not included in savepoints API response.
		// Whereas checkpoint API returns the timestamp latest_ack_timestamp.
		// Note: https://ci.apache.org/projects/flink/flink-docs-stable/ops/rest_api.html#jobs-jobid-checkpoints-details-checkpointid
		util.SetTimestamp(&newJob.SavepointTime)
	}

	return newJob
}

func (updater *ClusterStatusUpdater) isStatusChanged(
	currentStatus v1beta1.FlinkClusterStatus,
	newStatus v1beta1.FlinkClusterStatus) bool {
	var changed = false
	if newStatus.State != currentStatus.State {
		changed = true
		updater.log.Info(
			"Cluster state changed",
			"current",
			currentStatus.State,
			"new",
			newStatus.State)
	}
	if !reflect.DeepEqual(newStatus.Control, currentStatus.Control) {
		updater.log.Info(
			"Control status changed", "current",
			currentStatus.Control,
			"new",
			newStatus.Control)
		changed = true
	}
	if newStatus.Components.ConfigMap !=
		currentStatus.Components.ConfigMap {
		updater.log.Info(
			"ConfigMap status changed",
			"current",
			currentStatus.Components.ConfigMap,
			"new",
			newStatus.Components.ConfigMap)
		changed = true
	}
	if newStatus.Components.JobManagerStatefulSet !=
		currentStatus.Components.JobManagerStatefulSet {
		updater.log.Info(
			"JobManager StatefulSet status changed",
			"current", currentStatus.Components.JobManagerStatefulSet,
			"new",
			newStatus.Components.JobManagerStatefulSet)
		changed = true
	}
	if !reflect.DeepEqual(newStatus.Components.JobManagerService, currentStatus.Components.JobManagerService) {
		updater.log.Info(
			"JobManager service status changed",
			"current",
			currentStatus.Components.JobManagerService,
			"new", newStatus.Components.JobManagerService)
		changed = true
	}
	if currentStatus.Components.JobManagerIngress == nil {
		if newStatus.Components.JobManagerIngress != nil {
			updater.log.Info(
				"JobManager ingress status changed",
				"current",
				"nil",
				"new", *newStatus.Components.JobManagerIngress)
			changed = true
		}
	} else {
		if newStatus.Components.JobManagerIngress.State != currentStatus.Components.JobManagerIngress.State {
			updater.log.Info(
				"JobManager ingress status changed",
				"current",
				*currentStatus.Components.JobManagerIngress,
				"new",
				*newStatus.Components.JobManagerIngress)
			changed = true
		}
	}
	if newStatus.Components.TaskManagerStatefulSet !=
		currentStatus.Components.TaskManagerStatefulSet {
		updater.log.Info(
			"TaskManager StatefulSet status changed",
			"current",
			currentStatus.Components.TaskManagerStatefulSet,
			"new",
			newStatus.Components.TaskManagerStatefulSet)
		changed = true
	}
	if currentStatus.Components.Job == nil {
		if newStatus.Components.Job != nil {
			updater.log.Info(
				"Job status changed",
				"current",
				"nil",
				"new",
				*newStatus.Components.Job)
			changed = true
		}
	} else {
		if newStatus.Components.Job != nil {
			var isEqual = reflect.DeepEqual(
				newStatus.Components.Job, currentStatus.Components.Job)
			if !isEqual {
				updater.log.Info(
					"Job status changed",
					"current",
					*currentStatus.Components.Job,
					"new",
					*newStatus.Components.Job)
				changed = true
			}
		} else {
			changed = true
		}
	}
	if !reflect.DeepEqual(newStatus.Savepoint, currentStatus.Savepoint) {
		updater.log.Info(
			"Savepoint status changed", "current",
			currentStatus.Savepoint,
			"new",
			newStatus.Savepoint)
		changed = true
	}
	var nr = newStatus.Revision     // New revision status
	var cr = currentStatus.Revision // Current revision status
	if nr.CurrentRevision != cr.CurrentRevision ||
		nr.NextRevision != cr.NextRevision ||
		(nr.CollisionCount != nil && cr.CollisionCount == nil) ||
		(cr.CollisionCount != nil && *nr.CollisionCount != *cr.CollisionCount) {
		updater.log.Info(
			"FlinkCluster revision status changed", "current",
			fmt.Sprintf("currentRevision: %v, nextRevision: %v, collisionCount: %v", cr.CurrentRevision, cr.NextRevision, cr.CollisionCount),
			"new",
			fmt.Sprintf("currentRevision: %v, nextRevision: %v, collisionCount: %v", nr.CurrentRevision, nr.NextRevision, nr.CollisionCount))
		changed = true
	}
	return changed
}

func (updater *ClusterStatusUpdater) updateClusterStatus(
	status v1beta1.FlinkClusterStatus) error {
	var cluster = v1beta1.FlinkCluster{}
	updater.observed.cluster.DeepCopyInto(&cluster)
	cluster.Status = status
	updater.log.Info("(ClusterStatusUpdater) Updating cluster status", "clusterClone", cluster, "cluster.Status", cluster.Status)
	err := updater.k8sClient.Status().Update(updater.context, &cluster)
	// Clear control annotation after status update is complete.
	updater.clearControlAnnotation(status.Control)
	return err
}

// Clear finished or improper user control in annotations
func (updater *ClusterStatusUpdater) clearControlAnnotation(newControlStatus *v1beta1.FlinkClusterControlStatus) error {
	var userControl = updater.observed.cluster.Annotations[v1beta1.ControlAnnotation]
	if userControl == "" {
		return nil
	}
	if newControlStatus == nil || userControl != newControlStatus.Name || // refused control in updater
		(userControl == newControlStatus.Name && isUserControlFinished(newControlStatus)) /* finished control */ {
		// make annotation patch cleared
		annotationPatch := objectForPatch{
			Metadata: objectMetaForPatch{
				Annotations: map[string]interface{}{
					v1beta1.ControlAnnotation: nil,
				},
			},
		}
		patchBytes, err := json.Marshal(&annotationPatch)
		if err != nil {
			return err
		}
		return updater.k8sClient.Patch(updater.context, updater.observed.cluster, client.RawPatch(types.MergePatchType, patchBytes))
	}

	return nil
}

func (updater *ClusterStatusUpdater) deriveSavepointStatus(
	observedSavepoint *Savepoint,
	recordedSavepointStatus *v1beta1.SavepointStatus,
	newJobStatus *v1beta1.JobStatus,
	flinkJobID *string) *v1beta1.SavepointStatus {
	if recordedSavepointStatus == nil {
		return nil
	}

	// Derived savepoint status to return
	var s = recordedSavepointStatus.DeepCopy()
	var errMsg string

	// Update the savepoint status when observed savepoint is found.
	if s.State == v1beta1.SavepointStateInProgress {
		// Derive the state from the observed savepoint in JobManager.
		status := observedSavepoint.status
		switch {
		case status != nil && status.IsSuccessful():
			s.State = v1beta1.SavepointStateSucceeded
		case status != nil && status.IsFailed():
			s.State = v1beta1.SavepointStateFailed
			errMsg = fmt.Sprintf("Savepoint error: %v", observedSavepoint.status.FailureCause.StackTrace)
		case observedSavepoint.error != nil:
			s.State = v1beta1.SavepointStateFailed
			errMsg = fmt.Sprintf("Failed to get savepoint status: %v", observedSavepoint.error)
		}
		// Derive the failure state from Flink job status.
		// Append additional error message if it already exists.
		switch {
		case newJobStatus.IsStopped():
			errMsg = "Flink job is stopped: " + errMsg
			s.State = v1beta1.SavepointStateFailed
		case flinkJobID == nil || *flinkJobID != recordedSavepointStatus.JobID:
			errMsg = "Savepoint triggered Flink job is lost: " + errMsg
			s.State = v1beta1.SavepointStateFailed
		}
	}
	// TODO: Record event or introduce Condition in CRD status to notify update state pended.
	// https://github.com/kubernetes/apimachinery/blob/57f2a0733447cfd41294477d833cce6580faaca3/pkg/apis/meta/v1/types.go#L1376
	// Make up message.
	if errMsg != "" {
		if s.TriggerReason == v1beta1.SavepointReasonUpdate {
			errMsg =
				"Failed to take savepoint for update. " +
					"The update process is being postponed until a savepoint is available. " + errMsg
		}
		if len(errMsg) > 1024 {
			errMsg = errMsg[:1024]
		}
		s.Message = errMsg
	}

	return s
}

func deriveControlStatus(
	cluster *v1beta1.FlinkCluster,
	newSavepoint *v1beta1.SavepointStatus,
	newJob *v1beta1.JobStatus,
	recordedControl *v1beta1.FlinkClusterControlStatus) *v1beta1.FlinkClusterControlStatus {
	var controlRequest = getNewControlRequest(cluster)

	// Derived control status to return
	var c *v1beta1.FlinkClusterControlStatus

	// New control status
	if controlRequest != "" {
		c = getControlStatus(controlRequest, v1beta1.ControlStateRequested)
		return c
	}

	// Update control status in progress.
	if recordedControl != nil && recordedControl.State == v1beta1.ControlStateInProgress {
		c = recordedControl.DeepCopy()
		switch recordedControl.Name {
		case v1beta1.ControlNameJobCancel:
			switch {
			case newJob.State == v1beta1.JobStateCancelled:
				if newSavepoint != nil {
					if newSavepoint.State == v1beta1.SavepointStateSucceeded {
						c.State = v1beta1.ControlStateSucceeded
					} else if newSavepoint.IsFailed() && newSavepoint.TriggerReason == v1beta1.SavepointReasonJobCancel {
						c.Message = "Aborted job cancellation: failed to take savepoint."
						c.State = v1beta1.ControlStateFailed
					}
				} else {
					c.State = v1beta1.ControlStateSucceeded
				}
			case newJob.IsStopped():
				c.Message = "Aborted job cancellation: job is stopped already."
				c.State = v1beta1.ControlStateFailed
			}
		case v1beta1.ControlNameSavepoint:
			if newSavepoint == nil {
				c.Message = "Aborted: savepoint not defined"
				c.State = v1beta1.ControlStateFailed
			} else if newSavepoint.State == v1beta1.SavepointStateSucceeded {
				c.State = v1beta1.ControlStateSucceeded
			} else if newSavepoint.IsFailed() && newSavepoint.TriggerReason == v1beta1.SavepointReasonUserRequested {
				c.State = v1beta1.ControlStateFailed
			}
		}
		// Update time when state changed.
		if c.State != v1beta1.ControlStateInProgress {
			util.SetTimestamp(&c.UpdateTime)
		}
		return c
	}

	// Maintain control status if there is no change.
	if recordedControl != nil && c == nil {
		c = recordedControl.DeepCopy()
		return c
	}

	return nil
}

func deriveRevisionStatus(
	updateState UpdateState,
	observedRevision *Revision,
	recordedRevision *v1beta1.RevisionStatus,
) v1beta1.RevisionStatus {
	// Derived revision status
	var r = v1beta1.RevisionStatus{}

	// Finalize update process.
	if updateState == UpdateStateFinished {
		r.CurrentRevision = recordedRevision.NextRevision
	}

	// Update revision status.
	r.NextRevision = util.GetRevisionWithNameNumber(observedRevision.nextRevision)
	if r.CurrentRevision == "" {
		if recordedRevision.CurrentRevision == "" {
			r.CurrentRevision = util.GetRevisionWithNameNumber(observedRevision.currentRevision)
		} else {
			r.CurrentRevision = recordedRevision.CurrentRevision
		}
	}
	if observedRevision.collisionCount != 0 {
		r.CollisionCount = new(int32)
		*r.CollisionCount = observedRevision.collisionCount
	}

	return r
}

func getStatefulSetState(statefulSet *appsv1.StatefulSet) string {
	if statefulSet.Status.ReadyReplicas >= *statefulSet.Spec.Replicas {
		return v1beta1.ComponentStateReady
	}
	return v1beta1.ComponentStateNotReady
}
