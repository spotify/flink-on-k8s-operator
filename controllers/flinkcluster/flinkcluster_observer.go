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
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/controllers/history"
	flink "github.com/spotify/flink-on-k8s-operator/internal/flink"
	"github.com/spotify/flink-on-k8s-operator/internal/util"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStateObserver gets the observed state of the cluster.
type ClusterStateObserver struct {
	k8sClient    client.Client
	k8sClientset *kubernetes.Clientset
	flinkClient  *flink.Client
	request      ctrl.Request
	context      context.Context
	log          logr.Logger
	history      history.Interface
	recorder     record.EventRecorder
}

// ObservedClusterState holds observed state of a cluster.
type ObservedClusterState struct {
	cluster                *v1beta1.FlinkCluster
	revisions              []*appsv1.ControllerRevision
	configMap              *corev1.ConfigMap
	jmStatefulSet          *appsv1.StatefulSet
	jmService              *corev1.Service
	jmIngress              *networkingv1.Ingress
	tmStatefulSet          *appsv1.StatefulSet
	tmDeployment           *appsv1.Deployment
	tmService              *corev1.Service
	podDisruptionBudget    *policyv1.PodDisruptionBudget
	persistentVolumeClaims *corev1.PersistentVolumeClaimList
	flinkJob               FlinkJob
	flinkJobSubmitter      FlinkJobSubmitter
	savepoint              Savepoint
	revision               Revision
	observeTime            time.Time
	updateState            UpdateState
}

type FlinkJob struct {
	status     *flink.Job
	list       *flink.JobsOverview
	exceptions *flink.JobExceptions
	unexpected []string
}

type FlinkJobSubmitter struct {
	job *batchv1.Job
	pod *corev1.Pod
	log *SubmitterLog
}

type SubmitterLog struct {
	jobID   string
	message string
}

type Savepoint struct {
	status *flink.SavepointStatus
	error  error
}

type Revision struct {
	currentRevision *appsv1.ControllerRevision
	nextRevision    *appsv1.ControllerRevision
	collisionCount  int32
}

func (o *ObservedClusterState) isClusterUpdating() bool {
	return o.updateState == UpdateStateInProgress
}

// Job submitter status.
func (s *FlinkJobSubmitter) getState() JobSubmitState {
	switch {
	case s.log != nil && s.log.jobID != "":
		return JobDeployStateSucceeded
	// Job ID not found cases:
	// Failed and job ID not found.
	case s.job != nil && s.job.Status.Failed > 0:
		return JobDeployStateFailed
	// Ongoing job submission.
	case s.job != nil && s.job.Status.Succeeded == 0 && s.job.Status.Failed == 0:
		fallthrough
	// Finished, but failed to extract log.
	case s.log == nil:
		return JobDeployStateInProgress
	}
	// Abnormal case: successfully finished but job ID not found.
	return JobDeployStateUnknown
}

// Observes the state of the cluster and its components.
// NOT_FOUND error is ignored because it is normal, other errors are returned.
func (observer *ClusterStateObserver) observe(observed *ObservedClusterState) error {
	var log = observer.log

	// Cluster state.
	observed.cluster = new(v1beta1.FlinkCluster)
	if err := observer.observeCluster(observed.cluster); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get the cluster resource")
			return err
		}
		observer.sendDeletedEvent()
		observed.cluster = nil
	}

	if observed.cluster != nil {
		// Revisions.
		if err := observer.observeRevisions(observed); err != nil {
			log.Error(err, "Failed to get the controllerRevision resource list")
			return err
		}

		// ConfigMap.
		if err := observer.observeConfigMap(observed); err != nil {
			log.Error(err, "Failed to get configMap")
			return err
		}

		// PodDisruptionBudget.
		if err := observer.observePodDisruptionBudget(observed); err != nil {
			log.Error(err, "Failed to get PodDisruptionBudget")
			return err
		}

		// JobManager StatefulSet.
		if err := observer.observeJobManager(observed); err != nil {
			log.Error(err, "Failed to get JobManager StatefulSet")
			return err
		}

		// JobManager service.
		if err := observer.observeJobManagerService(observed); err != nil {
			log.Error(err, "Failed to get JobManager service")
			return err
		}

		// (Optional) JobManager ingress.
		if err := observer.observeJobManagerIngress(observed); err != nil {
			log.Error(err, "Failed to get JobManager ingress")
			return err
		}

		// TaskManager
		if err := observer.observeTaskManager(observed); err != nil {
			log.Error(err, "Failed to get TaskManager")
			return err
		}

		// TaskManager Service.
		if err := observer.observeTaskManagerService(observed); err != nil {
			log.Error(err, "Failed to get TaskManager Service")
			return err
		}

		// (Optional) Savepoint.
		if err := observer.observeSavepoint(observed.cluster, &observed.savepoint); err != nil {
			log.Error(err, "Failed to get Flink job savepoint status")
		}

		if err := observer.observePersistentVolumeClaims(observed); err != nil {
			log.Error(err, "Failed to get persistent volume claim list")
			return err
		}

		// (Optional) job.
		if err := observer.observeJob(observed); err != nil {
			log.Error(err, "Failed to get Flink job status")
			return err
		}
	}

	observed.observeTime = time.Now()
	observed.updateState = getUpdateState(observed)

	observer.logObservedState(observed)

	return nil
}

