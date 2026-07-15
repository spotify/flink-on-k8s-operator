package flinkcluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-logr/logr"
	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/flink"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
	"gotest.tools/v3/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestUpdateJobDeployStatusRetriesConflict(t *testing.T) {
	var scheme = runtime.NewScheme()
	assert.NilError(t, v1beta1.AddToScheme(scheme))

	var cluster = &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default"},
		Status: v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				Job: &v1beta1.JobStatus{
					State:             v1beta1.JobStateUpdating,
					StartTime:         "2026-07-08T14:00:00Z",
					SavepointLocation: "gs://bucket/old-savepoint",
				},
			},
		},
	}
	var fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(cluster).
		WithObjects(cluster).
		Build()

	var updateCalls atomic.Int32
	var interceptedClient = interceptor.NewClient(fakeClient, interceptor.Funcs{
		SubResourceUpdate: func(
			ctx context.Context,
			c client.Client,
			subResourceName string,
			obj client.Object,
			opts ...client.SubResourceUpdateOption,
		) error {
			if subResourceName == "status" && updateCalls.Add(1) == 1 {
				return apierrors.NewConflict(
					schema.GroupResource{Group: v1beta1.GroupVersion.Group, Resource: "flinkclusters"},
					obj.GetName(),
					errors.New("simulated conflict"),
				)
			}
			return c.SubResource(subResourceName).Update(ctx, obj, opts...)
		},
	})

	var desiredJob = &batchv1.Job{Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Args: []string{"standalone-job", "--fromSavepoint", "gs://bucket/new-savepoint"},
		}}},
	}}}
	var reconciler = &ClusterReconciler{
		k8sClient: interceptedClient,
		observed:  ObservedClusterState{cluster: cluster},
		desired:   model.DesiredClusterState{Job: desiredJob},
	}

	assert.NilError(t, reconciler.updateJobDeployStatus(context.Background()))
	assert.Equal(t, updateCalls.Load(), int32(2))

	var updated v1beta1.FlinkCluster
	assert.NilError(t, fakeClient.Get(
		context.Background(),
		types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace},
		&updated,
	))
	assert.Equal(t, updated.Status.Components.Job.State, v1beta1.JobStateUpdating)
	assert.Equal(t, updated.Status.Components.Job.StartTime, "")
	assert.Assert(t, updated.Status.Components.Job.DeployTime != "")
	assert.Equal(t, updated.Status.Components.Job.FromSavepoint, "gs://bucket/new-savepoint")
	assert.Equal(t, updated.Status.Components.Job.SavepointLocation, "gs://bucket/new-savepoint")
}

func TestReconcileJobDeletesSubmitterOlderThanCurrentRevision(t *testing.T) {
	var scheme = runtime.NewScheme()
	assert.NilError(t, v1beta1.AddToScheme(scheme))
	assert.NilError(t, batchv1.AddToScheme(scheme))

	var applicationMode = v1beta1.JobModeApplication
	var cluster = &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default"},
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{Mode: &applicationMode},
		},
		Status: v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				Job: &v1beta1.JobStatus{State: v1beta1.JobStateUpdating, FinalSavepoint: true},
			},
			Revision: v1beta1.RevisionStatus{
				CurrentRevision: "cluster-after-hpa-scale-2",
				NextRevision:    "cluster-update-3",
			},
		},
	}
	var oldSubmitter = &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name:      "cluster-jobmanager",
		Namespace: cluster.Namespace,
		Labels: map[string]string{
			RevisionNameLabel: "cluster-before-hpa-scale",
		},
	}}
	var desiredJob = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oldSubmitter.Name,
			Namespace: oldSubmitter.Namespace,
			Labels: map[string]string{
				RevisionNameLabel: "cluster-update",
			},
		},
		Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Args: []string{"standalone-job"}}}},
		}},
	}
	var fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(cluster).
		WithObjects(cluster, oldSubmitter).
		Build()
	var reconciler = &ClusterReconciler{
		k8sClient: fakeClient,
		observed: ObservedClusterState{
			cluster: cluster,
			flinkJobSubmitter: FlinkJobSubmitter{
				job: oldSubmitter,
			},
		},
		desired: model.DesiredClusterState{Job: desiredJob},
	}

	_, err := reconciler.reconcileJob(context.Background())
	assert.NilError(t, err)

	var submitter batchv1.Job
	err = fakeClient.Get(
		context.Background(),
		types.NamespacedName{Name: oldSubmitter.Name, Namespace: oldSubmitter.Namespace},
		&submitter,
	)
	assert.Assert(t, apierrors.IsNotFound(err))
}

