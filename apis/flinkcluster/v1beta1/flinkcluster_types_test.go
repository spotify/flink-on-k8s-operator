/*
Copyright 2019 Google LLC.

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

package v1beta1

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/types"
)

// These tests are written in BDD-style using Ginkgo framework. Refer to
// http://onsi.github.io/ginkgo to learn more.

var _ = Describe("FlinkCluster type", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		Namespace = "default"
		JobName   = "state-machine"

		timeout  = time.Second * 5
		interval = time.Millisecond * 250
	)

	var flinkJob FlinkCluster
	flinkJobManifestPath, _ := filepath.Abs("assets/test/flinkcluster_type_request.yaml")
	flinkJobManifest, _ := os.ReadFile(flinkJobManifestPath)
	yaml.Unmarshal(flinkJobManifest, &flinkJob)

	var expectedFlinkJob FlinkCluster
	expectedFlinkJobManifestPath, _ := filepath.Abs("assets/test/flinkcluster_type_expected.yaml")
	expectedFlinkJobManifest, _ := os.ReadFile(expectedFlinkJobManifestPath)
	yaml.Unmarshal(expectedFlinkJobManifest, &expectedFlinkJob)
	expectedSpecByte, _ := json.Marshal(expectedFlinkJob.Spec)
	expectedSpec := string(expectedSpecByte)

	Context("When creating FlinkCluster", func() {
		It("Should set default values", func() {
			By("By creating a new FlinkCluster")
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, &flinkJob)).Should(Succeed())

			// We'll need to retry getting this newly created FlinkCluster, given that creation may not immediately happen.
			GinkgoWriter.Println("FlinkCluster expected spec: ", expectedSpec)
			jobLookupKey := types.NamespacedName{Name: JobName, Namespace: Namespace}
			Eventually(func() string {
				createdJob := &FlinkCluster{}
				err := k8sClient.Get(ctx, jobLookupKey, createdJob)
				if err != nil {
					return ""
				}
				createdSpecBytes, _ := json.Marshal(createdJob.Spec)
				createdSpec := string(createdSpecBytes)
				GinkgoWriter.Println("FlinkCluster created spec: ", expectedSpec)

				return createdSpec
			}, timeout, interval).Should(Equal(expectedSpec))
		})
	})
})
