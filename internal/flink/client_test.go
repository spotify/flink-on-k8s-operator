package flink

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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
				assert.Equal(t, r.Method, http.MethodPost)
				assert.Equal(t, r.URL.Path, "/jobs/job-1/savepoints")
				body, err := io.ReadAll(r.Body)
				assert.NilError(t, err)
				assert.NilError(t, json.Unmarshal(body, &capturedBody))
				w.Header().Set("Content-Type", "application/json")
				_, err = w.Write([]byte(`{"request-id": "abc123"}`))
				assert.NilError(t, err)
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

func TestStopJobWithSavepointPayload(t *testing.T) {
	tests := []struct {
		name         string
		formatType   string
		expectedBody map[string]interface{}
	}{
		{
			name:       "with formatType NATIVE",
			formatType: "NATIVE",
			expectedBody: map[string]interface{}{
				"targetDirectory": "/savepoints",
				"drain":           false,
				"formatType":      "NATIVE",
			},
		},
		{
			name:       "with formatType CANONICAL",
			formatType: "CANONICAL",
			expectedBody: map[string]interface{}{
				"targetDirectory": "/savepoints",
				"drain":           false,
				"formatType":      "CANONICAL",
			},
		},
		{
			name:       "without formatType",
			formatType: "",
			expectedBody: map[string]interface{}{
				"targetDirectory": "/savepoints",
				"drain":           false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]interface{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.URL.Path, "/jobs/job-1/stop")
				body, err := io.ReadAll(r.Body)
				assert.NilError(t, err)
				assert.NilError(t, json.Unmarshal(body, &capturedBody))
				w.Header().Set("Content-Type", "application/json")
				_, err = w.Write([]byte(`{"request-id": "abc123"}`))
				assert.NilError(t, err)
			}))
			defer server.Close()

			client := NewClient(logr.Discard(), server.Client())
			triggerID, err := client.StopJobWithSavepoint(server.URL, "job-1", "/savepoints", tt.formatType)

			assert.NilError(t, err)
			assert.Equal(t, triggerID.RequestID, "abc123")
			assert.DeepEqual(t, capturedBody, tt.expectedBody)
		})
	}
}

func TestErrorResponseReusesConnection(t *testing.T) {
	var mu sync.Mutex
	var opened int

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"errors":["internal error"]}`)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			mu.Lock()
			opened++
			mu.Unlock()
		}
	}
	server.Start()
	defer server.Close()

	// Same shape as NewDefaultClient, with its own transport so the test does
	// not share http.DefaultTransport with the rest of the package.
	client := NewClient(logr.Discard(), &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{},
	})
	for i := 0; i < 5; i++ {
		_, err := client.GetJobsOverview(server.URL)
		assert.ErrorContains(t, err, "500 Internal Server Error")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, opened, 1)
}