func TestReconcileJobDeletesCompletedSubmitterForDetachedJobRestart(t *testing.T) {
	var scheme = runtime.NewScheme()
	assert.NilError(t, v1beta1.AddToScheme(scheme))
	assert.NilError(t, batchv1.AddToScheme(scheme))

	var detachedMode = v1beta1.JobModeDetached
	var restartPolicy = v1beta1.JobRestartPolicyFromSavepointOnFailure
	var cluster = &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default"},
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{
				Mode:          &detachedMode,
				RestartPolicy: &restartPolicy,
			},
		},
		Status: v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				Job: &v1beta1.JobStatus{State: v1beta1.JobStateFailed},
			},
			Revision: v1beta1.RevisionStatus{
				CurrentRevision: "cluster-current-1",
				NextRevision:    "cluster-current-1",
			},
		},
	}
	var completedSubmitter = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-jobmanager",
			Namespace: cluster.Namespace,
			Labels: map[string]string{
				RevisionNameLabel: "cluster-current",
			},
		},
		Status: batchv1.JobStatus{Succeeded: 1},
	}
	var desiredJob = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      completedSubmitter.Name,
			Namespace: completedSubmitter.Namespace,
			Labels: map[string]string{
				RevisionNameLabel: "cluster-current",
			},
		},
		Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Args: []string{"run"}}}},
		}},
	}
	var fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(cluster).
		WithObjects(cluster, completedSubmitter).
		Build()
	var reconciler = &ClusterReconciler{
		k8sClient: fakeClient,
		observed: ObservedClusterState{
			cluster: cluster,
			flinkJobSubmitter: FlinkJobSubmitter{
				job: completedSubmitter,
			},
		},
		desired: model.DesiredClusterState{Job: desiredJob},
	}

	_, err := reconciler.reconcileJob(context.Background())
	assert.NilError(t, err)

	var submitter batchv1.Job
	err = fakeClient.Get(
		context.Background(),
		types.NamespacedName{Name: completedSubmitter.Name, Namespace: completedSubmitter.Namespace},
		&submitter,
	)
	assert.Assert(t, apierrors.IsNotFound(err))
}

func TestReconcileJobKeepsSubmitterAtNextRevision(t *testing.T) {
	var scheme = runtime.NewScheme()
	assert.NilError(t, v1beta1.AddToScheme(scheme))
	assert.NilError(t, batchv1.AddToScheme(scheme))

	var applicationMode = v1beta1.JobModeApplication
	var cluster = &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default"},
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{Mode: &applicationMode},
		},
		Status: v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				Job: &v1beta1.JobStatus{State: v1beta1.JobStateUpdating},
			},
			Revision: v1beta1.RevisionStatus{
				CurrentRevision: "cluster-current-1",
				NextRevision:    "cluster-next-2",
			},
		},
	}
	var nextRevisionSubmitter = &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name:      "cluster-jobmanager",
		Namespace: cluster.Namespace,
		Labels: map[string]string{
			RevisionNameLabel: "cluster-next",
		},
	}}
	var desiredJob = &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nextRevisionSubmitter.Name,
			Namespace: nextRevisionSubmitter.Namespace,
			Labels: map[string]string{
				RevisionNameLabel: "cluster-next",
			},
		},
		Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Args: []string{"standalone-job"}}}},
		}},
	}
	var fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(cluster).
		WithObjects(cluster, nextRevisionSubmitter).
		Build()
	var reconciler = &ClusterReconciler{
		k8sClient: fakeClient,
		observed: ObservedClusterState{
			cluster: cluster,
			flinkJobSubmitter: FlinkJobSubmitter{
				job: nextRevisionSubmitter,
			},
		},
		desired: model.DesiredClusterState{Job: desiredJob},
	}

	_, err := reconciler.reconcileJob(context.Background())
	assert.NilError(t, err)

	var submitter batchv1.Job
	assert.NilError(t, fakeClient.Get(
		context.Background(),
		types.NamespacedName{Name: nextRevisionSubmitter.Name, Namespace: nextRevisionSubmitter.Namespace},
		&submitter,
	))
}

