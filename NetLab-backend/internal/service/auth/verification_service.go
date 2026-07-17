package auth

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"netlab-backend/internal/mailer"
	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
)

const (
	verificationCodeTTL  = 10 * time.Minute // 普通验证码有效期为 10 分钟。
	changeEmailCodeTTL   = 5 * time.Minute  // 修改邮箱验证码有效期为 5 分钟。
	verificationCooldown = time.Minute      // 同一邮箱同一用途的发送冷却期为 1 分钟。
)

type CaptchaVerifier interface {
	Verify(id, code string) (bool, error)
}

type VerificationService struct {
	userRepo      *repository.UserRepository
	tokenRepo     *repository.TokenRepository
	configService *sysconfig.Service
	emailSender   *mailer.Provider
	captcha       CaptchaVerifier
	logger        *zap.Logger
}

func NewVerificationService(userRepo *repository.UserRepository, tokenRepo *repository.TokenRepository, configService *sysconfig.Service, emailSender *mailer.Provider, captcha CaptchaVerifier, logger *zap.Logger) *VerificationService {
	return &VerificationService{userRepo: userRepo, tokenRepo: tokenRepo, configService: configService, emailSender: emailSender, captcha: captcha, logger: logger}
}

func (s *VerificationService) SendCode(ctx context.Context, email, purpose, locale, captchaID, captchaCode string) (int, *apperrors.AppError) {
	return s.sendCode(ctx, email, purpose, locale, captchaID, captchaCode, true)
}

func (s *VerificationService) SendCodeWithoutCaptcha(ctx context.Context, email, purpose, locale string) (int, *apperrors.AppError) {
	return s.sendCode(ctx, email, purpose, locale, "", "", false)
}

func (s *VerificationService) sendCode(ctx context.Context, email, purpose, locale, captchaID, captchaCode string, requireCaptcha bool) (int, *apperrors.AppError) {
	email, appErr := validation.NormalizeEmail(email)
	if appErr != nil {
		return 0, appErr
	}
	if purpose != "register" && purpose != "reset-password" && purpose != "change-email" && purpose != PasskeyEmailCodePurpose && purpose != twoFactorDisableEmailPurpose {
		return 0, validation.Invalid("invalid verification purpose")
	}
	if requireCaptcha && s.captcha != nil {
		if captchaID == "" || captchaCode == "" {
			return 0, apperrors.New(apperrors.ErrCodeInvalidCode, "captcha is required")
		}
		ok, err := s.captcha.Verify(captchaID, captchaCode)
		if err != nil || !ok {
			return 0, apperrors.ErrInvalidCode
		}
	}
	if s.emailSender == nil || !s.emailSender.IsEnabled(ctx) {
		return 0, apperrors.ErrEmailNotConfigured
	}
	ttl, err := s.tokenRepo.GetVerificationCooldown(ctx, email, purpose)
	if err == nil && ttl > 0 {
		cd := int(ttl.Seconds())
		return cd, apperrors.New(apperrors.ErrCodeRateLimited, fmt.Sprintf("please wait %d seconds before requesting another code", cd))
	}
	code, err := crypto.GenerateNumericCode(6)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to generate verification code", err)
	}
	if err := s.emailSender.SendVerificationCode(ctx, email, code, purpose, locale); err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeEmailSendFailed, "failed to send verification email", err)
	}
	codeTTL := verificationCodeTTL
	if purpose == "change-email" {
		codeTTL = changeEmailCodeTTL
	}
	if err := s.tokenRepo.StoreVerificationCode(ctx, email, code, purpose, codeTTL); err != nil {
		return 0, apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to store code", err)
	}
	_ = s.tokenRepo.SetVerificationCooldown(ctx, email, purpose, verificationCooldown)
	s.logger.Info("verification code sent", zap.String("email", email), zap.String("purpose", purpose))
	return int(verificationCooldown.Seconds()), nil
}

func (s *VerificationService) VerifyCode(ctx context.Context, email, code, purpose string) (bool, *apperrors.AppError) {
	email, appErr := validation.NormalizeEmail(email)
	if appErr != nil {
		return false, appErr
	}
	code, appErr = validation.NormalizeVerifyCode(code)
	if appErr != nil {
		return false, appErr
	}
	stored, err := s.tokenRepo.PeekVerificationCode(ctx, email, purpose)
	if err != nil {
		return false, apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", err)
	}
	return stored != "" && stored == code, nil
}

func (s *VerificationService) ConsumeCode(ctx context.Context, email, code, purpose string) *apperrors.AppError {
	email, appErr := validation.NormalizeEmail(email)
	if appErr != nil {
		return appErr
	}
	code, appErr = validation.NormalizeVerifyCode(code)
	if appErr != nil {
		return appErr
	}
	stored, err := s.tokenRepo.GetVerificationCode(ctx, email, purpose)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to consume code", err)
	}
	if stored == "" || stored != code {
		return apperrors.ErrInvalidCode
	}
	return nil
}
