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

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	var flinkJob FlinkCluster
	flinkJobManifestPath, _ := filepath.Abs("assets/test/flinkcluster_type_test.yaml")
	flinkJobManifest, _ := os.ReadFile(flinkJobManifestPath)
	yaml.Unmarshal(flinkJobManifest, &flinkJob)

	Context("When creating FlinkCluster", func() {
		It("Should set default values", func() {
			By("By creating a new FlinkCluster")
			ctx := context.Background()

			requested, _ := json.Marshal(flinkJob)
			GinkgoWriter.Println("FlinkCluster requested: ", string(requested))
			Expect(k8sClient.Create(ctx, &flinkJob)).Should(Succeed())

			jobLookupKey := types.NamespacedName{Name: JobName, Namespace: Namespace}
			createdjob := &FlinkCluster{}

			// We'll need to retry getting this newly created FlinkCluster, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, jobLookupKey, createdjob)
				if err != nil {
					return false
				}
				created, _ := json.Marshal(createdjob)
				GinkgoWriter.Println("FlinkCluster created: ", string(created))
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})
})