func TestCancelFlinkJob_StopWithSavepoint_Success(t *testing.T) {
	// given: Flink REST API that completes savepoint after 2 in-progress polls
	var pollCount atomic.Int32
	var stopBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/jobs/job-123/stop":
			body, err := io.ReadAll(r.Body)
			assert.NilError(t, err)
			assert.NilError(t, json.Unmarshal(body, &stopBody))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"request-id": "trigger-abc"}`)

		case r.Method == http.MethodGet && r.URL.Path == "/jobs/job-123/savepoints/trigger-abc":
			w.Header().Set("Content-Type", "application/json")
			n := pollCount.Add(1)
			if n < 3 {
				fmt.Fprint(w, `{"status":{"id":"IN_PROGRESS"}}`)
			} else {
				fmt.Fprint(w, `{"status":{"id":"COMPLETED"},"operation":{"location":"s3://bucket/sp-1"}}`)
			}

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// and: a running Flink 1.15 cluster with native savepoints configured
	savepointsDir := "s3://bucket/savepoints"
	formatType := v1beta1.SavepointFormatTypeNative
	cluster := newTestClusterWithJob(&savepointsDir, nil)
	cluster.Spec.FlinkVersion = "1.15.0"
	cluster.Spec.Job.SavepointFormatType = &formatType
	reconciler := newTestReconciler(cluster, newRedirectingHTTPClient(server.URL))

	// when: cancelFlinkJob is called with takeSavepoint=true
	err := reconciler.cancelFlinkJob(context.Background(), "job-123", true)

	// then: no error is returned
	requireNoError(t, err)

	// and: savepoint status was polled multiple times
	if pollCount.Load() < 3 {
		t.Errorf("expected at least 3 savepoint status polls, got %d", pollCount.Load())
	}

	// and: cluster status reflects a successful savepoint with correct metadata
	sp := requireSavepointStatus(t, reconciler, cluster)
	if sp.State != v1beta1.SavepointStateSucceeded {
		t.Errorf("expected savepoint state %q, got %q", v1beta1.SavepointStateSucceeded, sp.State)
	}
	if sp.TriggerReason != v1beta1.SavepointReasonJobCancel {
		t.Errorf("expected trigger reason %q, got %q", v1beta1.SavepointReasonJobCancel, sp.TriggerReason)
	}
	if sp.TriggerID != "trigger-abc" {
		t.Errorf("expected trigger ID %q, got %q", "trigger-abc", sp.TriggerID)
	}
	if sp.FormatType != v1beta1.SavepointFormatTypeNative {
		t.Errorf("expected format type %q, got %q", v1beta1.SavepointFormatTypeNative, sp.FormatType)
	}
	assert.DeepEqual(t, stopBody, map[string]interface{}{
		"targetDirectory": savepointsDir,
		"drain":           false,
		"formatType":      "NATIVE",
	})

	// and: the job status records the savepoint location as the job's final savepoint
	job := requireJobStatus(t, reconciler, cluster)
	if job.SavepointLocation != "s3://bucket/sp-1" {
		t.Errorf("expected job savepoint location %q, got %q", "s3://bucket/sp-1", job.SavepointLocation)
	}
	if !job.FinalSavepoint {
		t.Error("expected job final savepoint to be true")
	}
}

func TestCancelFlinkJob_StopWithSavepoint_SavepointFails(t *testing.T) {
	// given: Flink REST API that reports a failed savepoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/jobs/job-123/stop":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"request-id": "trigger-abc"}`)

		case r.Method == http.MethodGet && r.URL.Path == "/jobs/job-123/savepoints/trigger-abc":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":{"id":"COMPLETED"},"operation":{"failure-cause":{"class":"java.io.IOException","stack-trace":"java.io.IOException: Savepoint failed\n\tat ..."}}}`)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// and: a running cluster with savepoints configured
	savepointsDir := "s3://bucket/savepoints"
	cluster := newTestClusterWithJob(&savepointsDir, nil)
	reconciler := newTestReconciler(cluster, newRedirectingHTTPClient(server.URL))

	// when: cancelFlinkJob is called with takeSavepoint=true
	err := reconciler.cancelFlinkJob(context.Background(), "job-123", true)

	// then: an error is returned
	requireError(t, err)

	// and: cluster status reflects a failed savepoint
	sp := requireSavepointStatus(t, reconciler, cluster)
	if sp.State != v1beta1.SavepointStateFailed {
		t.Errorf("expected savepoint state %q, got %q", v1beta1.SavepointStateFailed, sp.State)
	}
}

