package flinkcluster

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("FlinkCluster Controller", Ordered, func() {
	// Utility constants and functions here
	const (
		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	var getDummyFlinkClusterWithJob = func() *v1beta1.FlinkCluster {
		fc := getDummyFlinkCluster()
		var blocking v1beta1.JobMode = v1beta1.JobModeBlocking
		fc.Spec.Job.Mode = &blocking
		return getDummyFlinkCluster()
	}

	BeforeAll(func() {
		By("Creating a new FlinkCluster")
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		Expect(k8sClient.Create(ctx, dummyFlinkCluster)).Should(Succeed())
	})

	It("Should create the JobManager Statefulset", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		expectedJobManagerName := dummyFlinkCluster.ObjectMeta.Name + "-jobmanager"
		jobManagerLookupKey := types.NamespacedName{Name: expectedJobManagerName, Namespace: dummyFlinkCluster.ObjectMeta.Namespace}
		createdJobManagerStatefulSet := &appsv1.StatefulSet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, jobManagerLookupKey, createdJobManagerStatefulSet)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	})

	It("Should create the TaskManager Statefulset", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		expectedTaskManagerName := dummyFlinkCluster.ObjectMeta.Name + "-taskmanager"
		taskManagerLookupKey := types.NamespacedName{Name: expectedTaskManagerName, Namespace: dummyFlinkCluster.ObjectMeta.Namespace}
		createdTaskManagerStatefulSet := &appsv1.StatefulSet{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, taskManagerLookupKey, createdTaskManagerStatefulSet)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	})

	It("Should create the JobSubmitter Job", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		expectedJobSubmitterName := dummyFlinkCluster.ObjectMeta.Name + "-job-submitter"
		jobSubmitterLookupKey := types.NamespacedName{Name: expectedJobSubmitterName, Namespace: dummyFlinkCluster.ObjectMeta.Namespace}
		createdJobSubmitterJob := &batchv1.Job{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, jobSubmitterLookupKey, createdJobSubmitterJob)
			return err == nil
		}, timeout, interval).Should(BeTrue())

	})

	It("Setting job-cancel annotation should kill-the-cluster", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		fetchedFlinkCluster := &v1beta1.FlinkCluster{}
		flinkClusterLookupKey := types.NamespacedName{Name: dummyFlinkCluster.ObjectMeta.Name, Namespace: dummyFlinkCluster.ObjectMeta.Namespace}

		Expect(k8sClient.Get(ctx, flinkClusterLookupKey, fetchedFlinkCluster)).Should(Succeed())
		fetchedFlinkCluster.ObjectMeta.Annotations = map[string]string{
			v1beta1.ControlAnnotation: v1beta1.ControlNameJobCancel,
		}
		Expect(k8sClient.Update(ctx, fetchedFlinkCluster)).Should(Succeed())

		expectedJobSubmitterName := dummyFlinkCluster.ObjectMeta.Name + "-job-submitter"
		jobSubmitterLookupKey := types.NamespacedName{Name: expectedJobSubmitterName, Namespace: dummyFlinkCluster.ObjectMeta.Namespace}
		createdJobSubmitterJob := &batchv1.Job{}

		// Check if the job is killed
		Eventually(func() bool {
			err := k8sClient.Get(ctx, jobSubmitterLookupKey, createdJobSubmitterJob)
			if err != nil {
				return errors.IsNotFound(err)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		// check if the FlinkCluster status is updated
		Eventually(func() bool {

			err := k8sClient.Get(ctx, flinkClusterLookupKey, fetchedFlinkCluster)
			if err != nil {
				return false
			}
			return fetchedFlinkCluster.Status.State == v1beta1.ClusterStateStopped
		}, timeout, interval).Should(BeTrue())

	})

	AfterAll(func() {
		By("Deleting the FlinkCluster")
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		Expect(k8sClient.Delete(ctx, dummyFlinkCluster)).Should(Succeed())
	})
})
