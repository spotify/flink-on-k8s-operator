package v1beta1

import (
	"fmt"
	"strings"
	"time"

	"github.com/spotify/flink-on-k8s-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
)

const (
	haConfigType       = "high-availability"
	haConfigTypeNew    = "high-availability.type"
	haConfigStorageDir = "high-availability.storageDir"
	haConfigClusterId  = "kubernetes.cluster-id"
)

func (j *JobStatus) IsActive() bool {
	return j != nil &&
		(j.State == JobStateRunning || j.State == JobStateDeploying)
}

func (j *JobStatus) IsPending() bool {
	return j != nil &&
		(j.State == JobStatePending ||
			j.State == JobStateUpdating ||
			j.State == JobStateRestarting)
}

func (j *JobStatus) IsFailed() bool {
	return j != nil &&
		(j.State == JobStateFailed ||
			j.State == JobStateLost ||
			j.State == JobStateDeployFailed)
}

func (j *JobStatus) IsStopped() bool {
	return j != nil &&
		(j.State == JobStateSucceeded ||
			j.State == JobStateCancelled ||
			j.IsFailed())
}

func (j *JobStatus) IsTerminated(spec *FlinkClusterSpec) bool {
	return j.IsStopped() && !j.ShouldRestart(spec)
}

// IsSavepointUpToDate check if the recorded savepoint is up-to-date compared to maxStateAgeToRestoreSeconds.
// If maxStateAgeToRestoreSeconds is not set,
// the savepoint is up-to-date only when the recorded savepoint is the final job state.
func (j *JobStatus) IsSavepointUpToDate(spec *JobSpec, compareTime time.Time) bool {
	if j.FinalSavepoint {
		return true
	}
	if compareTime.IsZero() ||
		spec.MaxStateAgeToRestoreSeconds == nil ||
		j.SavepointLocation == "" ||
		j.SavepointTime == "" {
		return false
	}

	var stateMaxAge = int(*spec.MaxStateAgeToRestoreSeconds)
	return !util.HasTimeElapsed(j.SavepointTime, compareTime, stateMaxAge)
}

// IsApplicationMode returns true if the job is deployed in application mode.
func (j *JobSpec) IsApplicationMode() bool {
	return j != nil && j.Mode != nil && *j.Mode == JobModeApplication
}

// ShouldRestart returns true if the controller should restart failed job.
// The controller can restart the job if policy is set to FromSavepointOnFailure.
// Job will restart from savepoint if the savepoint was taken successfully.
// The restart policy has no effect with a JobManager Deployment: Kubernetes recreates the JobManager pod,
// which recovers the job from its high availability metadata instead of a savepoint.
func (j *JobStatus) ShouldRestart(spec *FlinkClusterSpec) bool {
	if j == nil || !j.IsFailed() || spec == nil || spec.Job == nil ||
		spec.JobManager.GetDeploymentType() == JobManagerDeploymentTypeDeployment {
		return false
	}

	restartEnabled := spec.Job.RestartPolicy != nil && *spec.Job.RestartPolicy == JobRestartPolicyFromSavepointOnFailure
	return restartEnabled
}

// UpdateReady returns true if job is ready to proceed update.
func (j *JobStatus) UpdateReady(spec *JobSpec, observeTime time.Time) bool {
	var takeSavepointOnUpdate = spec.TakeSavepointOnUpdate == nil || *spec.TakeSavepointOnUpdate
	switch {
	case j == nil:
		fallthrough
	case !isBlank(spec.FromSavepoint):
		return true
	case j.IsActive():
		// When job is active and takeSavepointOnUpdate is true, only after taking savepoint with final job state,
		// proceed job update.
		if takeSavepointOnUpdate {
			if j.FinalSavepoint {
				return true
			}
		} else if j.IsSavepointUpToDate(spec, observeTime) {
			return true
		}
	case j.State == JobStateUpdating && !takeSavepointOnUpdate:
		return true
	default:
		// In other cases, check if savepoint is up-to-date compared to job end time.
		var jobCompletionTime time.Time
		if !j.CompletionTime.IsZero() {
			jobCompletionTime = j.CompletionTime.Time
		}
		if j.IsSavepointUpToDate(spec, jobCompletionTime) {
			return true
		}
	}
	return false
}

func (s *SavepointStatus) IsFailed() bool {
	return s != nil && (s.State == SavepointStateTriggerFailed || s.State == SavepointStateFailed)
}

func (r *RevisionStatus) IsUpdateTriggered() bool {
	return r.CurrentRevision != r.NextRevision
}

func isBlank(s *string) bool {
	return s == nil || strings.TrimSpace(*s) == ""
}

func (jm *JobManagerSpec) GetResources() *corev1.ResourceList {
	return util.UpperBoundedResourceList(jm.Resources)
}

// GetDeploymentType returns the JobManager workload type, treating unset as Job.
func (jm *JobManagerSpec) GetDeploymentType() JobManagerDeploymentType {
	if jm == nil || jm.DeploymentType == "" {
		return JobManagerDeploymentTypeJob
	}
	return jm.DeploymentType
}

func (tm *TaskManagerSpec) GetResources() *corev1.ResourceList {
	return util.UpperBoundedResourceList(tm.Resources)
}

func (fc *FlinkCluster) IsHighAvailabilityEnabled() bool {
	if fc.Spec.FlinkProperties == nil {
		return false
	}
	v, ok := fc.Spec.FlinkProperties[haConfigTypeNew]
	if !ok {
		v, ok = fc.Spec.FlinkProperties[haConfigType]
	}
	if !ok || strings.ToLower(v) == "none" {
		return false
	}
	v, ok = fc.Spec.FlinkProperties[haConfigClusterId]
	if !ok || strings.TrimSpace(v) == "" {
		return false
	}
	v, ok = fc.Spec.FlinkProperties[haConfigStorageDir]
	if !ok || strings.TrimSpace(v) == "" {
		return false
	}
	return true
}

func (fc *FlinkCluster) GetKubernetesClusterID() string {
	if !fc.IsHighAvailabilityEnabled() {
		return ""
	}
	return fc.Spec.FlinkProperties[haConfigClusterId]
}

func (fc *FlinkCluster) GetHAConfigMapName() string {
	id := fc.GetKubernetesClusterID()
	if id == "" {
		return ""
	}
	return fmt.Sprintf("%s-cluster-config-map", id)
}
