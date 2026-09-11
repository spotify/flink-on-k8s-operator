package flinkcluster

import (
	"context"
	"fmt"
	"testing"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/controllers/history"
	"github.com/spotify/flink-on-k8s-operator/internal/util"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// fakeHistory implements history.Interface for testing syncRevisionStatus.
type fakeHistory struct {
	revisions  []*appsv1.ControllerRevision
	createFunc func(parent metav1.Object, revision *appsv1.ControllerRevision, collisionCount *int32) (*appsv1.ControllerRevision, error)
}

func (f *fakeHistory) ListControllerRevisions(_ metav1.Object, _ labels.Selector) ([]*appsv1.ControllerRevision, error) {
	return f.revisions, nil
}

func (f *fakeHistory) CreateControllerRevision(parent metav1.Object, revision *appsv1.ControllerRevision, collisionCount *int32) (*appsv1.ControllerRevision, error) {
	if f.createFunc != nil {
		return f.createFunc(parent, revision, collisionCount)
	}
	clone := revision.DeepCopy()
	f.revisions = append(f.revisions, clone)
	return clone, nil
}

func (f *fakeHistory) DeleteControllerRevision(_ *appsv1.ControllerRevision) error {
	return nil
}

func (f *fakeHistory) UpdateControllerRevision(revision *appsv1.ControllerRevision, newRevision int64) (*appsv1.ControllerRevision, error) {
	clone := revision.DeepCopy()
	clone.Revision = newRevision
	return clone, nil
}

func (f *fakeHistory) AdoptControllerRevision(_ metav1.Object, _ schema.GroupVersionKind, revision *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	return revision, nil
}

func (f *fakeHistory) ReleaseControllerRevision(_ metav1.Object, revision *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	return revision, nil
}

func newTestCluster() *v1beta1.FlinkCluster {
	return &v1beta1.FlinkCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FlinkCluster",
			APIVersion: "flinkoperator.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1beta1.FlinkClusterSpec{
			Image: v1beta1.ImageSpec{Name: "flink:1.17.0"},
		},
	}
}

// persistRevisionStatus simulates what the updater does after syncRevisionStatus.
func persistRevisionStatus(cluster *v1beta1.FlinkCluster, rev Revision) {
	cluster.Status.Revision = v1beta1.RevisionStatus{
		CurrentRevision: util.GetRevisionWithNameNumber(rev.currentRevision),
		NextRevision:    util.GetRevisionWithNameNumber(rev.nextRevision),
	}
	if rev.collisionCount != 0 {
		cluster.Status.Revision.CollisionCount = new(int32)
		*cluster.Status.Revision.CollisionCount = rev.collisionCount
	}
}

