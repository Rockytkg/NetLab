package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Manager 负责 JWT 的创建、解析和校验。
type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	issuer        string
}

// Claims 表示 access token 的 JWT claims。
type Claims struct {
	jwt.RegisteredClaims
	UserID    string `json:"uid"`
	Username  string `json:"uname"`
	Role      string `json:"role"`
	SessionID string `json:"sid"`
	TokenType string `json:"typ"` // "access" 或 "refresh"
}

// NewManager 创建一个新的 JWT Manager。
func NewManager(accessSecret, refreshSecret string, accessExpiry time.Duration, issuer string) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  accessExpiry,
		issuer:        issuer,
	}
}

// TokenPair 包含一个 access token 和一个 refresh token。
type TokenPair struct {
	AccessToken   string    `json:"accessToken"`
	RefreshToken  string    `json:"refreshToken"`
	AccessExpiry  time.Time `json:"accessExpiry"`
	RefreshExpiry time.Time `json:"refreshExpiry"`
}

// IssueAccessToken 为给定用户创建一个新的 access JWT。
func (m *Manager) IssueAccessToken(userID, username, role, sessionID string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessExpiry)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:    userID,
		Username:  username,
		Role:      role,
		SessionID: sessionID,
		TokenType: "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.accessSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, expiresAt, nil
}

// IssueRefreshTokenUntil 创建一个在指定时间点过期的 refresh JWT。
func (m *Manager) IssueRefreshTokenUntil(userID, sessionID string, expiresAt time.Time) (string, time.Time, error) {
	now := time.Now()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:    userID,
		SessionID: sessionID,
		TokenType: "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.refreshSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return signed, expiresAt, nil
}

// IssueTokenPairUntil 同时创建 access token 和指定过期点的 refresh token。
func (m *Manager) IssueTokenPairUntil(userID, username, role, sessionID string, refreshExpiresAt time.Time) (*TokenPair, error) {
	accessToken, accessExp, err := m.IssueAccessToken(userID, username, role, sessionID)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshExp, err := m.IssueRefreshTokenUntil(userID, sessionID, refreshExpiresAt)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		AccessExpiry:  accessExp,
		RefreshExpiry: refreshExp,
	}, nil
}

// ParseAccessToken 解析并校验一个 access JWT。
func (m *Manager) ParseAccessToken(tokenString string) (*Claims, error) {
	claims, err := m.parseToken(tokenString, m.accessSecret)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "access" {
		return nil, fmt.Errorf("unexpected token type: %s", claims.TokenType)
	}
	if claims.SessionID == "" {
		return nil, fmt.Errorf("missing session id")
	}
	return claims, nil
}

// ParseRefreshToken 解析并校验一个 refresh JWT。
func (m *Manager) ParseRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.parseToken(tokenString, m.refreshSecret)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("unexpected token type: %s", claims.TokenType)
	}
	if claims.SessionID == "" {
		return nil, fmt.Errorf("missing session id")
	}
	return claims, nil
}

func (m *Manager) parseToken(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// 校验 access token 的令牌类型
	if claims.TokenType != "access" && claims.TokenType != "refresh" {
		return nil, fmt.Errorf("unknown token type: %s", claims.TokenType)
	}

	return claims, nil
}

// SessionValidator validates the Redis-backed single active login session.
type SessionValidator interface {
	// IsSessionActive returns true only when sid is the user's current session.
	IsSessionActive(ctx context.Context, userID, sessionID string) (bool, error)
}
