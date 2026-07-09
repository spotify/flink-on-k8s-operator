/*
Copyright 2026 Spotify AB.

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

package v1beta1

const logNilValue = "nil"

// FlinkClusterLogSummary returns a compact representation of a FlinkCluster for logging.
func FlinkClusterLogSummary(cluster *FlinkCluster) any {
	if cluster == nil {
		return logNilValue
	}

	kind := cluster.GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		kind = "FlinkCluster"
	}

	summary := map[string]any{
		"kind":            kind,
		"namespace":       cluster.Namespace,
		"name":            cluster.Name,
		"generation":      cluster.Generation,
		"resourceVersion": cluster.ResourceVersion,
		"flinkVersion":    cluster.Spec.FlinkVersion,
		"image":           cluster.Spec.Image.Name,
		"state":           cluster.Status.State,
		"revision":        RevisionStatusLogSummary(cluster.Status.Revision),
	}
	if cluster.Spec.Job != nil {
		summary["jobClass"] = cluster.Spec.Job.ClassName
	}
	if cluster.Spec.TaskManager != nil && cluster.Spec.TaskManager.Replicas != nil {
		summary["taskManagerReplicas"] = *cluster.Spec.TaskManager.Replicas
	}
	if cluster.Status.Components.Job != nil {
		summary["job"] = JobStatusLogSummary(cluster.Status.Components.Job)
	}
	if cluster.Status.Savepoint != nil {
		summary["savepoint"] = SavepointStatusLogSummary(cluster.Status.Savepoint)
	}
	if cluster.Status.Control != nil {
		summary["control"] = ControlStatusLogSummary(cluster.Status.Control)
	}
	return summary
}

// JobStatusLogSummary returns a compact representation of a job status for logging.
func JobStatusLogSummary(status *JobStatus) any {
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

// ControlStatusLogSummary returns a compact representation of a control status for logging.
func ControlStatusLogSummary(status *FlinkClusterControlStatus) any {
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

// SavepointStatusLogSummary returns a compact representation of a savepoint status for logging.
func SavepointStatusLogSummary(status *SavepointStatus) any {
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

// RevisionStatusLogSummary returns a compact representation of a revision status for logging.
func RevisionStatusLogSummary(status RevisionStatus) map[string]any {
	return map[string]any{
		"currentRevision": status.CurrentRevision,
		"nextRevision":    status.NextRevision,
	}
}
