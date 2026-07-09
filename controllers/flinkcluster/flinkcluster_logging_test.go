package flinkcluster

import (
	"encoding/json"
	"errors"
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

func TestLogClusterStatusSummaryDoesNotIncludeFullObjects(t *testing.T) {
	bigValue := strings.Repeat("x", 20_000)
	status := &v1beta1.FlinkClusterStatus{
		State: v1beta1.ClusterStateRunning,
		Components: v1beta1.FlinkClusterComponentsStatus{
			ConfigMap: &v1beta1.ConfigMapStatus{
				Name:  "test-config",
				State: v1beta1.ComponentStateReady,
			},
			JobManager: &v1beta1.JobManagerStatus{
				Name:          "test-jm",
				State:         v1beta1.ComponentStateReady,
				Replicas:      1,
				ReadyReplicas: 1,
				Ready:         "true",
			},
			JobManagerService: v1beta1.JobManagerServiceStatus{
				Name:                "test-jm-service",
				State:               v1beta1.ComponentStateReady,
				NodePort:            30081,
				LoadBalancerIngress: []corev1.LoadBalancerIngress{{Hostname: bigValue}},
			},
			JobManagerIngress: &v1beta1.JobManagerIngressStatus{
				Name:  "test-ingress",
				State: v1beta1.ComponentStateReady,
				URLs:  []string{"https://jm.example.com", bigValue},
			},
			TaskManager: &v1beta1.TaskManagerStatus{
				Name:          "test-tm",
				State:         v1beta1.ComponentStateReady,
				Replicas:      4,
				ReadyReplicas: 3,
				Ready:         "3/4",
				Selector:      bigValue,
			},
			Job: &v1beta1.JobStatus{
				ID:                  "flink-job-id",
				Name:                "test-job",
				SubmitterName:       "submitter",
				SubmitterExitCode:   0,
				State:               v1beta1.JobStateRunning,
				FromSavepoint:       bigValue,
				SavepointGeneration: 2,
				SavepointLocation:   bigValue,
				FinalSavepoint:      true,
				RestartCount:        1,
				FailureReasons:      []string{bigValue, "short reason"},
			},
		},
		Control: &v1beta1.FlinkClusterControlStatus{
			Name:       v1beta1.ControlNameSavepoint,
			Details:    map[string]string{"large": bigValue, "small": "value"},
			State:      v1beta1.ControlStateRequested,
			Message:    bigValue,
			UpdateTime: "2026-07-08T00:00:00Z",
		},
		Savepoint: &v1beta1.SavepointStatus{
			JobID:         "flink-job-id",
			TriggerID:     "trigger-id",
			TriggerTime:   "2026-07-08T00:00:01Z",
			TriggerReason: v1beta1.SavepointReasonUserRequested,
			UpdateTime:    "2026-07-08T00:00:02Z",
			State:         v1beta1.SavepointStateSucceeded,
			Message:       bigValue,
		},
		Revision: v1beta1.RevisionStatus{
			CurrentRevision: "current-revision",
			NextRevision:    "next-revision",
		},
	}

	summary := mustMap(t, logClusterStatusSummary(status))
	assertEqual(t, summary["state"], v1beta1.ClusterStateRunning)
	assertEqual(t, mustMap(t, summary["configMap"])["name"], "test-config")
	assertEqual(t, mustMap(t, summary["jobManager"])["readyReplicas"], int32(1))
	assertEqual(t, mustMap(t, summary["jobManagerService"])["name"], "test-jm-service")
	assertEqual(t, mustMap(t, summary["jobManagerIngress"])["urlCount"], 2)
	assertEqual(t, mustMap(t, summary["taskManager"])["replicas"], int32(4))
	assertEqual(t, mustMap(t, summary["job"])["failureReasonCount"], 2)
	assertEqual(t, mustMap(t, summary["job"])["hasSavepointLocation"], true)
	assertEqual(t, mustMap(t, summary["job"])["hasFromSavepoint"], true)
	assertEqual(t, mustMap(t, summary["control"])["detailCount"], 2)
	assertEqual(t, mustMap(t, summary["control"])["hasMessage"], true)
	assertEqual(t, mustMap(t, summary["savepoint"])["reason"], v1beta1.SavepointReasonUserRequested)
	assertEqual(t, mustMap(t, summary["savepoint"])["hasMessage"], true)
	assertEqual(t, mustMap(t, summary["revision"])["currentRevision"], "current-revision")

	payload := mustMarshalLogSummary(t, summary)
	payloadString := string(payload)
	if strings.Contains(payloadString, bigValue[:100]) {
		t.Fatalf("status summary leaked full object content")
	}
	if len(payload) > 5_000 {
		t.Fatalf("status summary is unexpectedly large: %d bytes", len(payload))
	}
}

func TestLogClusterStatusSummaryHandlesNilFields(t *testing.T) {
	status := &v1beta1.FlinkClusterStatus{
		State: v1beta1.ClusterStateCreating,
	}

	summary := mustMap(t, logClusterStatusSummary(status))
	assertEqual(t, summary["configMap"], logNilValue)
	assertEqual(t, summary["jobManager"], logNilValue)
	assertEqual(t, mustMap(t, summary["jobManagerService"])["name"], "")
	assertEqual(t, summary["jobManagerIngress"], logNilValue)
	assertEqual(t, summary["taskManager"], logNilValue)
	assertEqual(t, summary["job"], logNilValue)
	assertEqual(t, summary["control"], logNilValue)
	assertEqual(t, summary["savepoint"], logNilValue)
}

func TestLogFlinkSavepointSummaryBranches(t *testing.T) {
	err := errors.New("savepoint failed")
	status := &flink.SavepointStatus{
		JobID:     "flink-job-id",
		TriggerID: "trigger-id",
		Completed: true,
		Location:  "s3://bucket/savepoint",
	}

	tests := []struct {
		name   string
		status *flink.SavepointStatus
		err    error
		want   map[string]any
	}{
		{
			name:   "neither status nor error",
			status: nil,
			err:    nil,
			want:   nil,
		},
		{
			name:   "status only",
			status: status,
			err:    nil,
			want: map[string]any{
				"jobID":       "flink-job-id",
				"triggerID":   "trigger-id",
				"completed":   true,
				"hasLocation": true,
			},
		},
		{
			name:   "error only",
			status: nil,
			err:    err,
			want: map[string]any{
				"error": "savepoint failed",
			},
		},
		{
			name:   "status and error",
			status: status,
			err:    err,
			want: map[string]any{
				"jobID":       "flink-job-id",
				"triggerID":   "trigger-id",
				"completed":   true,
				"hasLocation": true,
				"error":       "savepoint failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logFlinkSavepointSummary(tt.status, tt.err)
			if tt.want == nil {
				assertEqual(t, got, logNilValue)
				return
			}

			gotMap := mustMap(t, got)
			for key, value := range tt.want {
				assertEqual(t, gotMap[key], value)
			}
			if len(gotMap) != len(tt.want) {
				t.Fatalf("unexpected summary keys: got %#v, want %#v", gotMap, tt.want)
			}
		})
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

func mustMap(t *testing.T, summary any) map[string]any {
	t.Helper()
	summaryMap, ok := summary.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T (%#v)", summary, summary)
	}
	return summaryMap
}

func assertEqual(t *testing.T, got any, want any) {
	t.Helper()
	if got != want {
		t.Fatalf("got %#v (%T), want %#v (%T)", got, got, want, want)
	}
}
