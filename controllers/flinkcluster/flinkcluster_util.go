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
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spotify/flink-on-k8s-operator/internal/flink"
	"github.com/spotify/flink-on-k8s-operator/internal/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/controllers/history"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

const (
	ControlRetries    = "retries"
	ControlMaxRetries = "3"

	RevisionNameLabel = "flinkoperator.k8s.io/revision-name"
	JobIdLabel        = "flinkoperator.k8s.io/job-id"

	SavepointRetryIntervalSeconds = 10
)

var (
	jobIdRegexp = regexp.MustCompile("JobID (.*)\n")
)

type UpdateState string
type JobSubmitState string

const (
	UpdateStateNoUpdate   UpdateState = "NoUpdate"
	UpdateStatePreparing  UpdateState = "Preparing"
	UpdateStateInProgress UpdateState = "InProgress"
	UpdateStateFinished   UpdateState = "Finished"

	JobDeployStateInProgress = "InProgress"
	JobDeployStateSucceeded  = "Succeeded"
	JobDeployStateFailed     = "Failed"
	JobDeployStateUnknown    = "Unknown"
)

type objectForPatch struct {
	Metadata objectMetaForPatch `json:"metadata"`
}

// objectMetaForPatch define object meta struct for patch operation
type objectMetaForPatch struct {
	Annotations map[string]interface{} `json:"annotations"`
}

func getFlinkAPIBaseURL(cluster *v1beta1.FlinkCluster) string {
	clusterDomain := os.Getenv("CLUSTER_DOMAIN")
	if clusterDomain == "" {
		clusterDomain = "cluster.local"
	}

	return fmt.Sprintf(
		"http://%s.%s.svc.%s:%d",
		getJobManagerServiceName(cluster.Name),
		cluster.Namespace,
		clusterDomain,
		*cluster.Spec.JobManager.Ports.UI)
}

// Gets ConfigMap name
func getConfigMapName(clusterName string) string {
	return clusterName + "-configmap"
}

// Gets PodDisruptionBudgetName name
func getPodDisruptionBudgetName(clusterName string) string {
	return "flink-" + clusterName
}

// Get HorizontalPodAutoscaler name
func getHorizontalPodAutoscalerName(clusterName string) string {
	return "flink-" + clusterName
}

// Gets JobManager StatefulSet name
func getJobManagerName(clusterName string) string {
	return clusterName + "-jobmanager"
}

// Gets JobManager service name
func getJobManagerServiceName(clusterName string) string {
	return clusterName + "-jobmanager"
}

// Gets JobManager ingress name
func getJobManagerIngressName(clusterName string) string {
	return clusterName + "-jobmanager"
}

// Gets TaskManager StatefulSet name
func getTaskManagerName(clusterName string) string {
	return clusterName + "-taskmanager"
}

func getJobManagerJobName(clusterName string) string {
	return getJobManagerName(clusterName)
}

func getSubmitterJobName(clusterName string) string {
	return clusterName + "-job-submitter"
}

// Checks whether it is possible to take savepoint.
func canTakeSavepoint(cluster *v1beta1.FlinkCluster) bool {
	var jobSpec = cluster.Spec.Job
	var savepointStatus = cluster.Status.Savepoint
	var job = cluster.Status.Components.Job
	return jobSpec != nil && jobSpec.SavepointsDir != nil &&
		!job.IsStopped() &&
		(savepointStatus == nil || savepointStatus.State != v1beta1.SavepointStateInProgress)
}

// Checks if the job should be stopped because a job-cancel was requested
func shouldStopJob(cluster *v1beta1.FlinkCluster) bool {
	var userControl = cluster.Annotations[v1beta1.ControlAnnotation]
	var cancelRequested = cluster.Spec.Job.CancelRequested
	return userControl == v1beta1.ControlNameJobCancel ||
		(cancelRequested != nil && *cancelRequested)
}

