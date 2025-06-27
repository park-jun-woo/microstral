package k8s

import (
	"context"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) TokenReview(token string) (*authenticationv1.TokenReview, error) {
	// 토큰이 비어 있는지 확인
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}

	// TokenReview 객체 생성: 여기서 Spec에 검증할 토큰을 설정합니다.
	tokenReview := &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{
			Token: token,
		},
	}

	// TokenReview 요청 보내기: 클러스터의 인증 API를 호출하여 토큰 검증 결과를 받습니다.
	result, err := c.clientset.AuthenticationV1().TokenReviews().Create(context.Background(), tokenReview, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	return result, nil
}
