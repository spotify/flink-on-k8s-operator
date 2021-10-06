/*
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

package controllers

import (
	"context"
	"time"

	"github.com/spotify/flink-on-k8s-operator/controllers/flink"
	"github.com/spotify/flink-on-k8s-operator/controllers/history"

	"github.com/go-logr/logr"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FlinkClusterHandlerV2 struct {
	k8sClient    client.Client
	k8sClientset *kubernetes.Clientset
	flinkClient  *flink.Client
	request      ctrl.Request
	log          logr.Logger
	recorder     record.EventRecorder
}

func (handler *FlinkClusterHandlerV2) reconcile(ctx context.Context) (ctrl.Result, error) {
	var log = handler.log

	var controllerHistory = history.NewHistory(handler.k8sClient, ctx)

	handler.logStart()
	log.Info("---------- 1. Observe the current state ----------")

	observed, observedErr := handler.observe(ctx, controllerHistory)
	if observedErr != nil {
		log.Error(observedErr, "Failed to observe the current state")
		handler.logEnd()
		return ctrl.Result{}, observedErr
	}

	// Sync history and observe revision status
	syncErr := handler.syncRevisionStatus(&observed, controllerHistory)
	if syncErr != nil {
		log.Error(syncErr, "Failed to sync flinkCluster history")
		handler.logEnd()
		return ctrl.Result{}, syncErr
	}

	log.Info("---------- 2. Update cluster status ----------")

	statusChanged, statusChangedErr := handler.updateStatusIfChanged(ctx, observed)
	if statusChangedErr != nil {
		log.Error(statusChangedErr, "Failed to update cluster status")
		handler.logEnd()
		return ctrl.Result{}, statusChangedErr
	}

	if statusChanged {
		log.Info("Wait status to be stable before taking further actions.")
		handler.logRequeue()
		return ctrl.Result{RequeueAfter: 5 * time.Second, Requeue: true}, nil
	}

	log.Info("---------- 3. Compute the desired state ----------")

	desired := handler.getDesiredClusterState(observed, time.Now())

	// desiredClusterState := StateFromDesired(*desired)
	// observedClusterState := StateFromObserved(*observed)

	// if diff := cmp.Diff(desiredClusterState, observedClusterState); diff != "" {
	// 	handler.log.Info("State diff", "diff", diff)
	// }

	log.Info("---------- 4. Take actions ----------")

	var reconciler = ClusterReconciler{
		context:     ctx,
		desired:     desired,
		flinkClient: handler.flinkClient,
		k8sClient:   handler.k8sClient,
		log:         log,
		observed:    observed,
		recorder:    handler.recorder,
	}

	result, err := reconciler.reconcile()

	if err != nil {
		log.Error(err, "Failed to reconcile")
	} else {
		log.Info("Reconcile completed")
	}
	if result.RequeueAfter > 0 {
		handler.logRequeue()
	} else {
		handler.logEnd()
	}

	return result, err
}

func (handler *FlinkClusterHandlerV2) logStart() {
	handler.log.Info("======================== START ========================")
}

func (handler *FlinkClusterHandlerV2) logEnd() {
	handler.log.Info("========================  END  ========================")
}

func (handler *FlinkClusterHandlerV2) logRequeue() {
	handler.log.Info("========================  REQ  ========================")
}