func getFromSavepoint(jobSpec batchv1.JobSpec) string {
	var jobArgs = jobSpec.Template.Spec.Containers[0].Args
	for i, arg := range jobArgs {
		if arg == "--fromSavepoint" && i < len(jobArgs)-1 {
			return jobArgs[i+1]
		}
	}
	return ""
}

// newRevision generates FlinkClusterSpec patch and makes new child ControllerRevision resource with it.
func newRevision(cluster *v1beta1.FlinkCluster, revision int64, collisionCount *int32) (*appsv1.ControllerRevision, error) {
	patch, err := newRevisionDataPatch(cluster)
	if err != nil {
		return nil, err
	}
	cr, err := history.NewControllerRevision(cluster,
		controllerKind,
		cluster.Labels,
		runtime.RawExtension{Raw: patch},
		revision,
		collisionCount)
	if err != nil {
		return nil, err
	}
	if cr.Annotations == nil {
		cr.Annotations = make(map[string]string)
	}
	for key, value := range cluster.Annotations {
		cr.Annotations[key] = value
	}
	cr.SetNamespace(cluster.GetNamespace())
	cr.GetLabels()[history.ControllerRevisionManagedByLabel] = cluster.GetName()
	return cr, nil
}

func newRevisionDataPatch(cluster *v1beta1.FlinkCluster) ([]byte, error) {
	// Ignore fields not related to rendering job resource.
	var c *v1beta1.FlinkCluster
	if cluster.Spec.Job != nil {
		c = cluster.DeepCopy()
		c.Spec.Job.CleanupPolicy = nil
		c.Spec.Job.RestartPolicy = nil
		c.Spec.Job.CancelRequested = nil
		c.Spec.Job.SavepointGeneration = 0
	} else {
		c = cluster
	}

	str := &bytes.Buffer{}
	err := unstructured.UnstructuredJSONScheme.Encode(c, str)

	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	json.Unmarshal([]byte(str.Bytes()), &raw)
	objCopy := make(map[string]interface{})
	spec := raw["spec"].(map[string]interface{})
	objCopy["spec"] = spec
	spec["$patch"] = "replace"

	// backward compatibility fix
	if c.Spec.Job != nil {
		job := spec["job"].(map[string]interface{})
		job["restartPolicy"] = nil
	}

	patch, err := json.Marshal(objCopy)
	return patch, err
}

func getCurrentRevisionName(r *v1beta1.RevisionStatus) string {
	return r.CurrentRevision[:strings.LastIndex(r.CurrentRevision, "-")]
}

func getNextRevisionName(r *v1beta1.RevisionStatus) string {
	return r.NextRevision[:strings.LastIndex(r.NextRevision, "-")]
}

func getRetryCount(data map[string]string) (string, error) {
	var err error
	var retries, ok = data["retries"]
	if ok {
		retryCount, err := strconv.Atoi(retries)
		if err == nil {
			retryCount++
			retries = strconv.Itoa(retryCount)
		}
	} else {
		retries = "1"
	}
	return retries, err
}

// getNewControlRequest returns new requested control that is not in progress now.
func getNewControlRequest(cluster *v1beta1.FlinkCluster) string {
	var userControl = cluster.Annotations[v1beta1.ControlAnnotation]
	var recorded = cluster.Status
	if recorded.Control == nil || recorded.Control.State != v1beta1.ControlStateInProgress {
		return userControl
	}
	return ""
}

func getControlStatus(controlName string, state string) *v1beta1.FlinkClusterControlStatus {
	var controlStatus = new(v1beta1.FlinkClusterControlStatus)
	controlStatus.Name = controlName
	controlStatus.State = state
	util.SetTimestamp(&controlStatus.UpdateTime)
	return controlStatus
}

func controlStatusChanged(cluster *v1beta1.FlinkCluster, controlName string) bool {
	if controlName == "" {
		return false
	}
	var recorded = cluster.Status
	if recorded.Control == nil || recorded.Control.Name != controlName {
		return true
	}
	return false
}

