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
// +kubebuilder:docs-gen:collapse=Apache License

package v1beta1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:docs-gen:collapse=Go imports

var log = logf.Log.WithName("webhook")

// flinkClusterDefaulter implements admission.CustomDefaulter for FlinkCluster.
type flinkClusterDefaulter struct{}

// flinkClusterValidator implements admission.CustomValidator for FlinkCluster.
type flinkClusterValidator struct {
	validator Validator
}

// SetupWebhookWithManager adds webhook for FlinkCluster.
func (cluster *FlinkCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, cluster).
		WithDefaulter(&flinkClusterDefaulter{}).
		WithValidator(&flinkClusterValidator{validator: Validator{}}).
		Complete()
}

/*
Kubebuilder markers to generate webhook manifests.
This marker is responsible for generating a mutating webhook manifest.
The meaning of each marker can be found [here](/reference/markers/webhook.md).
*/

// +kubebuilder:webhook:path=/mutate-flinkoperator-k8s-io-v1beta1-flinkcluster,admissionReviewVersions=v1,sideEffects=None,mutating=true,failurePolicy=fail,groups=flinkoperator.k8s.io,resources=flinkclusters,verbs=create;update,versions=v1beta1,name=mflinkcluster.flinkoperator.k8s.io

// Default implements admission.Defaulter[*FlinkCluster].
func (d *flinkClusterDefaulter) Default(ctx context.Context, cluster *FlinkCluster) error {
	log.Info("default", "name", cluster.Name, "original", *cluster)
	_SetDefault(cluster)
	log.Info("default", "name", cluster.Name, "augmented", *cluster)
	return nil
}

/*
This marker is responsible for generating a validating webhook manifest.
*/

// +kubebuilder:webhook:path=/validate-flinkoperator-k8s-io-v1beta1-flinkcluster,admissionReviewVersions=v1,sideEffects=None,mutating=false,failurePolicy=fail,groups=flinkoperator.k8s.io,resources=flinkclusters,verbs=create;update,versions=v1beta1,name=vflinkcluster.flinkoperator.k8s.io

var validator = Validator{}

// ValidateCreate implements admission.Validator[*FlinkCluster].
func (v *flinkClusterValidator) ValidateCreate(ctx context.Context, cluster *FlinkCluster) (admission.Warnings, error) {
	log.Info("Validate create", "name", cluster.Name)
	return nil, v.validator.ValidateCreate(cluster)
}

// ValidateUpdate implements admission.Validator[*FlinkCluster].
func (v *flinkClusterValidator) ValidateUpdate(ctx context.Context, oldCluster, cluster *FlinkCluster) (admission.Warnings, error) {
	log.Info("Validate update", "name", cluster.Name)
	return nil, v.validator.ValidateUpdate(oldCluster, cluster)
}

// ValidateDelete implements admission.Validator[*FlinkCluster].
func (v *flinkClusterValidator) ValidateDelete(ctx context.Context, cluster *FlinkCluster) (admission.Warnings, error) {
	log.Info("validate delete", "name", cluster.Name)
	// TODO
	return nil, nil
}

// +kubebuilder:docs-gen:collapse=Validate object name
