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
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

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
	autoscalingv2.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts)).
		WithName("controllers").
		WithName("FlinkCluster")
	ctrl.SetLogger(logger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:         scheme,
		Metrics:        metricsserver.Options{BindAddress: *metricsAddr},
		LeaderElection: *enableLeaderElection,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{*watchNamespace: {}},
		},
		LeaderElectionID: *leaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	reconciler, err := flinkcluster.NewReconciler(mgr)
	if err != nil {
		setupLog.Error(err, "Unable to create reconciler")
		os.Exit(1)
	}
	err = reconciler.SetupWithManager(mgr, *maxConcurrentReconciles)
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
