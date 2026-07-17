package jwt

import (
	"crypto/rsa"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type Signer interface {
	Sign(claims *Claims, secret []byte) (string, error)
	Verify(tokenString string, claims *Claims, secret []byte) (*Claims, error)
}

type hs256Signer struct{}

func (hs256Signer) Sign(claims *Claims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func (hs256Signer) Verify(tokenString string, claims *Claims, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	parsed, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return parsed, nil
}

type rs256Signer struct {
	privateKey interface{}
	publicKey  interface{}
}

func (s *rs256Signer) Sign(claims *Claims, _ []byte) (string, error) {
	key, ok := s.privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("invalid RSA private key")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(key)
}

func (s *rs256Signer) Verify(tokenString string, claims *Claims, _ []byte) (*Claims, error) {
	key, ok := s.publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid RSA public key")
	}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	parsed, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return parsed, nil
}

func NewSigner(mode, privateKeyPath, publicKeyPath string) (Signer, error) {
	switch mode {
	case "", "HS256":
		return hs256Signer{}, nil
	case "RS256":
		privatePEM, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read JWT private key: %w", err)
		}
		publicPEM, err := os.ReadFile(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read JWT public key: %w", err)
		}
		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
		if err != nil {
			return nil, fmt.Errorf("parse JWT private key: %w", err)
		}
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicPEM)
		if err != nil {
			return nil, fmt.Errorf("parse JWT public key: %w", err)
		}
		return &rs256Signer{privateKey: privateKey, publicKey: publicKey}, nil
	default:
		return nil, fmt.Errorf("unsupported JWT signing mode: %s", mode)
	}
}
