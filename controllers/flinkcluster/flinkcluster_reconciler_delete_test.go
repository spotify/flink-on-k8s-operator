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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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

func testDeleteConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config-map",
			Namespace: "default",
		},
	}
}
