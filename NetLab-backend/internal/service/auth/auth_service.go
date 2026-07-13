package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"netlab-backend/internal/mailer"
	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
)

// AuthService 包含身份认证的核心业务逻辑。
type AuthService struct {
	userRepo      *repository.UserRepository
	tokenRepo     *repository.TokenRepository
	configService *sysconfig.Service
	tokenService  *TokenService
	emailSender   *mailer.Provider
	logger        *zap.Logger

	maxFailedAttempts int
	lockDuration      time.Duration
}

// NewAuthService 创建一个新的 AuthService。
func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	configService *sysconfig.Service,
	tokenService *TokenService,
	emailSender *mailer.Provider,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:          userRepo,
		tokenRepo:         tokenRepo,
		configService:     configService,
		tokenService:      tokenService,
		emailSender:       emailSender,
		logger:            logger,
		maxFailedAttempts: 5,
		lockDuration:      15 * time.Minute,
	}
}

// Login 对用户进行身份认证并返回 token。
func (s *AuthService) Login(ctx context.Context, username, password, captchaID, captchaCode string) (*LoginServiceResult, *apperrors.AppError) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, apperrors.ErrInvalidCredentials
	}
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
	if locked, _, err := s.tokenRepo.IsLoginLocked(ctx, user.ID.String()); err == nil && locked {
		return nil, apperrors.ErrAccountLocked
	}

	if !crypto.VerifyPassword(user.PasswordHash, password) {
		_, _, _ = s.tokenRepo.IncrementLoginFailure(ctx, user.ID.String(), s.maxFailedAttempts, s.lockDuration)
		return nil, apperrors.ErrInvalidCredentials
	}

	// 密码校验通过。若用户已启用两步验证，则签发一次性挑战令牌，
	// 要求其提交动态码完成二次校验后再换取访问令牌（不直接签发 token）。
	if user.TwoFactorEnabled {
		_ = s.tokenRepo.ClearLoginFailures(ctx, user.ID.String())
		challenge, chErr := s.tokenRepo.StoreTwoFactorChallenge(ctx, user.ID.String(), twoFactorChallengeTTL)
		if chErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "failed to issue 2fa challenge", chErr)
		}
		return &LoginServiceResult{
			RequiresTwoFactor: true,
			TwoFactorToken:    challenge,
			User:              userToInfo(user),
		}, nil
	}

	tokens, appErr := s.tokenService.IssueTokens(ctx, user)
	if appErr != nil {
		return nil, appErr
	}

	_ = s.tokenRepo.ClearLoginFailures(ctx, user.ID.String())
	_ = s.userRepo.UpdateLoginSuccess(ctx, user.ID.String())

	s.logger.Info("user logged in",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
	)

	return &LoginServiceResult{
		Tokens:  tokens,
		User:    userToInfo(user),
		Actions: computeSecurityActions(ctx, s.configService, user),
	}, nil
}

