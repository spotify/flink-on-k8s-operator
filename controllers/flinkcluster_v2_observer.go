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
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/api/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/controllers/flink"
	"github.com/spotify/flink-on-k8s-operator/controllers/history"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Observes the state of the cluster and its components.
// NOT_FOUND error is ignored because it is normal, other errors are returned.
func (observer *FlinkClusterHandlerV2) observe(ctx context.Context, controllerHistory history.Interface) (ObservedClusterState, error) {
	var observed = ObservedClusterState{}

	var err error
	var log = observer.log

	// Cluster.
	observed.cluster, err = observer.observeCluster(ctx)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get the cluster resource")
		return observed, err
	}

	// Revisions.
	observed.revisions, err = observer.observeRevisions(observed.cluster, controllerHistory)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get the controllerRevision resource list")
		return observed, err
	}

	// ConfigMap.
	observed.configMap, err = observer.observeConfigMap(ctx)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get configMap")
		return observed, err
	}

	// JobManager StatefulSet.
	observed.jmStatefulSet, err = observer.observeStatefulSet(ctx, getJobManagerStatefulSetName(observer.request.Name))
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get JobManager StatefulSet")
		return observed, err
	}

	// JobManager Service.
	observed.jmService, err = observer.observeJobManagerService(ctx)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get JobManager service")
		return observed, err
	}

	// (Optional) JobManager Ingress.
	observed.jmIngress, err = observer.observeJobManagerIngress(ctx)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get JobManager ingress")
		return observed, err
	}

	// TaskManager StatefulSet.
	observed.tmStatefulSet, err = observer.observeStatefulSet(ctx, getTaskManagerStatefulSetName(observer.request.Name))
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get TaskManager StatefulSet")
		return observed, err
	}

	// PersistentVolumeClaims.
	observed.persistentVolumeClaims, _ = observer.observePersistentVolumeClaims(ctx)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get PersistentVolumeClaims")
	}

	// Job.
	observed.job, err = observer.observeJobSubmitterJob(ctx, observed)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get JobSubmitter Job")
		return observed, err
	}

	observed.jobPod, _ = observer.observeJobSubmitterPod(ctx, observed)
	observed.flinkJobSubmitLog, _ = observer.observeFlinkJobSubmitLog(observed)

	observed.flinkJobStatus = observer.observeFlinkJobStatus(observed)

	// Savepoint.
	observed.savepoint, observed.savepointErr = observer.observeSavepoint(observed)

	observed.observeTime = time.Now()

	return observed, nil
}

func (observer *FlinkClusterHandlerV2) observeCluster(ctx context.Context) (*v1beta1.FlinkCluster, error) {
	observedCluster := new(v1beta1.FlinkCluster)
	err := observer.k8sClient.Get(ctx, observer.request.NamespacedName, observedCluster)
	if err != nil {
		return nil, err
	} else {
		return observedCluster, nil
	}
}

func (observer *FlinkClusterHandlerV2) observeRevisions(cluster *v1beta1.FlinkCluster, controllerHistory history.Interface) ([]*appsv1.ControllerRevision, error) {
	if cluster == nil {
		return nil, nil
	}

	var observedRevisions []*appsv1.ControllerRevision

	selector := labels.SelectorFromSet(labels.Set(map[string]string{history.ControllerRevisionManagedByLabel: cluster.GetName()}))
	controllerRevisions, err := controllerHistory.ListControllerRevisions(cluster, selector)
	observedRevisions = append(observedRevisions, controllerRevisions...)

	if err != nil {
		return nil, err
	} else {
		var b strings.Builder
		for _, cr := range observedRevisions {
			fmt.Fprintf(&b, "{name: %v, revision: %v},", cr.Name, cr.Revision)
		}
		return observedRevisions, nil
	}
}