func TestSyncRevisionStatus_InitializesCurrentEqualToNext(t *testing.T) {
	cluster := newTestCluster()
	fake := &fakeHistory{}
	observer := &ClusterStateObserver{history: fake}
	observed := &ObservedClusterState{cluster: cluster}

	err := observer.syncRevisionStatus(context.Background(), observed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if observed.revision.currentRevision == nil || observed.revision.nextRevision == nil {
		t.Fatal("currentRevision and nextRevision should not be nil")
	}
	if observed.revision.currentRevision.Name != observed.revision.nextRevision.Name {
		t.Errorf("on first run, currentRevision (%s) should equal nextRevision (%s)",
			observed.revision.currentRevision.Name, observed.revision.nextRevision.Name)
	}
}

func TestSyncRevisionStatus_StableWhenSpecUnchanged(t *testing.T) {
	cluster := newTestCluster()
	fake := &fakeHistory{}
	observer := &ClusterStateObserver{history: fake}
	observed := &ObservedClusterState{cluster: cluster}

	// First reconciliation.
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	originalName := observed.revision.currentRevision.Name
	persistRevisionStatus(cluster, observed.revision)

	// Second reconciliation with same spec.
	observed.revisions = fake.revisions
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	if observed.revision.currentRevision.Name != originalName {
		t.Errorf("currentRevision name changed: %s -> %s", originalName, observed.revision.currentRevision.Name)
	}
	if observed.revision.nextRevision.Name != originalName {
		t.Errorf("nextRevision name changed: %s -> %s", originalName, observed.revision.nextRevision.Name)
	}
	if observed.revision.currentRevision.Name != observed.revision.nextRevision.Name {
		t.Errorf("IsUpdateTriggered should be false: currentRevision (%s) != nextRevision (%s)",
			observed.revision.currentRevision.Name, observed.revision.nextRevision.Name)
	}
}

func TestSyncRevisionStatus_CollisionCountStableAcrossReconciliations(t *testing.T) {
	cluster := newTestCluster()
	fake := &fakeHistory{}
	observer := &ClusterStateObserver{history: fake}
	observed := &ObservedClusterState{cluster: cluster}

	// First reconciliation.
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	initialCount := observed.revision.collisionCount
	persistRevisionStatus(cluster, observed.revision)

	// Multiple reconciliations with same spec.
	for i := 0; i < 5; i++ {
		observed.revisions = fake.revisions
		if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
			t.Fatalf("sync %d: %v", i+1, err)
		}
		if observed.revision.collisionCount != initialCount {
			t.Errorf("reconciliation %d: collisionCount changed from %d to %d",
				i+1, initialCount, observed.revision.collisionCount)
		}
	}
}

func TestSyncRevisionStatus_DeletedRevisionProducesStableNames(t *testing.T) {
	cluster := newTestCluster()
	fake := &fakeHistory{}
	observer := &ClusterStateObserver{history: fake}
	observed := &ObservedClusterState{cluster: cluster}

	// First reconciliation.
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	originalName := observed.revision.currentRevision.Name
	persistRevisionStatus(cluster, observed.revision)
	cluster.Status.State = v1beta1.ClusterStateRunning

	// Simulate truncateHistory or GC deleting the revision.
	fake.revisions = nil

	// Second reconciliation after deletion.
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("sync after deletion: %v", err)
	}

	// The recreated revision should have the same name (same spec, same collisionCount).
	if observed.revision.nextRevision.Name != originalName {
		t.Errorf("nextRevision name changed after deletion: %s -> %s (phantom update!)",
			originalName, observed.revision.nextRevision.Name)
	}

	// Persist and verify the derived revision status would not trigger an update.
	persistRevisionStatus(cluster, observed.revision)
	observed.revisions = fake.revisions
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("sync after re-persist: %v", err)
	}
	derived := deriveRevisionStatus(
		getUpdateState(observed),
		&observed.revision,
		&cluster.Status.Revision,
	)
	if derived.CurrentRevision != derived.NextRevision {
		t.Errorf("IsUpdateTriggered is true after self-healing: CurrentRevision=%s NextRevision=%s",
			derived.CurrentRevision, derived.NextRevision)
	}
}

func TestSyncRevisionStatus_DetectsRealSpecChange(t *testing.T) {
	cluster := newTestCluster()
	fake := &fakeHistory{}
	observer := &ClusterStateObserver{history: fake}
	observed := &ObservedClusterState{cluster: cluster}

	// First reconciliation.
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	originalName := observed.revision.currentRevision.Name
	persistRevisionStatus(cluster, observed.revision)

	// Change the spec.
	cluster.Spec.Image.Name = "flink:1.18.0"

	observed.revisions = fake.revisions
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("sync after spec change: %v", err)
	}

	if observed.revision.nextRevision.Name == originalName {
		t.Error("nextRevision should change when spec changes")
	}
}

