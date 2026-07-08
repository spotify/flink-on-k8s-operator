package flinkcluster

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
	"gotest.tools/v3/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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