func (observer *FlinkClusterHandlerV2) observeConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	observedConfigMap := new(corev1.ConfigMap)
	err := observer.k8sClient.Get(
		ctx,
		types.NamespacedName{
			Namespace: observer.request.Namespace,
			Name:      getConfigMapName(observer.request.Name),
		},
		observedConfigMap)

	if err != nil {
		return nil, err
	} else {
		return observedConfigMap, nil
	}
}

func (observer *FlinkClusterHandlerV2) observeStatefulSet(ctx context.Context, statefulSetName string) (*appsv1.StatefulSet, error) {
	observedStatefulSet := new(appsv1.StatefulSet)
	err := observer.k8sClient.Get(
		ctx,
		types.NamespacedName{
			Namespace: observer.request.Namespace,
			Name:      statefulSetName,
		},
		observedStatefulSet)

	if err != nil {
		return nil, err
	} else {
		return observedStatefulSet, nil
	}
}

func (observer *FlinkClusterHandlerV2) observeJobManagerService(ctx context.Context) (*corev1.Service, error) {
	observedJmService := new(corev1.Service)
	err := observer.k8sClient.Get(
		ctx,
		types.NamespacedName{
			Namespace: observer.request.Namespace,
			Name:      getJobManagerServiceName(observer.request.Name),
		},
		observedJmService)

	if err != nil {
		return nil, err
	} else {
		return observedJmService, nil
	}
}

func (observer *FlinkClusterHandlerV2) observeJobManagerIngress(ctx context.Context) (*extensionsv1beta1.Ingress, error) {
	observedJmIngress := new(extensionsv1beta1.Ingress)

	err := observer.k8sClient.Get(
		ctx,
		types.NamespacedName{
			Namespace: observer.request.Namespace,
			Name:      getJobManagerIngressName(observer.request.Name),
		},
		observedJmIngress)

	if err != nil {
		return nil, err
	} else {
		return observedJmIngress, nil
	}
}

func (observer *FlinkClusterHandlerV2) observePersistentVolumeClaims(ctx context.Context) (*corev1.PersistentVolumeClaimList, error) {
	observedClaims := new(corev1.PersistentVolumeClaimList)
	selector := labels.SelectorFromSet(map[string]string{"cluster": observer.request.Name})
	err := observer.k8sClient.List(
		ctx,
		observedClaims,
		client.InNamespace(observer.request.Namespace),
		client.MatchingLabelsSelector{Selector: selector})

	if err != nil {
		return nil, err
	} else {
		return observedClaims, nil
	}
}

func (observer *FlinkClusterHandlerV2) observeJobSubmitterJob(ctx context.Context, observed ObservedClusterState) (*batchv1.Job, error) {
	// Either the cluster has been deleted or it is a session cluster.
	if observed.cluster == nil || observed.cluster.Spec.Job == nil {
		return nil, nil
	}

	// Job resource.
	observedJob := new(batchv1.Job)

	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name

	err := observer.k8sClient.Get(
		ctx,
		types.NamespacedName{
			Namespace: clusterNamespace,
			Name:      getJobName(clusterName),
		},
		observedJob)

	if err != nil {
		return nil, err
	} else {
		return observedJob, nil
	}
}

func (observer *FlinkClusterHandlerV2) observeJobSubmitterPod(ctx context.Context, observed ObservedClusterState) (*corev1.Pod, error) {
	// Either the cluster has been deleted or it is a session cluster.
	if observed.cluster == nil || observed.cluster.Spec.Job == nil {
		return nil, nil
	}

	observedPod := new(corev1.Pod)

	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name
	var podSelector = labels.SelectorFromSet(map[string]string{"job-name": getJobName(clusterName)})
	var podList = new(corev1.PodList)

	var err = observer.k8sClient.List(
		ctx,
		podList,
		client.InNamespace(clusterNamespace),
		client.MatchingLabelsSelector{Selector: podSelector})
	if err != nil {
		return nil, err
	}

	if podList != nil && len(podList.Items) > 0 {
		podList.Items[0].DeepCopyInto(observedPod)
	} else {
		return nil, nil
	}

	return observedPod, nil
}

