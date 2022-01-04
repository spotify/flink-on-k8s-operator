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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"

	"github.com/spotify/flink-on-k8s-operator/api/v1beta1"
	schedulerinterface "github.com/spotify/flink-on-k8s-operator/controllers/batchscheduler/interface"
	"github.com/spotify/flink-on-k8s-operator/controllers/model"
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
func (v *VolcanoBatchScheduler) Schedule(cluster *v1beta1.FlinkCluster, state *model.DesiredClusterState) error {
	res, size := getClusterResource(state)
	pg, err := v.syncPodGroup(cluster, size, res)
	if err != nil {
		return err
	}
	v.setSchedulerMeta(pg, state)
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

func (v *VolcanoBatchScheduler) getPodGroupName(cluster *v1beta1.FlinkCluster) string {
	return fmt.Sprintf(podGroupNameFormat, cluster.Name)
}

// Converts the FlinkCluster as owner reference for its child resources.
func newOwnerReference(flinkCluster *v1beta1.FlinkCluster) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         flinkCluster.APIVersion,
		Kind:               flinkCluster.Kind,
		Name:               flinkCluster.Name,
		UID:                flinkCluster.UID,
		Controller:         &[]bool{true}[0],
		BlockOwnerDeletion: &[]bool{false}[0],
	}
}

func (v *VolcanoBatchScheduler) getPodGroup(cluster *v1beta1.FlinkCluster) (*scheduling.PodGroup, error) {
	podGroupName := v.getPodGroupName(cluster)
	return v.volcanoClient.
		SchedulingV1beta1().
		PodGroups(cluster.Namespace).
		Get(context.TODO(), podGroupName, metav1.GetOptions{})
}

func (v *VolcanoBatchScheduler) createPodGroup(cluster *v1beta1.FlinkCluster, size int32, minResource corev1.ResourceList) (*scheduling.PodGroup, error) {
	pg := scheduling.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       cluster.Namespace,
			Name:            v.getPodGroupName(cluster),
			OwnerReferences: []metav1.OwnerReference{newOwnerReference(cluster)},
		},
		Spec: scheduling.PodGroupSpec{
			MinMember:         size,
			MinResources:      &minResource,
			Queue:             cluster.Spec.BatchScheduler.Queue,
			PriorityClassName: cluster.Spec.BatchScheduler.PriorityClassName,
		},
	}

	return v.volcanoClient.
		SchedulingV1beta1().
		PodGroups(pg.Namespace).
		Create(context.TODO(), &pg, metav1.CreateOptions{})
}

func (v *VolcanoBatchScheduler) updatePodGroup(pg *scheduling.PodGroup) (*scheduling.PodGroup, error) {
	return v.volcanoClient.
		SchedulingV1beta1().
		PodGroups(pg.Namespace).
		Update(context.TODO(), pg, metav1.UpdateOptions{})
}

func (v *VolcanoBatchScheduler) syncPodGroup(cluster *v1beta1.FlinkCluster, size int32, minResource corev1.ResourceList) (*scheduling.PodGroup, error) {
	pg, err := v.getPodGroup(cluster)
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		return v.createPodGroup(cluster, size, minResource)
	}

	if pg.Spec.MinMember != size {
		pg.Spec.MinMember = size
		return v.updatePodGroup(pg)
	}

	return pg, nil
}

func getClusterResource(state *model.DesiredClusterState) (corev1.ResourceList, int32) {
	resource := corev1.ResourceList{}
	var size int32

	if state.JmStatefulSet != nil {
		size += *state.JmStatefulSet.Spec.Replicas
		for i := int32(0); i < *state.JmStatefulSet.Spec.Replicas; i++ {
			jmResource := getPodResource(&state.JmStatefulSet.Spec.Template.Spec)
			addResourceList(resource, jmResource, nil)
		}
	}

	if state.TmStatefulSet != nil {
		size += *state.TmStatefulSet.Spec.Replicas
		for i := int32(0); i < *state.TmStatefulSet.Spec.Replicas; i++ {
			tmResource := getPodResource(&state.TmStatefulSet.Spec.Template.Spec)
			addResourceList(resource, tmResource, nil)
		}
	}

	if state.Job != nil {
		size += 1
		jobResource := getPodResource(&state.Job.Spec.Template.Spec)
		addResourceList(resource, jobResource, nil)
	}

	return resource, size
}

func getPodResource(spec *corev1.PodSpec) corev1.ResourceList {
	resource := corev1.ResourceList{}
	for _, container := range spec.Containers {
		addResourceList(resource, container.Resources.Requests, container.Resources.Limits)
	}
	return resource
}

func addResourceList(list, req, limit corev1.ResourceList) {
	for name, quantity := range req {

		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			list[name] = value
		}
	}

	// If Requests is omitted for a container,
	// it defaults to Limits if that is explicitly specified.
	for name, quantity := range limit {
		if _, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
		}
	}
}
