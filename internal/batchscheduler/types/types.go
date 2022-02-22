package types

import (
	"github.com/spotify/flink-on-k8s-operator/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BatchScheduler is the general batch scheduler interface.
type BatchScheduler interface {
	// Name gets the name of the scheduler
	Name() string
	// Schedule reconciles batch scheduling
	Schedule(options SchedulerOptions, desired *model.DesiredClusterState) error
}

type SchedulerOptions struct {
	ClusterName       string
	ClusterNamespace  string
	Queue             string
	PriorityClassName string
	OwnerReferences   []metav1.OwnerReference
}
