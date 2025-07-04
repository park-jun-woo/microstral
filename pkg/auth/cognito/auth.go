// parkjunwoo.com/microstral/pkg/auth/cognito/auth.go
package cognito

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"parkjunwoo.com/microstral/pkg/auth"
	"parkjunwoo.com/microstral/pkg/env"
)

type Auth struct {
	Region             string
	UserPoolID         string
	ClientID           string
	ClientSecret       string
	SigninCallbackURI  string
	SignoutCallbackURI string
	ResponseType       string
	JWKSUrl            string

	jwks          *JWKS
	awsCfg        aws.Config
	cognitoClient *cognitoidentityprovider.Client
	once          sync.Once
}

func GetSecretValue(cfg aws.Config, secretName string) (string, error) {
	svc := secretsmanager.NewFromConfig(cfg)
	result, err := svc.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		return "", fmt.Errorf("unable to retrieve secret %s: %w", secretName, err)
	}
	return *result.SecretString, nil
}

func New(host, responseType string) (*Auth, error) {
	region := env.GetEnv("COGNITO_REGION", "")
	userPoolID := env.GetEnv("COGNITO_USERPOOL_ID", "")
	clientID := env.GetEnv("COGNITO_CLIENT_ID", "")
	clientSecretName := env.GetEnv("COGNITO_CLIENT_SECRET", "")
	signinCallbackURI := env.GetEnv("COGNITO_SIGNIN_CALLBACK", "")
	signoutCallbackURI := env.GetEnv("COGNITO_SIGNOUT_CALLBACK", "")

	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	clientSecret, err := GetSecretValue(awsCfg, clientSecretName)
	if err != nil {
		log.Printf("failed to get secret: %v", err)
		return nil, err
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(awsCfg)

	return &Auth{
		Region:             region,
		UserPoolID:         userPoolID,
		ClientID:           clientID,
		ClientSecret:       clientSecret,
		SigninCallbackURI:  signinCallbackURI,
		SignoutCallbackURI: signoutCallbackURI,
		ResponseType:       responseType,
		JWKSUrl: fmt.Sprintf(
			"https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json",
			region,
			userPoolID,
		),
		awsCfg:        awsCfg,
		cognitoClient: cognitoClient,
	}, nil
}

func (ca *Auth) issuer() string {
	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", ca.Region, ca.UserPoolID)
}

func (ca *Auth) domain() string {
	return fmt.Sprintf("%s.auth.%s.amazoncognito.com", ca.UserPoolID, ca.Region)
}

// JWT 기반 인증
func (ca *Auth) Authenticate(c *gin.Context) (auth.Claims, bool, error) {
	tokenStr := extractBearerToken(c.Request)
	if tokenStr == "" {
		return auth.Claims{
			Roles: []string{"Guest"},
		}, false, nil
	}

	token, err := jwt.Parse(tokenStr, ca.keyFunc)
	if err != nil || !token.Valid {
		return auth.Claims{}, false, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return auth.Claims{}, false, errors.New("invalid claims")
	}

	if claims["iss"] != ca.issuer() {
		return auth.Claims{}, false, errors.New("invalid issuer")
	}

	return auth.Claims{
		Usersub:  claims["sub"].(string),
		Username: claims["cognito:username"].(string),
		Name:     claims["name"].(string),
		Email:    claims["email"].(string),
		Roles:    parseRoles(claims),
	}, true, nil
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}

func (ca *Auth) keyFunc(token *jwt.Token) (interface{}, error) {
	ca.once.Do(func() {
		ca.jwks = fetchJWKS(ca.JWKSUrl)
	})

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("no kid found in token header")
	}

	return ca.jwks.getPublicKey(kid)
}

func parseRoles(claims jwt.MapClaims) []string {
	groups, ok := claims["cognito:groups"].([]interface{})
	if !ok {
		return []string{}
	}

	var roles []string
	for _, group := range groups {
		if role, ok := group.(string); ok {
			roles = append(roles, role)
		}
	}
	return roles
}