func (observer *ClusterStateObserver) sendDeletedEvent() {
	var eventCluster = &v1beta1.FlinkCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FlinkCluster",
			APIVersion: "flinkoperator.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      observer.request.Name,
			Namespace: observer.request.Namespace,
		},
	}
	observer.recorder.Event(
		eventCluster,
		"Normal",
		"StatusUpdate",
		"Cluster status: Deleted")
}

func (observer *ClusterStateObserver) observeJob(
	observed *ObservedClusterState) error {
	// Either the cluster has been deleted or it is a session cluster.
	if observed.cluster == nil || observed.cluster.Spec.Job == nil {
		return nil
	}
	var log = observer.log
	// Extract the log stream from pod only when the job state is Deploying.
	var recordedJob = observed.cluster.Status.Components.Job
	var jobName string
	var applicationMode = *observed.cluster.Spec.Job.Mode == v1beta1.JobModeApplication
	if applicationMode {
		jobName = getJobManagerJobName(observed.cluster.Name)
	} else {
		jobName = getSubmitterJobName(observed.cluster.Name)
	}

	// Job resource.
	job := new(batchv1.Job)
	if err := observer.observeObject(jobName, job); err != nil {
		log.Error(err, "Failed to get the job submitter")
		job = nil
	}

	// Get job submitter pod resource.
	jobPod := new(corev1.Pod)
	if err := observer.observeJobSubmitterPod(jobName, jobPod); err != nil {
		log.Error(err, "Failed to get the submitter pod")
		jobPod = nil
	}

	var submitterLog *SubmitterLog
	// Extract submission result only when it is in deployment progress.
	// It is not necessary to get the log stream from the submitter pod always.
	var jobDeployInProgress = recordedJob != nil && recordedJob.State == v1beta1.JobStateDeploying
	if jobPod != nil && jobDeployInProgress {
		var err error
		submitterLog, err = getFlinkJobSubmitLog(observer.k8sClientset, jobPod)
		if err != nil {
			// Error occurred while pulling log stream from the job submitter pod.
			// In this case the operator must return the error and retry in the next reconciliation iteration.
			log.Error(err, "Failed to get log stream from the job submitter pod. Will try again in the next iteration.")
			submitterLog = nil
		}
	}

	observed.flinkJobSubmitter = FlinkJobSubmitter{
		job: job,
		pod: jobPod,
		log: submitterLog,
	}

	// Wait until the job manager is ready.
	jmReady := applicationMode ||
		(observed.jmStatefulSet != nil && getStatefulSetState(observed.jmStatefulSet) == v1beta1.ComponentStateReady)
	if jmReady {
		// Observe the Flink job status.
		var flinkJobID string
		if jobID, ok := jobPod.Labels["job-id"]; ok {
			flinkJobID = jobID
		} else
		// Get the ID from the job submitter.
		if submitterLog != nil && submitterLog.jobID != "" {
			flinkJobID = submitterLog.jobID
		} else
		// Or get the job ID from the recorded job status which is written in previous iteration.
		if recordedJob != nil {
			flinkJobID = recordedJob.ID
		}
		observer.observeFlinkJobStatus(observed, flinkJobID, &observed.flinkJob)
	}

	return nil
}

