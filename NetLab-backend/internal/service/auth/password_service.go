package auth

import (
	"context"
	"strconv"

	"go.uber.org/zap"

	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
)

// PasswordService 承载密码相关业务：修改密码、忘记密码、重置密码。
type PasswordService struct {
	userRepo      *repository.UserRepository
	tokenRepo     *repository.TokenRepository
	configService *sysconfig.Service
	tokenService  *TokenService
	verification  *VerificationService
	logger        *zap.Logger
}

// NewPasswordService 创建一个新的 PasswordService。
func NewPasswordService(userRepo *repository.UserRepository, tokenRepo *repository.TokenRepository, configService *sysconfig.Service, tokenService *TokenService, verification *VerificationService, logger *zap.Logger) *PasswordService {
	return &PasswordService{userRepo: userRepo, tokenRepo: tokenRepo, configService: configService, tokenService: tokenService, verification: verification, logger: logger}
}

// ChangePassword 在校验当前密码后为用户设置新密码，并吊销其活跃会话。
func (s *PasswordService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) *apperrors.AppError {
	if currentPassword == "" {
		return apperrors.ErrInvalidCredentials
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		return apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return apperrors.ErrAccountDisabled
	}
	if user.PasswordHash == "" {
		return apperrors.New(apperrors.ErrCodeOperationDenied, "account has no local password")
	}
	if !crypto.VerifyPassword(user.PasswordHash, currentPassword) {
		return apperrors.ErrInvalidCredentials
	}
	if appErr := validation.ValidatePassword(newPassword); appErr != nil {
		return appErr
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update password", err)
	}
	_ = s.tokenService.RevokeTokens(ctx, userID)
	s.logger.Info("password changed", zap.String("user_id", userID))
	return nil
}

// SendResetCode 向邮箱发送重置密码验证码并返回冷却秒数；受系统重置开关与
// 图形验证码策略约束，邮箱未注册时静默返回以避免账号枚举。
func (s *PasswordService) SendResetCode(ctx context.Context, email, locale, captchaID, captchaCode string) (int, *apperrors.AppError) {
	email, appErr := validation.NormalizeEmail(email)
	if appErr != nil {
		return 0, appErr
	}
	if sec, err := s.configService.Security(ctx); err == nil && !sec.PasswordResetEnabled {
		return 0, apperrors.ErrPasswordResetClosed
	}
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if !exists {
		return 60, nil
	}
	return s.verification.SendCode(ctx, email, "reset-password", locale, captchaID, captchaCode)
}

// ForgotPassword 发起忘记密码流程，跳过图形验证码直接向邮箱发送重置验证码。
func (s *PasswordService) ForgotPassword(ctx context.Context, email, locale, captchaID, captchaCode string) *apperrors.AppError {
	_, appErr := s.verification.SendCodeWithoutCaptcha(ctx, email, "reset-password", locale)
	return appErr
}

// ResetPassword 在系统允许重置时校验并一次性消费邮箱验证码，为用户重置密码并吊销其活跃会话。
func (s *PasswordService) ResetPassword(ctx context.Context, email, verifyCode, newPassword string) *apperrors.AppError {
	if sec, err := s.configService.Security(ctx); err == nil && !sec.PasswordResetEnabled {
		return apperrors.ErrPasswordResetClosed
	}
	email, appErr := validation.NormalizeEmail(email)
	if appErr != nil {
		return appErr
	}
	if appErr := validation.ValidatePassword(newPassword); appErr != nil {
		return appErr
	}
	if appErr := s.verification.ConsumeCode(ctx, email, verifyCode, "reset-password"); appErr != nil {
		return appErr
	}
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return apperrors.ErrAccountDisabled
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, strconv.FormatUint(user.ID, 10), hash); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update password", err)
	}
	_ = s.tokenService.RevokeTokens(ctx, strconv.FormatUint(user.ID, 10))
	s.logger.Info("password reset", zap.String("user_id", strconv.FormatUint(user.ID, 10)))
	return nil
}
