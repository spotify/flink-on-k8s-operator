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

	"github.com/google/go-cmp/cmp"
	v1beta1 "github.com/spotify/flink-on-k8s-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Compares the current status recorded in the cluster's status field and the
// new status derived from the status of the components, updates the cluster
// status if it is changed, returns the new status.
func (updater *FlinkClusterHandlerV2) updateStatusIfChanged(ctx context.Context, observed ObservedClusterState) (bool, error) {
	if observed.cluster == nil {
		updater.log.Info("The cluster has been deleted, no status to update")
		return false, nil
	}

	// Current status recorded in the cluster's status field.
	var oldStatus = v1beta1.FlinkClusterStatus{}
	observed.cluster.Status.DeepCopyInto(&oldStatus)
	oldStatus.LastUpdateTime = ""

	// New status derived from the cluster's components.
	var newStatus = updater.deriveClusterStatus(observed)

	var changed = false
	if diff := cmp.Diff(oldStatus, newStatus); diff != "" {
		updater.log.Info("Status changed")
		updater.log.Info("Status diff", "diff", diff)
		changed = true
	}

	// Update
	if changed {
		updater.createStatusChangeEvents(observed, oldStatus, newStatus)
		var tc = &TimeConverter{}
		newStatus.LastUpdateTime = tc.ToString(time.Now())
		return true, updater.updateClusterStatus(ctx, observed, newStatus)
	} else {
		updater.log.Info("No status change", "state", oldStatus.State)
		return false, nil
	}
}

func (updater *FlinkClusterHandlerV2) createStatusChangeEvents(observed ObservedClusterState, oldStatus v1beta1.FlinkClusterStatus, newStatus v1beta1.FlinkClusterStatus) {
	if oldStatus.Components.JobManagerStatefulSet.State !=
		newStatus.Components.JobManagerStatefulSet.State {
		updater.createStatusChangeEvent(
			observed,
			"JobManager StatefulSet",
			oldStatus.Components.JobManagerStatefulSet.State,
			newStatus.Components.JobManagerStatefulSet.State)
	}

	// ConfigMap.
	if oldStatus.Components.ConfigMap.State !=
		newStatus.Components.ConfigMap.State {
		updater.createStatusChangeEvent(
			observed,
			"ConfigMap",
			oldStatus.Components.ConfigMap.State,
			newStatus.Components.ConfigMap.State)
	}

	// JobManager service.
	if oldStatus.Components.JobManagerService.State !=
		newStatus.Components.JobManagerService.State {
		updater.createStatusChangeEvent(
			observed,
			"JobManager service",
			oldStatus.Components.JobManagerService.State,
			newStatus.Components.JobManagerService.State)
	}

	// JobManager ingress.
	if oldStatus.Components.JobManagerIngress == nil && newStatus.Components.JobManagerIngress != nil {
		updater.createStatusChangeEvent(
			observed,
			"JobManager ingress",
			"",
			newStatus.Components.JobManagerIngress.State)
	}
	if oldStatus.Components.JobManagerIngress != nil && newStatus.Components.JobManagerIngress != nil &&
		oldStatus.Components.JobManagerIngress.State != newStatus.Components.JobManagerIngress.State {
		updater.createStatusChangeEvent(
			observed,
			"JobManager ingress",
			oldStatus.Components.JobManagerIngress.State,
			newStatus.Components.JobManagerIngress.State)
	}

	// TaskManager.
	if oldStatus.Components.TaskManagerStatefulSet.State !=
		newStatus.Components.TaskManagerStatefulSet.State {
		updater.createStatusChangeEvent(
			observed,
			"TaskManager StatefulSet",
			oldStatus.Components.TaskManagerStatefulSet.State,
			newStatus.Components.TaskManagerStatefulSet.State)
	}

	// Job.
	if oldStatus.Components.Job == nil && newStatus.Components.Job != nil {
		updater.createStatusChangeEvent(
			observed,
			"Job",
			"",
			newStatus.Components.Job.State)
	}
	if oldStatus.Components.Job != nil && newStatus.Components.Job != nil &&
		oldStatus.Components.Job.State != newStatus.Components.Job.State {
		updater.createStatusChangeEvent(
			observed,
			"Job",
			oldStatus.Components.Job.State,
			newStatus.Components.Job.State)
	}

	// Cluster.
	if oldStatus.State != newStatus.State {
		updater.createStatusChangeEvent(
			observed,
			"Cluster",
			oldStatus.State,
			newStatus.State)
	}

	// Savepoint.
	if newStatus.Savepoint != nil && !reflect.DeepEqual(oldStatus.Savepoint, newStatus.Savepoint) {
		eventType, eventReason, eventMessage := getSavepointEvent(*newStatus.Savepoint)
		updater.recorder.Event(observed.cluster, eventType, eventReason, eventMessage)
	}

	// Control.
	if newStatus.Control != nil && !reflect.DeepEqual(oldStatus.Control, newStatus.Control) {
		eventType, eventReason, eventMessage := getControlEvent(*newStatus.Control)
		updater.recorder.Event(observed.cluster, eventType, eventReason, eventMessage)
	}
}