func getControlEvent(status v1beta1.FlinkClusterControlStatus) (eventType string, eventReason string, eventMessage string) {
	var msg = status.Message
	if len(msg) > 100 {
		msg = msg[:100] + "..."
	}
	switch status.State {
	case v1beta1.ControlStateRequested:
		eventType = corev1.EventTypeNormal
		eventReason = "ControlRequested"
		eventMessage = fmt.Sprintf("Requested new user control %v", status.Name)
	case v1beta1.ControlStateInProgress:
		eventType = corev1.EventTypeNormal
		eventReason = "ControlInProgress"
		eventMessage = fmt.Sprintf("In progress user control %v", status.Name)
	case v1beta1.ControlStateSucceeded:
		eventType = corev1.EventTypeNormal
		eventReason = "ControlSucceeded"
		eventMessage = fmt.Sprintf("Succesfully completed user control %v", status.Name)
	case v1beta1.ControlStateFailed:
		eventType = corev1.EventTypeWarning
		eventReason = "ControlFailed"
		if status.Message != "" {
			eventMessage = fmt.Sprintf("User control %v failed: %v", status.Name, msg)
		} else {
			eventMessage = fmt.Sprintf("User control %v failed", status.Name)
		}
	}
	return
}

func getSavepointEvent(status v1beta1.SavepointStatus) (eventType string, eventReason string, eventMessage string) {
	var msg = status.Message
	if len(msg) > 100 {
		msg = msg[:100] + "..."
	}
	var triggerReason = status.TriggerReason
	if triggerReason == v1beta1.SavepointReasonJobCancel || triggerReason == v1beta1.SavepointReasonUpdate {
		triggerReason = "for " + triggerReason
	}
	switch status.State {
	case v1beta1.SavepointStateTriggerFailed:
		eventType = corev1.EventTypeWarning
		eventReason = "SavepointFailed"
		eventMessage = fmt.Sprintf("Failed to trigger savepoint %v: %v", triggerReason, msg)
	case v1beta1.SavepointStateInProgress:
		eventType = corev1.EventTypeNormal
		eventReason = "SavepointTriggered"
		eventMessage = fmt.Sprintf("Triggered savepoint %v: triggerID %v.", triggerReason, status.TriggerID)
	case v1beta1.SavepointStateSucceeded:
		eventType = corev1.EventTypeNormal
		eventReason = "SavepointCreated"
		eventMessage = "Successfully savepoint created"
	case v1beta1.SavepointStateFailed:
		eventType = corev1.EventTypeWarning
		eventReason = "SavepointFailed"
		eventMessage = fmt.Sprintf("Savepoint creation failed: %v", msg)
	}
	return
}

func isUserControlFinished(controlStatus *v1beta1.FlinkClusterControlStatus) bool {
	return controlStatus.State == v1beta1.ControlStateSucceeded ||
		controlStatus.State == v1beta1.ControlStateFailed
}

// Check time has passed
func hasTimeElapsed(timeToCheckStr string, now time.Time, intervalSec int) bool {
	tc := &util.TimeConverter{}
	timeToCheck := tc.FromString(timeToCheckStr)
	intervalPassedTime := timeToCheck.Add(time.Duration(int64(intervalSec) * int64(time.Second)))
	return now.After(intervalPassedTime)
}

