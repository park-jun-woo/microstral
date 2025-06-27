package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSecret: Secret 가져오기
func (c *Client) GetSecret(name string, namespace ...string) (*corev1.Secret, error) {
	ns := c.namespace
	if len(namespace) > 0 && namespace[0] != "" {
		ns = namespace[0]
	}

	secret, err := c.clientset.CoreV1().Secrets(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// SetSecret: Secret이 없으면 생성, 있으면 업데이트
func (c *Client) SetSecret(name string, data map[string][]byte, namespace ...string) (*corev1.Secret, error) {
	ns := c.namespace
	if len(namespace) > 0 && namespace[0] != "" {
		ns = namespace[0]
	}

	ctx := context.Background()

	// Secret 조회
	secret, err := c.clientset.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		// NotFound 에러면 새로 생성
		if k8serrors.IsNotFound(err) {
			newSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: ns,
				},
				Data: data,
				Type: corev1.SecretTypeOpaque, // 기본 Opaque 타입
			}

			created, createErr := c.clientset.CoreV1().Secrets(ns).Create(ctx, newSecret, metav1.CreateOptions{})
			if createErr != nil {
				return nil, createErr
			}
			return created, nil
		}
		// 그 외 에러면 바로 반환
		return nil, err
	}

	// 이미 Secret이 존재하면 데이터 덮어쓰고 업데이트
	secret.Data = data
	secret.Type = corev1.SecretTypeOpaque // 필요 시 기존 Type 유지 가능
	updated, updateErr := c.clientset.CoreV1().Secrets(ns).Update(ctx, secret, metav1.UpdateOptions{})
	if updateErr != nil {
		return nil, updateErr
	}

	return updated, nil
}
