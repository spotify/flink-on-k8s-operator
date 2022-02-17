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

package flink

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
)

const (
	savepointStateInProgress = "IN_PROGRESS"
	savepointStateCompleted  = "COMPLETED"
)

// Client - Flink API client.
type Client struct {
	log        logr.Logger
	httpClient *http.Client
}

type responseError struct {
	StatusCode int
	Status     string
}

func (e *responseError) Error() string {
	return e.Status
}

type roundTripper struct {
	Proxied http.RoundTripper
}

func (rt *roundTripper) RoundTrip(req *http.Request) (res *http.Response, e error) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "flink-operator")
	resp, err := rt.Proxied.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &responseError{StatusCode: resp.StatusCode, Status: resp.Status}
	}

	return resp, nil
}

func parseJson(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		err = json.Unmarshal(body, out)
	}
	return err
}

type JobException struct {
	Exception string `json:"exception"`
	Location  string `json:"location"`
}

type JobExceptions struct {
	Exceptions []JobException `json:"all-exceptions"`
}

// Job defines Flink job status.
type Job struct {
	Id        string `json:"jid"`
	State     string `json:"state"`
	Name      string `json:"name"`
	StartTime int64  `json:"start-time"`
	EndTime   int64  `json:"end-time"`
	Duration  int64  `json:"duration"`
}

// JobsOverview defines Flink job overview list.
type JobsOverview struct {
	Jobs []Job
}

type JobByStartTime []Job

func (jst JobByStartTime) Len() int           { return len(jst) }
func (jst JobByStartTime) Swap(i, j int)      { jst[i], jst[j] = jst[j], jst[i] }
func (jst JobByStartTime) Less(i, j int) bool { return jst[i].StartTime > jst[j].StartTime }

// SavepointTriggerID defines trigger ID of an async savepoint operation.
type SavepointTriggerID struct {
	RequestID string `json:"request-id"`
}

// SavepointFailureCause defines the cause of savepoint failure.
type SavepointFailureCause struct {
	ExceptionClass string `json:"class"`
	StackTrace     string `json:"stack-trace"`
}

// SavepointStateID - enum("IN_PROGRESS", "COMPLETED").
type SavepointStateID struct {
	ID string `json:"id"`
}

// SavepointStatus defines savepoint status of a job.
type SavepointStatus struct {
	// Flink job ID.
	JobID string
	// Savepoint operation trigger ID.
	TriggerID string
	// Completed or not.
	Completed bool
	// Savepoint location URI, non-empty when savepoint succeeded.
	Location string
	// Cause of the failure, non-empyt when savepoint failed
	FailureCause SavepointFailureCause
}

func (s *SavepointStatus) IsSuccessful() bool {
	return s.Completed && s.FailureCause.StackTrace == ""
}

func (s *SavepointStatus) IsFailed() bool {
	return s.Completed && s.FailureCause.StackTrace != ""
}

func (c *Client) GetJobsOverview(apiBaseURL string) (*JobsOverview, error) {
	resp, err := c.httpClient.Get(apiBaseURL + "/jobs/overview")
	if err != nil {
		return nil, err
	}

	jobsOverview := &JobsOverview{}
	if err := parseJson(resp, jobsOverview); err != nil {
		return nil, err
	}

	sort.Sort(JobByStartTime(jobsOverview.Jobs))

	return jobsOverview, err
}

// StopJob stops a job.
func (c *Client) StopJob(
	apiBaseURL string, jobID string) error {
	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/jobs/%s?mode=cancel", apiBaseURL, jobID), nil)
	if err != nil {
		return err
	}
	_, err = c.httpClient.Do(req)
	if err != nil {
		return err
	}

	return nil
}

// TriggerSavepoint triggers an async savepoint operation.
func (c *Client) TriggerSavepoint(apiBaseURL string, jobID string, dir string, cancel bool) (*SavepointTriggerID, error) {
	url := fmt.Sprintf("%s/jobs/%s/savepoints", apiBaseURL, jobID)
	jsonStr := fmt.Sprintf(`{
		"target-directory" : "%s",
		"cancel-job" : %v
	}`, dir, cancel)
	resp, err := c.httpClient.Post(url, "application/json", strings.NewReader(jsonStr))
	if err != nil {
		return nil, err
	}

	triggerID := &SavepointTriggerID{}
	err = parseJson(resp, triggerID)
	return triggerID, err
}

