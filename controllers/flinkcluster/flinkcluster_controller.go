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

package flinkcluster

import (
	"context"
	"time"

	"github.com/spotify/flink-on-k8s-operator/internal/controllers/history"
	"github.com/spotify/flink-on-k8s-operator/internal/flink"

	"github.com/go-logr/logr"
	v1beta1 "github.com/spotify/flink-on-k8s-operator/apis/flinkcluster/v1beta1"
	"github.com/spotify/flink-on-k8s-operator/internal/model"

	semver "github.com/hashicorp/go-version"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcontroller "sigs.k8s.io/controller-runtime/pkg/controller"
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = v1beta1.GroupVersion.WithKind("FlinkCluster")

// FlinkClusterReconciler reconciles a FlinkCluster object
type FlinkClusterReconciler struct {
	Client          client.Client
	Clientset       *kubernetes.Clientset
	DiscoveryClient *discovery.DiscoveryClient
	Log             logr.Logger
	Mgr             ctrl.Manager
}

// +kubebuilder:rbac:groups=flinkoperator.k8s.io,resources=flinkclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=flinkoperator.k8s.io,resources=flinkclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get
// +kubebuilder:rbac:groups=networking,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking,resources=ingresses/status,verbs=get

// Reconcile the observed state towards the desired state for a FlinkCluster custom resource.
func (reconciler *FlinkClusterReconciler) Reconcile(ctx context.Context,
	request ctrl.Request) (ctrl.Result, error) {
	var log = reconciler.Log.WithValues(
		"cluster", request.NamespacedName)
	var handler = FlinkClusterHandler{
		k8sClient:          reconciler.Client,
		k8sClientset:       reconciler.Clientset,
		k8sDiscoveryClient: reconciler.DiscoveryClient,
		flinkClient:        flink.NewDefaultClient(log),
		request:            request,
		context:            context.Background(),
		log:                log,
		recorder:           reconciler.Mgr.GetEventRecorderFor("FlinkOperator"),
		observed:           ObservedClusterState{},
	}
	return handler.reconcile(ctx, request)
}

// SetupWithManager registers this reconciler with the controller manager and
// starts watching FlinkCluster, Deployment and Service resources.
func (reconciler *FlinkClusterReconciler) SetupWithManager(
	mgr ctrl.Manager,
	maxConcurrentReconciles int) error {
	reconciler.Mgr = mgr
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(ctrlcontroller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		For(&v1beta1.FlinkCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&batchv1.Job{}).
		Complete(reconciler)
}

// FlinkClusterHandler holds the context and state for a
// reconcile request.
type FlinkClusterHandler struct {
	k8sClient          client.Client
	k8sClientset       *kubernetes.Clientset
	k8sDiscoveryClient *discovery.DiscoveryClient
	flinkClient        *flink.Client
	request            ctrl.Request
	context            context.Context
	log                logr.Logger
	recorder           record.EventRecorder
	observed           ObservedClusterState
	desired            model.DesiredClusterState
}

func (handler *FlinkClusterHandler) reconcile(ctx context.Context,
	request ctrl.Request) (ctrl.Result, error) {
	var k8sClient = handler.k8sClient
	var k8sDiscoveryClient = handler.k8sDiscoveryClient
	var flinkClient = handler.flinkClient
	var log = handler.log
	var context = handler.context
	var observed = &handler.observed
	var desired = &handler.desired
	var statusChanged bool
	var err error
	var k8sServerVersionInfo *version.Info
	var k8sServerVersion *semver.Version

	// History interface
	var history = history.NewHistory(k8sClient, context)
	k8sServerVersionInfo, err = k8sDiscoveryClient.ServerVersion()
	if err != nil {
		log.Error(err, "Failed to observe k8s server version")
		return ctrl.Result{}, err
	}
	k8sServerVersion, err = semver.NewVersion(k8sServerVersionInfo.String())
	if err != nil {
		log.Error(err, "Bad kubernetes server version.")
		return ctrl.Result{}, err
	}

	log.Info("============================================================")
	log.Info("Kubernetes server version:", "k8sServerVersion", k8sServerVersion.String())
	log.Info("---------- 1. Observe the current state ----------")

	var observer = ClusterStateObserver{
		k8sClient:        k8sClient,
		k8sClientset:     handler.k8sClientset,
		k8sServerVersion: k8sServerVersion,
		flinkClient:      flinkClient,
		request:          request,
		context:          context,
		log:              log,
		history:          history,
	}
	err = observer.observe(observed)
	if err != nil {
		log.Error(err, "Failed to observe the current state")
		return ctrl.Result{}, err
	}

	// Sync history and observe revision status
	err = observer.syncRevisionStatus(observed)
	if err != nil {
		log.Error(err, "Failed to sync flinkCluster history")
		return ctrl.Result{}, err
	}

	log.Info("---------- 2. Update cluster status ----------")

	var updater = ClusterStatusUpdater{
		k8sClient: k8sClient,
		context:   context,
		log:       log,
		recorder:  handler.recorder,
		observed:  handler.observed,
	}
	statusChanged, err = updater.updateStatusIfChanged()
	if err != nil {
		log.Error(err, "Failed to update cluster status")
		return ctrl.Result{}, err
	}
	if statusChanged {
		log.Info(
			"Wait status to be stable before taking further actions.",
			"requeueAfter",
			5)
		return ctrl.Result{
			Requeue: true, RequeueAfter: 5 * time.Second,
		}, nil
	}

	log.Info("---------- 3. Compute the desired state ----------")

	*desired = *getDesiredClusterState(observed)
	if desired.ConfigMap != nil {
		log.Info("Desired state", "ConfigMap", *desired.ConfigMap)
	} else {
		log.Info("Desired state", "ConfigMap", "nil")
	}
	if desired.PodDisruptionBudget.PodDisruptionBudgetV1 != nil {
		log.Info("Desired state", "PodDisruptionBudget", *desired.PodDisruptionBudget.PodDisruptionBudgetV1, "apiVersion", "policy/v1")
	} else if observed.podDisruptionBudget.PodDisruptionBudgetV1 != nil {
		log.Info("Desired state", "PodDisruptionBudget", "nil", "apiVersion", "policy/v1")
	}
	if desired.PodDisruptionBudget.PodDisruptionBudgetV1beta1 != nil {
		log.Info("Desired state", "PodDisruptionBudget", *desired.PodDisruptionBudget.PodDisruptionBudgetV1beta1, "apiVersion", "policy/v1beta1")
	} else if observed.podDisruptionBudget.PodDisruptionBudgetV1beta1 != nil {
		log.Info("Desired state", "PodDisruptionBudget", "nil", "apiVersion", "policy/v1beta1")
	}
	if desired.TmService != nil {
		log.Info("Desired state", "TaskManager Service", *desired.TmService)
	} else {
		log.Info("Desired state", "TaskManager Service", "nil")
	}
	if desired.JmStatefulSet != nil {
		log.Info("Desired state", "JobManager StatefulSet", *desired.JmStatefulSet)
	} else {
		log.Info("Desired state", "JobManager StatefulSet", "nil")
	}
	if desired.JmService != nil {
		log.Info("Desired state", "JobManager service", *desired.JmService)
	} else {
		log.Info("Desired state", "JobManager service", "nil")
	}
	if desired.JmIngress != nil {
		log.Info("Desired state", "JobManager ingress", *desired.JmIngress)
	} else {
		log.Info("Desired state", "JobManager ingress", "nil")
	}
	if desired.TmStatefulSet != nil {
		log.Info("Desired state", "TaskManager StatefulSet", *desired.TmStatefulSet)
	} else if desired.TmDeployment != nil {
		log.Info("Desired state", "TaskManager Deployment", *desired.TmDeployment)
	} else {
		log.Info("Desired state", "TaskManager", "nil")
	}

	if desired.Job != nil {
		log.Info("Desired state", "Job", *desired.Job)
	} else {
		log.Info("Desired state", "Job", "nil")
	}

	log.Info("---------- 4. Take actions ----------")

	var reconciler = ClusterReconciler{
		k8sClient:        k8sClient,
		k8sServerVersion: k8sServerVersion,
		flinkClient:      flinkClient,
		context:          context,
		log:              log,
		observed:         handler.observed,
		desired:          handler.desired,
		recorder:         handler.recorder,
	}
	result, err := reconciler.reconcile()
	if err != nil {
		log.Error(err, "Failed to reconcile")
	}
	if result.RequeueAfter > 0 {
		log.Info("Requeue reconcile request", "after", result.RequeueAfter)
	}

	return result, err
}
