package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodList는 지정된 서비스의 Pod 목록을 조회합니다.
func (c *Client) PodList(serviceName string, namespace ...string) (*corev1.PodList, error) {
	ns := ""
	if len(namespace) > 0 && namespace[0] != "" {
		ns = namespace[0]
	}
	if ns == "" {
		ns = c.namespace
	}

	// 먼저 서비스 객체를 조회하여, selector 정보를 얻습니다.
	svc, err := c.clientset.CoreV1().Services(ns).Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// 서비스에 지정된 selector를 이용해 Pod 목록을 조회합니다.
	// svc.Spec.Selector는 map[string]string 타입이므로, 이를 label selector 문자열로 변환합니다.
	selector := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: svc.Spec.Selector})

	podList, err := c.clientset.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	return podList, nil
}