func (observer *FlinkClusterHandlerV2) observeFlinkJobSubmitLog(observed ObservedClusterState) (*FlinkJobSubmitLog, error) {
	// Either the cluster has been deleted or it is a session cluster.
	if observed.cluster == nil || observed.jobPod == nil || observed.cluster.Spec.Job == nil {
		return nil, nil
	}

	// Extract submit result.
	observedFlinkJobSubmitLog, err := getFlinkJobSubmitLog(observer.k8sClientset, observed.jobPod)
	if err != nil {
		observer.log.Info("Failed to extract job submit result", "error", err.Error())
	}

	return observedFlinkJobSubmitLog, err
}

// Observes Flink job status through Flink API (instead of Kubernetes jobs through
// Kubernetes API).
//
// This needs to be done after the job manager is ready, because we use it to detect whether the Flink API server is up
// and running.
func (observer *FlinkClusterHandlerV2) observeFlinkJobStatus(observed ObservedClusterState) FlinkJobStatus {
	// Either the cluster has been deleted or it is a session cluster.
	if observed.cluster == nil || observed.cluster.Spec.Job == nil {
		return FlinkJobStatus{}
	}

	var log = observer.log
	flinkJobStatus := FlinkJobStatus{}

	// Wait until the job manager is ready.
	jmReady := observed.jmStatefulSet != nil && getStatefulSetState(observed.jmStatefulSet) == v1beta1.ComponentStateReady
	if !jmReady {
		log.Info("Skip getting Flink job status; JobManager is not ready")
		return flinkJobStatus
	}

	// Check expected job id from submission log
	var expectedJobId *string
	if observed.flinkJobSubmitLog != nil {
		expectedJobId = &observed.flinkJobSubmitLog.JobID
	} else {
		expectedJobId = nil
	}

	// Observe job from cluster
	flinkAPIBaseURL := getFlinkAPIBaseURL(observed.cluster)
	flinkJobList, err := observer.flinkClient.GetJobsOverview(flinkAPIBaseURL)
	if err != nil {
		// It is normal in many cases, not an error.
		log.Info("Failed to get Flink job status list.", "error", err)
		return flinkJobStatus
	}
	flinkJobStatus.flinkJobList = flinkJobList

	var runningFlinkJobIds []flink.Job
	var stoppedFlinkJobIds []flink.Job

	for _, job := range flinkJobList.Jobs {
		switch job.State {
		case "INITIALIZING", "CREATED", "RUNNING", "FAILING", "CANCELLING", "RESTARTING", "RECONCILING":
			runningFlinkJobIds = append(runningFlinkJobIds, job)
		case "FINISHED", "CANCELED", "FAILED", "SUSPENDED":
			stoppedFlinkJobIds = append(stoppedFlinkJobIds, job)
		default:
			log.Info("Unknown status", "status", job.State)
		}
	}

	startTimeComparator := func(p1, p2 *flink.Job) bool {
		return p1.StartTime > p2.StartTime
	}
	sort.Sort(&flink.JobSorter{
		Jobs: runningFlinkJobIds,
		By:   startTimeComparator,
	})
	sort.Sort(&flink.JobSorter{
		Jobs: stoppedFlinkJobIds,
		By:   startTimeComparator,
	})

	if len(runningFlinkJobIds) > 0 {
		flinkJobStatus.flinkJob = &runningFlinkJobIds[0]
	} else if len(stoppedFlinkJobIds) > 0 {
		flinkJobStatus.flinkJob = &stoppedFlinkJobIds[0]
	}

	if len(runningFlinkJobIds) > 1 {
		// Unexpected jobs
		flinkJobStatus.flinkJobsUnexpected = make([]string, 0)
		for _, job := range flinkJobList.Jobs {
			if flinkJobStatus.flinkJob.Id != job.Id {
				flinkJobStatus.flinkJobsUnexpected = append(flinkJobStatus.flinkJobsUnexpected, job.Id)
			}
		}
	}

	if expectedJobId == nil {
		if len(runningFlinkJobIds) > 0 {
			log.Info(fmt.Sprintf("No expected job. Job id %s is found.", flinkJobStatus.flinkJob.Id))
		} else {
			log.Info("No expected job, no job running.")
		}
	} else {
		if flinkJobStatus.flinkJob != nil {
			if *expectedJobId != flinkJobStatus.flinkJob.Id {
				log.Info(fmt.Sprintf("Job ID mismatch. Expected job id %s but job id %s is found", *expectedJobId, flinkJobStatus.flinkJob.Id))
				flinkJobStatus.flinkJobExceptions = observer.observeFlinkJobException(observed, flinkAPIBaseURL, *expectedJobId)
			} else {
				log.Info("Job id match.")
			}
		} else {
			log.Info(fmt.Sprintf("Job ID missing. Expected job id %s but no job is found", *expectedJobId))
		}
	}

	// Unexpected jobs
	if len(flinkJobStatus.flinkJobsUnexpected) > 1 {
		log.Error(errors.New("more than one unexpected Flink job were found"), "", "unexpected jobs", flinkJobStatus.flinkJobsUnexpected)
	}

	if flinkJobStatus.flinkJob != nil {
		log.Info("Observed Flink job", "flink job", *flinkJobStatus.flinkJob)
	}

	return flinkJobStatus
}

