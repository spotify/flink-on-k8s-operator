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

package main

import (
	"flag"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/controllers/flinkcluster"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var (
	metricsAddr             = flag.String("metrics-addr", ":8080", "The address the metric endpoint binds to.")
	enableLeaderElection    = flag.Bool("enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	leaderElectionID        = flag.String("leader-election-id", "flink-operator-lock", "The name that leader election will use for holding the leader lock")
	watchNamespace          = flag.String("watch-namespace", "", "Watch custom resources in the namespace, ignore other namespaces. If empty, all namespaces will be watched.")
	maxConcurrentReconciles = flag.Int("max-concurrent-reconciles", 1, "The maximum number of concurrent Reconciles which can be run. Defaults to 1.")
)

func init() {
	appsv1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	v1beta1.AddToScheme(scheme)
	networkingv1.AddToScheme(scheme)
	policyv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: *metricsAddr,
		LeaderElection:     *enableLeaderElection,
		Namespace:          *watchNamespace,
		LeaderElectionID:   *leaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	cs, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "Unable to create clientset")
		os.Exit(1)
	}

	err = (&flinkcluster.FlinkClusterReconciler{
		Client:    mgr.GetClient(),
		Clientset: cs,
		Log:       ctrl.Log.WithName("controllers").WithName("FlinkCluster"),
	}).SetupWithManager(mgr, *maxConcurrentReconciles)
	if err != nil {
		setupLog.Error(err, "Unable to create controller", "controller", "FlinkCluster")
		os.Exit(1)
	}

	// Set up webhooks for the custom resource.
	// Disable it with `FLINK_OPERATOR_ENABLE_WEBHOOKS=false` when we run locally.
	if os.Getenv("FLINK_OPERATOR_ENABLE_WEBHOOKS") != "false" {
		if err = (&v1beta1.FlinkCluster{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "Unable to setup webhooks", "webhook", "FlinkCluster")
			os.Exit(1)
		}
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}
