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
	}

	payload, err := json.Marshal(flinkClusterLogSummary(cluster))
	if err != nil {
		t.Fatalf("failed to marshal summary: %v", err)
	}
	payloadString := string(payload)
	if strings.Contains(payloadString, bigValue[:100]) {
		t.Fatalf("summary leaked full object content")
	}
	if len(payload) > 1_000 {
		t.Fatalf("summary is unexpectedly large: %d bytes", len(payload))
	}
	if !strings.Contains(payloadString, "test-cluster") {
		t.Fatalf("summary lost object identity: %s", payloadString)
	}
}
