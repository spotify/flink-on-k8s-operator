/*
Copyright 2026 Spotify AB.

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

package flinkcluster

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/flink"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
)

func TestDeleteComponent(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core types to scheme: %v", err)
	}

	t.Run("successful delete returns nil", func(t *testing.T) {
		configMap := testDeleteConfigMap()
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(configMap).
			Build()
		reconciler := ClusterReconciler{k8sClient: fakeClient}

		if err := reconciler.deleteComponent(ctx, configMap, "ConfigMap"); err != nil {
			t.Fatalf("deleteComponent returned error: %v", err)
		}
	})

	t.Run("not found returns nil", func(t *testing.T) {
		configMap := testDeleteConfigMap()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		reconciler := ClusterReconciler{k8sClient: fakeClient}

		if err := reconciler.deleteComponent(ctx, configMap, "ConfigMap"); err != nil {
			t.Fatalf("deleteComponent returned error for an absent object: %v", err)
		}
	})

	t.Run("non-NotFound error is returned", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
					return deleteErr
				},
			}).
			Build()
		reconciler := ClusterReconciler{k8sClient: fakeClient}

		err := reconciler.deleteComponent(ctx, testDeleteConfigMap(), "ConfigMap")
		if !errors.Is(err, deleteErr) {
			t.Fatalf("deleteComponent returned %v, want %v", err, deleteErr)
		}
	})
}

func TestEnsureFinalizer(t *testing.T) {
	ctx := context.Background()
	scheme := testHADeleteScheme(t)

	t.Run("adds finalizer to HA-enabled cluster", func(t *testing.T) {
		// given: an HA-enabled cluster (not being deleted) without our finalizer yet.
		cluster := testHACluster(false)
		cluster.DeletionTimestamp = nil
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed:  ObservedClusterState{cluster: cluster},
		}

		// when
		added, err := reconciler.ensureFinalizer(ctx)

		// then: the finalizer is now present on the persisted cluster.
		assert.NilError(t, err)
		assert.Assert(t, added)
		var updated v1beta1.FlinkCluster
		assert.NilError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &updated))
		assert.Assert(t, controllerutil.ContainsFinalizer(&updated, jobManagerShutdownFinalizer))
	})

	t.Run("is a no-op when already present", func(t *testing.T) {
		// given: the finalizer is already present.
		cluster := testHACluster(true)
		cluster.DeletionTimestamp = nil
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
					t.Fatal("Update should not be called when the finalizer is already present")
					return nil
				},
			}).
			WithObjects(cluster).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed:  ObservedClusterState{cluster: cluster},
		}

		// expect: succeeds without ever calling Update (enforced by the interceptor above).
		added, err := reconciler.ensureFinalizer(ctx)
		assert.NilError(t, err)
		assert.Assert(t, !added)
	})

	t.Run("is a no-op when deletion has started", func(t *testing.T) {
		// given: a deleting HA-enabled cluster whose finalizer was manually removed
		cluster := testHACluster(false)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(context.Context, client.WithWatch, client.Object, client.Patch, ...client.PatchOption) error {
					t.Fatal("Patch should not be called after deletion has started")
					return nil
				},
			}).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed:  ObservedClusterState{cluster: cluster},
		}

		// when: the finalizer is ensured
		added, err := reconciler.ensureFinalizer(ctx)

		// then: reconciliation succeeds without restoring the finalizer
		assert.NilError(t, err)
		assert.Assert(t, !added)
		assert.Assert(t, !controllerutil.ContainsFinalizer(cluster, jobManagerShutdownFinalizer))
	})
}

func TestReconcileDeletion(t *testing.T) {
	ctx := context.Background()
	scheme := testHADeleteScheme(t)

	t.Run("is a no-op when finalizer is not present", func(t *testing.T) {
		// given: a non-HA cluster being deleted
		cluster := testNonHACluster()
		submitterJob := testSubmitterJob(cluster.Namespace)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, submitterJob).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster:           cluster,
				flinkJobSubmitter: FlinkJobSubmitter{job: submitterJob},
			},
		}

		// when
		result, err := reconciler.reconcileDeletion(ctx)

		// then: no requeue and nothing touched - reconcileDeletion takes no action at all.
		assert.NilError(t, err)
		assert.Equal(t, result, ctrl.Result{})
		assert.NilError(t, fakeClient.Get(ctx, types.NamespacedName{Name: submitterJob.Name, Namespace: submitterJob.Namespace}, &batchv1.Job{}))
	})

	t.Run("application mode: deletes job submitter and requeues without removing the finalizer while the JM pod is still running", func(t *testing.T) {
		// given: an Application-mode HA cluster whose submitter Job's pod is still running.
		cluster := testHACluster(true)
		submitterJob := testSubmitterJob(cluster.Namespace)
		submitterPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ha-cluster-jobmanager-abcde",
				Namespace: cluster.Namespace,
				Labels:    map[string]string{"cluster": cluster.Name, "component": "jobmanager"},
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, submitterJob, submitterPod).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster:           cluster,
				flinkJobSubmitter: FlinkJobSubmitter{job: submitterJob},
			},
		}

		// when
		result, err := reconciler.reconcileDeletion(ctx)

		// then: requeued
		assert.NilError(t, err)
		assert.Equal(t, result, requeueResult)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: submitterJob.Name, Namespace: submitterJob.Namespace}, &batchv1.Job{})
		assert.Assert(t, apierrors.IsNotFound(err))

		// The finalizer stays while the JM pod is still running.
		var updated v1beta1.FlinkCluster
		assert.NilError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &updated))
		assert.Assert(t, controllerutil.ContainsFinalizer(&updated, jobManagerShutdownFinalizer))
	})

	t.Run("application mode: deletes job submitter and removes finalizer once JM pod is gone", func(t *testing.T) {
		// given: an Application-mode HA cluster being deleted, with a submitter Job but no
		// running submitter pod (JM already terminated), and a still-present HA ConfigMap.
		cluster := testHACluster(true)
		submitterJob := testSubmitterJob(cluster.Namespace)
		haConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.GetHAConfigMapName(),
				Namespace: cluster.Namespace,
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, submitterJob, haConfigMap).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster:           cluster,
				flinkJobSubmitter: FlinkJobSubmitter{job: submitterJob}, // No submitter pod exists.
			},
		}

		// when
		result, err := reconciler.reconcileDeletion(ctx)

		// then: no requeue
		assert.NilError(t, err)
		assert.Equal(t, result, ctrl.Result{})

		err = fakeClient.Get(ctx, types.NamespacedName{Name: submitterJob.Name, Namespace: submitterJob.Namespace}, &batchv1.Job{})
		assert.Assert(t, apierrors.IsNotFound(err))

		// jobManagerShutdownFinalizer was removed, which lets the API server (and the fake client) complete the cluster's deletion.
		getErr := fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &v1beta1.FlinkCluster{})
		assert.Assert(t, apierrors.IsNotFound(getErr))
	})

	t.Run("application mode: removes finalizer immediately when the submitter job is already gone", func(t *testing.T) {
		// given: an Application-mode HA cluster whose job already reached a terminal state and
		// was cleaned up (default CleanupPolicy) before deletion started.
		cluster := testHACluster(true)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster: cluster,
				// No submitter job observed (FlinkJobSubmitter zero value).
			},
		}

		// when
		result, err := reconciler.reconcileDeletion(ctx)

		// then: no requeue
		assert.NilError(t, err)
		assert.Equal(t, result, ctrl.Result{})

		// jobManagerShutdownFinalizer was removed, which lets the API server (and the fake client) complete the cluster's deletion.
		getErr := fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &v1beta1.FlinkCluster{})
		assert.Assert(t, apierrors.IsNotFound(getErr))
	})

	t.Run("session mode: deletes JM StatefulSet and requeues without removing the finalizer while the JM pod is still running", func(t *testing.T) {
		// given: a session-mode HA cluster whose JM StatefulSet still has a running pod.
		cluster := testSessionHACluster(true)
		jmStatefulSet := testJMStatefulSet(cluster.Namespace, cluster.Name)
		jmPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-jobmanager-0",
				Namespace: cluster.Namespace,
				Labels:    map[string]string{"cluster": cluster.Name, "component": "jobmanager"},
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, jmStatefulSet, jmPod).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster:       cluster,
				jmStatefulSet: jmStatefulSet,
			},
		}

		// when
		result, err := reconciler.reconcileDeletion(ctx)

		// then: requeued
		assert.NilError(t, err)
		assert.Equal(t, result, requeueResult)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jmStatefulSet.Name, Namespace: jmStatefulSet.Namespace}, &appsv1.StatefulSet{})
		assert.Assert(t, apierrors.IsNotFound(err))

		// The finalizer stays while the JM pod is still running.
		var updated v1beta1.FlinkCluster
		assert.NilError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &updated))
		assert.Assert(t, controllerutil.ContainsFinalizer(&updated, jobManagerShutdownFinalizer))
	})

	t.Run("session mode: deletes JM StatefulSet and removes finalizer once JM pod is gone", func(t *testing.T) {
		// given: a session-mode HA cluster whose JM StatefulSet has no running pods.
		cluster := testSessionHACluster(true)
		jmStatefulSet := testJMStatefulSet(cluster.Namespace, cluster.Name)
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, jmStatefulSet).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster:       cluster,
				jmStatefulSet: jmStatefulSet,
			},
		}

		// when
		result, err := reconciler.reconcileDeletion(ctx)

		// then: no requeue
		assert.NilError(t, err)
		assert.Equal(t, result, ctrl.Result{})

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jmStatefulSet.Name, Namespace: jmStatefulSet.Namespace}, &appsv1.StatefulSet{})
		assert.Assert(t, apierrors.IsNotFound(err))

		// jobManagerShutdownFinalizer was removed, which lets the API server (and the fake client) complete the cluster's deletion.
		getErr := fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &v1beta1.FlinkCluster{})
		assert.Assert(t, apierrors.IsNotFound(getErr))
	})
}

func TestReconcileFlinkNativeConfigMaps(t *testing.T) {
	ctx := context.Background()
	scheme := testHADeleteScheme(t)

	newCluster := func() *v1beta1.FlinkCluster {
		return &v1beta1.FlinkCluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "flinkoperator.k8s.io/v1beta1",
				Kind:       "FlinkCluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cluster",
				Namespace: "default",
				UID:       "test-uid",
			},
			Spec: v1beta1.FlinkClusterSpec{
				FlinkProperties: map[string]string{
					"high-availability":            "kubernetes",
					"kubernetes.cluster-id":        "my-cluster",
					"high-availability.storageDir": "s3://bucket/ha",
				},
			},
		}
	}

	t.Run("patches owner references on Flink-native ConfigMaps", func(t *testing.T) {
		// given: two Flink-native ConfigMaps (HA + checkpoint) observed with no owner reference.
		cluster := newCluster()
		cm1 := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cluster-cluster-config-map",
				Namespace: cluster.Namespace,
			},
		}
		cm2 := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cluster-abc123-config-map",
				Namespace: cluster.Namespace,
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(cluster, cm1, cm2).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster: cluster,
				flinkNativeConfigMaps: &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*cm1, *cm2},
				},
			},
		}

		// when
		err := reconciler.reconcileFlinkNativeConfigMaps(ctx)

		// then: both ConfigMaps now have the cluster as their sole owner.
		assert.NilError(t, err)
		for _, name := range []string{cm1.Name, cm2.Name} {
			var updated corev1.ConfigMap
			assert.NilError(t, fakeClient.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, &updated))
			assert.Equal(t, len(updated.OwnerReferences), 1)
			assert.Equal(t, updated.OwnerReferences[0].Name, cluster.Name)
			assert.Equal(t, updated.OwnerReferences[0].UID, cluster.UID)
		}
	})

	t.Run("skips ConfigMaps that already have owner references", func(t *testing.T) {
		// given: a Flink-native ConfigMap that already has an (unrelated) owner reference.
		cluster := newCluster()
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cluster-cluster-config-map",
				Namespace: cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{Name: "someone-else", UID: "other-uid"},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
					t.Fatal("Update should not be called for an already-owned ConfigMap")
					return nil
				},
			}).
			WithObjects(cluster, cm).
			Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster: cluster,
				flinkNativeConfigMaps: &corev1.ConfigMapList{
					Items: []corev1.ConfigMap{*cm},
				},
			},
		}

		// expect: succeeds without ever calling Update (enforced by the interceptor above).
		assert.NilError(t, reconciler.reconcileFlinkNativeConfigMaps(ctx))
	})

	t.Run("is a no-op when list is nil", func(t *testing.T) {
		// given: no Flink-native ConfigMaps were observed at all.
		cluster := newCluster()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed:  ObservedClusterState{cluster: cluster},
		}

		// expect:
		assert.NilError(t, reconciler.reconcileFlinkNativeConfigMaps(ctx))
	})

	t.Run("is a no-op when list is empty", func(t *testing.T) {
		// given: an empty (non-nil) list of observed Flink-native ConfigMaps.
		cluster := newCluster()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster).Build()
		reconciler := ClusterReconciler{
			k8sClient: fakeClient,
			observed: ObservedClusterState{
				cluster:               cluster,
				flinkNativeConfigMaps: &corev1.ConfigMapList{},
			},
		}

		// when / then
		assert.NilError(t, reconciler.reconcileFlinkNativeConfigMaps(ctx))
	})
}

func TestReconcileReturnsEarlyAfterAddingFinalizer(t *testing.T) {
	// A reconcile that adds the finalizer must return before taking any Flink-side action,
	// so the watch event enqueued by the patch cannot observe a status that predates an
	// action taken after it. The cluster below is set up so that reconcile would go on to
	// trigger stop-with-savepoint for the pending update if it didn't return early.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Not t.Fatalf: FailNow must be called from the goroutine running the test.
		t.Errorf("unexpected Flink API call: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected call", http.StatusInternalServerError)
	}))
	defer server.Close()

	// given: an HA cluster without the finalizer, with a running job and an update triggered.
	savepointsDir := "s3://bucket/savepoints"
	uiPort := int32(8081) // dereferenced when building the Flink API base URL
	cluster := testHACluster(false)
	cluster.DeletionTimestamp = nil
	cluster.Spec.Job.SavepointsDir = &savepointsDir
	cluster.Spec.JobManager = &v1beta1.JobManagerSpec{Ports: v1beta1.JobManagerPorts{UI: &uiPort}}
	cluster.Status.Revision = v1beta1.RevisionStatus{CurrentRevision: "rev-1", NextRevision: "rev-2"}
	cluster.Status.Components.Job = &v1beta1.JobStatus{ID: "job-123", State: v1beta1.JobStateRunning}

	fakeClient := fake.NewClientBuilder().
		WithScheme(testHADeleteScheme(t)).
		WithStatusSubresource(&v1beta1.FlinkCluster{}).
		WithObjects(cluster).
		Build()
	reconciler := ClusterReconciler{
		k8sClient:   fakeClient,
		flinkClient: flink.NewClient(logr.Discard(), newRedirectingHTTPClient(server.URL)),
		observed: ObservedClusterState{
			cluster:                cluster,
			persistentVolumeClaims: &corev1.PersistentVolumeClaimList{},
		},
		desired:  model.DesiredClusterState{Job: testSubmitterJob(cluster.Namespace)},
		recorder: record.NewFakeRecorder(16),
	}

	// when
	ctx := logr.NewContext(context.Background(), logr.Discard())
	result, err := reconciler.reconcile(ctx)

	// then: early return with a 5s requeue
	assert.Equal(t, result, ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second})
	assert.NilError(t, err)

	// and: the finalizer patch is the only thing that happened — no Flink API call (enforced by
	//the handler above) and no savepoint recorded.
	var updated v1beta1.FlinkCluster
	assert.NilError(t, fakeClient.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, &updated))
	assert.Assert(t, controllerutil.ContainsFinalizer(&updated, jobManagerShutdownFinalizer))
	assert.Assert(t, updated.Status.Savepoint == nil)
}

func TestReconcileDeletingClusterDoesNotReAddFinalizer(t *testing.T) {
	scheme := testHADeleteScheme(t)

	// given: a deleting HA cluster whose finalizer was already removed.
	// The fake client refuses to store an object with deletionTimestamp but no finalizers,
	// so don't register the cluster as a stored object — reconcile only reads from observed.
	cluster := testHACluster(false)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Patch: func(_ context.Context, _ client.WithWatch, obj client.Object, _ client.Patch, _ ...client.PatchOption) error {
				if _, ok := obj.(*v1beta1.FlinkCluster); ok {
					t.Fatal("Patch should not be called on the FlinkCluster during deletion without a finalizer")
				}
				return nil
			},
		}).
		Build()
	reconciler := ClusterReconciler{
		k8sClient: fakeClient,
		observed:  ObservedClusterState{cluster: cluster},
		recorder:  record.NewFakeRecorder(16),
	}

	// when
	result, err := reconciler.reconcile(logr.NewContext(context.Background(), logr.Discard()))

	// then: reconcileDeletion returns immediately (no finalizer to process).
	assert.NilError(t, err)
	assert.Equal(t, result, ctrl.Result{})
	assert.Assert(t, !controllerutil.ContainsFinalizer(cluster, jobManagerShutdownFinalizer))
}

func testDeleteConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config-map",
			Namespace: "default",
		},
	}
}

func testHADeleteScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	assert.NilError(t, v1beta1.AddToScheme(scheme))
	assert.NilError(t, corev1.AddToScheme(scheme))
	assert.NilError(t, appsv1.AddToScheme(scheme))
	assert.NilError(t, batchv1.AddToScheme(scheme))
	return scheme
}

func testNonHACluster() *v1beta1.FlinkCluster {
	return &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "non-ha-cluster",
			Namespace: "default",
		},
	}
}

func testHACluster(withFinalizer bool) *v1beta1.FlinkCluster {
	applicationMode := v1beta1.JobModeApplication
	cluster := &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ha-cluster",
			Namespace:         "default",
			DeletionTimestamp: &metav1.Time{Time: metav1.Now().Time},
		},
		Spec: v1beta1.FlinkClusterSpec{
			Job: &v1beta1.JobSpec{Mode: &applicationMode},
			FlinkProperties: map[string]string{
				"high-availability":            "kubernetes",
				"kubernetes.cluster-id":        "ha-cluster",
				"high-availability.storageDir": "s3://bucket/ha",
			},
		},
	}
	if withFinalizer {
		controllerutil.AddFinalizer(cluster, jobManagerShutdownFinalizer)
	}
	return cluster
}

func testSessionHACluster(withFinalizer bool) *v1beta1.FlinkCluster {
	cluster := &v1beta1.FlinkCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ha-session",
			Namespace:         "default",
			DeletionTimestamp: &metav1.Time{Time: metav1.Now().Time},
		},
		Spec: v1beta1.FlinkClusterSpec{
			FlinkProperties: map[string]string{
				"high-availability":            "kubernetes",
				"kubernetes.cluster-id":        "ha-session",
				"high-availability.storageDir": "s3://bucket/ha",
			},
		},
	}
	if withFinalizer {
		controllerutil.AddFinalizer(cluster, jobManagerShutdownFinalizer)
	}
	return cluster
}

func testJMStatefulSet(namespace, clusterName string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName + "-jobmanager",
			Namespace: namespace,
		},
	}
}

func testSubmitterJob(namespace string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "ha-cluster-jobmanager", Namespace: namespace},
	}
}
