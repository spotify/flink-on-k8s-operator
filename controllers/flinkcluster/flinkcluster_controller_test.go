package flinkcluster

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("FlinkCluster Controller", Ordered, func() {
	// Utility constants and functions here
	const (
		timeout  = time.Second * 20
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	var getDummyFlinkClusterWithJob = func() *v1beta1.FlinkCluster {
		fc := getDummyFlinkCluster()
		var blocking = v1beta1.JobModeBlocking
		fc.Spec.Job.Mode = &blocking
		fc.Spec.PodDisruptionBudget = &policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
			MaxUnavailable: &intstr.IntOrString{StrVal: "0%"},
		}
		return fc
	}

	BeforeAll(func() {
		By("Creating a new FlinkCluster")
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		Expect(k8sClient.Create(ctx, dummyFlinkCluster)).Should(Succeed())
	})

	It("Should create the JobManager Statefulset", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		expectedJobManagerName := getJobManagerName(dummyFlinkCluster.Name)
		jobManagerLookupKey := types.NamespacedName{
			Name:      expectedJobManagerName,
			Namespace: dummyFlinkCluster.Namespace,
		}
		createdJobManagerStatefulSet := &appsv1.StatefulSet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, jobManagerLookupKey, createdJobManagerStatefulSet)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	})

	It("Should create the TaskManager Statefulset", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		expectedTaskManagerName := getTaskManagerName(dummyFlinkCluster.Name)
		taskManagerLookupKey := types.NamespacedName{
			Name:      expectedTaskManagerName,
			Namespace: dummyFlinkCluster.Namespace,
		}
		createdTaskManagerStatefulSet := &appsv1.StatefulSet{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, taskManagerLookupKey, createdTaskManagerStatefulSet)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	})

	It("Should create the JobSubmitter Job", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		expectedJobSubmitterName := dummyFlinkCluster.Name + "-job-submitter"
		jobSubmitterLookupKey := types.NamespacedName{
			Name:      expectedJobSubmitterName,
			Namespace: dummyFlinkCluster.Namespace,
		}
		createdJobSubmitterJob := &batchv1.Job{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, jobSubmitterLookupKey, createdJobSubmitterJob)
			return err == nil
		}, timeout, interval).Should(BeTrue())

	})

	It("Should create the PodDisruptionBudget", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		expectedPdbName := getPodDisruptionBudgetName(dummyFlinkCluster.Name)
		pdbLookupKey := types.NamespacedName{
			Name:      expectedPdbName,
			Namespace: dummyFlinkCluster.Namespace,
		}
		createdPdb := &policyv1.PodDisruptionBudget{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, pdbLookupKey, createdPdb)
			return err == nil
		}, timeout, interval).Should(BeTrue())

	})

	It("Setting job-cancel annotation should kill-the-cluster", func() {
		dummyFlinkCluster := getDummyFlinkClusterWithJob()
		fetchedFlinkCluster := &v1beta1.FlinkCluster{}
		flinkClusterLookupKey := types.NamespacedName{
			Name:      dummyFlinkCluster.Name,
			Namespace: dummyFlinkCluster.Namespace,
		}

		Eventually(func() bool {
			e := k8sClient.Get(ctx, flinkClusterLookupKey, fetchedFlinkCluster)
			if e != nil {
				return false
			}
			fetchedFlinkCluster.Annotations = map[string]string{
				v1beta1.ControlAnnotation: v1beta1.ControlNameJobCancel,
			}
			e = k8sClient.Update(ctx, fetchedFlinkCluster)
			return e == nil
		}, timeout, interval).Should(BeTrue())

		expectedJobSubmitterName := dummyFlinkCluster.Name + "-job-submitter"
		jobSubmitterLookupKey := types.NamespacedName{
			Name:      expectedJobSubmitterName,
			Namespace: dummyFlinkCluster.Namespace,
		}
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