// isComponentUpdated checks whether the component updated.
// If the component is observed as well as the next revision name in status.nextRevision and component's label `flinkoperator.k8s.io/hash` are equal, then it is updated already.
// If the component is not observed and it is required, then it is not updated yet.
// If the component is not observed and it is optional, but it is specified in the spec, then it is not updated yet.
func isComponentUpdated(component client.Object, cluster *v1beta1.FlinkCluster) bool {
	if !cluster.Status.Revision.IsUpdateTriggered() {
		return true
	}
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
	case *policyv1.PodDisruptionBudget:
		if o == nil {
			return false
		}
	case *corev1.Service:
		if o == nil {
			return false
		}
	case *batchv1.Job:
		if o == nil {
			return cluster.Spec.Job == nil
		}
	case *networkingv1.Ingress:
		if o == nil {
			jm := cluster.Spec.JobManager
			return jm == nil || jm.Ingress == nil
		}
	case *autoscalingv2.HorizontalPodAutoscaler:
		if o == nil {
			return false
		}
	}

	labels := component.GetLabels()
	nextRevisionName := getNextRevisionName(&cluster.Status.Revision)

	return labels[RevisionNameLabel] == nextRevisionName
}

func areComponentsUpdated(components []client.Object, cluster *v1beta1.FlinkCluster) bool {
	for _, c := range components {
		if !isComponentUpdated(c, cluster) {
			return false
		}
	}
	return true
}

// isClusterUpdateToDate checks whether all cluster components are replaced to next revision.
func isClusterUpdateToDate(observed *ObservedClusterState) bool {
	if !observed.cluster.Status.Revision.IsUpdateTriggered() {
		return true
	}

	components := []client.Object{
		observed.configMap,
		observed.tmService,
		observed.jmService,
	}

	if !IsApplicationModeCluster(observed.cluster) {
		components = append(components, observed.jmStatefulSet)
	}

	if observed.cluster.Spec.PodDisruptionBudget != nil {
		components = append(components, observed.podDisruptionBudget)
	}

	if observed.cluster.Spec.TaskManager.HorizontalPodAutoscaler != nil {
		components = append(components, observed.horizontalPodAutoscaler)
	}

	switch observed.cluster.Spec.TaskManager.DeploymentType {
	case v1beta1.DeploymentTypeDeployment:
		components = append(components, observed.tmDeployment)
	case v1beta1.DeploymentTypeStatefulSet:
		components = append(components, observed.tmStatefulSet)
	}

	return areComponentsUpdated(components, observed.cluster)
}

// isFlinkAPIReady checks whether cluster is ready to submit job.
func isFlinkAPIReady(list *flink.JobsOverview) bool {
	// If the observed Flink job status list is not nil (e.g., emtpy list),
	// it means Flink REST API server is up and running. It is the source of
	// truth of whether we can submit a job.
	return list != nil
}

// jobStateFinalized returns true, if job state is saved so that it can be resumed later.
func finalSavepointRequested(jobID string, s *v1beta1.SavepointStatus) bool {
	return s != nil && s.JobID == jobID &&
		(s.TriggerReason == v1beta1.SavepointReasonUpdate ||
			s.TriggerReason == v1beta1.SavepointReasonJobCancel)
}

func getUpdateState(observed *ObservedClusterState) UpdateState {
	if observed.cluster == nil {
		return UpdateStateNoUpdate
	}

	clusterStatus := observed.cluster.Status
	if !clusterStatus.Revision.IsUpdateTriggered() {
		return UpdateStateNoUpdate
	}

	jobStatus := clusterStatus.Components.Job
	switch {
	case !isScaleUpdate(observed.revisions, observed.cluster) &&
		!jobStatus.UpdateReady(observed.cluster.Spec.Job, observed.observeTime):
		return UpdateStatePreparing
	case !isClusterUpdateToDate(observed):
		return UpdateStateInProgress
	}
	return UpdateStateFinished
}

func revisionDiff(revisions []*appsv1.ControllerRevision) map[string]util.DiffValue {
	if len(revisions) < 2 {
		return map[string]util.DiffValue{}
	}

	patchSpec := func(bytes []byte) map[string]any {
		var raw map[string]any
		json.Unmarshal(bytes, &raw)
		return raw["spec"].(map[string]any)
	}

	history.SortControllerRevisions(revisions)
	a, b := revisions[len(revisions)-2], revisions[len(revisions)-1]
	aSpec := patchSpec(a.Data.Raw)
	bSpec := patchSpec(b.Data.Raw)

	return util.MapDiff(aSpec, bSpec)
}