func (updater *FlinkClusterHandlerV2) createStatusChangeEvent(observed ObservedClusterState, name string, oldStatus string, newStatus string) {
	if len(oldStatus) == 0 {
		updater.recorder.Event(
			observed.cluster,
			corev1.EventTypeNormal,
			"StatusUpdate",
			fmt.Sprintf("%v status: %v", name, newStatus))
	} else {
		updater.recorder.Event(
			observed.cluster,
			corev1.EventTypeNormal,
			"StatusUpdate",
			fmt.Sprintf("%v status changed: %v -> %v", name, oldStatus, newStatus))
	}
}

func (updater *FlinkClusterHandlerV2) deriveClusterStatus(observed ObservedClusterState) v1beta1.FlinkClusterStatus {
	recorded := observed.cluster.Status

	var status = v1beta1.FlinkClusterStatus{}
	// jmStatefulSet, jmService, tmStatefulSet.
	var isJobUpdating = recorded.Components.Job != nil && recorded.Components.Job.State == v1beta1.JobStateUpdating

	status.Components.ConfigMap = updater.deriveConfigMapStatus(observed)
	status.Components.JobManagerStatefulSet = updater.deriveStatefulSetStatus(observed, observed.jmStatefulSet, observed.cluster.Status.Components.JobManagerStatefulSet)
	status.Components.JobManagerService = updater.deriveServiceStatus(observed)
	status.Components.TaskManagerStatefulSet = updater.deriveStatefulSetStatus(observed, observed.tmStatefulSet, observed.cluster.Status.Components.TaskManagerStatefulSet)

	// (Optional) JobManager ingress.
	var observedJmIngress = observed.jmIngress
	if !isComponentUpdated(observedJmIngress, *observed.cluster) && isJobUpdating {
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

	// (Optional) Job.
	var jobStatus = updater.deriveJobStatus(observed)
	status.Components.Job = jobStatus

	// Derive the new cluster state.
	status.State = updater.deriveClusterState(observed, status)

	// Savepoint status
	// update savepoint status if it is in progress
	if recorded.Savepoint != nil {
		var newSavepointStatus = recorded.Savepoint.DeepCopy()
		if recorded.Savepoint.State == v1beta1.SavepointStateInProgress && observed.savepoint != nil {
			switch {
			case observed.savepoint.IsSuccessful():
				newSavepointStatus.State = v1beta1.SavepointStateSucceeded
			case observed.savepoint.IsFailed():
				var msg string
				newSavepointStatus.State = v1beta1.SavepointStateFailed
				if observed.savepoint.FailureCause.StackTrace != "" {
					msg = fmt.Sprintf("Savepoint error: %v", observed.savepoint.FailureCause.StackTrace)
				} else if observed.savepointErr != nil {
					msg = fmt.Sprintf("Failed to get triggered savepoint status: %v", observed.savepointErr)
				} else {
					msg = "Failed to get triggered savepoint status"
				}
				if len(msg) > 1024 {
					msg = msg[:1024] + "..."
				}
				newSavepointStatus.Message = msg
				// TODO: organize more making savepoint status
				if newSavepointStatus.TriggerReason == v1beta1.SavepointTriggerReasonUpdate {
					newSavepointStatus.Message =
						"Failed to take savepoint for update. " +
							"The update process is being postponed until a savepoint is available. " + newSavepointStatus.Message
				}
			}
		}
		if newSavepointStatus.State == v1beta1.SavepointStateNotTriggered || newSavepointStatus.State == v1beta1.SavepointStateInProgress {
			var flinkJobID = observed.getFlinkJobId()
			switch {
			case savepointTimeout(newSavepointStatus):
				newSavepointStatus.State = v1beta1.SavepointStateFailed
				newSavepointStatus.Message = "Timed out taking savepoint."
			case isJobStopped(recorded.Components.Job):
				newSavepointStatus.Message = "Flink job is stopped."
				newSavepointStatus.State = v1beta1.SavepointStateFailed
			case observed.flinkJobStatus.flinkJobList == nil:
				newSavepointStatus.Message = "Flink API is not available."
				newSavepointStatus.State = v1beta1.SavepointStateFailed
			case flinkJobID == nil:
				newSavepointStatus.Message = "Flink job is not submitted or identified."
				newSavepointStatus.State = v1beta1.SavepointStateFailed
			case flinkJobID != nil && (recorded.Savepoint.TriggerID != "" && *flinkJobID != recorded.Savepoint.JobID):
				newSavepointStatus.Message = "Savepoint triggered Flink job is lost."
				newSavepointStatus.State = v1beta1.SavepointStateFailed
			}
			// TODO: organize more making savepoint status
			if newSavepointStatus.State == v1beta1.SavepointStateFailed &&
				newSavepointStatus.TriggerReason == v1beta1.SavepointTriggerReasonUpdate {
				newSavepointStatus.Message =
					"Failed to take savepoint for update. " +
						"The update process is being postponed until a savepoint is available. " + newSavepointStatus.Message
			}
		}
		status.Savepoint = newSavepointStatus
	}

	// User requested control
	var userControl = observed.cluster.Annotations[v1beta1.ControlAnnotation]

	// Update job control status in progress
	var controlStatus *v1beta1.FlinkClusterControlStatus
	if recorded.Control != nil && userControl == recorded.Control.Name &&
		recorded.Control.State == v1beta1.ControlStateProgressing {
		controlStatus = recorded.Control.DeepCopy()
		var savepointStatus = status.Savepoint
		switch recorded.Control.Name {
		case v1beta1.ControlNameJobCancel:
			if status.Components.Job.State == v1beta1.JobStateCancelled {
				controlStatus.State = v1beta1.ControlStateSucceeded
				setTimestamp(&controlStatus.UpdateTime)
			} else if isJobTerminated(observed.cluster.Spec.Job.RestartPolicy, recorded.Components.Job) {
				controlStatus.Message = "Aborted job cancellation: Job is terminated."
				controlStatus.State = v1beta1.ControlStateFailed
				setTimestamp(&controlStatus.UpdateTime)
			} else if savepointStatus != nil && savepointStatus.State == v1beta1.SavepointStateFailed {
				controlStatus.Message = "Aborted job cancellation: failed to create savepoint."
				controlStatus.State = v1beta1.ControlStateFailed
				setTimestamp(&controlStatus.UpdateTime)
			} else if recorded.Control.Message != "" {
				controlStatus.State = v1beta1.ControlStateFailed
				setTimestamp(&controlStatus.UpdateTime)
			}
		case v1beta1.ControlNameSavepoint:
			if savepointStatus != nil {
				switch savepointStatus.State {
				case v1beta1.SavepointStateSucceeded:
					controlStatus.State = v1beta1.ControlStateSucceeded
					setTimestamp(&controlStatus.UpdateTime)
				case v1beta1.SavepointStateFailed, v1beta1.SavepointStateTriggerFailed:
					controlStatus.State = v1beta1.ControlStateFailed
					setTimestamp(&controlStatus.UpdateTime)
				}
			}
		}
		// aborted by max retry reach
		var retries = controlStatus.Details[ControlRetries]
		if retries == ControlMaxRetries {
			controlStatus.Message = fmt.Sprintf("Aborted control %v. The maximum number of retries has been reached.", controlStatus.Name)
			controlStatus.State = v1beta1.ControlStateFailed
			setTimestamp(&controlStatus.UpdateTime)
		}
	} else if userControl != "" {
		// Handle new user control.
		updater.log.Info("New user control requested: " + userControl)
		if userControl != v1beta1.ControlNameJobCancel && userControl != v1beta1.ControlNameSavepoint {
			if userControl != "" {
				updater.log.Info(fmt.Sprintf(v1beta1.InvalidControlAnnMsg, v1beta1.ControlAnnotation, userControl))
			}
		} else if recorded.Control != nil && recorded.Control.State == v1beta1.ControlStateProgressing {
			updater.log.Info(fmt.Sprintf(v1beta1.ControlChangeWarnMsg, v1beta1.ControlAnnotation), "current control", recorded.Control.Name, "new control", userControl)
		} else {
			switch userControl {
			case v1beta1.ControlNameSavepoint:
				if observed.cluster.Spec.Job.SavepointsDir == nil || *observed.cluster.Spec.Job.SavepointsDir == "" {
					updater.log.Info(fmt.Sprintf(v1beta1.InvalidSavepointDirMsg, v1beta1.ControlAnnotation))
					break
				} else if isJobStopped(observed.cluster.Status.Components.Job) {
					updater.log.Info(fmt.Sprintf(v1beta1.InvalidJobStateForSavepointMsg, v1beta1.ControlAnnotation))
					break
				}
				// Clear status for new savepoint
				status.Savepoint = getRequestedSavepointStatus(v1beta1.SavepointTriggerReasonUserRequested)
				controlStatus = getNewUserControlStatus(userControl)
			case v1beta1.ControlNameJobCancel:
				if isJobTerminated(observed.cluster.Spec.Job.RestartPolicy, recorded.Components.Job) {
					updater.log.Info(fmt.Sprintf(v1beta1.InvalidJobStateForJobCancelMsg, v1beta1.ControlAnnotation))
					break
				}
				// Savepoint for job-cancel
				var observedSavepoint = observed.cluster.Status.Savepoint
				if observedSavepoint == nil ||
					(observedSavepoint.State != v1beta1.SavepointStateInProgress && observedSavepoint.State != v1beta1.SavepointStateNotTriggered) {
					updater.log.Info("There is no savepoint in progress. Trigger savepoint in reconciler.")
					status.Savepoint = getRequestedSavepointStatus(v1beta1.SavepointTriggerReasonJobCancel)
				} else {
					updater.log.Info("There is a savepoint in progress. Skip new savepoint.")
				}
				controlStatus = getNewUserControlStatus(userControl)
			}
		}
	}
	// Maintain control status if there is no change.
	if recorded.Control != nil && controlStatus == nil {
		controlStatus = recorded.Control.DeepCopy()
	}
	status.Control = controlStatus

	// Handle update.
	var savepointForJobUpdate *v1beta1.SavepointStatus
	switch updater.deriveUpdateState(observed, status) {
	case UpdateStatePreparing:
		// Even if savepoint has been created for update already, we check the age of savepoint continually.
		// If created savepoint is old and savepoint can be triggered, we should take savepoint again.
		// (e.g., for the case update is not progressed by accidents like network partition)
		if !isSavepointUpToDate(observed.observeTime, *jobStatus) &&
			canTakeSavepoint(*observed.cluster) &&
			(recorded.Savepoint == nil || recorded.Savepoint.State != v1beta1.SavepointStateNotTriggered) {
			// If failed to take savepoint, retry after SavepointRequestRetryIntervalSec.
			if recorded.Savepoint != nil &&
				!hasTimeElapsed(recorded.Savepoint.RequestTime, time.Now(), SavepointRequestRetryIntervalSec) {
				updater.log.Info(fmt.Sprintf("Will retry to trigger savepoint for update, in %v seconds because previous request was failed", SavepointRequestRetryIntervalSec))
			} else {
				status.Savepoint = getRequestedSavepointStatus(v1beta1.SavepointTriggerReasonUpdate)
				updater.log.Info("Savepoint will be triggered for update")
			}
		} else if recorded.Savepoint != nil && recorded.Savepoint.State == v1beta1.SavepointStateInProgress {
			updater.log.Info("Savepoint for update is in progress")
		} else {
			updater.log.Info("Stopping job for update")
		}
	case UpdateStateInProgress:
		updater.log.Info("Updating cluster")
	case UpdateStateFinished:
		status.CurrentRevision = observed.cluster.Status.NextRevision
		updater.log.Info("Finished update")
	}
	if savepointForJobUpdate != nil {
		status.Savepoint = savepointForJobUpdate
	}

	// Update revision status
	status.NextRevision = getRevisionWithNameNumber(observed.revisionStatus.nextRevision)
	if status.CurrentRevision == "" {
		if recorded.CurrentRevision == "" {
			status.CurrentRevision = getRevisionWithNameNumber(observed.revisionStatus.currentRevision)
		} else {
			status.CurrentRevision = recorded.CurrentRevision
		}
	}

	if observed.revisionStatus.collisionCount != 0 {
		status.CollisionCount = new(int32)
		*status.CollisionCount = observed.revisionStatus.collisionCount
	}

	return status
}

func (updater *FlinkClusterHandlerV2) deriveConfigMapStatus(observed ObservedClusterState) v1beta1.FlinkClusterComponentState {
	observedConfigMap := observed.configMap
	if observed.configMap == nil {
		if observed.cluster.Status.Components.ConfigMap.Name != "" {
			return v1beta1.FlinkClusterComponentState{
				Name:  observed.cluster.Status.Components.ConfigMap.Name,
				State: v1beta1.ComponentStateDeleted,
			}
		} else {
			return v1beta1.FlinkClusterComponentState{
				State: v1beta1.ComponentStateUpdating,
			}
		}
	} else {
		if observed.isComponentUpdated(observedConfigMap) {
			return v1beta1.FlinkClusterComponentState{
				Name:  observed.configMap.Name,
				State: v1beta1.ComponentStateReady,
			}
		} else {
			return v1beta1.FlinkClusterComponentState{
				Name:  observed.configMap.Name,
				State: v1beta1.ComponentStateUpdating,
			}
		}
	}
}

func (updater *FlinkClusterHandlerV2) deriveStatefulSetStatus(observed ObservedClusterState, observedStatefulSet *appsv1.StatefulSet, previousStatus v1beta1.FlinkClusterComponentState) v1beta1.FlinkClusterComponentState {
	if observedStatefulSet == nil {
		if previousStatus.Name != "" {
			return v1beta1.FlinkClusterComponentState{
				Name:  previousStatus.Name,
				State: v1beta1.ComponentStateDeleted,
			}
		} else {
			return v1beta1.FlinkClusterComponentState{
				State: v1beta1.ComponentStateUpdating,
			}
		}
	} else {
		if observed.isComponentUpdated(observedStatefulSet) {
			return v1beta1.FlinkClusterComponentState{
				Name:  observedStatefulSet.Name,
				State: v1beta1.ComponentStateReady,
			}
		} else {
			return v1beta1.FlinkClusterComponentState{
				Name:  observedStatefulSet.Name,
				State: v1beta1.ComponentStateUpdating,
			}
		}
	}
}

func (updater *FlinkClusterHandlerV2) deriveServiceStatus(observed ObservedClusterState) v1beta1.JobManagerServiceStatus {
	var resultStatus v1beta1.JobManagerServiceStatus

	observedService := observed.jmService
	currentStatus := observed.cluster.Status.Components.JobManagerService

	if observedService == nil {
		if currentStatus.Name != "" {
			return v1beta1.JobManagerServiceStatus{
				Name:  currentStatus.Name,
				State: v1beta1.ComponentStateDeleted,
			}
		} else {
			currentStatus.DeepCopyInto(&resultStatus)
			resultStatus.State = v1beta1.ComponentStateUpdating
			return resultStatus
		}
	} else {
		if !isComponentUpdated(observedService, *observed.cluster) {
			currentStatus.DeepCopyInto(&resultStatus)
			resultStatus.State = v1beta1.ComponentStateUpdating
			return resultStatus
		} else {
			var nodePort int32
			var loadBalancerIngress []corev1.LoadBalancerIngress
			state := v1beta1.ComponentStateNotReady

			switch observedService.Spec.Type {
			case corev1.ServiceTypeClusterIP:
				if observedService.Spec.ClusterIP != "" {
					state = v1beta1.ComponentStateReady
				}
			case corev1.ServiceTypeLoadBalancer:
				if len(observedService.Status.LoadBalancer.Ingress) > 0 {
					state = v1beta1.ComponentStateReady
					loadBalancerIngress = observedService.Status.LoadBalancer.Ingress
				}
			case corev1.ServiceTypeNodePort:
				if len(observedService.Spec.Ports) > 0 {
					state = v1beta1.ComponentStateReady
					for _, port := range observedService.Spec.Ports {
						if port.Name == "ui" {
							nodePort = port.NodePort
						}
					}
				}
			}

			return v1beta1.JobManagerServiceStatus{
				Name:                observedService.Name,
				State:               state,
				NodePort:            nodePort,
				LoadBalancerIngress: loadBalancerIngress,
			}
		}
	}
}

func (updater *FlinkClusterHandlerV2) deriveJobStatus(observed ObservedClusterState) *v1beta1.JobStatus {
	var observedJobStatus = observed.cluster.Status.Components.Job

	if observedJobStatus == nil {
		return nil
	}

	newJobStatus := observedJobStatus.DeepCopy()

	// Flink Job ID
	if observed.flinkJobStatus.flinkJob != nil {
		newJobStatus.ID = observed.flinkJobStatus.flinkJob.Id
		newJobStatus.Name = observed.flinkJobStatus.flinkJob.Name
	}

	if observed.job != nil {
		newJobStatus.SubmitterName = observed.job.Name
	}

	// Update State
	newJobStatus.State, newJobStatus.FailureReasons = updater.deriveJobStatusState(observed)

	updater.log.Info("Job state", "jobstate", newJobStatus.State)

	// Update Job Status info
	if isJobStopped(newJobStatus) && newJobStatus.CompletionTime.IsZero() {
		now := metav1.Now()
		newJobStatus.CompletionTime = &now
	}

	if newJobStatus.State == v1beta1.JobStateFailed || newJobStatus.State == v1beta1.JobStateLost {
		if len(newJobStatus.FailureReasons) == 0 {
			newJobStatus.FailureReasons = []string{}
			exceptions := observed.flinkJobStatus.flinkJobExceptions
			if exceptions != nil && len(exceptions.Exceptions) > 0 {
				for _, e := range exceptions.Exceptions {
					newJobStatus.FailureReasons = append(newJobStatus.FailureReasons, e.Exception)
				}
			} else {
				newJobStatus.FailureReasons = append(newJobStatus.FailureReasons, "Failed to get job exceptions")
			}
			if observed.flinkJobSubmitLog != nil {
				newJobStatus.FailureReasons = append(newJobStatus.FailureReasons, observed.flinkJobSubmitLog.Message)
			}
		}
	}

	// Savepoint
	if observed.savepoint != nil && observed.savepoint.IsSuccessful() {
		newJobStatus.SavepointGeneration++
		newJobStatus.LastSavepointTriggerID = observed.savepoint.TriggerID
		newJobStatus.SavepointLocation = observed.savepoint.Location
		setTimestamp(&newJobStatus.LastSavepointTime)
	}

	return newJobStatus
}

func (updater *FlinkClusterHandlerV2) deriveJobStatusState(observed ObservedClusterState) (string, []string) {
	var errorMessage []string

	if observed.isNewRevision() {
		if observed.flinkJobStatus.flinkJob == nil {
			return v1beta1.JobStateUpdating, errorMessage
		} else {
			switch observed.flinkJobStatus.flinkJob.State {
			case "INITIALIZING", "CREATED", "RUNNING", "FAILING", "CANCELLING", "RESTARTING", "RECONCILING":
				return v1beta1.JobStateRunning, errorMessage
			case "FINISHED", "CANCELED", "FAILED", "SUSPENDED":
				return v1beta1.JobStateUpdating, errorMessage
			default:
				errorMessage = append(errorMessage, "Unknown flink job state")
				return v1beta1.JobStateUnknown, errorMessage
			}
		}
	} else {
		if observed.flinkJobStatus.flinkJob == nil {
			if observed.job == nil {
				if shouldRestartJobV2(observed.cluster.Spec.Job.RestartPolicy, observed.cluster.Status.Components.Job) {
					return v1beta1.JobStateUpdating, errorMessage
				} else {
					return observed.cluster.Status.Components.Job.State, observed.cluster.Status.Components.Job.FailureReasons
				}
			} else {
				if observed.job.Status.Succeeded == 0 && observed.job.Status.Failed == 0 {
					return v1beta1.JobStatePending, errorMessage
				}
				if observed.cluster.Status.Components.Job != nil && observed.cluster.Status.Components.Job.State == v1beta1.JobStateUpdating {
					return v1beta1.JobStateUpdating, errorMessage
				}
				if observed.job.Status.Failed != 0 {
					errorMessage = append(errorMessage, "Job pod failed")
					return v1beta1.JobStateFailed, errorMessage
				}
				if observed.job.Status.Succeeded != 0 {
					if observed.cluster.Status.Components.Job != nil && observed.cluster.Status.Components.Job.State == v1beta1.JobStateRunning {
						errorMessage = append(errorMessage, "Missing flink job with successful submission")
						return v1beta1.JobStateLost, errorMessage
					}
					if shouldRestartJobV2(observed.cluster.Spec.Job.RestartPolicy, observed.cluster.Status.Components.Job) {
						return v1beta1.JobStateUpdating, errorMessage
					}
					// Return job state
					return observed.cluster.Status.Components.Job.State, errorMessage
				}
				errorMessage = append(errorMessage, "Unknown job pod status")
				return v1beta1.JobStateUnknown, errorMessage
			}
		} else {
			switch observed.flinkJobStatus.flinkJob.State {
			case "INITIALIZING", "CREATED", "RUNNING", "FAILING", "CANCELLING", "RESTARTING", "RECONCILING":
				return v1beta1.JobStateRunning, errorMessage
			case "FINISHED", "CANCELED", "FAILED", "SUSPENDED":
				if observed.cluster.Status.Components.Job != nil && observed.cluster.Status.Components.Job.State == v1beta1.JobStateUpdating {
					return v1beta1.JobStateUpdating, errorMessage
				} else if observed.job != nil && observed.job.Status.Succeeded == 0 && observed.job.Status.Failed == 0 {
					return v1beta1.JobStatePending, errorMessage
				} else if observed.job != nil && observed.job.Status.Failed != 0 {
					errorMessage = append(errorMessage, "Job pod failed")
					return v1beta1.JobStateFailed, errorMessage
				} else {
					switch observed.flinkJobStatus.flinkJob.State {
					case "FINISHED":
						return v1beta1.JobStateSucceeded, errorMessage
					case "CANCELED":
						return v1beta1.JobStateCancelled, errorMessage
					case "FAILED":
						return v1beta1.JobStateFailed, errorMessage
					case "SUSPENDED":
						return v1beta1.JobStateSuspended, errorMessage
					default:
						// Impossible to reach, but required for compiler
						errorMessage = append(errorMessage, "Unknown flink job state")
						return v1beta1.JobStateUnknown, errorMessage
					}
				}
			default:
				errorMessage = append(errorMessage, "Unknown flink job state")
				return v1beta1.JobStateUnknown, errorMessage
			}
		}
	}
}

func (updater *FlinkClusterHandlerV2) deriveClusterState(observed ObservedClusterState, updatedStatus v1beta1.FlinkClusterStatus) string {

	currentStatus := observed.cluster.Status

	runningComponents, totalComponents := updater.deriveRunningComponents(updatedStatus)

	switch currentStatus.State {
	case "", v1beta1.ClusterStateCreating:
		if observed.isNewRevision() {
			return v1beta1.ClusterStateUpdating
		}
		if runningComponents < totalComponents {
			return v1beta1.ClusterStateCreating
		}
		return v1beta1.ClusterStateRunning

	case v1beta1.ClusterStateUpdating:
		if observed.isNewRevision() {
			return v1beta1.ClusterStateUpdating
		}
		if runningComponents < totalComponents {
			return v1beta1.ClusterStateReconciling
		}
		return v1beta1.ClusterStateRunning

	case v1beta1.ClusterStateRunning, v1beta1.ClusterStateReconciling:
		if observed.isNewRevision() {
			return v1beta1.ClusterStateUpdating
		}
		if updatedStatus.Components.Job != nil {
			var policy = observed.cluster.Spec.Job.CleanupPolicy
			switch updatedStatus.Components.Job.State {
			case v1beta1.JobStateSucceeded, v1beta1.JobStateFailed, v1beta1.JobStateCancelled:
				if policy.AfterJobSucceeds != v1beta1.CleanupActionKeepCluster {
					return v1beta1.ClusterStateStopping
				}
			case v1beta1.JobStatePending, v1beta1.JobStateRunning, v1beta1.JobStateUpdating, v1beta1.JobStateSuspended, v1beta1.JobStateUnknown, v1beta1.JobStateLost:
				return v1beta1.ClusterStateRunning
			default:
				return v1beta1.ClusterStateRunning
			}
		}
		if runningComponents < totalComponents {
			return v1beta1.ClusterStateReconciling
		}
		return v1beta1.ClusterStateRunning

	case v1beta1.ClusterStateStopping, v1beta1.ClusterStatePartiallyStopped:
		if observed.isNewRevision() {
			return v1beta1.ClusterStateUpdating
		}
		if runningComponents == 0 {
			return v1beta1.ClusterStateStopped
		}
		if runningComponents < totalComponents {
			return v1beta1.ClusterStatePartiallyStopped
		}
		return v1beta1.ClusterStateStopping

	case v1beta1.ClusterStateStopped:
		if observed.isNewRevision() {
			return v1beta1.ClusterStateUpdating
		}
		return v1beta1.ClusterStateStopped

	default:
		panic(fmt.Sprintf("Unknown cluster state: %v", currentStatus.State))
	}
}

func (updater *FlinkClusterHandlerV2) deriveUpdateState(observed ObservedClusterState, updatedStatus v1beta1.FlinkClusterStatus) UpdateState {
	currentStatus := observed.cluster.Status

	if !observed.isNewRevision() {
		// irrelevant
		return ""
	}

	if observed.cluster.Spec.Job != nil && currentStatus.Components.Job != nil && currentStatus.Components.Job.State == v1beta1.JobStateRunning {
		return UpdateStatePreparing
	}

	runningComponents, totalComponents := updater.deriveRunningComponents(updatedStatus)
	if runningComponents != totalComponents {
		return UpdateStateInProgress
	}

	return UpdateStateFinished
}

func (updater *FlinkClusterHandlerV2) deriveRunningComponents(updatedStatus v1beta1.FlinkClusterStatus) (int, int) {
	var runningComponents = 0
	var totalComponents = 3
	if updatedStatus.Components.JobManagerStatefulSet.State == v1beta1.ComponentStateReady {
		runningComponents++
	}
	if updatedStatus.Components.JobManagerService.State == v1beta1.ComponentStateReady {
		runningComponents++
	}
	if updatedStatus.Components.TaskManagerStatefulSet.State == v1beta1.ComponentStateReady {
		runningComponents++
	}

	return runningComponents, totalComponents
}

func (updater *FlinkClusterHandlerV2) updateClusterStatus(ctx context.Context, observed ObservedClusterState, status v1beta1.FlinkClusterStatus) error {
	var cluster = v1beta1.FlinkCluster{}
	observed.cluster.DeepCopyInto(&cluster)
	cluster.Status = status

	updater.log.Info("(FlinkClusterHandlerV2) Updating cluster status")

	err := updater.k8sClient.Status().Update(ctx, &cluster)

	// Clear control annotation after status update is complete.
	updater.clearControlAnnotation(ctx, observed, status.Control)
	return err
}

// Clear finished or improper user control in annotations
func (updater *FlinkClusterHandlerV2) clearControlAnnotation(ctx context.Context, observed ObservedClusterState, newControlStatus *v1beta1.FlinkClusterControlStatus) error {
	var userControl = observed.cluster.Annotations[v1beta1.ControlAnnotation]
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
		return updater.k8sClient.Patch(ctx, observed.cluster, client.RawPatch(types.MergePatchType, patchBytes))
	}

	return nil
}
