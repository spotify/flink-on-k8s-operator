package flinkcluster

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("FlinkCluster Controller", func() {
	// Utility constants and functions here
	const (
		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a new FlinkCluster", func() {
		It("Should create the JobManager Statefulset", func() {
			// Test code here
			By("By creating a new FlinkCluster")
			dummyFlinkCluster := getDummyFlinkCluster()
			Expect(k8sClient.Create(ctx, dummyFlinkCluster)).Should(Succeed())

			expectedJobManagerName := dummyFlinkCluster.ObjectMeta.Name + "-jobmanager"
			jobManagerLookupKey := types.NamespacedName{Name: expectedJobManagerName, Namespace: dummyFlinkCluster.ObjectMeta.Namespace}
			createdJobManagerStatefulSet := &appsv1.StatefulSet{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, jobManagerLookupKey, createdJobManagerStatefulSet)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

		})
	})
})