// Register 创建一个新的用户账户。
func (s *AuthService) Register(ctx context.Context, username, email, password, verifyCode string) *apperrors.AppError {
	sec, err := s.configService.Security(ctx)
	if err == nil && !sec.RegistrationEnabled {
		return apperrors.ErrOperationDenied
	}
	var appErr *apperrors.AppError
	username, appErr = validation.NormalizeUsername(username)
	if appErr != nil {
		return appErr
	}
	email, appErr = validation.NormalizeEmail(email)
	if appErr != nil {
		return appErr
	}
	verifyCode, appErr = validation.NormalizeVerifyCode(verifyCode)
	if appErr != nil {
		return appErr
	}
	if appErr := validation.ValidatePassword(password); appErr != nil {
		return appErr
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

	now := time.Now()
	user := &model.User{
		Username:          username,
		Email:             email,
		PasswordHash:      passwordHash,
		Role:              model.RoleViewer,
		Status:            model.StatusActive,
		PasswordChangedAt: &now,
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
	email, appErr = validation.NormalizeEmail(email)
	if appErr != nil {
		return 0, appErr
	}
	if purpose != "register" && purpose != "reset-password" && purpose != "change-email" && purpose != PasskeyEmailCodePurpose && purpose != twoFactorDisableEmailPurpose {
		return 0, validation.Invalid("invalid verification purpose")
	}
	// 如果邮件发送不可用则快速失败。若没有这层保护，
	// 该接口会报告成功但实际上悄无声息地什么也没发送。
	if s.emailSender == nil || !s.emailSender.IsEnabled(ctx) {
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

	code, err := crypto.GenerateNumericCode(6)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to generate verification code", err)
	}

	// 先发送邮件。只有在发送成功后才持久化验证码并开始冷却计时，
	// 这样瞬时的 SMTP 故障不会把用户锁在门外。
	if err := s.emailSender.SendVerificationCode(ctx, email, code, purpose, locale); err != nil {
		s.logger.Error("failed to send verification email",
			zap.String("email", email),
			zap.String("purpose", purpose),
			zap.Error(err),
		)
		return 0, apperrors.Wrap(apperrors.ErrCodeEmailSendFailed, "failed to send verification email", err)
	}

	codeTTL := 10 * time.Minute
	if purpose == "change-email" {
		codeTTL = 5 * time.Minute
	}
	if err := s.tokenRepo.StoreVerificationCode(ctx, email, code, purpose, codeTTL); err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to store code", err)
	}

	_ = s.tokenRepo.SetVerificationCooldown(ctx, email, purpose, 60*time.Second)

	s.logger.Info("verification code sent",
		zap.String("email", email),
		zap.String("purpose", purpose),
	)

	return 60, nil
}

// SendPasswordResetCode 为密码重置流程发送验证码。
// 对不存在的邮箱返回与成功相同的结果，但不发送邮件，避免用户枚举并节约邮件资源。
func (s *AuthService) SendPasswordResetCode(ctx context.Context, email, locale string) (cooldownSec int, appErr *apperrors.AppError) {
	email, appErr = validation.NormalizeEmail(email)
	if appErr != nil {
		return 0, appErr
	}
	// 尊重密码重置功能开关。
	if sec, err := s.configService.Security(ctx); err == nil && !sec.PasswordResetEnabled {
		return 0, apperrors.ErrPasswordResetClosed
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if !exists {
		// 不泄露该邮箱是否存在；同时避免对不存在用户发送邮件。
		return 60, nil
	}

	return s.SendVerificationCode(ctx, email, "reset-password", locale)
}

// ForgotPassword 发起密码重置流程。
func (s *AuthService) ForgotPassword(ctx context.Context, email, locale string) *apperrors.AppError {
	_, appErr := s.SendPasswordResetCode(ctx, email, locale)
	return appErr
}

// ResetPassword 在邮箱验证通过后重置用户密码。
func (s *AuthService) ResetPassword(ctx context.Context, email, verifyCode, newPassword string) *apperrors.AppError {
	if sec, err := s.configService.Security(ctx); err == nil && !sec.PasswordResetEnabled {
		return apperrors.ErrPasswordResetClosed
	}

	var appErr *apperrors.AppError
	email, appErr = validation.NormalizeEmail(email)
	if appErr != nil {
		return appErr
	}
	verifyCode, appErr = validation.NormalizeVerifyCode(verifyCode)
	if appErr != nil {
		return appErr
	}
	if appErr := validation.ValidatePassword(newPassword); appErr != nil {
		return appErr
	}
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

	_ = s.tokenService.RevokeTokens(ctx, user.ID.String())

	s.logger.Info("password reset", zap.String("user_id", user.ID.String()))

	return nil
}

// VerifyCode 将验证码与 Redis 中存储的值进行校验。
// 如果验证码匹配返回 true，如果不存在或错误则返回 false。
func (s *AuthService) VerifyCode(ctx context.Context, email, code, purpose string) (bool, *apperrors.AppError) {
	var appErr *apperrors.AppError
	email, appErr = validation.NormalizeEmail(email)
	if appErr != nil {
		return false, appErr
	}
	code, appErr = validation.NormalizeVerifyCode(code)
	if appErr != nil {
		return false, appErr
	}
	storedCode, err := s.tokenRepo.PeekVerificationCode(ctx, email, purpose)
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

// ChangePassword 修改已登录用户的密码：先校验当前密码，成功后更新并
// 撤销该用户的全部 token（要求重新登录），符合改密后失效旧会话的实践。
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) *apperrors.AppError {
	if strings.TrimSpace(currentPassword) == "" {
		return apperrors.ErrInvalidCredentials
	}
	if appErr := validation.ValidatePassword(newPassword); appErr != nil {
		return appErr
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

	// OAuth-only 账户没有本地密码，无法通过“修改密码”流程设置。
	if user.PasswordHash == "" {
		return apperrors.New(apperrors.ErrCodeOperationDenied, "account has no local password")
	}
	if !crypto.VerifyPassword(user.PasswordHash, currentPassword) {
		return apperrors.ErrInvalidCredentials
	}

	newHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, newHash); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update password", err)
	}

	// 改密后吊销全部会话，强制重新登录。
	_ = s.tokenService.RevokeTokens(ctx, userID)

	s.logger.Info("password changed", zap.String("user_id", userID))
	return nil
}

// CompleteRequiredSecurityUpdate updates mandatory account fields before the
// user can enter the application.
func (s *AuthService) CompleteRequiredSecurityUpdate(ctx context.Context, userID, newPassword, newEmail, verifyCode string) (*model.User, *apperrors.AppError) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return nil, apperrors.ErrAccountDisabled
	}
	if user.PasswordHash == "" {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "account has no local password")
	}

	needsPassword := user.ForcePasswordChange || passwordExpiredFor(ctx, s.configService, user)
	needsEmail := user.ForcePasswordChange && user.ForceEmailChange
	if !needsPassword {
		return user, nil
	}
	if appErr := validation.ValidatePassword(newPassword); appErr != nil {
		return nil, appErr
	}
	hash, hashErr := crypto.HashPassword(newPassword)
	if hashErr != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", hashErr)
	}
	if needsEmail {
		email, appErr := validation.NormalizeEmail(newEmail)
		if appErr != nil {
			return nil, appErr
		}
		existing, dbErr := s.userRepo.FindByEmail(ctx, email)
		if dbErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", dbErr)
		}
		if existing != nil && existing.ID.String() != userID {
			return nil, apperrors.ErrEmailExists
		}
		isDefaultAdmin := user.Username == "admin" && user.Role == model.RoleSuperAdmin
		if !isDefaultAdmin {
			code, appErr := validation.NormalizeVerifyCode(verifyCode)
			if appErr != nil {
				return nil, appErr
			}
			stored, dbErr := s.tokenRepo.GetVerificationCode(ctx, email, "change-email")
			if dbErr != nil {
				return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", dbErr)
			}
			if stored == "" || stored != code {
				return nil, apperrors.ErrInvalidCode
			}
		}
		if err := s.userRepo.UpdateEmail(ctx, userID, email); err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update email", err)
		}
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update password", err)
	}
	_ = s.tokenService.RevokeTokens(ctx, userID)
	updated, dbErr := s.userRepo.FindByID(ctx, userID)
	if dbErr != nil || updated == nil {
		return nil, apperrors.ErrUserNotFound
	}
	return updated, nil
}

