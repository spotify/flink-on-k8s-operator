package history

import (
	"context"
	"sync/atomic"
	"testing"

	apps "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestCreateControllerRevision_NotFoundRace_DoesNotIncrementCollisionCount(t *testing.T) {
	// This test verifies that when Create returns AlreadyExists and the
	// subsequent Get returns NotFound (a race condition where the revision
	// was deleted between the two calls), collisionCount is NOT incremented.
	//
	// Incrementing collisionCount on NotFound changes the hash, causing the
	// next revision to have a different name despite identical data. This
	// makes IsUpdateTriggered() permanently true, blocking cluster state
	// transitions (e.g., Running -> Stopping).

	scheme := runtime.NewScheme()
	apps.AddToScheme(scheme)

	parent := &metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: "default",
		UID:       "test-uid",
	}

	revision := &apps.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Labels:    map[string]string{},
		},
		Data:     runtime.RawExtension{Raw: []byte(`{"spec":"test-data"}`)},
		Revision: 1,
	}

	var createCalls atomic.Int32
	var getCalls atomic.Int32

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Wrap the fake client with interceptors to simulate the race.
	interceptedClient := interceptor.NewClient(fakeClient, interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			n := createCalls.Add(1)
			if n == 1 {
				// First Create: return AlreadyExists to trigger the Get path.
				return apierrors.NewAlreadyExists(
					schema.GroupResource{Group: "apps", Resource: "controllerrevisions"},
					obj.GetName(),
				)
			}
			// Second Create: succeed (the slot is now free).
			return c.Create(ctx, obj, opts...)
		},
		Get: func(ctx context.Context, c client.WithWatch, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
			n := getCalls.Add(1)
			if n == 1 {
				// First Get (after AlreadyExists): return NotFound — the race condition.
				return apierrors.NewNotFound(
					schema.GroupResource{Group: "apps", Resource: "controllerrevisions"},
					key.Name,
				)
			}
			return c.Get(ctx, key, obj, opts...)
		},
	})

	h := &realHistory{
		Client:  interceptedClient,
		context: context.Background(),
	}

	var collisionCount int32 = 0
	originalCollisionCount := collisionCount

	result, err := h.CreateControllerRevision(parent, revision, &collisionCount)
	if err != nil {
		t.Fatalf("CreateControllerRevision returned error: %v", err)
	}
	if result == nil {
		t.Fatal("CreateControllerRevision returned nil result")
	}

	// THE CRITICAL ASSERTION: collisionCount must NOT be incremented.
	// If it is, the hash changes, producing a different revision name for
	// identical data, which causes IsUpdateTriggered() to be permanently true.
	if collisionCount != originalCollisionCount {
		t.Errorf("REGRESSION: collisionCount was incremented from %d to %d on NotFound race.\n"+
			"This causes phantom updates: the revision gets a different name despite identical data,\n"+
			"making IsUpdateTriggered() permanently true and blocking Running -> Stopping transitions.",
			originalCollisionCount, collisionCount)
	}

	// Verify the revision was actually created (second Create call succeeded).
	if createCalls.Load() != 2 {
		t.Errorf("expected 2 Create calls (AlreadyExists + success), got %d", createCalls.Load())
	}
	if getCalls.Load() != 1 {
		t.Errorf("expected 1 Get call (NotFound), got %d", getCalls.Load())
	}

	// Verify the revision name is based on the original hash (not a collision hash).
	expectedHash := HashControllerRevision(revision, &originalCollisionCount)
	expectedName := ControllerRevisionName(parent.Name, expectedHash)
	if result.Name != expectedName {
		t.Errorf("revision name mismatch: got %s, want %s (hash should use original collisionCount)",
			result.Name, expectedName)
	}
}

// objectMetaParent implements metav1.Object for testing.
type objectMetaParent struct {
	*metav1.ObjectMeta
}

func (o *objectMetaParent) GetNamespace() string                          { return o.Namespace }
func (o *objectMetaParent) SetNamespace(ns string)                        { o.Namespace = ns }
func (o *objectMetaParent) GetName() string                               { return o.Name }
func (o *objectMetaParent) SetName(name string)                           { o.Name = name }
func (o *objectMetaParent) GetGenerateName() string                       { return "" }
func (o *objectMetaParent) SetGenerateName(string)                        {}
func (o *objectMetaParent) GetUID() types.UID                             { return o.UID }
func (o *objectMetaParent) SetUID(uid types.UID)                          { o.UID = uid }
func (o *objectMetaParent) GetResourceVersion() string                    { return "" }
func (o *objectMetaParent) SetResourceVersion(string)                     {}
func (o *objectMetaParent) GetGeneration() int64                          { return 0 }
func (o *objectMetaParent) SetGeneration(int64)                           {}
func (o *objectMetaParent) GetSelfLink() string                           { return "" }
func (o *objectMetaParent) SetSelfLink(string)                            {}
func (o *objectMetaParent) GetCreationTimestamp() metav1.Time             { return metav1.Time{} }
func (o *objectMetaParent) SetCreationTimestamp(metav1.Time)              {}
func (o *objectMetaParent) GetDeletionTimestamp() *metav1.Time            { return nil }
func (o *objectMetaParent) SetDeletionTimestamp(*metav1.Time)             {}
func (o *objectMetaParent) GetDeletionGracePeriodSeconds() *int64         { return nil }
func (o *objectMetaParent) SetDeletionGracePeriodSeconds(*int64)          {}
func (o *objectMetaParent) GetLabels() map[string]string                  { return nil }
func (o *objectMetaParent) SetLabels(map[string]string)                   {}
func (o *objectMetaParent) GetAnnotations() map[string]string             { return nil }
func (o *objectMetaParent) SetAnnotations(map[string]string)              {}
func (o *objectMetaParent) GetFinalizers() []string                       { return nil }
func (o *objectMetaParent) SetFinalizers([]string)                        {}
func (o *objectMetaParent) GetOwnerReferences() []metav1.OwnerReference   { return nil }
func (o *objectMetaParent) SetOwnerReferences([]metav1.OwnerReference)    {}
func (o *objectMetaParent) GetManagedFields() []metav1.ManagedFieldsEntry { return nil }
func (o *objectMetaParent) SetManagedFields([]metav1.ManagedFieldsEntry)  {}
