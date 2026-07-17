package auth

import (
	"context"
	"strings"
	"time"

	"strconv"

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
	verification  *VerificationService
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
	verification ...*VerificationService,
) *AuthService {
	var verificationService *VerificationService
	if len(verification) > 0 {
		verificationService = verification[0]
	}
	return &AuthService{
		userRepo:          userRepo,
		tokenRepo:         tokenRepo,
		configService:     configService,
		tokenService:      tokenService,
		emailSender:       emailSender,
		verification:      verificationService,
		logger:            logger,
		maxFailedAttempts: 5,
		lockDuration:      15 * time.Minute,
	}
}

// Login 对用户进行身份认证并返回 token。
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginServiceResult, *apperrors.AppError) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, apperrors.ErrInvalidCredentials
	}
	user, err := s.userRepo.FindByUsernameOrEmail(ctx, username)
	if err != nil {
		s.logger.Error("login: find user failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "database error", err)
	}
	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	if user.Status == model.StatusDisabled {
		return nil, apperrors.ErrAccountDisabled
	}
	if locked, _, err := s.tokenRepo.IsLoginLocked(ctx, strconv.FormatUint(user.ID, 10)); err == nil && locked {
		return nil, apperrors.ErrAccountLocked
	}

	if !crypto.VerifyPassword(user.PasswordHash, password) {
		_, _, _ = s.tokenRepo.IncrementLoginFailure(ctx, strconv.FormatUint(user.ID, 10), s.maxFailedAttempts, s.lockDuration)
		return nil, apperrors.ErrInvalidCredentials
	}

	// 密码校验通过。若用户已启用两步验证，则签发一次性挑战令牌，
	// 要求其提交动态码完成二次校验后再换取访问令牌（不直接签发 token）。
	if user.TwoFactorEnabled {
		_ = s.tokenRepo.ClearLoginFailures(ctx, strconv.FormatUint(user.ID, 10))
		challenge, chErr := s.tokenRepo.StoreTwoFactorChallenge(ctx, strconv.FormatUint(user.ID, 10), twoFactorChallengeTTL)
		if chErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeInternal, "failed to issue 2fa challenge", chErr)
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

	_ = s.tokenRepo.ClearLoginFailures(ctx, strconv.FormatUint(user.ID, 10))
	s.logger.Info("user logged in",
		zap.String("user_id", strconv.FormatUint(user.ID, 10)),
		zap.String("username", user.Username),
	)

	return &LoginServiceResult{
		Tokens:  tokens,
		User:    userToInfo(user),
		Actions: computeSecurityActions(ctx, s.configService, user),
	}, nil
}

// Register 创建一个新的用户账户。
func (s *AuthService) Register(ctx context.Context, username, nickname, phone, email, password, verifyCode string) *apperrors.AppError {
	sec, err := s.configService.Security(ctx)
	if err == nil && !sec.RegistrationEnabled {
		return apperrors.ErrOperationDenied
	}
	var appErr *apperrors.AppError
	username, appErr = validation.NormalizeUsername(username)
	if appErr != nil {
		return appErr
	}
	nickname, appErr = validation.NormalizeNickname(nickname)
	if appErr != nil {
		return appErr
	}
	phone, appErr = validation.NormalizePhone(phone)
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
	exists, err = s.userRepo.ExistsByPhone(ctx, phone)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	}
	if exists {
		return apperrors.ErrDuplicateEntry
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
		Nickname:          nickname,
		Phone:             phone,
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
		zap.String("user_id", strconv.FormatUint(user.ID, 10)),
		zap.String("username", user.Username),
	)

	return nil
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
		if existing != nil && strconv.FormatUint(existing.ID, 10) != userID {
			return nil, apperrors.ErrEmailExists
		}
		isDefaultAdmin := user.Username == "admin" || user.Username == "superadmin"
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
	return s.verification.SendCodeWithoutCaptcha(ctx, email, purpose, locale)
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
	if existing != nil && strconv.FormatUint(existing.ID, 10) != userID {
		return 0, apperrors.ErrEmailExists
	}
	return s.verification.SendCodeWithoutCaptcha(ctx, newEmail, "change-email", locale)
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
	if existing != nil && strconv.FormatUint(existing.ID, 10) != userID {
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
	Nickname            string
	Phone               string
	Avatar              string
	Email               string
	Role                string
	Permissions         []string
	TwoFactorEnabled    bool
	PreferredAuthMethod string
	HasPasskey          bool
}

func userToInfo(u *model.User) *UserInfoResult {
	return &UserInfoResult{
		ID:                  strconv.FormatUint(u.ID, 10),
		Username:            u.Username,
		Nickname:            u.Nickname,
		Phone:               u.Phone,
		Avatar:              u.Avatar,
		Email:               u.Email,
		Role:                string(u.Role),
		TwoFactorEnabled:    u.TwoFactorEnabled,
		PreferredAuthMethod: u.PreferredAuthMethod,
	}
}
