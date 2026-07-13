package auth

import (
	"context"
	"time"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
	pkgjwt "netlab-backend/pkg/jwt"
)

// TokenService 处理 JWT 生命周期。
type TokenService struct {
	jwtManager            *pkgjwt.Manager
	tokenRepo             *repository.TokenRepository
	userRepo              *repository.UserRepository
	refreshExpiry         time.Duration
	sessionAbsoluteExpiry time.Duration
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
			cfg.Issuer,
		),
		tokenRepo:             tokenRepo,
		userRepo:              userRepo,
		refreshExpiry:         cfg.RefreshExpiry,
		sessionAbsoluteExpiry: cfg.SessionAbsoluteExpiry,
	}
}

// JWTManager 返回底层的 JWT manager。
func (s *TokenService) JWTManager() *pkgjwt.Manager {
	return s.jwtManager
}

// IssueTokens 为用户创建 access + refresh token。
func (s *TokenService) IssueTokens(ctx context.Context, user *model.User) (*TokenResult, *apperrors.AppError) {
	sessionID, err := crypto.GenerateRandomBase64URL(16)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to generate session id", err)
	}
	absoluteExp := time.Now().Add(s.sessionAbsoluteExpiry)
	refreshExp := s.nextRefreshExpiry(absoluteExp)
	pair, err := s.jwtManager.IssueTokenPairUntil(user.ID.String(), user.Username, string(user.Role), sessionID, refreshExp)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to issue tokens", err)
	}

	if err := s.tokenRepo.SaveSession(ctx, user.ID.String(), sessionID, pair.RefreshToken, pair.RefreshExpiry, absoluteExp); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to save token session", err)
	}

	return &TokenResult{
		AccessToken:   pair.AccessToken,
		RefreshToken:  pair.RefreshToken,
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
	claims, err := s.jwtManager.ParseRefreshToken(refreshTokenValue)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	reused, err := s.tokenRepo.IsRefreshTokenReused(ctx, refreshTokenValue)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidRefreshToken, "failed to check token reuse", err)
	}
	if reused {
		_ = s.tokenRepo.RevokeAllUserTokens(ctx, claims.UserID)
		return nil, apperrors.ErrInvalidRefreshToken
	}

	active, absoluteExp, err := s.tokenRepo.IsRefreshTokenActive(ctx, claims.UserID, claims.SessionID, refreshTokenValue)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidRefreshToken, "failed to check token status", err)
	}
	if !active {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return nil, apperrors.ErrAccountDisabled
	}

	refreshExp := s.nextRefreshExpiry(absoluteExp)
	if !refreshExp.After(time.Now()) {
		_ = s.tokenRepo.RevokeAllUserTokens(ctx, claims.UserID)
		return nil, apperrors.ErrInvalidRefreshToken
	}
	pair, issueErr := s.jwtManager.IssueTokenPairUntil(user.ID.String(), user.Username, string(user.Role), claims.SessionID, refreshExp)
	if issueErr != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to issue tokens", issueErr)
	}
	if err := s.tokenRepo.RotateSession(ctx, user.ID.String(), claims.SessionID, refreshTokenValue, pair.RefreshToken, pair.RefreshExpiry); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidRefreshToken, "failed to rotate token session", err)
	}

	return &TokenResult{
		AccessToken:   pair.AccessToken,
		RefreshToken:  pair.RefreshToken,
		AccessExpiry:  pair.AccessExpiry,
		RefreshExpiry: pair.RefreshExpiry,
	}, nil
}

// RevokeTokens 撤销某个用户的当前会话。
func (s *TokenService) RevokeTokens(ctx context.Context, userID string) error {
	return s.tokenRepo.RevokeAllUserTokens(ctx, userID)
}

// TokenResult 包含登录/刷新后返回的 token。
type TokenResult struct {
	AccessToken   string
	RefreshToken  string
	AccessExpiry  time.Time
	RefreshExpiry time.Time
}

func (s *TokenService) nextRefreshExpiry(absoluteExp time.Time) time.Time {
	idleExp := time.Now().Add(s.refreshExpiry)
	if absoluteExp.Before(idleExp) {
		return absoluteExp
	}
	return idleExp
}
