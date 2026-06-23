package flink

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
)

func TestTriggerSavepointPayload(t *testing.T) {
	tests := []struct {
		name         string
		cancel       bool
		formatType   string
		expectedBody map[string]interface{}
	}{
		{
			name:       "with formatType NATIVE",
			formatType: "NATIVE",
			expectedBody: map[string]interface{}{
				"target-directory": "/savepoints",
				"cancel-job":       false,
				"formatType":       "NATIVE",
			},
		},
		{
			name:       "with formatType CANONICAL",
			formatType: "CANONICAL",
			expectedBody: map[string]interface{}{
				"target-directory": "/savepoints",
				"cancel-job":       false,
				"formatType":       "CANONICAL",
			},
		},
		{
			name:       "without formatType",
			formatType: "",
			expectedBody: map[string]interface{}{
				"target-directory": "/savepoints",
				"cancel-job":       false,
			},
		},
		{
			name:       "cancel with formatType",
			cancel:     true,
			formatType: "NATIVE",
			expectedBody: map[string]interface{}{
				"target-directory": "/savepoints",
				"cancel-job":       true,
				"formatType":       "NATIVE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]interface{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"request-id": "abc123"}`))
			}))
			defer server.Close()

			client := NewClient(logr.Discard(), server.Client())
			triggerID, err := client.TriggerSavepoint(server.URL, "job-1", "/savepoints", tt.cancel, tt.formatType)

			assert.NilError(t, err)
			assert.Equal(t, triggerID.RequestID, "abc123")
			assert.DeepEqual(t, capturedBody, tt.expectedBody)
		})
	}
}
