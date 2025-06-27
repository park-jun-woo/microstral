package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// CreateSignedCert는 "부모 CA"(Root이든 Middle이든)로부터 서명받아
// 새 "서버/클라이언트" 용 인증서를 (certPEM, keyPEM, error)로 반환합니다.
//
// dnsName, organizationName, expire(예: "24h")를 지정하면,
// Leaf 인증서를 발급받을 수 있습니다. (IsCA=false)
func CreateSignedCert(dnsName string, organizationName string, expire string, parentCertPEM string, parentKeyPEM string, bits ...int) (string, string, error) {
	// 1) bits 인자 처리 (기본값 4096)
	switch len(bits) {
	case 0:
		bits = append(bits, 4096)
	case 1:
		// 그대로 사용
	default:
		return "", "", fmt.Errorf("too many arguments")
	}

	// 2) 상위 CA(cert) 파싱
	block, _ := pem.Decode([]byte(parentCertPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return "", "", fmt.Errorf("failed to parse parent CA certificate PEM")
	}
	parentCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", "", err
	}

	// (옵션) 부모 CA가 IsCA=true인지 확인
	if !parentCert.IsCA {
		return "", "", fmt.Errorf("the parent certificate is not a CA (IsCA=false)")
	}

	// 3) 상위 CA(key) 파싱
	block, _ = pem.Decode([]byte(parentKeyPEM))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return "", "", fmt.Errorf("failed to parse parent CA key PEM")
	}
	parentKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", "", err
	}

	// 4) 새로 발급할 "leaf" 인증서의 개인키 생성
	leafPriv, err := rsa.GenerateKey(rand.Reader, bits[0])
	if err != nil {
		return "", "", err
	}

	// 5) 만료 기간
	expireTime, err := time.ParseDuration(expire)
	if err != nil {
		return "", "", err
	}

	// 6) 시리얼 넘버
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	// 7) "서버/클라이언트" 인증서 템플릿(IsCA=false)
	leafTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   dnsName,
			Organization: []string{organizationName},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(expireTime),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false, // leaf cert
		DNSNames:              []string{dnsName},
	}

	// 8) 부모 CA로 서명
	leafDER, err := x509.CreateCertificate(
		rand.Reader,
		&leafTemplate,
		parentCert,
		&leafPriv.PublicKey,
		parentKey,
	)
	if err != nil {
		return "", "", err
	}

	// 9) PEM 인코딩
	leafCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: leafDER,
	})
	leafKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(leafPriv),
	})

	// 반환
	return string(leafCertPEM), string(leafKeyPEM), nil
}

// CreateMiddleCert는 rootCertPEM/rootKeyPEM(루트 CA)로부터 서명받아
// 새 “중간 CA” (Intermediate CA) 인증서를 발급합니다.
//
// dnsName, organizationName, expire(예: "24h") 지정 시,
// 중간 CA 인증서를 (certPEM, keyPEM, error) 형태로 반환합니다.
func CreateMiddleCert(dnsName string, organizationName string, expire string,
	rootCertPEM string, rootKeyPEM string, bits ...int) (string, string, error) {

	// 1) bits 인자 처리 (디폴트 4096 등)
	switch len(bits) {
	case 0:
		bits = append(bits, 4096)
	case 1:
		// 그대로 사용
	default:
		return "", "", fmt.Errorf("too many arguments")
	}

	// 2) 상위(루트) CA 인증서와 키 파싱
	block, _ := pem.Decode([]byte(rootCertPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return "", "", fmt.Errorf("failed to parse root CA certificate PEM")
	}
	rootCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", "", err
	}

	block, _ = pem.Decode([]byte(rootKeyPEM))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return "", "", fmt.Errorf("failed to parse root CA key PEM")
	}
	rootKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", "", err
	}

	// 3) 새로 발급할 “중간 CA” 용 RSA 키 생성
	priv, err := rsa.GenerateKey(rand.Reader, bits[0])
	if err != nil {
		return "", "", err
	}

	// 4) 만료 기간 파싱
	expireTime, err := time.ParseDuration(expire)
	if err != nil {
		return "", "", err
	}

	// 5) 시리얼 번호 생성
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	// 6) 중간 CA 인증서 템플릿 (IsCA = true, KeyUsageCertSign 등)
	certTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   dnsName,
			Organization: []string{organizationName},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(expireTime),

		// CA 역할, 서명 가능
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign |
			x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           nil, // 중간 CA이므로 서버/클라이언트 용도는 지정 X (optional)
		BasicConstraintsValid: true,
		IsCA:                  true,

		// MaxPathLen: 원하는 CA 체인의 길이 조절
		// MaxPathLenZero: ...
	}

	// 7) “루트 CA”로 서명 (template, parent, pubKey, privKey)
	interDER, err := x509.CreateCertificate(rand.Reader, &certTemplate, rootCert,
		&priv.PublicKey, rootKey)
	if err != nil {
		return "", "", err
	}

	// 8) PEM 인코딩
	interCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: interDER,
	})
	interKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	// 9) 반환
	return string(interCertPEM), string(interKeyPEM), nil
}

// CreateRoot는 자체 서명된 Root CA(루트 인증서 + 개인키)를 생성합니다.
// dnsName, organizationName, expire(예: "24h", "8760h" 등) 를 받아,
// CA 용 인증서와 개인키를 PEM 형식으로 반환합니다.
func CreateRoot(dnsName string, organizationName string, expire string, bits ...int) (string, string, error) {
	switch len(bits) {
	case 0:
		bits = append(bits, 4096)
	case 1:

	default:
		return "", "", fmt.Errorf("too many arguments")
	}
	// 개인키 생성 (루트 CA용)
	caPriv, err := rsa.GenerateKey(rand.Reader, bits[0])
	if err != nil {
		return "", "", err
	}

	// 만료 기간 설정
	expireTime, err := time.ParseDuration(expire)
	if err != nil {
		return "", "", err
	}

	// SerialNumber 랜덤생성
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	// 루트 CA 인증서 템플릿
	caTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   dnsName,
			Organization: []string{organizationName},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(expireTime),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           nil, // 루트 CA이므로 ExtKeyUsage 생략 (서버/클라이언트 목적 X)
		BasicConstraintsValid: true,
		IsCA:                  true,  // CA 권한
		MaxPathLen:            2,     // 필요에 따라 조정 가능
		MaxPathLenZero:        false, // 필요에 따라 조정 가능
	}

	// 루트 CA 자체 서명
	caDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPriv.PublicKey, caPriv)
	if err != nil {
		return "", "", err
	}

	// CA 인증서 PEM 인코딩
	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caDER,
	})

	// CA 개인키 PEM 인코딩
	caKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPriv),
	})

	return string(caCertPEM), string(caKeyPEM), nil
}
