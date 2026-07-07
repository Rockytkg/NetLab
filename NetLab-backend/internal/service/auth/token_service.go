package auth

import (
	"context"
	"encoding/hex"
	"time"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/pkg/apperrors"
	cryptopkg "netlab-backend/pkg/crypto"
	pkgjwt "netlab-backend/pkg/jwt"
)

// TokenService 处理 JWT 生命周期与会话密钥管理。
type TokenService struct {
	jwtManager     *pkgjwt.Manager
	tokenRepo      *repository.TokenRepository
	userRepo       *repository.UserRepository
	accessExpiry   time.Duration
	refreshExpiry  time.Duration
	sessionKeyTTL  time.Duration
}

// NewTokenService 创建一个新的 TokenService。
func NewTokenService(
	cfg config.JWTConfig,
	tokenRepo *repository.TokenRepository,
	userRepo *repository.UserRepository,
) *TokenService {
	return &TokenService{
		jwtManager: pkgjwt.NewManager(
			cfg.AccessSecret,
			cfg.RefreshSecret,
			cfg.AccessExpiry,
			cfg.RefreshExpiry,
			cfg.Issuer,
		),
		tokenRepo:     tokenRepo,
		userRepo:      userRepo,
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
		sessionKeyTTL: 24 * time.Hour,
	}
}

// JWTManager 返回底层的 JWT manager。
func (s *TokenService) JWTManager() *pkgjwt.Manager {
	return s.jwtManager
}

// IssueTokens 为用户创建 access + refresh token 以及会话密钥。
func (s *TokenService) IssueTokens(ctx context.Context, user *model.User) (*TokenResult, *apperrors.AppError) {
	pair, err := s.jwtManager.IssueTokenPair(user.ID.String(), user.Username, user.Roles)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to issue tokens", err)
	}

	if err := s.tokenRepo.SaveRefreshToken(ctx, user.ID.String(), pair.RefreshToken, pair.RefreshExpiry); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to save refresh token", err)
	}

	signingKey, err := cryptopkg.GenerateRandomKeyHex()
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to generate signing key", err)
	}

	if err := s.tokenRepo.SaveSessionKeys(ctx, user.ID.String(), signingKey, s.sessionKeyTTL); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to save session keys", err)
	}

	return &TokenResult{
		AccessToken:   pair.AccessToken,
		RefreshToken:  pair.RefreshToken,
		SigningKey:    signingKey,
		AccessExpiry:  pair.AccessExpiry,
		RefreshExpiry: pair.RefreshExpiry,
	}, nil
}

// RefreshTokens 校验 refresh token 并通过轮换签发一对新 token。
//
// 该方法实现了自动重用检测：如果出示的是一个之前已被轮换的 token，
// 则表明可能发生了 token 窃取。在这种情况下，该用户的所有 refresh
// token 都会被撤销，强制其重新完整认证。
func (s *TokenService) RefreshTokens(ctx context.Context, refreshTokenValue string) (*TokenResult, *apperrors.AppError) {
	// ── 重用检测：检查此 token 是否已被轮换 ──
	reused, err := s.tokenRepo.IsRefreshTokenReused(ctx, refreshTokenValue)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidRefreshToken, "failed to check token reuse", err)
	}
	if reused {
		// 检测到 token 重用 —— 可能发生窃取！
		// 解析 token 以识别受害用户，然后撤销其所有 token。
		claims, parseErr := s.jwtManager.ParseRefreshToken(refreshTokenValue)
		if parseErr == nil && claims != nil {
			_ = s.tokenRepo.RevokeAllUserTokens(ctx, claims.UserID)
			_ = s.tokenRepo.DeleteSessionKeys(ctx, claims.UserID)
		}
		return nil, apperrors.ErrInvalidRefreshToken
	}

	// ── 检查 token 是否已被撤销 ──
	revoked, err := s.tokenRepo.IsRefreshTokenRevoked(ctx, refreshTokenValue)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidRefreshToken, "failed to check token status", err)
	}
	if revoked {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	// ── 解析并校验 token ──
	claims, err := s.jwtManager.ParseRefreshToken(refreshTokenValue)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return nil, apperrors.ErrAccountDisabled
	}

	// ── 将旧 token 标记为已使用（用于重用检测）并撤销它 ──
	_ = s.tokenRepo.MarkRefreshTokenUsed(ctx, refreshTokenValue, s.refreshExpiry)
	_ = s.tokenRepo.RevokeRefreshToken(ctx, refreshTokenValue)

	return s.IssueTokens(ctx, user)
}

// RevokeTokens 撤销某个用户的所有 token 并清除会话密钥。
func (s *TokenService) RevokeTokens(ctx context.Context, userID, jti string, tokenExp time.Time) error {
	if jti != "" {
		_ = s.tokenRepo.AddToBlacklist(ctx, jti, tokenExp)
	}
	if err := s.tokenRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		return err
	}
	_ = s.tokenRepo.DeleteSessionKeys(ctx, userID)
	return nil
}

// TokenResult 包含登录/刷新后返回的 token 和会话签名密钥。
type TokenResult struct {
	AccessToken   string
	RefreshToken  string
	SigningKey    string
	AccessExpiry  time.Time
	RefreshExpiry time.Time
}

// GetSessionSigningKey 获取某个用户会话的原始签名密钥字节。
// 如果未存储会话密钥（例如遗留会话）则返回 nil。
//
// 存储的密钥采用十六进制编码（参见 GenerateRandomKeyHex）。在
// 十六进制迁移之前创建的会话持有 base64 密钥；这些会话会在部署时
// 被清除，因此此处无需 base64 兜底。
func (s *TokenService) GetSessionSigningKey(ctx context.Context, userID string) ([]byte, error) {
	signingKeyHex, err := s.tokenRepo.GetSessionKeys(ctx, userID)
	if err != nil || signingKeyHex == "" {
		return nil, err
	}
	return hex.DecodeString(signingKeyHex)
}