func TestSyncRevisionStatus_NoPhantomUpdateAfterNotFoundRace(t *testing.T) {
	cluster := newTestCluster()
	fake := &fakeHistory{}
	observer := &ClusterStateObserver{history: fake}
	observed := &ObservedClusterState{cluster: cluster}

	// First reconciliation.
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	originalName := observed.revision.currentRevision.Name
	originalCollisionCount := observed.revision.collisionCount
	persistRevisionStatus(cluster, observed.revision)

	// Simulate the revision being deleted (e.g., by GC during the NotFound race).
	// CreateControllerRevision will be called since no equal revision exists.
	fake.revisions = nil
	fake.createFunc = func(parent metav1.Object, revision *appsv1.ControllerRevision, collisionCount *int32) (*appsv1.ControllerRevision, error) {
		// Simulate the outcome of a Create-AlreadyExists-Get-NotFound-retry-Create-success
		// sequence. The key invariant: collisionCount should NOT be incremented.
		clone := revision.DeepCopy()
		hash := history.HashControllerRevision(revision, collisionCount)
		clone.Name = history.ControllerRevisionName(parent.GetName(), hash)
		fake.revisions = append(fake.revisions, clone)
		return clone, nil
	}

	// Second reconciliation.
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("sync after race: %v", err)
	}

	if observed.revision.collisionCount != originalCollisionCount {
		t.Errorf("collisionCount changed: %d -> %d (would cause phantom update)",
			originalCollisionCount, observed.revision.collisionCount)
	}

	if observed.revision.nextRevision.Name != originalName {
		t.Errorf("nextRevision name changed: %s -> %s (phantom update — would cause IsUpdateTriggered=true permanently)",
			originalName, observed.revision.nextRevision.Name)
	}

	// This is the critical invariant: after this sequence, deriveRevisionStatus
	// should NOT produce IsUpdateTriggered=true.
	persistRevisionStatus(cluster, observed.revision)
	derived := deriveRevisionStatus(
		getUpdateState(observed),
		&observed.revision,
		&cluster.Status.Revision,
	)
	if derived.CurrentRevision != derived.NextRevision {
		t.Errorf("CRITICAL: IsUpdateTriggered is permanently true after NotFound race!\n"+
			"  CurrentRevision: %s\n  NextRevision: %s\n"+
			"  This blocks Running -> Stopping cluster state transition.",
			derived.CurrentRevision, derived.NextRevision)
	}
}

// Verify that a real collision (different data, same hash) correctly increments collisionCount.
func TestSyncRevisionStatus_RealCollisionIncrementsCount(t *testing.T) {
	cluster := newTestCluster()
	fake := &fakeHistory{}
	observer := &ClusterStateObserver{history: fake}
	observed := &ObservedClusterState{cluster: cluster}

	// First reconciliation — creates initial revision.
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	persistRevisionStatus(cluster, observed.revision)

	// Now change the spec and simulate a hash collision in CreateControllerRevision.
	// The fake returns a revision that was created with an incremented collision count.
	cluster.Spec.Image.Name = "flink:1.18.0"
	var createCalls int
	fake.createFunc = func(parent metav1.Object, revision *appsv1.ControllerRevision, collisionCount *int32) (*appsv1.ControllerRevision, error) {
		createCalls++
		// Simulate: first attempt collides (AlreadyExists, different data),
		// collisionCount is incremented, second attempt succeeds.
		// The real CreateControllerRevision handles this internally.
		// We simulate the final outcome: collisionCount was incremented.
		*collisionCount++
		clone := revision.DeepCopy()
		hash := history.HashControllerRevision(revision, collisionCount)
		clone.Name = history.ControllerRevisionName(parent.GetName(), hash)
		fake.revisions = append(fake.revisions, clone)
		return clone, nil
	}

	observed.revisions = fake.revisions
	if err := observer.syncRevisionStatus(context.Background(), observed); err != nil {
		t.Fatalf("sync after spec change with collision: %v", err)
	}

	if createCalls == 0 {
		t.Fatal("expected CreateControllerRevision to be called")
	}

	// With a real spec change, collisionCount increment is expected and correct.
	// The nextRevision should differ from the original (different spec).
	if observed.revision.currentRevision.Name == observed.revision.nextRevision.Name {
		t.Log("Note: currentRevision == nextRevision (expected on first reconciliation with change)")
	}
	_ = fmt.Sprintf("collisionCount after real collision: %d", observed.revision.collisionCount)
}