// Observes Flink job status through Flink API (instead of Kubernetes jobs through
// Kubernetes API).
//
// This needs to be done after the job manager is ready, because we use it to detect whether the Flink API server is up
// and running.
func (observer *ClusterStateObserver) observeFlinkJobStatus(observed *ObservedClusterState, flinkJobID string, flinkJob *FlinkJob) {
	var log = observer.log
	// Observe following
	var flinkJobStatus *flink.Job
	var flinkJobList *flink.JobsOverview
	var flinkJobsUnexpected []string

	// Get Flink job status list.
	flinkAPIBaseURL := getFlinkAPIBaseURL(observed.cluster)
	flinkJobList, err := observer.flinkClient.GetJobsOverview(flinkAPIBaseURL)
	if err != nil {
		// It is normal in many cases, not an error.
		log.Info("Failed to get Flink job status list.", "error", err)
		return
	}
	flinkJob.list = flinkJobList

	// Extract the current job status and unexpected jobs.
	for _, job := range flinkJobList.Jobs {
		if flinkJobID == job.Id {
			flinkJobStatus = &job
		} else if getFlinkJobDeploymentState(job.State) == v1beta1.JobStateRunning {
			flinkJobsUnexpected = append(flinkJobsUnexpected, job.Id)
		}
	}
	flinkJob.status = flinkJobStatus
	flinkJob.unexpected = flinkJobsUnexpected

	log.Info("Observed Flink job",
		"submitted job status", flinkJob.status,
		"all job list", flinkJob.list,
		"unexpected job list", flinkJob.unexpected)
	if len(flinkJobsUnexpected) > 0 {
		log.Info("More than one unexpected Flink job were found!")
	}

	if flinkJobID == "" {
		log.Info("No flinkJobID given. Skipping get exceptions")
	} else {
		flinkJobExceptions, err := observer.flinkClient.GetJobExceptions(flinkAPIBaseURL, flinkJobID)
		if err != nil {
			// It is normal in many cases, not an error.
			log.Info("Failed to get Flink job exceptions.", "error", err)
		} else {
			log.Info("Observed Flink job exceptions", "jobs", flinkJobExceptions)
			flinkJob.exceptions = flinkJobExceptions
		}
	}

}

func (observer *ClusterStateObserver) observeSavepoint(cluster *v1beta1.FlinkCluster, savepoint *Savepoint) error {
	if cluster == nil ||
		cluster.Status.Savepoint == nil ||
		cluster.Status.Savepoint.State != v1beta1.SavepointStateInProgress {
		return nil
	}

	// Get savepoint status in progress.
	var flinkAPIBaseURL = getFlinkAPIBaseURL(cluster)
	var recordedSavepoint = cluster.Status.Savepoint
	var jobID = recordedSavepoint.JobID
	var triggerID = recordedSavepoint.TriggerID

	savepointStatus, err := observer.flinkClient.GetSavepointStatus(flinkAPIBaseURL, jobID, triggerID)
	savepoint.status = savepointStatus
	savepoint.error = err

	return err
}

func (observer *ClusterStateObserver) observeCluster(
	cluster *v1beta1.FlinkCluster) error {
	return observer.k8sClient.Get(
		observer.context, observer.request.NamespacedName, cluster)
}

func (observer *ClusterStateObserver) observeRevisions(
	observed *ObservedClusterState) error {
	observed.revisions = []*appsv1.ControllerRevision{}
	selector := labels.SelectorFromSet(labels.Set(map[string]string{history.ControllerRevisionManagedByLabel: observed.cluster.GetName()}))
	controllerRevisions, err := observer.history.ListControllerRevisions(observed.cluster, selector)
	observed.revisions = append(observed.revisions, controllerRevisions...)

	if client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}

func (observer *ClusterStateObserver) observePodDisruptionBudget(
	observed *ObservedClusterState) error {
	var clusterName = observer.request.Name
	observed.podDisruptionBudget = new(policyv1.PodDisruptionBudget)
	pdbName := getPodDisruptionBudgetName(clusterName)
	if err := observer.observeObject(pdbName, observed.podDisruptionBudget); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		observed.podDisruptionBudget = nil
	}
	return nil
}

func (observer *ClusterStateObserver) observeConfigMap(
	observed *ObservedClusterState) error {
	var clusterName = observer.request.Name
	observed.configMap = new(corev1.ConfigMap)
	configMapName := getConfigMapName(clusterName)
	if err := observer.observeObject(configMapName, observed.configMap); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		observed.configMap = nil
	}

	return nil
}