func (observer *FlinkClusterHandlerV2) observeFlinkJobException(observed ObservedClusterState, flinkAPIBaseURL string, flinkJobID string) *flink.JobExceptions {
	var log = observer.log
	flinkJobExceptions, err := observer.flinkClient.GetJobExceptions(flinkAPIBaseURL, flinkJobID)
	if err != nil {
		// It is normal in many cases, not an error.
		log.Info("Failed to get Flink job exceptions.", "error", err)
		return nil
	}
	log.Info("Observed Flink job exceptions", "jobs", flinkJobExceptions)
	return flinkJobExceptions
}

func (observer *FlinkClusterHandlerV2) observeSavepoint(observed ObservedClusterState) (*flink.SavepointStatus, error) {
	var log = observer.log

	if observed.cluster == nil {
		return nil, nil
	}

	// Get savepoint status in progress.
	var savepointStatus = observed.cluster.Status.Savepoint

	if savepointStatus != nil && savepointStatus.State == v1beta1.SavepointStateInProgress {
		var flinkAPIBaseURL = getFlinkAPIBaseURL(observed.cluster)
		var jobID = savepointStatus.JobID
		var triggerID = savepointStatus.TriggerID

		savepoint, err := observer.flinkClient.GetSavepointStatus(flinkAPIBaseURL, jobID, triggerID)

		if err == nil && len(savepoint.FailureCause.StackTrace) > 0 {
			err = fmt.Errorf("%s", savepoint.FailureCause.StackTrace)
		}
		if err != nil {
			log.Info("Failed to get savepoint.", "error", err, "jobID", jobID, "triggerID", triggerID)
		}

		return savepoint, err
	}

	return nil, nil
}