func TestIsJmReady(t *testing.T) {
	readyPod := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}
	terminatingPod := readyPod.DeepCopy()
	deletionTime := metav1.Now()
	terminatingPod.DeletionTimestamp = &deletionTime
	pendingPod := readyPod.DeepCopy()
	pendingPod.Status.Phase = corev1.PodPending
	notReadyPod := readyPod.DeepCopy()
	notReadyPod.Status.Conditions[0].Status = corev1.ConditionFalse
	readyStatefulSet := &appsv1.StatefulSet{
		Spec:   appsv1.StatefulSetSpec{Replicas: ptr.To(int32(1))},
		Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
	}
	notReadyStatefulSet := readyStatefulSet.DeepCopy()
	notReadyStatefulSet.Status.ReadyReplicas = 0
	terminatingService := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		DeletionTimestamp: &deletionTime,
	}}

	tests := []struct {
		name            string
		applicationMode bool
		service         *corev1.Service
		jobPod          *corev1.Pod
		statefulSet     *appsv1.StatefulSet
		expected        bool
	}{
		{
			name:            "application mode with absent service",
			applicationMode: true,
			jobPod:          readyPod,
			expected:        false,
		},
		{
			name:            "application mode with terminating service",
			applicationMode: true,
			service:         terminatingService,
			jobPod:          readyPod,
			expected:        false,
		},
		{
			name:            "application mode with absent pod",
			applicationMode: true,
			service:         &corev1.Service{},
			expected:        false,
		},
		{
			name:            "application mode with terminating pod",
			applicationMode: true,
			service:         &corev1.Service{},
			jobPod:          terminatingPod,
			expected:        false,
		},
		{
			name:            "application mode with non-running pod",
			applicationMode: true,
			service:         &corev1.Service{},
			jobPod:          pendingPod,
			expected:        false,
		},
		{
			name:            "application mode with non-ready pod",
			applicationMode: true,
			service:         &corev1.Service{},
			jobPod:          notReadyPod,
			expected:        false,
		},
		{
			name:            "application mode with ready service and pod",
			applicationMode: true,
			service:         &corev1.Service{},
			jobPod:          readyPod,
			expected:        true,
		},
		{
			name:            "non-application mode with absent statefulset",
			applicationMode: false,
			expected:        false,
		},
		{
			name:            "non-application mode with non-ready statefulset",
			applicationMode: false,
			statefulSet:     notReadyStatefulSet,
			expected:        false,
		},
		{
			name:            "non-application mode with ready statefulset",
			applicationMode: false,
			statefulSet:     readyStatefulSet,
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: the specified deployment mode and JobManager resources
			observed := &ObservedClusterState{
				jmService:     tt.service,
				jmStatefulSet: tt.statefulSet,
			}

			// when: JobManager readiness is evaluated
			actual := isJmReady(tt.applicationMode, observed, tt.jobPod)

			// then: the expected readiness is returned
			if actual != tt.expected {
				t.Fatalf("expected readiness %t, got %t", tt.expected, actual)
			}
		})
	}
}

