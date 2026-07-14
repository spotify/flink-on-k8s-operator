package flinkcluster

import (
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	flink "github.com/spotify/flink-on-k8s-operator/internal/flink"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
)

const (
	logNilValue = "nil"
)

func logObjectSummary(obj client.Object) any {
	if isNilClientObject(obj) {
		return logNilValue
	}

	summary := map[string]any{
		"kind":      logObjectKind(obj),
		"namespace": obj.GetNamespace(),
		"name":      obj.GetName(),
	}
	if apiVersion := logObjectAPIVersion(obj); apiVersion != "" {
		summary["apiVersion"] = apiVersion
	}
	if generation := obj.GetGeneration(); generation != 0 {
		summary["generation"] = generation
	}
	if resourceVersion := obj.GetResourceVersion(); resourceVersion != "" {
		summary["resourceVersion"] = resourceVersion
	}
	return summary
}

func logFlinkClusterSummary(cluster *v1beta1.FlinkCluster) any {
	if cluster == nil {
		return logNilValue
	}

	summary := map[string]any{
		"kind":            logObjectKind(cluster),
		"namespace":       cluster.Namespace,
		"name":            cluster.Name,
		"generation":      cluster.Generation,
		"resourceVersion": cluster.ResourceVersion,
		"flinkVersion":    cluster.Spec.FlinkVersion,
		"image":           cluster.Spec.Image.Name,
		"state":           cluster.Status.State,
		"revision":        logRevisionStatusSummary(cluster.Status.Revision),
	}
	if cluster.Spec.Job != nil {
		summary["jobClass"] = cluster.Spec.Job.ClassName
	}
	if cluster.Spec.TaskManager != nil && cluster.Spec.TaskManager.Replicas != nil {
		summary["taskManagerReplicas"] = *cluster.Spec.TaskManager.Replicas
	}
	if cluster.Status.Components.Job != nil {
		summary["job"] = logJobStatusSummary(cluster.Status.Components.Job)
	}
	if cluster.Status.Savepoint != nil {
		summary["savepoint"] = logSavepointStatusSummary(cluster.Status.Savepoint)
	}
	if cluster.Status.Control != nil {
		summary["control"] = logControlStatusSummary(cluster.Status.Control)
	}
	return summary
}

func logObservedClusterStateSummary(observed *ObservedClusterState) map[string]any {
	if observed == nil {
		return map[string]any{"cluster": logNilValue}
	}

	summary := map[string]any{
		"cluster":                 logFlinkClusterSummary(observed.cluster),
		"controllerRevisions":     logControllerRevisionsSummary(observed.revisions),
		"configMap":               logObjectSummary(observed.configMap),
		"haConfigMap":             logObjectSummary(observed.haConfigMap),
		"podDisruptionBudget":     logObjectSummary(observed.podDisruptionBudget),
		"jobManagerStatefulSet":   logObjectSummary(observed.jmStatefulSet),
		"jobManagerService":       logObjectSummary(observed.jmService),
		"jobManagerIngress":       logObjectSummary(observed.jmIngress),
		"taskManagerStatefulSet":  logObjectSummary(observed.tmStatefulSet),
		"taskManagerDeployment":   logObjectSummary(observed.tmDeployment),
		"taskManagerService":      logObjectSummary(observed.tmService),
		"horizontalPodAutoscaler": logObjectSummary(observed.horizontalPodAutoscaler),
		"flinkJob":                logFlinkJobSummary(observed.flinkJob.status),
		"flinkJobCount":           logFlinkJobCount(observed.flinkJob.list),
		"flinkJobExceptionCount":  logFlinkJobExceptionCount(observed.flinkJob.exceptions),
		"unexpectedFlinkJobCount": len(observed.flinkJob.unexpected),
		"jobSubmitter":            logObjectSummary(observed.flinkJobSubmitter.job),
		"jobSubmitterPod":         logObjectSummary(observed.flinkJobSubmitter.pod),
		"jobSubmitterLog":         logSubmitterLogSummary(observed.flinkJobSubmitter.log),
		"savepoint":               logFlinkSavepointSummary(observed.savepoint.status, observed.savepoint.error),
	}
	if observed.persistentVolumeClaims != nil {
		summary["persistentVolumeClaimCount"] = len(observed.persistentVolumeClaims.Items)
	} else {
		summary["persistentVolumeClaimCount"] = logNilValue
	}
	return summary
}

func logDesiredClusterStateSummary(desired *model.DesiredClusterState) map[string]any {
	if desired == nil {
		return map[string]any{"components": logNilValue}
	}

	return map[string]any{
		"configMap":               logObjectSummary(desired.ConfigMap),
		"podDisruptionBudget":     logObjectSummary(desired.PodDisruptionBudget),
		"jobManagerStatefulSet":   logObjectSummary(desired.JmStatefulSet),
		"jobManagerService":       logObjectSummary(desired.JmService),
		"jobManagerIngress":       logObjectSummary(desired.JmIngress),
		"taskManagerStatefulSet":  logObjectSummary(desired.TmStatefulSet),
		"taskManagerDeployment":   logObjectSummary(desired.TmDeployment),
		"taskManagerService":      logObjectSummary(desired.TmService),
		"horizontalPodAutoscaler": logObjectSummary(desired.HorizontalPodAutoscaler),
		"job":                     logObjectSummary(desired.Job),
	}
}