func TestCancelFlinkJob_StopWithSavepoint_TriggerFails(t *testing.T) {
	// given: Flink REST API that returns 500 on stop request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/jobs/job-123/stop":
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// and: a running cluster with savepoints configured
	savepointsDir := "s3://bucket/savepoints"
	cluster := newTestClusterWithJob(&savepointsDir, nil)
	reconciler := newTestReconciler(cluster, newRedirectingHTTPClient(server.URL))

	// when: cancelFlinkJob is called with takeSavepoint=true
	err := reconciler.cancelFlinkJob(context.Background(), "job-123", true)

	// then: an error is returned
	requireError(t, err)
	// and: no savepoint status is written to the cluster status
	assertNoSavepointStatus(t, reconciler, cluster)
}

func TestCancelFlinkJob_WithoutSavepoint(t *testing.T) {
	// given: Flink REST API that accepts cancel requests
	var cancelCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/jobs/job-123":
			cancelCalled.Store(true)
			w.WriteHeader(http.StatusAccepted)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// and: a running cluster with savepoints configured
	savepointsDir := "s3://bucket/savepoints"
	cluster := newTestClusterWithJob(&savepointsDir, nil)
	reconciler := newTestReconciler(cluster, newRedirectingHTTPClient(server.URL))

	// when: cancelFlinkJob is called with takeSavepoint=false
	err := reconciler.cancelFlinkJob(context.Background(), "job-123", false)

	// then: no error is returned
	requireNoError(t, err)
	// and: the job was cancelled without savepoint
	assertCancelCalled(t, &cancelCalled)
	// and: no savepoint status is written to the cluster status
	assertNoSavepointStatus(t, reconciler, cluster)
}

func TestCancelFlinkJob_NoSavepointsDir_FallsBackToCancel(t *testing.T) {
	// given: Flink REST API that accepts cancel requests
	var cancelCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/jobs/job-123":
			cancelCalled.Store(true)
			w.WriteHeader(http.StatusAccepted)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// and: a running cluster without savepointsDir configured
	cluster := newTestClusterWithJob(nil, nil)
	reconciler := newTestReconciler(cluster, newRedirectingHTTPClient(server.URL))

	// when: cancelFlinkJob is called with takeSavepoint=true
	err := reconciler.cancelFlinkJob(context.Background(), "job-123", true)

	// then: no error is returned
	requireNoError(t, err)
	// and: the job was cancelled without savepoint
	assertCancelCalled(t, &cancelCalled)
	// and: no savepoint status is written to the cluster status
	assertNoSavepointStatus(t, reconciler, cluster)
}

