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
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"

	"github.com/go-logr/logr"
	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/batchscheduler"
	schedulerTypes "github.com/spotify/flink-on-k8s-operator/internal/batchscheduler/types"
	"github.com/spotify/flink-on-k8s-operator/internal/flink"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
	"github.com/spotify/flink-on-k8s-operator/internal/util"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterReconciler takes actions to drive the observed state towards the
// desired state.
type ClusterReconciler struct {
	k8sClient   client.Client
	flinkClient *flink.Client
	observed    ObservedClusterState
	desired     model.DesiredClusterState
	recorder    record.EventRecorder
}

const JobCheckInterval = 10 * time.Second

var requeueResult = ctrl.Result{RequeueAfter: JobCheckInterval, Requeue: true}

// Compares the desired state and the observed state, if there is a difference,
// takes actions to drive the observed state towards the desired state.
func (reconciler *ClusterReconciler) reconcile(ctx context.Context) (ctrl.Result, error) {
	var err error
	log := logr.FromContextOrDiscard(ctx)

	// Child resources of the cluster CR will be automatically reclaimed by K8S.
	if reconciler.observed.cluster == nil {
		log.Info("The cluster has been deleted, no action to take")
		return ctrl.Result{}, nil
	}

	if shouldUpdateCluster(&reconciler.observed) {
		log.Info("The cluster update is in progress")
	}

	err = reconciler.reconcileBatchScheduler()
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileConfigMap(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileHAConfigMap(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcilePodDisruptionBudget(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileJobManagerStatefulSet(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileJobManagerService(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileJobManagerIngress(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileTaskManagerStatefulSet(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileTaskManagerDeployment(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileHorizontalPodAutoscaler(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcileTaskManagerService(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconciler.reconcilePersistentVolumeClaims(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	result, err := reconciler.reconcileJob(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	return result, nil
}

func (reconciler *ClusterReconciler) reconcileBatchScheduler() error {
	cluster := reconciler.observed.cluster
	schedulerSpec := cluster.Spec.BatchScheduler
	if schedulerSpec == nil || schedulerSpec.Name == "" {
		return nil
	}

	scheduler, err := batchscheduler.GetScheduler(schedulerSpec.Name)
	if err != nil {
		return err
	}

	options := schedulerTypes.SchedulerOptions{
		ClusterName:       cluster.Name,
		ClusterNamespace:  cluster.Namespace,
		Queue:             schedulerSpec.Queue,
		PriorityClassName: schedulerSpec.PriorityClassName,
		OwnerReferences:   []metav1.OwnerReference{ToOwnerReference(cluster)},
	}
	err = scheduler.Schedule(options, &reconciler.desired)
	if err != nil {
		return err
	}

	return nil
}

func (reconciler *ClusterReconciler) reconcileJobManagerStatefulSet(ctx context.Context) error {
	return reconciler.reconcileComponent(
		ctx,
		"JobManager",
		reconciler.desired.JmStatefulSet,
		reconciler.observed.jmStatefulSet)
}

func (reconciler *ClusterReconciler) reconcileTaskManagerStatefulSet(ctx context.Context) error {
	return reconciler.reconcileComponent(
		ctx,
		"TaskManager",
		reconciler.desired.TmStatefulSet,
		reconciler.observed.tmStatefulSet)
}

func (reconciler *ClusterReconciler) reconcileTaskManagerDeployment(ctx context.Context) error {
	return reconciler.reconcileComponent(
		ctx,
		"TaskManager",
		reconciler.desired.TmDeployment,
		reconciler.observed.tmDeployment)
}

func (reconciler *ClusterReconciler) reconcileComponent(
	ctx context.Context,
	component string,
	desiredObj client.Object,
	observedObj client.Object) error {
	log := logr.FromContextOrDiscard(ctx).WithValues("component", component)
	desiredObjIsNil := reflect.ValueOf(desiredObj).IsNil()
	observedObjIsNil := reflect.ValueOf(observedObj).IsNil()

	if !desiredObjIsNil && observedObjIsNil {
		return reconciler.createComponent(ctx, desiredObj, component)
	}

	if !desiredObjIsNil && !observedObjIsNil {
		var cluster = reconciler.observed.cluster
		if shouldUpdateCluster(&reconciler.observed) && !isComponentUpdated(observedObj, cluster) {
			var err error
			if shouldRecreateOnUpdate(&reconciler.observed) {
				err = reconciler.deleteComponent(ctx, desiredObj, component)
			} else {
				err = reconciler.updateComponent(ctx, desiredObj, component)
			}
			if err != nil {
				return err
			}
			return nil
		}
		log.Info("Component already exists, no action", "component", component)
		return nil
	}

	if desiredObjIsNil && !observedObjIsNil {
		return reconciler.deleteComponent(ctx, observedObj, component)
	}

	return nil
}

func (reconciler *ClusterReconciler) reconcileHorizontalPodAutoscaler(ctx context.Context) error {
	return reconciler.reconcileComponent(
		ctx,
		"HorizontalPodAutoscaler",
		reconciler.desired.HorizontalPodAutoscaler,
		reconciler.observed.horizontalPodAutoscaler)
}

func (reconciler *ClusterReconciler) reconcileTaskManagerService(ctx context.Context) error {
	var desiredTmService = reconciler.desired.TmService
	var observedTmService = reconciler.observed.tmService

	if desiredTmService != nil && observedTmService != nil {
		// v1.Service API does not handle update correctly when below values are empty.
		desiredTmService.SetResourceVersion(observedTmService.GetResourceVersion())
		desiredTmService.Spec.ClusterIP = observedTmService.Spec.ClusterIP
	}
	return reconciler.reconcileComponent(ctx, "TaskManagerService", desiredTmService, observedTmService)
}

func (reconciler *ClusterReconciler) createComponent(
	ctx context.Context, obj client.Object, component string) error {
	log := logr.FromContextOrDiscard(ctx).
		WithValues("component", component).
		WithValues("object", obj)

	if err := reconciler.k8sClient.Create(ctx, obj); err != nil {
		log.Error(err, "Failed to create")
		return err
	}

	log.Info("Created")
	return nil
}

func (reconciler *ClusterReconciler) updateComponent(ctx context.Context, desired client.Object, component string) error {
	log := logr.FromContextOrDiscard(ctx).
		WithValues("component", component).
		WithValues("object", desired)
	var k8sClient = reconciler.k8sClient

	if err := k8sClient.Update(ctx, desired); err != nil {
		log.Error(err, "Failed to update component for update")
		return err
	}

	log.Info("Updated")
	return nil
}

func (reconciler *ClusterReconciler) deleteComponent(
	ctx context.Context, obj client.Object, component string) error {
	log := logr.FromContextOrDiscard(ctx).
		WithValues("component", component).
		WithValues("object", obj)
	var k8sClient = reconciler.k8sClient

	var err = k8sClient.Delete(ctx, obj)
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete", component, obj)
	}

	log.Info("Deleted")
	return nil
}

func (reconciler *ClusterReconciler) reconcileJobManagerService(ctx context.Context) error {
	var desiredJmService = reconciler.desired.JmService
	var observedJmService = reconciler.observed.jmService

	if desiredJmService != nil && observedJmService != nil {
		// v1.Service API does not handle update correctly when below values are empty.
		desiredJmService.SetResourceVersion(observedJmService.GetResourceVersion())
		desiredJmService.Spec.ClusterIP = observedJmService.Spec.ClusterIP
	}

	return reconciler.reconcileComponent(ctx, "JobManagerService", desiredJmService, observedJmService)
}

func (reconciler *ClusterReconciler) reconcileJobManagerIngress(ctx context.Context) error {
	var desiredJmIngress = reconciler.desired.JmIngress
	var observedJmIngress = reconciler.observed.jmIngress

	return reconciler.reconcileComponent(ctx, "JobManagerIngress", desiredJmIngress, observedJmIngress)
}

func (reconciler *ClusterReconciler) reconcileConfigMap(ctx context.Context) error {
	var desiredConfigMap = reconciler.desired.ConfigMap
	var observedConfigMap = reconciler.observed.configMap

	return reconciler.reconcileComponent(ctx, "ConfigMap", desiredConfigMap, observedConfigMap)
}

// Set the owner reference of the cluster to the HA ConfigMap (if it doesn't already have one)
func (reconciler *ClusterReconciler) reconcileHAConfigMap(ctx context.Context) error {
	var observedHAConfigMap = reconciler.observed.haConfigMap
	if observedHAConfigMap == nil {
		return nil
	}
	if observedHAConfigMap.OwnerReferences == nil || len(observedHAConfigMap.OwnerReferences) == 0 {
		observedHAConfigMap.OwnerReferences = []metav1.OwnerReference{ToOwnerReference(reconciler.observed.cluster)}
		err := reconciler.updateComponent(ctx, observedHAConfigMap, "HA ConfigMap")
		if err != nil {
			return err
		}
	}
	return nil
}

func (reconciler *ClusterReconciler) reconcilePodDisruptionBudget(ctx context.Context) error {
	desiredPodDisruptionBudget := reconciler.desired.PodDisruptionBudget
	observedPodDisruptionBudget := reconciler.observed.podDisruptionBudget

	if desiredPodDisruptionBudget != nil && observedPodDisruptionBudget != nil {
		// When updating a PodDisruptionBudget, the resource version must be set!
		// Setting the resource version to the observed resource version ensures that the update is not rejected
		desiredPodDisruptionBudget.SetResourceVersion(observedPodDisruptionBudget.ResourceVersion)
	}

	return reconciler.reconcileComponent(
		ctx,
		"PodDisruptionBudget",
		desiredPodDisruptionBudget,
		observedPodDisruptionBudget)

}

func (reconciler *ClusterReconciler) reconcilePersistentVolumeClaims(ctx context.Context) error {
	observed := reconciler.observed
	pvcs := observed.persistentVolumeClaims
	jm := observed.jmStatefulSet
	tm := observed.tmStatefulSet

	for _, pvc := range pvcs.Items {
		if c, ok := pvc.Labels["component"]; ok && c == "jobmanager" && jm != nil {
			reconciler.reconcilePersistentVolumeClaim(ctx, &pvc, jm)
		}
		if c, ok := pvc.Labels["component"]; ok && c == "taskmanager" && tm != nil {
			reconciler.reconcilePersistentVolumeClaim(ctx, &pvc, tm)
		}
	}

	return nil
}

func (reconciler *ClusterReconciler) reconcilePersistentVolumeClaim(ctx context.Context, pvc *corev1.PersistentVolumeClaim, sset *appsv1.StatefulSet) error {
	log := logr.FromContextOrDiscard(ctx)
	k8sClient := reconciler.k8sClient

	for _, ownerRef := range pvc.GetOwnerReferences() {
		if ownerRef.Kind == sset.Kind {
			return nil
		}
	}

	patch := fmt.Sprintf(
		`{"metadata":{"ownerReferences":[{"apiVersion":"%s","kind":"%s","name":"%s","uid":"%s","controller":true,"blockOwnerDeletion":false}],"uid":"%s"}}`,
		sset.APIVersion,
		sset.Kind,
		sset.GetName(),
		sset.GetUID(),
		pvc.GetUID(),
	)
	err := k8sClient.Patch(ctx, pvc, client.RawPatch(types.MergePatchType, []byte(patch)))
	if err != nil {
		log.Error(err, "Failed to update PersistentVolumeClaim")
	} else {
		log.Info("PersistentVolumeClaim patched")
	}

	return err
}

func (reconciler *ClusterReconciler) reconcileJob(ctx context.Context) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx)
	var desiredJob = reconciler.desired.Job
	var observed = reconciler.observed
	var recorded = observed.cluster.Status
	var jobSpec = observed.cluster.Spec.Job
	var job = recorded.Components.Job
	var err error
	var jobID = reconciler.getFlinkJobID()

	// Update status changed via job reconciliation.
	var newSavepointStatus *v1beta1.SavepointStatus
	var newControlStatus *v1beta1.FlinkClusterControlStatus
	defer reconciler.updateStatus(ctx, &newSavepointStatus, &newControlStatus)

	observedSubmitter := observed.flinkJobSubmitter.job

	if desiredJob != nil && job.IsTerminated(jobSpec) {
		return ctrl.Result{}, nil
	}

	if wasJobCancelRequested(observed.cluster.Status.Control) {
		log.Info("Force tearing down the job")
		userControl := getNewControlRequest(observed.cluster)
		if userControl == v1beta1.ControlNameJobCancel {
			newControlStatus = getControlStatus(userControl, v1beta1.ControlStateInProgress)
		}
		// cancel all running jobs
		if job.IsActive() {
			if err := reconciler.cancelRunningJobs(ctx, true /* takeSavepoint */); err != nil && !errors.IsResourceExpired(err) {
				return requeueResult, err
			}
		}
		// kill job submitter pod
		if observedSubmitter != nil {
			if err := reconciler.deleteJob(ctx, observedSubmitter); err != nil {
				return requeueResult, err
			}
		}
	}

	// Create new Flink job submitter when starting new job, updating job or restarting job in failure.
	if desiredJob != nil && !job.IsActive() {
		log.Info("Deploying Flink job")

		// TODO: Record event or introduce Condition in CRD status to notify update state pended.
		// https://github.com/kubernetes/apimachinery/blob/57f2a0733447cfd41294477d833cce6580faaca3/pkg/apis/meta/v1/types.go#L1376
		var unexpectedJobs = observed.flinkJob.unexpected
		if len(unexpectedJobs) > 0 {
			// This is an exceptional situation.
			// There should be no jobs because all jobs are terminated in the previous iterations.
			// In this case user should identify the problem so that the job is not executed multiple times unintentionally
			// cause of Flink error, Flink operator error or other unknown error.
			// If user want to proceed, unexpected jobs should be terminated.
			log.Error(errors.NewInternalError(fmt.Errorf("unexpected jobs found")), "Failed to create job submitter", "unexpected jobs", unexpectedJobs)
			return ctrl.Result{}, nil
		}

		// Create Flink job submitter
		log.Info("Updating job status to proceed creating new job submitter")
		// Job status must be updated before creating a job submitter to ensure the observed job is the job submitted by the operator.
		err = reconciler.updateJobDeployStatus(ctx)
		if err != nil {
			log.Info("Failed to update the job status for job submission")
			return requeueResult, err
		}

		cr := getCurrentRevisionName(&observed.cluster.Status.Revision)
		if observedSubmitter != nil {
			if observedSubmitter.Labels[RevisionNameLabel] == cr {
				log.Info("Found old job submitter")
				err = reconciler.deleteJob(ctx, observedSubmitter)
				if err != nil {
					return requeueResult, err
				}
			} else if observedSubmitter.Status.Failed >= 1 {
				log.Info("Found failed job submitter")
				err = reconciler.deleteJob(ctx, observedSubmitter)
				if err != nil {
					return requeueResult, err
				}
			} else {
				log.Info("Found job submitter, wait for it to be active or failed")
				return requeueResult, nil
			}
		} else {
			err = reconciler.createJob(ctx, desiredJob)
		}

		return requeueResult, err
	}

	if desiredJob != nil && job.IsActive() {
		if job.State == v1beta1.JobStateDeploying {
			log.Info("Job submitter is deployed, wait until completed")
			return requeueResult, nil
		}

		// Suspend or stop job to proceed update.
		if recorded.Revision.IsUpdateTriggered() && isJobUpdate(observed.revisions, observed.cluster) {
			log.Info("Preparing job update")
			var takeSavepoint = jobSpec.TakeSavepointOnUpdate == nil || *jobSpec.TakeSavepointOnUpdate
			var shouldSuspend = takeSavepoint && util.IsBlank(jobSpec.FromSavepoint)
			if shouldSuspend {
				newSavepointStatus, err = reconciler.trySuspendJob(ctx)
			} else if shouldUpdateJob(&observed) {
				err = reconciler.cancelJob(ctx)
			}
			return requeueResult, err
		}

		// Trigger savepoint if required.
		if len(jobID) > 0 {
			var savepointReason = reconciler.shouldTakeSavepoint()
			if savepointReason != "" {
				newSavepointStatus, err = reconciler.triggerSavepoint(ctx, jobID, savepointReason, false)
			}
			// Get new control status when the savepoint reason matches the requested control.
			var userControl = getNewControlRequest(observed.cluster)
			if userControl == v1beta1.ControlNameSavepoint && savepointReason == v1beta1.SavepointReasonUserRequested {
				newControlStatus = getControlStatus(userControl, v1beta1.ControlStateInProgress)
			}
			return requeueResult, err
		}

		log.Info("Job is not finished yet, no action", "jobID", jobID)
		return requeueResult, nil
	}

	// Job finished. Stop Flink job and kill job-submitter.
	if desiredJob == nil && (!job.IsStopped() || observedSubmitter != nil) {
		if job.IsActive() {
			userControl := getNewControlRequest(observed.cluster)
			if userControl == v1beta1.ControlNameJobCancel {
				newControlStatus = getControlStatus(userControl, v1beta1.ControlStateInProgress)
			}

			log.Info("Stopping job", "jobID", jobID)
			if err := reconciler.cancelRunningJobs(ctx, true /* takeSavepoint */); err != nil {
				return requeueResult, err
			}
		} else if job.IsStopped() && observedSubmitter != nil {
			if observed.cluster.Status.Components.Job.SubmitterExitCode == -1 {
				log.Info("Job submitter has not finished yet")
				return requeueResult, err
			}
			if err := reconciler.deleteJob(ctx, observedSubmitter); err != nil {
				return requeueResult, err
			}
		}

		// to make sure the job is stopped
		return requeueResult, nil
	}

	if job.IsStopped() {
		log.Info("Job has finished, no action")
	}

	return ctrl.Result{}, nil
}

func (reconciler *ClusterReconciler) createJob(ctx context.Context, job *batchv1.Job) error {
	log := logr.FromContextOrDiscard(ctx)
	var k8sClient = reconciler.k8sClient

	log.Info("Creating job submitter", "resource", *job)
	var err = k8sClient.Create(ctx, job)
	if err != nil {
		log.Info("Failed to created job submitter", "error", err)
	} else {
		log.Info("Job submitter created")
	}
	return err
}

func (reconciler *ClusterReconciler) deleteJob(ctx context.Context, job *batchv1.Job) error {
	log := logr.FromContextOrDiscard(ctx)
	var k8sClient = reconciler.k8sClient

	var deletePolicy = metav1.DeletePropagationBackground
	var deleteOption = client.DeleteOptions{PropagationPolicy: &deletePolicy}

	log.Info("Deleting job submitter", "job", job)
	var err = k8sClient.Delete(ctx, job, &deleteOption)
	err = client.IgnoreNotFound(err)
	if err != nil {
		log.Error(err, "Failed to delete job submitter")
	} else {
		log.Info("Job submitter deleted")
	}
	return err
}

func (reconciler *ClusterReconciler) getFlinkJobID() string {
	var jobStatus = reconciler.observed.cluster.Status.Components.Job
	if jobStatus != nil && len(jobStatus.ID) > 0 {
		return jobStatus.ID
	}
	return ""
}

func (reconciler *ClusterReconciler) trySuspendJob(ctx context.Context) (*v1beta1.SavepointStatus, error) {
	log := logr.FromContextOrDiscard(ctx)
	var recorded = reconciler.observed.cluster.Status

	if !canTakeSavepoint(reconciler.observed.cluster) {
		return nil, nil
	}

	var jobID = reconciler.getFlinkJobID()

	log.Info("Checking the conditions for progressing")
	var canSuspend = reconciler.canSuspendJob(ctx, jobID, recorded.Savepoint)
	if canSuspend {
		log.Info("Triggering savepoint for suspending job")
		var newSavepointStatus, err = reconciler.triggerSavepoint(ctx, jobID, v1beta1.SavepointReasonUpdate, true)
		if err != nil {
			log.Info("Failed to trigger savepoint", "jobID", jobID, "triggerID", newSavepointStatus.TriggerID, "error", err)
		} else {
			log.Info("Successfully savepoint triggered", "jobID", jobID, "triggerID", newSavepointStatus.TriggerID)
		}
		return newSavepointStatus, err
	}

	return nil, nil
}

func (reconciler *ClusterReconciler) cancelJob(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)
	var observedFlinkJob = reconciler.observed.flinkJob.status

	log.Info("Stopping Flink job", "", observedFlinkJob)
	var err = reconciler.cancelRunningJobs(ctx, false /* takeSavepoint */)
	if err != nil {
		log.Info("Failed to stop Flink job")
		return err
	}

	// TODO: Not to delete the job submitter immediately, and retain the latest ones for inspection.
	var observedSubmitter = reconciler.observed.flinkJobSubmitter.job
	if observedSubmitter != nil {
		var err = reconciler.deleteJob(ctx, observedSubmitter)
		if err != nil {
			log.Error(
				err, "Failed to delete job submitter", "job", observedSubmitter)
			return err
		}
	}

	return nil
}

func (reconciler *ClusterReconciler) cancelUnexpectedJobs(
	ctx context.Context,
	takeSavepoint bool) error {
	var unexpectedJobs = reconciler.observed.flinkJob.unexpected
	return reconciler.cancelJobs(ctx, takeSavepoint, unexpectedJobs)
}

// Cancel running jobs.
func (reconciler *ClusterReconciler) cancelRunningJobs(
	ctx context.Context,
	takeSavepoint bool) error {
	var runningJobs = reconciler.observed.flinkJob.unexpected
	var flinkJob = reconciler.observed.flinkJob.status
	if flinkJob != nil && flinkJob.Id != "" &&
		getFlinkJobDeploymentState(flinkJob.State) == v1beta1.JobStateRunning {
		runningJobs = append(runningJobs, flinkJob.Id)
	}
	if len(runningJobs) == 0 {
		return errors.NewResourceExpired("no running Flink jobs to stop")
	}
	return reconciler.cancelJobs(ctx, takeSavepoint, runningJobs)
}

// Cancel jobs.
func (reconciler *ClusterReconciler) cancelJobs(
	ctx context.Context,
	takeSavepoint bool,
	jobs []string) error {
	log := logr.FromContextOrDiscard(ctx)
	for _, jobID := range jobs {
		log.Info("Cancel running job", "jobID", jobID)
		var err = reconciler.cancelFlinkJob(ctx, jobID, takeSavepoint)
		if err != nil {
			log.Error(err, "Failed to cancel running job", "jobID", jobID)
			return err
		}
	}
	return nil
}

// Takes a savepoint if possible then stops the job.
func (reconciler *ClusterReconciler) cancelFlinkJob(ctx context.Context, jobID string, takeSavepoint bool) error {
	log := logr.FromContextOrDiscard(ctx)
	if takeSavepoint && canTakeSavepoint(reconciler.observed.cluster) {
		log.Info("Taking savepoint before stopping job", "jobID", jobID)
		var err = reconciler.takeSavepoint(ctx, jobID)
		if err != nil {
			return err
		}
	}

	var apiBaseURL = getFlinkAPIBaseURL(reconciler.observed.cluster)
	log.Info("Stoping job", "jobID", jobID)
	return reconciler.flinkClient.StopJob(apiBaseURL, jobID)
}

// canSuspendJob
func (reconciler *ClusterReconciler) canSuspendJob(ctx context.Context, jobID string, s *v1beta1.SavepointStatus) bool {
	log := logr.FromContextOrDiscard(ctx)
	var firstTry = !finalSavepointRequested(jobID, s)
	if firstTry {
		return true
	}

	switch s.State {
	case v1beta1.SavepointStateSucceeded:
		log.Info("Successfully savepoint completed, wait until the job stops")
		return false
	case v1beta1.SavepointStateInProgress:
		log.Info("Savepoint is in progress, wait until it is completed")
		return false
	case v1beta1.SavepointStateTriggerFailed:
		log.Info("Savepoint trigger failed in previous request")
	case v1beta1.SavepointStateFailed:
		log.Info("Savepoint failed on previous request")
	}

	var retryTimeArrived = hasTimeElapsed(s.UpdateTime, time.Now(), SavepointRetryIntervalSeconds)
	if !retryTimeArrived {
		log.Info("Wait until next retry time arrived")
	}
	return retryTimeArrived
}

func (reconciler *ClusterReconciler) shouldTakeSavepoint() v1beta1.SavepointReason {
	var observed = reconciler.observed
	var cluster = observed.cluster
	var jobSpec = observed.cluster.Spec.Job
	var job = observed.cluster.Status.Components.Job
	var savepoint = observed.cluster.Status.Savepoint
	var newRequestedControl = getNewControlRequest(cluster)

	if !canTakeSavepoint(reconciler.observed.cluster) {
		return ""
	}

	// Savepoint trigger priority is user request including update and job stop.
	switch {
	// TODO: spec.job.cancelRequested will be deprecated
	// Should stop job with savepoint by user requested control
	case newRequestedControl == v1beta1.ControlNameJobCancel || (jobSpec.CancelRequested != nil && *jobSpec.CancelRequested):
		return v1beta1.SavepointReasonJobCancel
	// Take savepoint by user request
	case newRequestedControl == v1beta1.ControlNameSavepoint:
		fallthrough
	// TODO: spec.job.savepointGeneration will be deprecated
	case jobSpec.SavepointGeneration > job.SavepointGeneration:
		// Triggered by savepointGeneration increased.
		// When previous savepoint is failed, savepoint trigger by spec.job.savepointGeneration is not possible
		// because the field cannot be increased more.
		// Note: checkSavepointGeneration in flinkcluster_validate.go
		return v1beta1.SavepointReasonUserRequested
	// Scheduled auto savepoint
	case jobSpec.AutoSavepointSeconds != nil:
		// When previous try was failed, check retry interval.
		if savepoint.IsFailed() && savepoint.TriggerReason == v1beta1.SavepointReasonScheduled {
			var nextRetryTime = util.GetTime(savepoint.UpdateTime).Add(SavepointRetryIntervalSeconds * time.Second)
			if time.Now().After(nextRetryTime) {
				return v1beta1.SavepointReasonScheduled
			} else {
				return ""
			}
		}
		// Check if next trigger time arrived.
		var compareTime string
		if len(job.SavepointTime) == 0 {
			compareTime = job.StartTime
		} else {
			compareTime = job.SavepointTime
		}
		var nextTime = getTimeAfterAddedSeconds(compareTime, int64(*jobSpec.AutoSavepointSeconds))
		if time.Now().After(nextTime) {
			return v1beta1.SavepointReasonScheduled
		}
	}
	return ""
}

// Trigger savepoint for a job then return savepoint status to update.
func (reconciler *ClusterReconciler) triggerSavepoint(
	ctx context.Context,
	jobID string,
	triggerReason v1beta1.SavepointReason,
	cancel bool) (*v1beta1.SavepointStatus, error) {
	log := logr.FromContextOrDiscard(ctx)
	var cluster = reconciler.observed.cluster
	var apiBaseURL = getFlinkAPIBaseURL(reconciler.observed.cluster)
	var triggerSuccess bool
	var savepointTriggerID *flink.SavepointTriggerID
	var triggerID string
	var message string
	var err error

	log.Info(fmt.Sprintf("Trigger savepoint for %s", triggerReason), "jobID", jobID)
	savepointTriggerID, err = reconciler.flinkClient.TriggerSavepoint(apiBaseURL, jobID, *cluster.Spec.Job.SavepointsDir, cancel)
	if err != nil {
		// limit message size to 1KiB
		if message = err.Error(); len(message) > 1024 {
			message = message[:1024] + "..."
		}
		triggerSuccess = false
		log.Info("Failed to trigger savepoint", "jobID", jobID, "triggerID", triggerID, "error", err)
	} else {
		triggerSuccess = true
		triggerID = savepointTriggerID.RequestID
		log.Info("Successfully savepoint triggered", "jobID", jobID, "triggerID", triggerID)
	}
	newSavepointStatus := reconciler.getNewSavepointStatus(triggerID, triggerReason, message, triggerSuccess)

	return newSavepointStatus, err
}

// Takes savepoint for a job then update job status with the info.
func (reconciler *ClusterReconciler) takeSavepoint(ctx context.Context, jobID string) error {
	log := logr.FromContextOrDiscard(ctx)
	apiBaseURL := getFlinkAPIBaseURL(reconciler.observed.cluster)

	log.Info("Taking savepoint.", "jobID", jobID)
	status, err := reconciler.flinkClient.TakeSavepoint(apiBaseURL, jobID, *reconciler.observed.cluster.Spec.Job.SavepointsDir)
	log.Info("Savepoint status.", "status", status, "error", err)

	if err == nil && len(status.FailureCause.StackTrace) > 0 {
		err = fmt.Errorf("%s", status.FailureCause.StackTrace)
	}

	if err != nil || !status.Completed {
		log.Info("Failed to take savepoint.", "jobID", jobID)
	}

	return err
}

func (reconciler *ClusterReconciler) updateStatus(
	ctx context.Context, ss **v1beta1.SavepointStatus, cs **v1beta1.FlinkClusterControlStatus) {
	log := logr.FromContextOrDiscard(ctx)

	var savepointStatus = *ss
	var controlStatus = *cs

	if savepointStatus == nil && controlStatus == nil {
		return
	}

	// Record events
	if savepointStatus != nil {
		eventType, eventReason, eventMessage := getSavepointEvent(*savepointStatus)
		reconciler.recorder.Event(reconciler.observed.cluster, eventType, eventReason, eventMessage)
	}
	if controlStatus != nil {
		eventType, eventReason, eventMessage := getControlEvent(*controlStatus)
		reconciler.recorder.Event(reconciler.observed.cluster, eventType, eventReason, eventMessage)
	}

	// Update status
	var clusterClone = reconciler.observed.cluster.DeepCopy()
	var statusUpdateErr error
	retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var newStatus = &clusterClone.Status
		if savepointStatus != nil {
			newStatus.Savepoint = savepointStatus
		}
		if controlStatus != nil {
			newStatus.Control = controlStatus
		}
		util.SetTimestamp(&newStatus.LastUpdateTime)
		log.Info("Updating cluster status", "clusterClone", clusterClone, "newStatus", newStatus)
		statusUpdateErr = reconciler.k8sClient.Status().Update(ctx, clusterClone)
		if statusUpdateErr == nil {
			return nil
		}
		var clusterUpdated v1beta1.FlinkCluster
		if err := reconciler.k8sClient.Get(
			ctx,
			types.NamespacedName{Namespace: clusterClone.Namespace, Name: clusterClone.Name}, &clusterUpdated); err == nil {
			clusterClone = clusterUpdated.DeepCopy()
		}
		return statusUpdateErr
	})
	if statusUpdateErr != nil {
		log.Error(
			statusUpdateErr, "Failed to update status.", "error", statusUpdateErr)
	}
}

func (reconciler *ClusterReconciler) updateJobDeployStatus(ctx context.Context) error {
	var log = logr.FromContextOrDiscard(ctx)
	var observedCluster = reconciler.observed.cluster
	var desiredJobSubmitter = reconciler.desired.Job
	var err error

	var clusterClone = observedCluster.DeepCopy()
	var newJob = clusterClone.Status.Components.Job

	// Reset running job information.
	newJob.ID = ""
	newJob.StartTime = ""
	newJob.CompletionTime = nil

	// Mark as job submitter is deployed.
	util.SetTimestamp(&newJob.DeployTime)
	util.SetTimestamp(&clusterClone.Status.LastUpdateTime)

	// Latest savepoint location should be fromSavepoint.
	var fromSavepoint = getFromSavepoint(desiredJobSubmitter.Spec)
	newJob.FromSavepoint = fromSavepoint
	if newJob.SavepointLocation != "" {
		newJob.SavepointLocation = fromSavepoint
	}

	// Update job status.
	err = reconciler.k8sClient.Status().Update(ctx, clusterClone)
	if err != nil {
		log.Error(
			err, "Failed to update job status for new job submitter", "error", err)
	} else {
		log.Info("Succeeded to update job status for new job submitter.", "job status", newJob)
	}
	return err
}

// getNewSavepointStatus returns newly triggered savepoint status.
func (reconciler *ClusterReconciler) getNewSavepointStatus(triggerID string, triggerReason v1beta1.SavepointReason, message string, triggerSuccess bool) *v1beta1.SavepointStatus {
	var jobID = reconciler.getFlinkJobID()
	var savepointState string
	var now string
	util.SetTimestamp(&now)

	if triggerSuccess {
		savepointState = v1beta1.SavepointStateInProgress
	} else {
		savepointState = v1beta1.SavepointStateTriggerFailed
	}
	var savepointStatus = &v1beta1.SavepointStatus{
		JobID:         jobID,
		TriggerID:     triggerID,
		TriggerReason: triggerReason,
		TriggerTime:   now,
		UpdateTime:    now,
		Message:       message,
		State:         savepointState,
	}
	return savepointStatus
}

// Convert raw time to object and add `addedSeconds` to it,
// getting a time object for the parsed `rawTime` with `addedSeconds` added to it.
func getTimeAfterAddedSeconds(rawTime string, addedSeconds int64) time.Time {
	var tc = &util.TimeConverter{}
	var lastTriggerTime = time.Time{}
	if len(rawTime) != 0 {
		lastTriggerTime = tc.FromString(rawTime)
	}
	return lastTriggerTime.Add(time.Duration(addedSeconds * int64(time.Second)))
}
