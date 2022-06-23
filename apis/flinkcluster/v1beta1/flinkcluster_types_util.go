package v1beta1

import (
	"fmt"
	"strings"
	"time"
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

func (j *JobStatus) IsTerminated(spec *JobSpec) bool {
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
	return !hasTimeElapsed(j.SavepointTime, compareTime, stateMaxAge)
}

// ShouldRestart returns true if the controller should restart failed job.
// The controller can restart the job only if there is a savepoint that is close to the end time of the job.
func (j *JobStatus) ShouldRestart(spec *JobSpec) bool {
	if j == nil || !j.IsFailed() || spec == nil {
		return false
	}

	restartEnabled := spec.RestartPolicy != nil && *spec.RestartPolicy == JobRestartPolicyFromSavepointOnFailure

	var jobCompletionTime time.Time
	if j.CompletionTime != nil {
		jobCompletionTime = j.CompletionTime.Time
	}

	return restartEnabled && j.IsSavepointUpToDate(spec, jobCompletionTime)
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

// TimeConverter converts between time.Time and string.
type TimeConverter struct{}

// FromString converts string to time.Time.
func (tc *TimeConverter) FromString(timeStr string) time.Time {
	timestamp, err := time.Parse(
		time.RFC3339, timeStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse time string: %s", timeStr))
	}
	return timestamp
}

// ToString converts time.Time to string.
func (tc *TimeConverter) ToString(timestamp time.Time) string {
	return timestamp.Format(time.RFC3339)
}

// Check time has passed
func hasTimeElapsed(timeToCheckStr string, now time.Time, intervalSec int) bool {
	tc := &TimeConverter{}
	timeToCheck := tc.FromString(timeToCheckStr)
	intervalPassedTime := timeToCheck.Add(time.Duration(int64(intervalSec) * int64(time.Second)))
	return now.After(intervalPassedTime)
}

func isBlank(s *string) bool {
	return s == nil || strings.TrimSpace(*s) == ""
}
