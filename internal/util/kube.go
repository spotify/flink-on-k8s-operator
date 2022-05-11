package util

import (
	"bytes"
	"context"
	"fmt"
	"io"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func GetPodLogs(clientset *kubernetes.Clientset, pod *corev1.Pod) (string, error) {
	if pod == nil {
		return "", fmt.Errorf("no job pod found, even though submission completed")
	}
	pods := clientset.CoreV1().Pods(pod.Namespace)

	req := pods.GetLogs(pod.Name, &corev1.PodLogOptions{Container: "main"})
	podLogs, err := req.Stream(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to get logs for pod %s: %v", pod.Name, err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error in copy information from pod logs to buf")
	}
	str := buf.String()

	return str, nil
}

func GetNextRevisionNumber(revisions []*appsv1.ControllerRevision) int64 {
	count := len(revisions)
	if count <= 0 {
		return 1
	}
	return revisions[count-1].Revision + 1
}

// Compose revision in FlinkClusterStatus with name and number of ControllerRevision
func GetRevisionWithNameNumber(cr *appsv1.ControllerRevision) string {
	return fmt.Sprintf("%v-%v", cr.Name, cr.Revision)
}

func GetNonLiveHistory(revisions []*appsv1.ControllerRevision, historyLimit int) []*appsv1.ControllerRevision {

	history := append([]*appsv1.ControllerRevision{}, revisions...)
	nonLiveHistory := make([]*appsv1.ControllerRevision, 0)

	historyLen := len(history)
	if historyLen <= historyLimit {
		return nonLiveHistory
	}

	nonLiveHistory = append(nonLiveHistory, history[:(historyLen-historyLimit)]...)
	return nonLiveHistory
}