// SendAccountEmailCode 向当前已登录用户自己的邮箱发送验证码。
// 用于账户内的敏感操作（如添加/删除 Passkey）的二次校验，避免公开的
// send-code 端点被滥用于向任意邮箱发信。
func (s *AuthService) SendAccountEmailCode(ctx context.Context, userID, purpose, locale string) (cooldownSec int, appErr *apperrors.AppError) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		return 0, apperrors.ErrUserNotFound
	}
	email, appErr := validation.NormalizeEmail(user.Email)
	if appErr != nil {
		return 0, appErr
	}
	return s.SendVerificationCode(ctx, email, purpose, locale)
}

// SendChangeEmailCode 向新的邮箱地址发送 5 分钟有效的验证码。
func (s *AuthService) SendChangeEmailCode(ctx context.Context, userID, newEmail, locale string) (cooldownSec int, appErr *apperrors.AppError) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		return 0, apperrors.ErrUserNotFound
	}
	newEmail, appErr = validation.NormalizeEmail(newEmail)
	if appErr != nil {
		return 0, appErr
	}
	existing, err := s.userRepo.FindByEmail(ctx, newEmail)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	}
	if existing != nil && existing.ID.String() != userID {
		return 0, apperrors.ErrEmailExists
	}
	return s.SendVerificationCode(ctx, newEmail, "change-email", locale)
}