func isScaleUpdate(revisions []*appsv1.ControllerRevision, cluster *v1beta1.FlinkCluster) bool {
	if cluster != nil && cluster.Spec.Job == nil {
		return false
	}

	diff := revisionDiff(revisions)
	tmDiff, ok := diff["taskManager"]
	if len(diff) != 1 || !ok {
		return false
	}

	left := tmDiff.Left.(map[string]any)["replicas"]
	right := tmDiff.Right.(map[string]any)["replicas"]

	return left != right
}

func shouldUpdateJob(observed *ObservedClusterState) bool {
	return observed.updateState == UpdateStateInProgress && !isScaleUpdate(observed.revisions, observed.cluster)
}

func shouldUpdateCluster(observed *ObservedClusterState) bool {
	if isScaleUpdate(observed.revisions, observed.cluster) {
		return observed.updateState == UpdateStateInProgress
	}

	var job = observed.cluster.Status.Components.Job
	return !job.IsActive() && observed.updateState == UpdateStateInProgress
}

func shouldRecreateOnUpdate(observed *ObservedClusterState) bool {
	ru := observed.cluster.Spec.RecreateOnUpdate
	return *ru && !isScaleUpdate(observed.revisions, observed.cluster)
}

func getFlinkJobDeploymentState(flinkJobState string) v1beta1.JobState {
	switch flinkJobState {
	case "INITIALIZING", "CREATED", "RUNNING", "FAILING", "CANCELLING", "RESTARTING", "RECONCILING", "SUSPENDED":
		return v1beta1.JobStateRunning
	case "FINISHED":
		return v1beta1.JobStateSucceeded
	case "CANCELED":
		return v1beta1.JobStateCancelled
	case "FAILED":
		return v1beta1.JobStateFailed
	default:
		return v1beta1.JobStateUnknown
	}
}

// getFlinkJobSubmitLog extract logs from the job submitter pod.
func getFlinkJobSubmitLog(clientset *kubernetes.Clientset, observedPod *corev1.Pod) (*SubmitterLog, error) {
	log, err := util.GetPodLogs(clientset, observedPod)
	if err != nil {
		return nil, err
	}

	return getFlinkJobSubmitLogFromString(log), nil
}

func getFlinkJobSubmitLogFromString(podLog string) *SubmitterLog {
	if result := jobIdRegexp.FindStringSubmatch(podLog); len(result) > 0 {
		return &SubmitterLog{jobID: result[1], message: podLog}
	} else {
		return &SubmitterLog{jobID: "", message: podLog}
	}
}

func IsApplicationModeCluster(cluster *v1beta1.FlinkCluster) bool {
	jobSpec := cluster.Spec.Job
	return jobSpec != nil && jobSpec.Mode != nil && *jobSpec.Mode == v1beta1.JobModeApplication
}

// checks if job-cancel was requested
func wasJobCancelRequested(controlStatus *v1beta1.FlinkClusterControlStatus) bool {
	return controlStatus != nil && controlStatus.Name == v1beta1.ControlNameJobCancel
}

func GenJobId(cluster *v1beta1.FlinkCluster) (string, error) {
	if cluster != nil && cluster.Status.Components.Job != nil && cluster.Status.Components.Job.ID != "" {
		return cluster.Status.Components.Job.ID, nil
	}

	if cluster == nil || len(cluster.Status.Revision.NextRevision) == 0 {
		return "", fmt.Errorf("error generating job id: cluster or next revision is nil")
	}

	hash := md5.Sum([]byte(cluster.Status.Revision.NextRevision))
	return hex.EncodeToString(hash[:]), nil
}

func isJobInitialising(jobStatus batchv1.JobStatus) bool {
	return jobStatus.Active == 0 && jobStatus.Succeeded == 0 && jobStatus.Failed == 0
}