func logObservedClusterStateFull(observed *ObservedClusterState) map[string]any {
	if observed == nil {
		return map[string]any{"cluster": logNilValue}
	}

	summary := map[string]any{
		"cluster":                 logFullObject(observed.cluster),
		"controllerRevisions":     observed.revisions,
		"configMap":               logFullObject(observed.configMap),
		"haConfigMap":             logFullObject(observed.haConfigMap),
		"podDisruptionBudget":     logFullObject(observed.podDisruptionBudget),
		"jobManagerStatefulSet":   logFullObject(observed.jmStatefulSet),
		"jobManagerService":       logFullObject(observed.jmService),
		"jobManagerIngress":       logFullObject(observed.jmIngress),
		"taskManagerStatefulSet":  logFullObject(observed.tmStatefulSet),
		"taskManagerDeployment":   logFullObject(observed.tmDeployment),
		"taskManagerService":      logFullObject(observed.tmService),
		"horizontalPodAutoscaler": logFullObject(observed.horizontalPodAutoscaler),
		"flinkJob":                logFullObject(observed.flinkJob.status),
		"flinkJobList":            logFullObject(observed.flinkJob.list),
		"flinkJobExceptions":      logFullObject(observed.flinkJob.exceptions),
		"unexpectedFlinkJobs":     observed.flinkJob.unexpected,
		"jobSubmitter":            logFullObject(observed.flinkJobSubmitter.job),
		"jobSubmitterPod":         logFullObject(observed.flinkJobSubmitter.pod),
		"jobSubmitterLog":         logSubmitterLogFull(observed.flinkJobSubmitter.log),
		"savepoint":               logFullObject(observed.savepoint.status),
	}
	if observed.persistentVolumeClaims != nil {
		summary["persistentVolumeClaims"] = observed.persistentVolumeClaims.Items
	} else {
		summary["persistentVolumeClaims"] = logNilValue
	}
	return summary
}

func logDesiredClusterStateFull(desired *model.DesiredClusterState) map[string]any {
	if desired == nil {
		return map[string]any{"components": logNilValue}
	}

	return map[string]any{
		"configMap":               logFullObject(desired.ConfigMap),
		"podDisruptionBudget":     logFullObject(desired.PodDisruptionBudget),
		"jobManagerStatefulSet":   logFullObject(desired.JmStatefulSet),
		"jobManagerService":       logFullObject(desired.JmService),
		"jobManagerIngress":       logFullObject(desired.JmIngress),
		"taskManagerStatefulSet":  logFullObject(desired.TmStatefulSet),
		"taskManagerDeployment":   logFullObject(desired.TmDeployment),
		"taskManagerService":      logFullObject(desired.TmService),
		"horizontalPodAutoscaler": logFullObject(desired.HorizontalPodAutoscaler),
		"job":                     logFullObject(desired.Job),
	}
}

func logFullObject(obj any) any {
	if obj == nil {
		return logNilValue
	}

	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return logNilValue
		}
		return value.Elem().Interface()
	}
	return obj
}

func logClusterStatusSummary(status *v1beta1.FlinkClusterStatus) any {
	if status == nil {
		return logNilValue
	}

	return map[string]any{
		"state":             status.State,
		"configMap":         logConfigMapStatusSummary(status.Components.ConfigMap),
		"jobManager":        logJobManagerStatusSummary(status.Components.JobManager),
		"jobManagerService": logJobManagerServiceStatusSummary(status.Components.JobManagerService),
		"jobManagerIngress": logJobManagerIngressStatusSummary(status.Components.JobManagerIngress),
		"taskManager":       logTaskManagerStatusSummary(status.Components.TaskManager),
		"job":               logJobStatusSummary(status.Components.Job),
		"control":           logControlStatusSummary(status.Control),
		"savepoint":         logSavepointStatusSummary(status.Savepoint),
		"revision":          logRevisionStatusSummary(status.Revision),
	}
}

func logConfigMapStatusSummary(status *v1beta1.ConfigMapStatus) any {
	if status == nil {
		return logNilValue
	}
	return map[string]any{
		"name":  status.Name,
		"state": status.State,
	}
}

func logJobManagerStatusSummary(status *v1beta1.JobManagerStatus) any {
	if status == nil {
		return logNilValue
	}
	return map[string]any{
		"name":          status.Name,
		"state":         status.State,
		"replicas":      status.Replicas,
		"readyReplicas": status.ReadyReplicas,
		"ready":         status.Ready,
	}
}