func (observer *ClusterStateObserver) observeJobManager(
	observed *ObservedClusterState) error {
	var clusterName = observer.request.Name
	var jmStatefulSetName = getJobManagerStatefulSetName(clusterName)
	observed.jmStatefulSet = new(appsv1.StatefulSet)
	if err := observer.observeObject(jmStatefulSetName, observed.jmStatefulSet); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		observed.jmStatefulSet = nil
	}

	return nil
}

func (observer *ClusterStateObserver) observeTaskManager(
	observed *ObservedClusterState) error {
	var clusterName = observer.request.Name
	// TaskManager StatefulSet
	tmDeploymentType := observed.cluster.Spec.TaskManager.DeploymentType
	if tmDeploymentType == "" || tmDeploymentType == v1beta1.DeploymentTypeStatefulSet {
		observed.tmStatefulSet = new(appsv1.StatefulSet)
		tmName := getTaskManagerStatefulSetName(clusterName)
		if err := observer.observeObject(tmName, observed.tmStatefulSet); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return err
			}
			observed.tmStatefulSet = nil
		}
	}

	// TaskManager Deployment
	if tmDeploymentType == v1beta1.DeploymentTypeDeployment {
		observed.tmDeployment = new(appsv1.Deployment)
		tmName := getTaskManagerDeploymentName(clusterName)
		if err := observer.observeObject(tmName, observed.tmDeployment); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return err
			}
			observed.tmDeployment = nil
		}
	}

	return nil
}

func (observer *ClusterStateObserver) observeTaskManagerService(
	observed *ObservedClusterState) error {
	var clusterName = observer.request.Name
	observed.tmService = new(corev1.Service)
	name := getTaskManagerStatefulSetName(clusterName)
	if err := observer.observeObject(name, observed.tmService); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		observed.tmService = nil
	}
	return nil
}

func (observer *ClusterStateObserver) observeJobManagerService(
	observed *ObservedClusterState) error {
	var clusterName = observer.request.Name
	observed.jmService = new(corev1.Service)
	jmSvcName := getJobManagerServiceName(clusterName)
	if err := observer.observeObject(jmSvcName, observed.jmService); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		observed.jmService = nil
	}
	return nil
}

func (observer *ClusterStateObserver) observeJobManagerIngress(
	observed *ObservedClusterState) error {
	var clusterName = observer.request.Name
	observed.jmIngress = new(networkingv1.Ingress)
	jmIngressName := getJobManagerIngressName(clusterName)
	if err := observer.observeObject(jmIngressName, observed.jmIngress); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		observed.jmIngress = nil
	}

	return nil
}

// observeJobSubmitterPod observes job submitter pod.
func (observer *ClusterStateObserver) observeJobSubmitterPod(
	jobName string,
	observedPod *corev1.Pod) error {
	var clusterNamespace = observer.request.Namespace
	var podSelector = labels.SelectorFromSet(map[string]string{"job-name": jobName})
	var podList = new(corev1.PodList)

	var err = observer.k8sClient.List(
		observer.context,
		podList,
		client.InNamespace(clusterNamespace),
		client.MatchingLabelsSelector{Selector: podSelector})
	if err != nil {
		return err
	}
	if len(podList.Items) == 0 {
		observedPod = nil
	} else {
		podList.Items[0].DeepCopyInto(observedPod)
	}

	return nil
}

