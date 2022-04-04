/*
Copyright 2020 Google LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by statelicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package volcano

import (
	"fmt"

	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	schedulerinterface "github.com/spotify/flink-on-k8s-operator/internal/batchscheduler/types"
	"github.com/spotify/flink-on-k8s-operator/internal/model"
)

const (
	schedulerName      = "volcano"
	podGroupNameFormat = "flink-%s"
)

// volcano scheduler implements the BatchScheduler interface.
type VolcanoBatchScheduler struct {
	volcanoClient volcanoclient.Interface
}

// Create volcano BatchScheduler
func New() (schedulerinterface.BatchScheduler, error) {
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}

	vcClient, err := volcanoclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize volcano client with error %v", err)
	}
	return &VolcanoBatchScheduler{
		volcanoClient: vcClient,
	}, nil
}

// Name returns the current scheduler name.
func (v *VolcanoBatchScheduler) Name() string {
	return schedulerName
}

// Schedule reconciles batch scheduling
func (v *VolcanoBatchScheduler) Schedule(
	options schedulerinterface.SchedulerOptions,
	state *model.DesiredClusterState) error {
	pg, err := v.syncPodGroup(options, state)
	if err != nil {
		return err
	}

	if pg != nil {
		v.setSchedulerMeta(pg, state)
	}
	return nil
}

func (v *VolcanoBatchScheduler) setSchedulerMeta(pg *scheduling.PodGroup, state *model.DesiredClusterState) {
	setMeta := func(podTemplateSpec *corev1.PodTemplateSpec) {
		if podTemplateSpec != nil {
			podTemplateSpec.Spec.SchedulerName = v.Name()
			podTemplateSpec.Spec.PriorityClassName = pg.Spec.PriorityClassName
			if podTemplateSpec.Annotations == nil {
				podTemplateSpec.Annotations = make(map[string]string)
			}
			podTemplateSpec.Annotations[scheduling.KubeGroupNameAnnotationKey] = pg.Name
		}
	}

	if state.TmStatefulSet != nil {
		setMeta(&state.TmStatefulSet.Spec.Template)
	}
	if state.JmStatefulSet != nil {
		setMeta(&state.JmStatefulSet.Spec.Template)
	}
	if state.Job != nil {
		setMeta(&state.Job.Spec.Template)
	}
}

func (v *VolcanoBatchScheduler) getPodGroup(podGroupName, namespace string) (*scheduling.PodGroup, error) {
	return v.volcanoClient.
		SchedulingV1beta1().
		PodGroups(namespace).
		Get(context.TODO(), podGroupName, metav1.GetOptions{})
}

func (v *VolcanoBatchScheduler) createPodGroup(pg *scheduling.PodGroup) (*scheduling.PodGroup, error) {
	return v.volcanoClient.
		SchedulingV1beta1().
		PodGroups(pg.Namespace).
		Create(context.TODO(), pg, metav1.CreateOptions{})
}

func (v *VolcanoBatchScheduler) updatePodGroup(pg *scheduling.PodGroup) (*scheduling.PodGroup, error) {
	return v.volcanoClient.
		SchedulingV1beta1().
		PodGroups(pg.Namespace).
		Update(context.TODO(), pg, metav1.UpdateOptions{})
}

func (v *VolcanoBatchScheduler) deletePodGroup(podGroupName, namespace string) error {
	return v.volcanoClient.
		SchedulingV1beta1().
		PodGroups(namespace).
		Delete(context.TODO(), podGroupName, metav1.DeleteOptions{})
}

func (v *VolcanoBatchScheduler) syncPodGroup(
	options schedulerinterface.SchedulerOptions,
	state *model.DesiredClusterState) (*scheduling.PodGroup, error) {
	podGroupName := fmt.Sprintf(podGroupNameFormat, options.ClusterName)
	namespace := options.ClusterNamespace

	if state.JmStatefulSet == nil && state.TmStatefulSet == nil {
		// remove the podgroup if the JobManager/TaskManager statefulset are not set
		err := v.deletePodGroup(podGroupName, namespace)
		if !errors.IsNotFound(err) {
			return nil, err
		}

		return nil, nil
	}

	resourceRequirements, size := getClusterResource(state)
	pg, err := v.getPodGroup(podGroupName, namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		pg := scheduling.PodGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:       namespace,
				Name:            podGroupName,
				OwnerReferences: options.OwnerReferences,
			},
			Spec: scheduling.PodGroupSpec{
				MinMember:         size,
				MinResources:      buildMinResource(resourceRequirements),
				Queue:             options.Queue,
				PriorityClassName: options.PriorityClassName,
			},
		}
		return v.createPodGroup(&pg)
	}

	if pg.Spec.MinMember != size {
		pg.Spec.MinMember = size
		return v.updatePodGroup(pg)
	}

	return pg, nil
}

func getClusterResource(state *model.DesiredClusterState) (*corev1.ResourceRequirements, int32) {
	reqs := &corev1.ResourceRequirements{
		Limits:   map[corev1.ResourceName]resource.Quantity{},
		Requests: map[corev1.ResourceName]resource.Quantity{},
	}
	var size int32

	if state.JmStatefulSet != nil {
		spec := state.JmStatefulSet.Spec
		size += *spec.Replicas
		resources := getStatefulSetResources(&spec)
		addResourceRequirements(reqs, resources)
	}

	if state.TmStatefulSet != nil {
		spec := state.TmStatefulSet.Spec
		size += *spec.Replicas
		resources := getStatefulSetResources(&spec)
		addResourceRequirements(reqs, resources)
	}

	if state.Job != nil {
		size += 1
		jobResource := getPodResource(&state.Job.Spec.Template.Spec)
		addResourceRequirements(reqs, jobResource)
	}

	return reqs, size
}

func getStatefulSetResources(spec *appsv1.StatefulSetSpec) *corev1.ResourceRequirements {
	reqs := &corev1.ResourceRequirements{
		Limits:   map[corev1.ResourceName]resource.Quantity{},
		Requests: map[corev1.ResourceName]resource.Quantity{},
	}

	for i := int32(0); i < *spec.Replicas; i++ {
		tmResource := getPodResource(&spec.Template.Spec)
		addResourceRequirements(reqs, tmResource)

		for _, pvc := range spec.VolumeClaimTemplates {
			addResourceRequirements(reqs, &pvc.Spec.Resources)
		}
	}

	return reqs
}

func getPodResource(spec *corev1.PodSpec) *corev1.ResourceRequirements {
	reqs := &corev1.ResourceRequirements{
		Limits:   map[corev1.ResourceName]resource.Quantity{},
		Requests: map[corev1.ResourceName]resource.Quantity{},
	}
	for _, container := range spec.Containers {
		addResourceRequirements(reqs, &container.Resources)
	}
	return reqs
}

func addResourceRequirements(acc, req *corev1.ResourceRequirements) {
	for name, quantity := range req.Requests {
		if value, ok := acc.Requests[name]; !ok {
			acc.Requests[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			acc.Requests[name] = value
		}
	}

	for name, quantity := range req.Limits {
		if value, ok := acc.Limits[name]; !ok {
			acc.Limits[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			acc.Limits[name] = value
		}
	}
}

func buildMinResource(req *corev1.ResourceRequirements) *corev1.ResourceList {
	minResource := corev1.ResourceList{}
	for name, quantity := range req.Requests {
		minResource[name] = quantity.DeepCopy()
		n := corev1.ResourceName(fmt.Sprintf("requests.%s", name))
		minResource[n] = quantity.DeepCopy()
	}

	for name, quantity := range req.Limits {
		n := corev1.ResourceName(fmt.Sprintf("limits.%s", name))
		minResource[n] = quantity.DeepCopy()
	}

	return &minResource
}