func TestCancelFlinkJob_StopWithSavepoint_Timeout(t *testing.T) {
	// given: Flink REST API that always reports savepoint in progress
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/jobs/job-123/stop":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"request-id": "trigger-abc"}`)

		case r.Method == http.MethodGet && r.URL.Path == "/jobs/job-123/savepoints/trigger-abc":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":{"id":"IN_PROGRESS"}}`)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// and: a running cluster with a short checkpointing timeout
	savepointsDir := "s3://bucket/savepoints"
	cluster := newTestClusterWithJob(&savepointsDir, map[string]string{
		"execution.checkpointing.timeout": "500ms",
	})
	reconciler := newTestReconciler(cluster, newRedirectingHTTPClient(server.URL))

	// when: cancelFlinkJob is called with takeSavepoint=true
	err := reconciler.cancelFlinkJob(context.Background(), "job-123", true)

	// then: a timeout error is returned
	requireError(t, err, "timed out")

	// and: cluster status reflects a failed savepoint
	sp := requireSavepointStatus(t, reconciler, cluster)
	if sp.State != v1beta1.SavepointStateFailed {
		t.Errorf("expected savepoint state %q, got %q", v1beta1.SavepointStateFailed, sp.State)
	}
}

// --- Test helpers ---

// redirectTransport rewrites every request to target the httptest server,
// preserving the original path and query.
type redirectTransport struct {
	target    string
	transport http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.target[len("http://"):]
	return t.transport.RoundTrip(req)
}

func newTestReconciler(cluster *v1beta1.FlinkCluster, httpClient *http.Client) *ClusterReconciler {
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&v1beta1.FlinkCluster{}).
		WithObjects(cluster).
		Build()

	return &ClusterReconciler{
		k8sClient:   k8sClient,
		flinkClient: flink.NewClient(logr.Discard(), httpClient),
		observed:    ObservedClusterState{cluster: cluster},
		recorder:    record.NewFakeRecorder(16),
	}
}

func newTestClusterWithJob(savepointsDir *string, flinkProperties map[string]string) *v1beta1.FlinkCluster {
	var uiPort int32 = 8081
	return &v1beta1.FlinkCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FlinkCluster",
			APIVersion: "flinkoperator.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{
				SavepointsDir: savepointsDir,
			},
			JobManager: &v1beta1.JobManagerSpec{
				Ports: v1beta1.JobManagerPorts{UI: &uiPort},
			},
			FlinkProperties: flinkProperties,
		},
		Status: v1beta1.FlinkClusterStatus{
			Components: v1beta1.FlinkClusterComponentsStatus{
				Job: &v1beta1.JobStatus{
					ID:    "job-123",
					State: v1beta1.JobStateRunning,
				},
			},
		},
	}
}

func newRedirectingHTTPClient(serverURL string) *http.Client {
	return &http.Client{Transport: &redirectTransport{target: serverURL, transport: http.DefaultTransport}}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func requireError(t *testing.T, err error, contains ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, s := range contains {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("expected error containing %q, got: %v", s, err)
		}
	}
}

func assertCancelCalled(t *testing.T, called *atomic.Bool) {
	t.Helper()
	if !called.Load() {
		t.Error("expected cancel without savepoint to be called")
	}
}

func requireSavepointStatus(t *testing.T, reconciler *ClusterReconciler, cluster *v1beta1.FlinkCluster) *v1beta1.SavepointStatus {
	t.Helper()
	updated := &v1beta1.FlinkCluster{}
	if err := reconciler.k8sClient.Get(context.Background(),
		types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, updated); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}
	if updated.Status.Savepoint == nil {
		t.Fatal("expected savepoint status to be set")
	}
	return updated.Status.Savepoint
}

func requireJobStatus(t *testing.T, reconciler *ClusterReconciler, cluster *v1beta1.FlinkCluster) *v1beta1.JobStatus {
	t.Helper()
	updated := &v1beta1.FlinkCluster{}
	if err := reconciler.k8sClient.Get(context.Background(),
		types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, updated); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}
	if updated.Status.Components.Job == nil {
		t.Fatal("expected job status to be set")
	}
	return updated.Status.Components.Job
}

func assertNoSavepointStatus(t *testing.T, reconciler *ClusterReconciler, cluster *v1beta1.FlinkCluster) {
	t.Helper()
	updated := &v1beta1.FlinkCluster{}
	if err := reconciler.k8sClient.Get(context.Background(),
		types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, updated); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}
	if updated.Status.Savepoint != nil {
		t.Errorf("expected no savepoint status, got %+v", updated.Status.Savepoint)
	}
}