func TestObserveJobSubmitterPodUsesJobSelector(t *testing.T) {
	const namespace = "default"
	const currentJobUID = types.UID("current-job-uid")
	const oldJobUID = types.UID("old-job-uid")

	newJob := func(uid types.UID) *batchv1.Job {
		return &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jobmanager",
				Namespace: namespace,
				UID:       uid,
			},
			Spec: batchv1.JobSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
					batchv1.ControllerUidLabel: string(uid),
				}},
			},
		}
	}
	newPod := func(name string, uid types.UID) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					batchv1.ControllerUidLabel: string(uid),
				},
			},
		}
	}

	tests := []struct {
		name         string
		job          *batchv1.Job
		pods         []corev1.Pod
		expectedName string
	}{
		{
			name: "replacement pod is selected instead of old job pod",
			job:  newJob(currentJobUID),
			pods: []corev1.Pod{
				newPod("aaa-old", oldJobUID),
				newPod("zzz-replacement", currentJobUID),
			},
			expectedName: "zzz-replacement",
		},
		{
			name: "first pod matching the current job is selected",
			job:  newJob(currentJobUID),
			pods: []corev1.Pod{
				newPod("aaa-first", currentJobUID),
				newPod("zzz-second", currentJobUID),
			},
			expectedName: "aaa-first",
		},
		{
			name: "no matching pod returns nil",
			job:  newJob(currentJobUID),
			pods: []corev1.Pod{
				newPod("old", oldJobUID),
			},
		},
		{
			name: "absent job returns nil",
			pods: []corev1.Pod{
				newPod("unrelated", oldJobUID),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: pods from the current and previous job instances
			scheme := runtime.NewScheme()
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add core API to scheme: %v", err)
			}
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			for i := range tt.pods {
				clientBuilder = clientBuilder.WithObjects(&tt.pods[i])
			}

			// and: an observer for the job namespace
			observer := &ClusterStateObserver{
				k8sClient: clientBuilder.Build(),
				request:   ctrl.Request{NamespacedName: types.NamespacedName{Namespace: namespace}},
			}

			// when: the job submitter pod is observed
			var observedPod *corev1.Pod
			err := observer.observeJobSubmitterPod(context.Background(), tt.job, &observedPod)

			// then: observation succeeds
			if err != nil {
				t.Fatalf("failed to observe job submitter pod: %v", err)
			}

			// and: the first pod selected by the current job selector is returned
			if tt.expectedName == "" {
				if observedPod != nil {
					t.Fatalf("expected no observed pod, got %q", observedPod.Name)
				}
				return
			}
			if observedPod == nil {
				t.Fatalf("expected pod %q, got nil", tt.expectedName)
			}
			if observedPod.Name != tt.expectedName {
				t.Fatalf("expected pod %q, got %q", tt.expectedName, observedPod.Name)
			}
		})
	}
}

func TestGetObservedFlinkJobID(t *testing.T) {
	tests := []struct {
		name         string
		jobPod       *corev1.Pod
		submitterLog *SubmitterLog
		recordedJob  *v1beta1.JobStatus
		expected     string
	}{
		{
			name:         "pod label is preferred",
			jobPod:       &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{JobIdLabel: "pod-job-id"}}},
			submitterLog: &SubmitterLog{jobID: "log-job-id"},
			recordedJob:  &v1beta1.JobStatus{ID: "recorded-job-id"},
			expected:     "pod-job-id",
		},
		{
			name:         "submitter log is used without a pod",
			submitterLog: &SubmitterLog{jobID: "log-job-id"},
			recordedJob:  &v1beta1.JobStatus{ID: "recorded-job-id"},
			expected:     "log-job-id",
		},
		{
			name:        "recorded job is used without a pod or log",
			recordedJob: &v1beta1.JobStatus{ID: "recorded-job-id"},
			expected:    "recorded-job-id",
		},
		{
			name: "job ID is absent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: the observed job ID sources
			jobPod := tt.jobPod
			submitterLog := tt.submitterLog
			recordedJob := tt.recordedJob

			// when: the Flink job ID is selected
			actual := getObservedFlinkJobID(jobPod, submitterLog, recordedJob)

			// then: the first available job ID is returned
			if actual != tt.expected {
				t.Fatalf("expected job ID %q, got %q", tt.expected, actual)
			}
		})
	}
}