// GetSavepointStatus returns savepoint status.
//
// Flink API response examples:
//
// 1) success:
//
// {
//    "status":{"id":"COMPLETED"},
//    "operation":{
//      "location":"file:/tmp/savepoint-ad4025-dd46c1bd1c80"
//    }
// }
//
// 2) failure:
//
// {
//    "status":{"id":"COMPLETED"},
//    "operation":{
//      "failure-cause":{
//        "class": "java.util.concurrent.CompletionException",
//        "stack-trace": "..."
//      }
//    }
// }
func (c *Client) GetSavepointStatus(
	apiBaseURL string, jobID string, triggerID string) (*SavepointStatus, error) {
	var url = fmt.Sprintf("%s/jobs/%s/savepoints/%s", apiBaseURL, jobID, triggerID)
	var status = &SavepointStatus{JobID: jobID, TriggerID: triggerID}
	var rootJSON map[string]*json.RawMessage
	var stateID SavepointStateID
	var opJSON map[string]*json.RawMessage

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	err = parseJson(resp, &rootJSON)
	if err != nil {
		return nil, err
	}

	c.log.Info("Savepoint status json", "json", rootJSON)
	if state, ok := rootJSON["status"]; ok && state != nil {
		err = json.Unmarshal(*state, &stateID)
		if err != nil {
			return nil, err
		}
		if stateID.ID == savepointStateCompleted {
			status.Completed = true
		} else {
			status.Completed = false
		}
	}
	if op, ok := rootJSON["operation"]; ok && op != nil {
		err = json.Unmarshal(*op, &opJSON)
		if err != nil {
			return nil, err
		}
		// Success
		if location, ok := opJSON["location"]; ok && location != nil {
			err = json.Unmarshal(*location, &status.Location)
			if err != nil {
				return nil, err
			}
		}
		// Failure
		if failureCause, ok := opJSON["failure-cause"]; ok && failureCause != nil {
			err = json.Unmarshal(*failureCause, &status.FailureCause)
			if err != nil {
				return nil, err
			}
		}
	}
	return status, err
}

// TakeSavepoint takes savepoint, blocks until it succeeds or fails.
func (c *Client) TakeSavepoint(apiBaseURL string, jobID string, dir string) (*SavepointStatus, error) {
	status := &SavepointStatus{JobID: jobID}

	triggerID, err := c.TriggerSavepoint(apiBaseURL, jobID, dir, false)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 12; i++ {
		status, err = c.GetSavepointStatus(apiBaseURL, jobID, triggerID.RequestID)
		if err == nil && status.Completed {
			return status, nil
		}
		time.Sleep(5 * time.Second)
	}

	return status, err
}

func (c *Client) TakeSavepointAsync(apiBaseURL string, jobID string, dir string) (string, error) {
	triggerID, err := c.TriggerSavepoint(apiBaseURL, jobID, dir, false)
	if err != nil {
		return "", err
	}

	return triggerID.RequestID, err
}

func (c *Client) GetJobExceptions(apiBaseURL string, jobId string) (*JobExceptions, error) {
	url := fmt.Sprintf("%s/jobs/%s/exceptions", apiBaseURL, jobId)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	exp := &JobExceptions{}
	if err := parseJson(resp, exp); err != nil {
		return nil, err
	}

	return exp, nil
}

func NewDefaultClient(log logr.Logger) *Client {
	return NewClient(log, &http.Client{})
}

func NewClient(log logr.Logger, httpClient *http.Client) *Client {
	if httpClient.Transport == nil {
		httpClient.Transport = http.DefaultTransport
	}
	httpClient.Transport = &roundTripper{Proxied: httpClient.Transport}

	return &Client{log: log, httpClient: httpClient}
}