// syncRevisionStatus synchronizes current FlinkCluster resource and its child ControllerRevision resources.
// When FlinkCluster resource is edited, the operator creates new child ControllerRevision for it
// and updates nextRevision in FlinkClusterStatus to the name of the new ControllerRevision.
// At that time, the name of the ControllerRevision is composed with the hash string generated
// from the FlinkClusterSpec which is to be stored in it.
// Therefore the contents of the ControllerRevision resources are maintained not duplicate.
// If edited FlinkClusterSpec is the same with the content of any existing ControllerRevision resources,
// the operator will only update nextRevision of the FlinkClusterStatus to the name of the ControllerRevision
// that has the same content, instead of creating new ControllerRevision.
// Finally, it maintains the number of child ControllerRevision resources according to RevisionHistoryLimit.
func (observer *FlinkClusterHandlerV2) syncRevisionStatus(observed *ObservedClusterState, controllerHistory history.Interface) error {
	if observed.cluster == nil {
		return nil
	}

	var revisions = observed.revisions
	var cluster = observed.cluster
	var recordedStatus = cluster.Status
	var currentRevision, nextRevision *appsv1.ControllerRevision
	var revisionStatus = observed.revisionStatus

	revisionCount := len(revisions)
	history.SortControllerRevisions(revisions)

	// Use a local copy of cluster.Status.CollisionCount to avoid modifying cluster.Status directly.
	var collisionCount int32
	if recordedStatus.CollisionCount != nil {
		collisionCount = *recordedStatus.CollisionCount
	}

	// create a new revision from the current cluster
	nextRevision, err := newRevision(cluster, getNextRevisionNumber(revisions), &collisionCount)
	if err != nil {
		return err
	}

	// find any equivalent revisions
	equalRevisions := history.FindEqualRevisions(revisions, nextRevision)
	equalCount := len(equalRevisions)
	if equalCount > 0 && history.EqualRevision(revisions[revisionCount-1], equalRevisions[equalCount-1]) {
		// if the equivalent revision is immediately prior the next revision has not changed
		nextRevision = revisions[revisionCount-1]
	} else if equalCount > 0 {
		// if the equivalent revision is not immediately prior we will roll back by incrementing the
		// Revision of the equivalent revision
		nextRevision, err = controllerHistory.UpdateControllerRevision(
			equalRevisions[equalCount-1],
			nextRevision.Revision)
		if err != nil {
			return err
		}
	} else {
		//if there is no equivalent revision we create a new one
		nextRevision, err = controllerHistory.CreateControllerRevision(cluster, nextRevision, &collisionCount)
		if err != nil {
			return err
		}
	}

	// if the current revision is nil we initialize the history by setting it to the next revision
	if recordedStatus.CurrentRevision == "" {
		currentRevision = nextRevision
		// attempt to find the revision that corresponds to the current revision
	} else {
		for i := range revisions {
			if revisions[i].Name == getCurrentRevisionName(recordedStatus) {
				currentRevision = revisions[i]
				break
			}
		}
	}
	if currentRevision == nil {
		return fmt.Errorf("current ControlRevision resoucre not found")
	}

	// update revision status
	revisionStatus = new(RevisionStatus)
	revisionStatus.currentRevision = currentRevision.DeepCopy()
	revisionStatus.nextRevision = nextRevision.DeepCopy()
	revisionStatus.collisionCount = collisionCount
	observed.revisionStatus = revisionStatus

	// maintain the revision history limit
	err = observer.truncateHistory(observed, controllerHistory)
	if err != nil {
		return err
	}

	return nil
}

func (observer *FlinkClusterHandlerV2) truncateHistory(observed *ObservedClusterState, controllerHistory history.Interface) error {
	var cluster = observed.cluster
	var revisions = observed.revisions
	// TODO: default limit
	var historyLimit int
	if cluster.Spec.RevisionHistoryLimit != nil {
		historyLimit = int(*cluster.Spec.RevisionHistoryLimit)
	} else {
		historyLimit = 10
	}

	nonLiveHistory := getNonLiveHistory(revisions, historyLimit)

	// delete any non-live history to maintain the revision limit.
	for i := 0; i < len(nonLiveHistory); i++ {
		if err := controllerHistory.DeleteControllerRevision(nonLiveHistory[i]); err != nil {
			return err
		}
	}
	return nil
}

// ======================================================
// Utils

func (observed ObservedClusterState) isNewRevision() bool {
	return observed.cluster.Status.CurrentRevision != observed.cluster.Status.NextRevision
}

