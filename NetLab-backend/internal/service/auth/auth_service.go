package auth

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/captcha"
	"netlab-backend/pkg/crypto"
	"netlab-backend/pkg/email"
)

// AuthService 包含身份认证的核心业务逻辑。
type AuthService struct {
	db              *gorm.DB
	userRepo        *repository.UserRepository
	tokenRepo       *repository.TokenRepository
	passkeyRepo     *repository.PasskeyRepository
	configRepo      *repository.ConfigRepository
	tokenService    *TokenService
	captchaMgr      *captcha.Manager
	emailSender     *email.SMTPSender
	logger          *zap.Logger

	maxFailedAttempts int
	lockDuration      time.Duration
}

// NewAuthService 创建一个新的 AuthService。
func NewAuthService(
	db *gorm.DB,
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	passkeyRepo *repository.PasskeyRepository,
	configRepo *repository.ConfigRepository,
	tokenService *TokenService,
	captchaMgr *captcha.Manager,
	emailSender *email.SMTPSender,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		db:                db,
		userRepo:          userRepo,
		tokenRepo:         tokenRepo,
		passkeyRepo:       passkeyRepo,
		configRepo:        configRepo,
		tokenService:      tokenService,
		captchaMgr:        captchaMgr,
		emailSender:       emailSender,
		logger:            logger,
		maxFailedAttempts: 5,
		lockDuration:      15 * time.Minute,
	}
}

// Login 对用户进行身份认证并返回 token。
func (s *AuthService) Login(ctx context.Context, username, password, captchaID, captchaCode string) (*LoginServiceResult, *apperrors.AppError) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		s.logger.Error("login: find user failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "database error", err)
	}
	if user == nil {
		user, err = s.userRepo.FindByEmail(ctx, username)
		if err != nil {
			s.logger.Error("login: find user by email failed", zap.Error(err))
			return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "database error", err)
		}
	}
	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	if user.Status == model.StatusDisabled {
		return nil, apperrors.ErrAccountDisabled
	}
	if user.IsLocked() {
		return nil, apperrors.ErrAccountLocked
	}

	if !crypto.VerifyPassword(user.PasswordHash, password) {
		_ = s.userRepo.IncrementFailedLogin(ctx, user.ID.String(), s.maxFailedAttempts, s.lockDuration)
		return nil, apperrors.ErrInvalidCredentials
	}

	tokens, appErr := s.tokenService.IssueTokens(ctx, user)
	if appErr != nil {
		return nil, appErr
	}

	_ = s.userRepo.UpdateLoginSuccess(ctx, user.ID.String())

	s.logger.Info("user logged in",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
	)

	return &LoginServiceResult{
		Tokens: tokens,
		User:   userToInfo(user),
	}, nil
}

// Register 创建一个新的用户账户。
func (s *AuthService) Register(ctx context.Context, username, email, password, verifyCode string) *apperrors.AppError {
	cfg, err := s.configRepo.GetSystemConfig(ctx)
	if err == nil && !cfg.RegistrationEnabled {
		return apperrors.ErrOperationDenied
	}

	exists, err := s.userRepo.ExistsByUsername(ctx, username)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	}
	if exists {
		return apperrors.ErrUsernameExists
	}

	exists, err = s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	}
	if exists {
		return apperrors.ErrEmailExists
	}

	storedCode, err := s.tokenRepo.GetVerificationCode(ctx, email, "register")
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", err)
	}
	if storedCode == "" || storedCode != verifyCode {
		return apperrors.ErrInvalidCode
	}

	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Roles:        []string{string(model.RoleViewer)},
		Status:       model.StatusActive,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("register: create user failed", zap.Error(err))
		return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "failed to create user", err)
	}

	s.logger.Info("user registered",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
	)

	return nil
}

