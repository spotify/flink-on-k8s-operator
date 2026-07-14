package v1beta1

import (
	"encoding/json"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFlinkClusterLogSummaryDoesNotIncludeFullObject(t *testing.T) {
	bigValue := strings.Repeat("x", 20_000)
	cluster := &FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"large": bigValue,
			},
		},
		Spec: FlinkClusterSpec{
			FlinkVersion: "1.20.0",
			Image: ImageSpec{
				Name: "flink:1.20.0",
			},
			LogConfig: map[string]string{
				"log4j-console.properties": bigValue,
			},
		},
		Status: FlinkClusterStatus{
			Components: FlinkClusterComponentsStatus{
				Job: &JobStatus{
					ID:             "job-id",
					State:          JobStateRunning,
					FailureReasons: []string{bigValue},
				},
			},
			Control: &FlinkClusterControlStatus{
				Name:    "control",
				Message: bigValue,
			},
			Savepoint: &SavepointStatus{
				JobID:   "job-id",
				Message: bigValue,
			},
			Revision: RevisionStatus{
				CurrentRevision: "revision-1",
				NextRevision:    "revision-2",
			},
		},
	}

	summary := mustLogSummaryMap(t, FlinkClusterLogSummary(cluster))
	if summary["kind"] != "FlinkCluster" {
		t.Fatalf("summary has unexpected kind: %#v", summary["kind"])
	}
	job := mustLogSummaryMap(t, summary["job"])
	if job["id"] != "job-id" || job["state"] != JobStateRunning {
		t.Fatalf("summary has unexpected nested job: %#v", job)
	}
	if mustLogSummaryMap(t, summary["revision"])["currentRevision"] != "revision-1" {
		t.Fatalf("summary lost revision: %#v", summary["revision"])
	}
	if mustLogSummaryMap(t, summary["savepoint"])["jobID"] != "job-id" {
		t.Fatalf("summary lost savepoint: %#v", summary["savepoint"])
	}
	if mustLogSummaryMap(t, summary["control"])["name"] != "control" {
		t.Fatalf("summary lost control: %#v", summary["control"])
	}
	if _, found := summary["jobID"]; found {
		t.Fatalf("summary retained deprecated flat jobID key: %#v", summary)
	}
	if _, found := summary["jobState"]; found {
		t.Fatalf("summary retained deprecated flat jobState key: %#v", summary)
	}

	payload, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal summary: %v", err)
	}
	payloadString := string(payload)
	if strings.Contains(payloadString, bigValue[:100]) {
		t.Fatalf("summary leaked full object content")
	}
	if len(payload) > 2_000 {
		t.Fatalf("summary is unexpectedly large: %d bytes", len(payload))
	}
	if !strings.Contains(payloadString, "test-cluster") {
		t.Fatalf("summary lost object identity: %s", payloadString)
	}
}

func TestFlinkClusterLogSummaryHandlesNilCluster(t *testing.T) {
	if got := FlinkClusterLogSummary(nil); got != logNilValue {
		t.Fatalf("got %#v, want %#v", got, logNilValue)
	}
}

func mustLogSummaryMap(t *testing.T, value any) map[string]any {
	t.Helper()
	summary, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T (%#v)", value, value)
	}
	return summary
}
