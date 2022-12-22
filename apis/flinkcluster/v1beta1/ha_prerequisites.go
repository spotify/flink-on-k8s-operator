package v1beta1

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
)

func saHasConfigMapUpdatePermissions(saName string, namespace string) (bool, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return false, err
	}

	config.Impersonate = rest.ImpersonationConfig{
		UserName: fmt.Sprintf("system:serviceaccount:%s:%s", namespace, saName),
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return false, err
	}

	action := authv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      "*",
		Resource:  "configmaps",
	}

	selfCheck := authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &action,
		},
	}

	resp, err := clientset.AuthorizationV1().
		SelfSubjectAccessReviews().
		Create(context.TODO(), &selfCheck, metav1.CreateOptions{})

	if err != nil {
		return false, err
	}

	if resp.Status.Allowed {
		return true, nil
	} else {
		return false, nil
	}
}
