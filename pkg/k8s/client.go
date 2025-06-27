package k8s

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"parkjunwoo.com/microstral/pkg/file"
)

type Client struct {
	clientset *kubernetes.Clientset
	namespace string
	podName   string
}

func NewClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	podName := os.Getenv("HOSTNAME")

	nsBytes, err := file.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, err
	}

	return &Client{
		clientset: clientset,
		podName:   podName,
		namespace: string(nsBytes),
	}, nil
}

func (c *Client) GetClientset() *kubernetes.Clientset {
	return c.clientset
}

func (c *Client) GetNamespace() string {
	return c.namespace
}

// GetPodName은 HOSTNAME 환경 변수로부터 현재 Pod의 이름을 반환합니다.
func (c *Client) GetPodName() string {
	return c.podName
}

func (c *Client) GetServiceAccountToken() (string, error) {
	saTokenBytes, err := file.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", fmt.Errorf("failed to read ServiceAccountToken: %v", err)
	}
	return string(saTokenBytes), nil
}
