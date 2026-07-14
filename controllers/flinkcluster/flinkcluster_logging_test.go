package flinkcluster

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/flink"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
)

func TestLogObjectSummaryHandlesTypedNil(t *testing.T) {
	var configMap *corev1.ConfigMap
	var obj client.Object = configMap

	if got := logObjectSummary(obj); got != logNilValue {
		t.Fatalf("expected typed nil object to summarize as %q, got %#v", logNilValue, got)
	}
}

func TestIsNilClientObjectHandlesNonPointerKinds(t *testing.T) {
	var nilChan chan struct{}
	var nilFunc func()
	var nilMap map[string]string
	var nilSlice []string

	for name, value := range map[string]any{
		"chan":  nilChan,
		"func":  nilFunc,
		"map":   nilMap,
		"slice": nilSlice,
	} {
		t.Run(name+" nil", func(t *testing.T) {
			if !isNilClientObject(value) {
				t.Fatalf("expected nil %s to be detected", name)
			}
		})
	}

	for name, value := range map[string]any{
		"chan":    make(chan struct{}),
		"func":    func() {},
		"map":     map[string]string{},
		"slice":   []string{},
		"default": 1,
	} {
		t.Run(name+" non-nil", func(t *testing.T) {
			if isNilClientObject(value) {
				t.Fatalf("expected non-nil %s not to be detected as nil", name)
			}
		})
	}

	// reflect.ValueOf unwraps an interface to its dynamic value, so exercise the
	// interface branch with an addressable interface value directly.
	var nilInterface any
	if !isNilReflectValue(reflect.ValueOf(&nilInterface).Elem()) {
		t.Fatal("expected nil interface to be detected")
	}
	nonNilInterface := any("value")
	if isNilReflectValue(reflect.ValueOf(&nonNilInterface).Elem()) {
		t.Fatal("expected non-nil interface not to be detected as nil")
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
	haConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ha-config",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"large": bigValue,
		},
	}
	observed := &ObservedClusterState{
		cluster:     cluster,
		configMap:   configMap,
		haConfigMap: haConfigMap,
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
	haConfigMapSummary, ok := logObservedClusterStateSummary(observed)["haConfigMap"].(map[string]any)
	if !ok || haConfigMapSummary["name"] != "test-ha-config" {
		t.Fatalf("summary lost HA ConfigMap identity: %#v", haConfigMapSummary)
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
	haConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ha-config"},
		Data:       map[string]string{"ha-marker": bigValue},
	}
	observed := &ObservedClusterState{
		cluster:     cluster,
		configMap:   configMap,
		haConfigMap: haConfigMap,
		flinkJob: FlinkJob{
			status: &flink.Job{Id: "submitted-job-id", Name: "submitted-job-marker"},
			list: &flink.JobsOverview{Jobs: []flink.Job{
				{Id: "listed-job-id", Name: "listed-job-marker"},
			}},
			exceptions: &flink.JobExceptions{Exceptions: []flink.JobException{
				{Exception: "exception-marker", Location: "exception-location"},
			}},
			unexpected: []string{"unexpected-job-marker"},
		},
		flinkJobSubmitter: FlinkJobSubmitter{
			job: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
				Name:        "submitter-job-marker",
				Annotations: map[string]string{"large": bigValue},
			}},
			pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name:        "submitter-pod-marker",
				Annotations: map[string]string{"large": bigValue},
			}},
			log: &SubmitterLog{jobID: "submitter-log-job-id", message: "submitter-log-message-marker"},
		},
	}

	full := logObservedClusterStateFull(observed)
	for _, key := range []string{
		"haConfigMap", "flinkJob", "flinkJobList", "flinkJobExceptions",
		"unexpectedFlinkJobs", "jobSubmitter", "jobSubmitterPod", "jobSubmitterLog",
	} {
		if _, ok := full[key]; !ok {
			t.Errorf("full observed state log is missing %q", key)
		}
	}

	payload := mustMarshalLogSummary(t, full)
	payloadString := string(payload)
	for _, marker := range []string{
		bigValue[:100], "test-ha-config", "submitted-job-marker", "listed-job-marker",
		"exception-marker", "unexpected-job-marker", "submitter-job-marker",
		"submitter-pod-marker", "submitter-log-message-marker",
	} {
		if !strings.Contains(payloadString, marker) {
			t.Errorf("full observed state log did not include marker %q", marker)
		}
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