func (observed ObservedClusterState) isClusterStopped() bool {
	return observed.cluster.Status.State == v1beta1.ClusterStateStopped
}

func (observed ObservedClusterState) isComponentUpdated(component runtime.Object) bool {
	// MUST: Add is new revision check
	// if !isUpdateTriggered(cluster.Status) {
	// 	return true
	// }
	switch o := component.(type) {
	case *appsv1.Deployment:
		if o == nil {
			return false
		}
	case *appsv1.StatefulSet:
		if o == nil {
			return false
		}
	case *corev1.ConfigMap:
		if o == nil {
			return false
		}
	case *corev1.Service:
		if o == nil {
			return false
		}
	case *batchv1.Job:
		if o == nil {
			return observed.cluster.Spec.Job == nil
		}
	case *extensionsv1beta1.Ingress:
		if o == nil {
			return observed.cluster.Spec.JobManager.Ingress == nil
		}
	}

	var labels, err = meta.NewAccessor().Labels(component)
	var nextRevisionName = getNextRevisionName(observed.cluster.Status)

	if err != nil {
		return false
	}

	return labels[RevisionNameLabel] == nextRevisionName
}

func (observed ObservedClusterState) isJobStopped() bool {
	status := observed.cluster.Status.Components.Job
	return status != nil &&
		(status.State == v1beta1.JobStateSucceeded ||
			status.State == v1beta1.JobStateFailed ||
			status.State == v1beta1.JobStateCancelled ||
			status.State == v1beta1.JobStateSuspended ||
			status.State == v1beta1.JobStateLost)
}

func (observed ObservedClusterState) isJobUpdating() bool {
	return observed.cluster.Status.Components.Job != nil &&
		observed.cluster.Status.Components.Job.State == v1beta1.JobStateUpdating
}

// Gets Flink job ID based on the observed state and the recorded state.
//
// It is possible that the recorded is not nil, but the observed is, due
// to transient error or being skiped as an optimization.
// If this returned nil, it is the state that job is not submitted or not identified yet.
func (observed ObservedClusterState) getFlinkJobId() *string {
	var observedFlinkJob = observed.flinkJobStatus.flinkJob
	if observedFlinkJob != nil && len(observedFlinkJob.Id) > 0 {
		return &observedFlinkJob.Id
	}

	// Observed from job submitter (when job manager is not ready yet)
	var observedJobSubmitLog = observed.flinkJobSubmitLog
	if observedJobSubmitLog != nil && observedJobSubmitLog.JobID != "" {
		return &observedJobSubmitLog.JobID
	}

	// Recorded.
	var recordedJobStatus = observed.cluster.Status.Components.Job
	if recordedJobStatus != nil && len(recordedJobStatus.ID) > 0 {
		return &recordedJobStatus.ID
	}

	return nil
}

// Checks whether the component should be deleted according to the cleanup
// policy. Always return false for session cluster.
func (observed ObservedClusterState) shouldCleanup(cluster *v1beta1.FlinkCluster, component string) bool {
	var jobStatus = cluster.Status.Components.Job

	// Session cluster.
	if jobStatus == nil {
		return false
	}

	if isUpdateTriggered(cluster.Status) {
		return false
	}

	var action v1beta1.CleanupAction
	switch jobStatus.State {
	case v1beta1.JobStateSucceeded:
		action = cluster.Spec.Job.CleanupPolicy.AfterJobSucceeds
	case v1beta1.JobStateFailed, v1beta1.JobStateLost:
		action = cluster.Spec.Job.CleanupPolicy.AfterJobFails
	case v1beta1.JobStateCancelled:
		action = cluster.Spec.Job.CleanupPolicy.AfterJobCancelled
	default:
		return false
	}

	switch action {
	case v1beta1.CleanupActionDeleteCluster:
		return true
	case v1beta1.CleanupActionDeleteTaskManager:
		return component == "TaskManagerStatefulSet"
	}

	return false
}