func (observer *ClusterStateObserver) observePersistentVolumeClaims(
	observed *ObservedClusterState) error {
	var clusterNamespace = observer.request.Namespace
	var clusterName = observer.request.Name
	var selector = labels.SelectorFromSet(map[string]string{"cluster": clusterName})

	observed.persistentVolumeClaims = new(corev1.PersistentVolumeClaimList)
	err := observer.k8sClient.List(
		observer.context,
		observed.persistentVolumeClaims,
		client.InNamespace(clusterNamespace),
		client.MatchingLabelsSelector{Selector: selector})
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
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
func (observer *ClusterStateObserver) syncRevisionStatus(observed *ObservedClusterState) error {
	if observed.cluster == nil {
		return nil
	}

	var cluster = observed.cluster
	var revisions = observed.revisions
	var recorded = cluster.Status
	var currentRevision, nextRevision *appsv1.ControllerRevision
	var controllerHistory = observer.history

	revisionCount := len(revisions)
	history.SortControllerRevisions(revisions)

	// Use a local copy of cluster.Status.CollisionCount to avoid modifying cluster.Status directly.
	var collisionCount int32
	if recorded.Revision.CollisionCount != nil {
		collisionCount = *recorded.Revision.CollisionCount
	}

	// create a new revision from the current cluster
	nextRevision, err := newRevision(cluster, util.GetNextRevisionNumber(revisions), &collisionCount)
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
	if recorded.Revision.CurrentRevision == "" {
		currentRevision = nextRevision
		// attempt to find the revision that corresponds to the current revision
	} else {
		for i := range revisions {
			if revisions[i].Name == getCurrentRevisionName(&recorded.Revision) {
				currentRevision = revisions[i]
				break
			}
		}
	}
	if currentRevision == nil {
		return fmt.Errorf("current ControlRevision resoucre not found")
	}

	// Update revision status.
	observed.revision = Revision{
		currentRevision: currentRevision.DeepCopy(),
		nextRevision:    nextRevision.DeepCopy(),
		collisionCount:  collisionCount,
	}

	// maintain the revision history limit
	err = observer.truncateHistory(observed)
	if err != nil {
		return err
	}

	return nil
}

func (observer *ClusterStateObserver) truncateHistory(observed *ObservedClusterState) error {
	var cluster = observed.cluster
	var revisions = observed.revisions
	// TODO: default limit
	var historyLimit int
	if cluster.Spec.RevisionHistoryLimit != nil {
		historyLimit = int(*cluster.Spec.RevisionHistoryLimit)
	} else {
		historyLimit = 10
	}

	nonLiveHistory := util.GetNonLiveHistory(revisions, historyLimit)

	// delete any non-live history to maintain the revision limit.
	for i := 0; i < len(nonLiveHistory); i++ {
		if err := observer.history.DeleteControllerRevision(nonLiveHistory[i]); err != nil {
			return err
		}
	}
	return nil
}

func (observer *ClusterStateObserver) observeObject(name string, obj client.Object) error {
	var namespace = observer.request.Namespace
	return observer.k8sClient.Get(
		observer.context,
		types.NamespacedName{Namespace: namespace, Name: name},
		obj)
}

func (observer *ClusterStateObserver) logObservedState(observed *ObservedClusterState) error {
	log := observer.log

	if observed.cluster == nil {
		log = log.WithValues("cluster", "nil")
	} else {
		log = log.WithValues("cluster", *observed.cluster)

		var b strings.Builder
		for _, cr := range observed.revisions {
			fmt.Fprintf(&b, "{name: %v, revision: %v},", cr.Name, cr.Revision)
		}
		log = log.WithValues("controllerRevisions", fmt.Sprintf("[%v]", b.String()))
		if observed.configMap != nil {
			log = log.WithValues("configMap", *observed.configMap)
		} else {
			log = log.WithValues("configMap", "nil")
		}
		if observed.podDisruptionBudget != nil {
			log = log.WithValues("podDisruptionBudget", *observed.podDisruptionBudget)
		} else {
			log = log.WithValues("podDisruptionBudget", "nil")
		}
		if observed.jmStatefulSet != nil {
			log = log.WithValues("jmStatefulSet", *observed.jmStatefulSet)
		} else {
			log = log.WithValues("jmStatefulSet", "nil")
		}
		if observed.jmService != nil {
			log = log.WithValues("jmService", *observed.jmService)
		} else {
			log = log.WithValues("jmService", "nil")
		}
		if observed.jmIngress != nil {
			log = log.WithValues("jmIngress", *observed.jmIngress)
		} else {
			log = log.WithValues("jmIngress", "nil")
		}
		if observed.tmStatefulSet != nil {
			log = log.WithValues("tmStatefulSet", *observed.tmStatefulSet)
		} else {
			log = log.WithValues("tmStatefulSet", "nil")
		}
		if observed.tmDeployment != nil {
			log = log.WithValues("tmDeployment", *observed.tmDeployment)
		} else {
			log = log.WithValues("tmDeployment", "nil")
		}
		if observed.tmService != nil {
			log = log.WithValues("tmService", *observed.tmService)
		} else {
			log = log.WithValues("tmService", "nil")
		}
		if observed.savepoint.status != nil {
			log = log.WithValues("savepoint", *observed.savepoint.status)
		} else {
			log = log.WithValues("savepoint", "nil")
		}
		if observed.persistentVolumeClaims != nil {
			log = log.WithValues("persistentVolumeClaims", len(observed.persistentVolumeClaims.Items))
		} else {
			log = log.WithValues("persistentVolumeClaims", "nil")
		}

	}
	log.Info("Observed state")
	return nil
}
