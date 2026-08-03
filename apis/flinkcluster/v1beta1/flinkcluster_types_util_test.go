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

package v1beta1

import (
	"testing"
	"time"

	"github.com/spotify/flink-on-k8s-operator/internal/util"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsSavepointUpToDate(t *testing.T) {
	var tc = &util.TimeConverter{}
	var savepointTime = time.Now()
	var jobCompletionTime = savepointTime.Add(time.Second * 100)
	var maxStateAgeToRestoreSeconds = int32(300)

	// When maxStateAgeToRestoreSeconds is not provided
	var jobSpec = JobSpec{
		MaxStateAgeToRestoreSeconds: nil,
	}
	var jobStatus = JobStatus{
		SavepointTime:     tc.ToString(savepointTime),
		SavepointLocation: "gs://my-bucket/savepoint-123",
	}
	var update = jobStatus.IsSavepointUpToDate(&jobSpec, jobCompletionTime)
	assert.Equal(t, update, false)

	// Old savepoint
	savepointTime = time.Now()
	jobCompletionTime = savepointTime.Add(time.Second * 500)
	jobSpec = JobSpec{
		MaxStateAgeToRestoreSeconds: &maxStateAgeToRestoreSeconds,
	}
	jobStatus = JobStatus{
		SavepointTime:     tc.ToString(savepointTime),
		SavepointLocation: "gs://my-bucket/savepoint-123",
		CompletionTime:    &metav1.Time{Time: jobCompletionTime},
	}
	update = jobStatus.IsSavepointUpToDate(&jobSpec, jobCompletionTime)
	assert.Equal(t, update, false)

	// Fails without savepointLocation
	savepointTime = time.Now()
	jobCompletionTime = savepointTime.Add(time.Second * 100)
	jobSpec = JobSpec{
		MaxStateAgeToRestoreSeconds: &maxStateAgeToRestoreSeconds,
	}
	jobStatus = JobStatus{
		SavepointTime:  tc.ToString(savepointTime),
		CompletionTime: &metav1.Time{Time: jobCompletionTime},
	}
	update = jobStatus.IsSavepointUpToDate(&jobSpec, jobCompletionTime)
	assert.Equal(t, update, false)

	// Up-to-date savepoint
	jobCompletionTime = savepointTime.Add(time.Second * 100)
	jobSpec = JobSpec{
		MaxStateAgeToRestoreSeconds: &maxStateAgeToRestoreSeconds,
	}
	jobStatus = JobStatus{
		SavepointTime:     tc.ToString(savepointTime),
		SavepointLocation: "gs://my-bucket/savepoint-123",
	}
	update = jobStatus.IsSavepointUpToDate(&jobSpec, jobCompletionTime)
	assert.Equal(t, update, true)

	// A savepoint of the final job state.
	jobSpec = JobSpec{
		MaxStateAgeToRestoreSeconds: &maxStateAgeToRestoreSeconds,
	}
	jobStatus = JobStatus{
		FinalSavepoint: true,
	}
	update = jobStatus.IsSavepointUpToDate(&jobSpec, time.Time{})
	assert.Equal(t, update, true)
}

func TestShouldRestartJob(t *testing.T) {
	var tc = &util.TimeConverter{}
	var restartOnFailure = JobRestartPolicyFromSavepointOnFailure
	var neverRestart = JobRestartPolicyNever
	var maxStateAgeToRestoreSeconds = int32(300) // 5 min

	// Restart with savepoint up to date
	var savepointTime = time.Now()
	var jobCompletionTime = savepointTime.Add(time.Second * 60) // savepointTime + 1 min
	var jobSpec = JobSpec{
		RestartPolicy:               &restartOnFailure,
		MaxStateAgeToRestoreSeconds: &maxStateAgeToRestoreSeconds,
	}
	var jobStatus = JobStatus{
		State:             JobStateFailed,
		SavepointLocation: "gs://my-bucket/savepoint-123",
		SavepointTime:     tc.ToString(savepointTime),
		CompletionTime:    &metav1.Time{Time: jobCompletionTime},
	}
	var restart = jobStatus.ShouldRestart(&jobSpec)
	assert.Equal(t, restart, true)

	// Not restart without savepoint
	jobSpec = JobSpec{
		RestartPolicy:               &restartOnFailure,
		MaxStateAgeToRestoreSeconds: &maxStateAgeToRestoreSeconds,
	}
	jobStatus = JobStatus{
		State:          JobStateFailed,
		CompletionTime: &metav1.Time{Time: jobCompletionTime},
	}
	restart = jobStatus.ShouldRestart(&jobSpec)
	assert.Equal(t, restart, true)

	// Not restart with restartPolicy Never
	jobSpec = JobSpec{
		RestartPolicy:               &neverRestart,
		MaxStateAgeToRestoreSeconds: &maxStateAgeToRestoreSeconds,
	}
	jobStatus = JobStatus{
		State:             JobStateFailed,
		SavepointLocation: "gs://my-bucket/savepoint-123",
		SavepointTime:     tc.ToString(savepointTime),
		CompletionTime:    &metav1.Time{Time: jobCompletionTime},
	}
	restart = jobStatus.ShouldRestart(&jobSpec)
	assert.Equal(t, restart, false)

	// Not restart with old savepoint
	jobCompletionTime = savepointTime.Add(time.Second * 300) // savepointTime + 5 min
	jobSpec = JobSpec{
		RestartPolicy:               &neverRestart,
		MaxStateAgeToRestoreSeconds: &maxStateAgeToRestoreSeconds,
	}
	jobStatus = JobStatus{
		State:             JobStateFailed,
		SavepointLocation: "gs://my-bucket/savepoint-123",
		SavepointTime:     tc.ToString(savepointTime),
		CompletionTime:    &metav1.Time{Time: jobCompletionTime},
	}
	restart = jobStatus.ShouldRestart(&jobSpec)
	assert.Equal(t, restart, false)
}

func TestIsHighAvailabilityEnabled(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]string
		want       bool
	}{
		{
			name:       "nil properties",
			properties: nil,
			want:       false,
		},
		{
			name:       "empty properties",
			properties: map[string]string{},
			want:       false,
		},
		{
			name: "deprecated high-availability property",
			properties: map[string]string{
				"high-availability":            "kubernetes",
				"kubernetes.cluster-id":        "my-cluster",
				"high-availability.storageDir": "s3://bucket/ha",
			},
			want: true,
		},
		{
			name: "new high-availability.type property",
			properties: map[string]string{
				"high-availability.type":       "kubernetes",
				"kubernetes.cluster-id":        "my-cluster",
				"high-availability.storageDir": "s3://bucket/ha",
			},
			want: true,
		},
		{
			name: "high-availability set to none",
			properties: map[string]string{
				"high-availability":            "NONE",
				"kubernetes.cluster-id":        "my-cluster",
				"high-availability.storageDir": "s3://bucket/ha",
			},
			want: false,
		},
		{
			name: "new property takes precedence",
			properties: map[string]string{
				"high-availability":            "none",
				"high-availability.type":       "kubernetes",
				"kubernetes.cluster-id":        "my-cluster",
				"high-availability.storageDir": "s3://bucket/ha",
			},
			want: true,
		},
		{
			name: "missing cluster-id",
			properties: map[string]string{
				"high-availability":            "kubernetes",
				"high-availability.storageDir": "s3://bucket/ha",
			},
			want: false,
		},
		{
			name: "missing storageDir",
			properties: map[string]string{
				"high-availability":     "kubernetes",
				"kubernetes.cluster-id": "my-cluster",
			},
			want: false,
		},
		{
			name: "missing high-availability",
			properties: map[string]string{
				"kubernetes.cluster-id":        "my-cluster",
				"high-availability.storageDir": "s3://bucket/ha",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FlinkCluster{Spec: FlinkClusterSpec{FlinkProperties: tt.properties}}
			assert.Equal(t, fc.IsHighAvailabilityEnabled(), tt.want)
		})
	}
}

func TestGetKubernetesClusterID(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]string
		want       string
	}{
		{
			name:       "returns cluster-id when HA enabled",
			properties: map[string]string{"high-availability": "kubernetes", "kubernetes.cluster-id": "my-id", "high-availability.storageDir": "s3://bucket"},
			want:       "my-id",
		},
		{
			name:       "returns cluster-id with new HA property",
			properties: map[string]string{"high-availability.type": "kubernetes", "kubernetes.cluster-id": "my-id", "high-availability.storageDir": "s3://bucket"},
			want:       "my-id",
		},
		{
			name:       "returns empty when HA disabled",
			properties: nil,
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FlinkCluster{Spec: FlinkClusterSpec{FlinkProperties: tt.properties}}
			assert.Equal(t, fc.GetKubernetesClusterID(), tt.want)
		})
	}
}