func logJobManagerServiceStatusSummary(status v1beta1.JobManagerServiceStatus) any {
	return map[string]any{
		"name":  status.Name,
		"state": status.State,
	}
}

func logJobManagerIngressStatusSummary(status *v1beta1.JobManagerIngressStatus) any {
	if status == nil {
		return logNilValue
	}
	return map[string]any{
		"name":     status.Name,
		"state":    status.State,
		"urlCount": len(status.URLs),
	}
}

func logTaskManagerStatusSummary(status *v1beta1.TaskManagerStatus) any {
	if status == nil {
		return logNilValue
	}
	return map[string]any{
		"name":          status.Name,
		"state":         status.State,
		"replicas":      status.Replicas,
		"readyReplicas": status.ReadyReplicas,
		"ready":         status.Ready,
	}
}

func logJobStatusSummary(status *v1beta1.JobStatus) any {
	if status == nil {
		return logNilValue
	}
	return map[string]any{
		"id":                   status.ID,
		"name":                 status.Name,
		"submitterName":        status.SubmitterName,
		"submitterExitCode":    status.SubmitterExitCode,
		"state":                status.State,
		"savepointGeneration":  status.SavepointGeneration,
		"finalSavepoint":       status.FinalSavepoint,
		"restartCount":         status.RestartCount,
		"failureReasonCount":   len(status.FailureReasons),
		"hasSavepointLocation": status.SavepointLocation != "",
		"hasFromSavepoint":     status.FromSavepoint != "",
	}
}

func logControlStatusSummary(status *v1beta1.FlinkClusterControlStatus) any {
	if status == nil {
		return logNilValue
	}
	return map[string]any{
		"name":        status.Name,
		"state":       status.State,
		"detailCount": len(status.Details),
		"hasMessage":  status.Message != "",
		"updateTime":  status.UpdateTime,
	}
}

func logSavepointStatusSummary(status *v1beta1.SavepointStatus) any {
	if status == nil {
		return logNilValue
	}
	return map[string]any{
		"jobID":       status.JobID,
		"triggerID":   status.TriggerID,
		"reason":      status.TriggerReason,
		"state":       status.State,
		"hasMessage":  status.Message != "",
		"triggerTime": status.TriggerTime,
		"updateTime":  status.UpdateTime,
	}
}

func logRevisionStatusSummary(status v1beta1.RevisionStatus) map[string]any {
	return map[string]any{
		"currentRevision": status.CurrentRevision,
		"nextRevision":    status.NextRevision,
	}
}

func logControllerRevisionsSummary(revisions []*appsv1.ControllerRevision) []map[string]any {
	summary := make([]map[string]any, 0, len(revisions))
	for _, revision := range revisions {
		if revision == nil {
			continue
		}
		summary = append(summary, map[string]any{
			"name":     revision.Name,
			"revision": revision.Revision,
		})
	}
	return summary
}

func logFlinkJobSummary(job *flink.Job) any {
	if job == nil {
		return logNilValue
	}
	return map[string]any{
		"id":        job.Id,
		"name":      job.Name,
		"state":     job.State,
		"startTime": job.StartTime,
		"endTime":   job.EndTime,
		"duration":  job.Duration,
	}
}

func logFlinkJobCount(jobs *flink.JobsOverview) any {
	if jobs == nil {
		return logNilValue
	}
	return len(jobs.Jobs)
}

func logFlinkJobExceptionCount(exceptions *flink.JobExceptions) any {
	if exceptions == nil {
		return logNilValue
	}
	return len(exceptions.Exceptions)
}

func logSubmitterLogSummary(submitterLog *SubmitterLog) any {
	if submitterLog == nil {
		return logNilValue
	}
	return map[string]any{
		"jobID":      submitterLog.jobID,
		"hasMessage": submitterLog.message != "",
	}
}

func logSubmitterLogFull(submitterLog *SubmitterLog) any {
	if submitterLog == nil {
		return logNilValue
	}
	return map[string]any{
		"jobID":   submitterLog.jobID,
		"message": submitterLog.message,
	}
}

func logFlinkSavepointSummary(status *flink.SavepointStatus, err error) any {
	if status == nil && err == nil {
		return logNilValue
	}

	summary := map[string]any{}
	if status != nil {
		summary["jobID"] = status.JobID
		summary["triggerID"] = status.TriggerID
		summary["completed"] = status.Completed
		summary["hasLocation"] = status.Location != ""
	}
	if err != nil {
		summary["error"] = err.Error()
	}
	return summary
}

func logObjectKind(obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Kind != "" {
		return gvk.Kind
	}

	objType := reflect.TypeOf(obj)
	for objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}
	return objType.Name()
}

func logObjectAPIVersion(obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk == (schema.GroupVersionKind{}) {
		return ""
	}
	return gvk.GroupVersion().String()
}

func isNilClientObject(obj any) bool {
	if obj == nil {
		return true
	}

	return isNilReflectValue(reflect.ValueOf(obj))
}

func isNilReflectValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
