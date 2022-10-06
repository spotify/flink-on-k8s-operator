package util

import (
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestUpperBoundedResourceList(t *testing.T) {
	t.Run("only requests set", func(t *testing.T) {
		resourceList := UpperBoundedResourceList(corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		})
		assert.Equal(t, *resourceList.Cpu(), resource.MustParse("1"))
		assert.Equal(t, *resourceList.Memory(), resource.MustParse("1Gi"))
	})

	t.Run("only limits set", func(t *testing.T) {
		resourceList := UpperBoundedResourceList(corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		})
		assert.Equal(t, *resourceList.Cpu(), resource.MustParse("1"))
		assert.Equal(t, *resourceList.Memory(), resource.MustParse("1Gi"))
	})

	t.Run("only limits and requests set", func(t *testing.T) {
		resourceList := UpperBoundedResourceList(corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		})
		assert.Equal(t, *resourceList.Cpu(), resource.MustParse("2"))
		assert.Equal(t, *resourceList.Memory(), resource.MustParse("2Gi"))
	})
}