// SendVerificationCode 向指定邮箱发送 6 位验证码。
// locale 用于选择邮件模板的语言（zh-CN 或 en-US）。
func (s *AuthService) SendVerificationCode(ctx context.Context, email, purpose, locale string) (cooldownSec int, appErr *apperrors.AppError) {
	// 如果邮件发送不可用则快速失败。若没有这层保护，
	// 该接口会报告成功但实际上悄无声息地什么也没发送。
	if s.emailSender == nil || !s.emailSender.IsEnabled() {
		s.logger.Warn("verification code requested but SMTP is not configured",
			zap.String("email", email),
			zap.String("purpose", purpose),
		)
		return 0, apperrors.ErrEmailNotConfigured
	}

	ttl, err := s.tokenRepo.GetVerificationCooldown(ctx, email, purpose)
	if err == nil && ttl > 0 {
		cd := int(ttl.Seconds())
		return cd, apperrors.New(apperrors.ErrCodeRateLimited,
			fmt.Sprintf("please wait %d seconds before requesting another code", cd))
	}

	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// 先发送邮件。只有在发送成功后才持久化验证码并开始冷却计时，
	// 这样瞬时的 SMTP 故障不会把用户锁在门外。
	if err := s.emailSender.SendVerificationCode(email, code, purpose, locale); err != nil {
		s.logger.Error("failed to send verification email",
			zap.String("email", email),
			zap.String("purpose", purpose),
			zap.Error(err),
		)
		return 0, apperrors.Wrap(apperrors.ErrCodeEmailSendFailed, "failed to send verification email", err)
	}

	if err := s.tokenRepo.StoreVerificationCode(ctx, email, code, purpose, 10*time.Minute); err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to store code", err)
	}

	_ = s.tokenRepo.SetVerificationCooldown(ctx, email, purpose, 60*time.Second)

	s.logger.Info("verification code sent",
		zap.String("email", email),
		zap.String("purpose", purpose),
	)

	return 60, nil
}

// ForgotPassword 发起密码重置流程。
func (s *AuthService) ForgotPassword(ctx context.Context, email, locale string) *apperrors.AppError {
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if !exists {
		// 不泄露该邮箱是否存在
		return nil
	}

	_, appErr := s.SendVerificationCode(ctx, email, "reset-password", locale)
	return appErr
}

// ResetPassword 在邮箱验证通过后重置用户密码。
func (s *AuthService) ResetPassword(ctx context.Context, email, verifyCode, newPassword string) *apperrors.AppError {
	storedCode, err := s.tokenRepo.GetVerificationCode(ctx, email, "reset-password")
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", err)
	}
	if storedCode == "" || storedCode != verifyCode {
		return apperrors.ErrInvalidCode
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return apperrors.ErrAccountDisabled
	}

	passwordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, user.ID.String(), passwordHash); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update password", err)
	}

	_ = s.tokenService.RevokeTokens(ctx, user.ID.String(), "", time.Time{})

	s.logger.Info("password reset", zap.String("user_id", user.ID.String()))

	return nil
}

// VerifyCode 将验证码与 Redis 中存储的值进行校验。
// 如果验证码匹配返回 true，如果不存在或错误则返回 false。
func (s *AuthService) VerifyCode(ctx context.Context, email, code, purpose string) (bool, *apperrors.AppError) {
	storedCode, err := s.tokenRepo.GetVerificationCode(ctx, email, purpose)
	if err != nil {
		return false, apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", err)
	}
	if storedCode == "" || storedCode != code {
		return false, nil
	}
	return true, nil
}

// GetUserInfo 返回当前用户的信息。
func (s *AuthService) GetUserInfo(ctx context.Context, userID string) (*model.User, *apperrors.AppError) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	return user, nil
}

// Logout 撤销当前会话。
func (s *AuthService) Logout(ctx context.Context, userID, jti string, tokenExp time.Time) error {
	return s.tokenService.RevokeTokens(ctx, userID, jti, tokenExp)
}

// GetSystemConfig 返回用于对外公开展示的系统配置。
func (s *AuthService) GetSystemConfig(ctx context.Context) (*repository.SystemConfigResult, *apperrors.AppError) {
	config, err := s.configRepo.GetSystemConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to load config", err)
	}
	return config, nil
}

// LoginServiceResult 封装登录响应数据。
type LoginServiceResult struct {
	Tokens *TokenResult
	User   *UserInfoResult
}

// UserInfoResult 是从模型中提取的用户信息。
type UserInfoResult struct {
	ID       string
	Username string
	Avatar   string
	Email    string
	Roles    []string
}

func userToInfo(u *model.User) *UserInfoResult {
	return &UserInfoResult{
		ID:       u.ID.String(),
		Username: u.Username,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Roles:    u.Roles,
	}
}
