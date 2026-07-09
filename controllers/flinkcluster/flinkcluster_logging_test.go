package flinkcluster

import (
	"encoding/json"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
)

func TestLogObjectSummaryHandlesTypedNil(t *testing.T) {
	var configMap *corev1.ConfigMap
	var obj client.Object = configMap

	if got := logObjectSummary(obj); got != logNilValue {
		t.Fatalf("expected typed nil object to summarize as %q, got %#v", logNilValue, got)
	}
}

func TestLogObservedClusterStateSummaryDoesNotIncludeFullObjects(t *testing.T) {
	bigValue := strings.Repeat("x", 20_000)
	cluster := &v1beta1.FlinkCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "flinkoperator.k8s.io/v1beta1",
			Kind:       "FlinkCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-cluster",
			Namespace:       "test-namespace",
			Generation:      7,
			ResourceVersion: "12345",
			Annotations: map[string]string{
				"large": bigValue,
			},
		},
		Spec: v1beta1.FlinkClusterSpec{
			FlinkVersion: "1.20.0",
			Image: v1beta1.ImageSpec{
				Name: "flink:1.20.0",
			},
			LogConfig: map[string]string{
				"log4j-console.properties": bigValue,
			},
		},
		Status: v1beta1.FlinkClusterStatus{
			State: v1beta1.ClusterStateRunning,
			Components: v1beta1.FlinkClusterComponentsStatus{
				Job: &v1beta1.JobStatus{
					ID:    "flink-job-id",
					State: v1beta1.JobStateRunning,
				},
			},
		},
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"large": bigValue,
		},
	}
	observed := &ObservedClusterState{
		cluster:   cluster,
		configMap: configMap,
		revisions: []*appsv1.ControllerRevision{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "test-cluster-abc-1"},
				Revision:   1,
			},
		},
	}

	payload := mustMarshalLogSummary(t, logObservedClusterStateSummary(observed))
	payloadString := string(payload)
	if strings.Contains(payloadString, bigValue[:100]) {
		t.Fatalf("summary leaked full object content")
	}
	if len(payload) > 5_000 {
		t.Fatalf("summary is unexpectedly large: %d bytes", len(payload))
	}
	if !strings.Contains(payloadString, "test-cluster") {
		t.Fatalf("summary lost object identity: %s", payloadString)
	}
}

func TestLogObservedClusterStateFullIncludesFullObjects(t *testing.T) {
	bigValue := strings.Repeat("x", 20_000)
	cluster := &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"large": bigValue,
			},
		},
		Spec: v1beta1.FlinkClusterSpec{
			LogConfig: map[string]string{
				"log4j-console.properties": bigValue,
			},
		},
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"large": bigValue,
		},
	}
	observed := &ObservedClusterState{
		cluster:   cluster,
		configMap: configMap,
	}

	payload := mustMarshalLogSummary(t, logObservedClusterStateFull(observed))
	payloadString := string(payload)
	if !strings.Contains(payloadString, bigValue[:100]) {
		t.Fatalf("full observed state log did not include full object content")
	}
	if len(payload) < 20_000 {
		t.Fatalf("full observed state log is unexpectedly small: %d bytes", len(payload))
	}
}

func TestLogDesiredClusterStateSummaryDoesNotIncludeFullObjects(t *testing.T) {
	bigValue := strings.Repeat("x", 20_000)
	desired := &model.DesiredClusterState{
		ConfigMap: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				"large": bigValue,
			},
		},
	}

	payload := mustMarshalLogSummary(t, logDesiredClusterStateSummary(desired))
	payloadString := string(payload)
	if strings.Contains(payloadString, bigValue[:100]) {
		t.Fatalf("summary leaked full desired object content")
	}
	if len(payload) > 2_000 {
		t.Fatalf("summary is unexpectedly large: %d bytes", len(payload))
	}
	if !strings.Contains(payloadString, "test-config") {
		t.Fatalf("summary lost desired object identity: %s", payloadString)
	}
}

func TestLogDesiredClusterStateFullIncludesFullObjects(t *testing.T) {
	bigValue := strings.Repeat("x", 20_000)
	desired := &model.DesiredClusterState{
		ConfigMap: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: "test-namespace",
			},
			Data: map[string]string{
				"large": bigValue,
			},
		},
	}

	payload := mustMarshalLogSummary(t, logDesiredClusterStateFull(desired))
	payloadString := string(payload)
	if !strings.Contains(payloadString, bigValue[:100]) {
		t.Fatalf("full desired state log did not include full object content")
	}
	if len(payload) < 20_000 {
		t.Fatalf("full desired state log is unexpectedly small: %d bytes", len(payload))
	}
}

func mustMarshalLogSummary(t *testing.T, summary any) []byte {
	t.Helper()
	payload, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("failed to marshal summary: %v", err)
	}
	return payload
}