// ChangeEmail 使用新邮箱收到的验证码更新当前账户邮箱。
func (s *AuthService) ChangeEmail(ctx context.Context, userID, newEmail, verifyCode string) *apperrors.AppError {
	newEmail, appErr := validation.NormalizeEmail(newEmail)
	if appErr != nil {
		return appErr
	}
	verifyCode, appErr = validation.NormalizeVerifyCode(verifyCode)
	if appErr != nil {
		return appErr
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		return apperrors.ErrUserNotFound
	}
	existing, err := s.userRepo.FindByEmail(ctx, newEmail)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	}
	if existing != nil && existing.ID.String() != userID {
		return apperrors.ErrEmailExists
	}
	storedCode, err := s.tokenRepo.GetVerificationCode(ctx, newEmail, "change-email")
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", err)
	}
	if storedCode == "" || storedCode != verifyCode {
		return apperrors.ErrInvalidCode
	}
	if err := s.userRepo.UpdateEmail(ctx, userID, newEmail); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update email", err)
	}
	s.logger.Info("email changed", zap.String("user_id", userID))
	return nil
}

// Logout 撤销当前会话。
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	return s.tokenService.RevokeTokens(ctx, userID)
}

// GetSystemConfig 返回用于对外公开展示的系统配置。
func (s *AuthService) GetSystemConfig(ctx context.Context) (*repository.SystemConfigResult, *apperrors.AppError) {
	config, err := s.configService.PublicConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to load config", err)
	}
	return config, nil
}

// LoginServiceResult 封装登录响应数据。
type LoginServiceResult struct {
	Tokens              *TokenResult
	User                *UserInfoResult
	Actions             SecurityActionsResult
	RequiresTwoFactor   bool
	TwoFactorToken      string
	PendingOAuthBinding *PendingOAuthBindingResult
}

type SecurityActionsResult struct {
	RequirePasswordChange bool   `json:"requirePasswordChange"`
	RequireEmailChange    bool   `json:"requireEmailChange"`
	RequireTwoFactorSetup bool   `json:"requireTwoFactorSetup"`
	Reason                string `json:"reason,omitempty"`
}

// UserInfoResult 是从模型中提取的用户信息。
type UserInfoResult struct {
	ID                  string
	Username            string
	Avatar              string
	Email               string
	Role                string
	TwoFactorEnabled    bool
	PreferredAuthMethod string
	HasPasskey          bool
}

func userToInfo(u *model.User) *UserInfoResult {
	return &UserInfoResult{
		ID:                  u.ID.String(),
		Username:            u.Username,
		Avatar:              u.Avatar,
		Email:               u.Email,
		Role:                string(u.Role),
		TwoFactorEnabled:    u.TwoFactorEnabled,
		PreferredAuthMethod: u.PreferredAuthMethod,
	}
}
