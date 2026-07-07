package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Manager 负责 JWT 的创建、解析和校验。
type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// Claims 表示 access token 的 JWT claims。
type Claims struct {
	jwt.RegisteredClaims
	UserID   string   `json:"uid"`
	Username string   `json:"uname"`
	Roles    []string `json:"roles"`
	TokenType string  `json:"typ"` // "access" 或 "refresh"
}

// NewManager 创建一个新的 JWT Manager。
func NewManager(accessSecret, refreshSecret string, accessExpiry, refreshExpiry time.Duration, issuer string) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        issuer,
	}
}

// TokenPair 包含一个 access token 和一个 refresh token。
type TokenPair struct {
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	AccessExpiry  time.Time `json:"access_expiry"`
	RefreshExpiry time.Time `json:"refresh_expiry"`
}

// IssueAccessToken 为给定用户创建一个新的 access JWT。
func (m *Manager) IssueAccessToken(userID, username string, roles []string) (string, string, time.Time, error) { //nolint:unparam
	now := time.Now()
	expiresAt := now.Add(m.accessExpiry)
	jti := uuid.New().String()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:    userID,
		Username:  username,
		Roles:     roles,
		TokenType: "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.accessSecret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, jti, expiresAt, nil
}

// IssueRefreshToken 创建一个新的 refresh JWT。
func (m *Manager) IssueRefreshToken(userID string) (string, string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshExpiry)
	jti := uuid.New().String()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:    userID,
		TokenType: "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.refreshSecret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return signed, jti, expiresAt, nil
}

// IssueTokenPair 同时创建 access 和 refresh token。
func (m *Manager) IssueTokenPair(userID, username string, roles []string) (*TokenPair, error) {
	accessToken, _, accessExp, err := m.IssueAccessToken(userID, username, roles)
	if err != nil {
		return nil, err
	}

	refreshToken, _, refreshExp, err := m.IssueRefreshToken(userID)
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
	return m.parseToken(tokenString, m.accessSecret)
}

// ParseRefreshToken 解析并校验一个 refresh JWT。
func (m *Manager) ParseRefreshToken(tokenString string) (*Claims, error) {
	return m.parseToken(tokenString, m.refreshSecret)
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

// BlacklistManager 通过 Redis 处理 JWT 黑名单。
type BlacklistManager interface {
	// AddToBlacklist 将一个 JTI 加入黑名单，其 TTL 与令牌的剩余生命周期一致。
	AddToBlacklist(ctx context.Context, jti string, expiresAt time.Time) error
	// IsBlacklisted 检查某个 JTI 是否在黑名单中。
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

