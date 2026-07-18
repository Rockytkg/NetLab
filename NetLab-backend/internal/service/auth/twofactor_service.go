package auth

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"image/png"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
)

// 两步验证相关参数。
const (
	twoFactorChallengeTTL        = 5 * time.Minute  // 登录挑战令牌有效期
	twoFactorSetupTTL            = 5 * time.Minute  // 绑定流程暂存密钥有效期
	twoFactorMaxAttempts         = 5                // 动态码最大尝试次数
	twoFactorLockDuration        = 15 * time.Minute // 超限后的临时锁定时长
	twoFactorRecoveryCount       = 10               // 启用 2FA 时生成的恢复码数量
	twoFactorDisableEmailPurpose = "disable-2fa"    // 关闭 2FA 邮箱验证码用途
	twoFactorTOTPSkew            = 1                // TOTP 容忍度：前后各 1 个时间步长
)

// recoveryCodeCharset 去除了易混淆字符（0/O/1/I/L），便于人工抄写。
const recoveryCodeCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// TwoFactorService 实现基于 TOTP 的两步验证业务逻辑：
// 绑定（生成密钥 + 二维码 + 校验动态码）、关闭以及登录时的二次校验，
// 并提供一次性恢复码与首选验证方式管理。
type TwoFactorService struct {
	userRepo      *repository.UserRepository
	tokenRepo     *repository.TokenRepository
	tokenService  *TokenService
	configService *sysconfig.Service
	logger        *zap.Logger
}

// NewTwoFactorService 创建一个新的 TwoFactorService。
func NewTwoFactorService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	tokenService *TokenService,
	configService *sysconfig.Service,
	logger *zap.Logger,
) *TwoFactorService {
	return &TwoFactorService{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		tokenService:  tokenService,
		configService: configService,
		logger:        logger,
	}
}

// TwoFactorSetupResult 是绑定流程首步返回给前端的数据。
type TwoFactorSetupResult struct {
	Secret     string
	OtpauthURL string
	QRCode     string // data:image/png;base64
}

// TwoFactorEnableResult 是启用 2FA 后返回的数据，包含一次性恢复码明文。
// 恢复码仅在此刻返回一次，后端只保存其哈希。
type TwoFactorEnableResult struct {
	RecoveryCodes []string
}

// BeginSetup 生成一个新的 TOTP 密钥与二维码，并将密钥暂存于 Redis
// （短期有效），等待 ConfirmSetup 校验动态码通过后落库。
func (s *TwoFactorService) BeginSetup(ctx context.Context, userID string) (*TwoFactorSetupResult, *apperrors.AppError) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		return nil, apperrors.ErrUserNotFound
	}

	accountName := user.Email
	if accountName == "" {
		accountName = user.Username
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "NetLab",
		AccountName: accountName,
	})
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to generate totp key", err)
	}

	if err := s.tokenRepo.StoreTwoFactorSetupSecret(ctx, userID, key.Secret(), twoFactorSetupTTL); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to store totp secret", err)
	}

	qrCode, appErr := encodeQRCode(key)
	if appErr != nil {
		return nil, appErr
	}

	return &TwoFactorSetupResult{
		Secret:     key.Secret(),
		OtpauthURL: key.URL(),
		QRCode:     qrCode,
	}, nil
}

// ConfirmSetup 校验用户输入的动态码，通过后将（加密后的）密钥落库、启用 2FA，
// 并生成一次性恢复码返回给用户保存。
func (s *TwoFactorService) ConfirmSetup(ctx context.Context, userID, code string) (*TwoFactorEnableResult, *apperrors.AppError) {
	code = normalizeTwoFactorCode(code)
	if code == "" {
		return nil, apperrors.ErrInvalidTwoFactorCode
	}

	secret, err := s.tokenRepo.GetTwoFactorSetupSecret(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to read totp secret", err)
	}
	if secret == "" {
		return nil, apperrors.ErrTwoFactorNotConfigured
	}

	if !validateTOTP(code, secret) {
		return nil, apperrors.ErrInvalidTwoFactorCode
	}

	encrypted, err := s.configService.EncryptSecret(secret)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to encrypt totp secret", err)
	}
	if err := s.userRepo.EnableTwoFactor(ctx, userID, encrypted); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to enable two-factor", err)
	}
	_ = s.tokenRepo.DeleteTwoFactorSetupSecret(ctx, userID)

	codes, hashes := generateRecoveryCodes(twoFactorRecoveryCount)
	uidVal, _ := strconv.ParseUint(userID, 10, 64)
	if err := s.userRepo.StoreRecoveryCodes(ctx, uidVal, hashes); err != nil {
		s.logger.Warn("failed to store recovery codes", zap.String("user_id", userID), zap.Error(err))
	}

	s.logger.Info("two-factor authentication enabled", zap.String("user_id", userID))
	return &TwoFactorEnableResult{RecoveryCodes: codes}, nil
}

// Disable 关闭两步验证。需校验发送到用户绑定邮箱的一次性验证码；若系统
// 策略强制开启 2FA 则禁止关闭。关闭后同时清除该用户的恢复码。
func (s *TwoFactorService) Disable(ctx context.Context, userID, verifyCode string) *apperrors.AppError {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		return apperrors.ErrUserNotFound
	}
	if !user.TwoFactorEnabled {
		return nil // 已关闭，幂等返回
	}
	sec, _ := s.configService.Security(ctx)
	if sec.TwoFactorRequired {
		return apperrors.New(apperrors.ErrCodeOperationDenied, "two-factor authentication is required by policy and cannot be disabled")
	}
	if appErr := s.verifyEmailCode(ctx, user, verifyCode); appErr != nil {
		return appErr
	}
	if err := s.userRepo.DisableTwoFactor(ctx, userID); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to disable two-factor", err)
	}
	_ = s.userRepo.DeleteRecoveryCodes(ctx, user.ID)
	s.logger.Info("two-factor authentication disabled", zap.String("user_id", userID))
	return nil
}

// VerifyLogin 校验登录第二步提交的动态码，通过后签发访问令牌。
// 挑战令牌一次性有效；动态码尝试受频率限制保护。
func (s *TwoFactorService) VerifyLogin(ctx context.Context, token, code string) (*LoginServiceResult, *apperrors.AppError) {
	code = normalizeTwoFactorCode(code)
	if code == "" {
		return nil, apperrors.ErrInvalidTwoFactorCode
	}

	user, appErr := s.resolveChallengeUser(ctx, token)
	if appErr != nil {
		return nil, appErr
	}

	if locked, _, _ := s.tokenRepo.IsTwoFactorLocked(ctx, strconv.FormatUint(user.ID, 10)); locked {
		return nil, apperrors.ErrRateLimited
	}

	secret, decErr := s.configService.DecryptSecret(user.TwoFactorSecret)
	if decErr != nil || secret == "" {
		return nil, apperrors.ErrTwoFactorNotConfigured
	}

	if !validateTOTP(code, secret) {
		_, locked, _ := s.tokenRepo.IncrementTwoFactorFailure(ctx, strconv.FormatUint(user.ID, 10), twoFactorMaxAttempts, twoFactorLockDuration)
		if locked {
			// 达到阈值：作废挑战令牌，强制用户重新登录。
			s.tokenRepo.ConsumeTwoFactorChallenge(ctx, token)
		}
		return nil, apperrors.ErrInvalidTwoFactorCode
	}

	return s.completeChallenge(ctx, user, token)
}

// VerifyLoginWithRecovery 使用一次性恢复码完成登录第二步校验。
// 恢复码仅可使用一次，成功消费后立即标记为失效。
func (s *TwoFactorService) VerifyLoginWithRecovery(ctx context.Context, token, recoveryCode string) (*LoginServiceResult, *apperrors.AppError) {
	normalized := normalizeRecoveryCode(recoveryCode)
	if normalized == "" {
		return nil, apperrors.ErrInvalidTwoFactorCode
	}

	user, appErr := s.resolveChallengeUser(ctx, token)
	if appErr != nil {
		return nil, appErr
	}

	if locked, _, _ := s.tokenRepo.IsTwoFactorLocked(ctx, strconv.FormatUint(user.ID, 10)); locked {
		return nil, apperrors.ErrRateLimited
	}

	hash := sha256Hex(normalized)
	consumed, err := s.userRepo.ConsumeRecoveryCode(ctx, user.ID, hash)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to consume recovery code", err)
	}
	if !consumed {
		_, locked, _ := s.tokenRepo.IncrementTwoFactorFailure(ctx, strconv.FormatUint(user.ID, 10), twoFactorMaxAttempts, twoFactorLockDuration)
		if locked {
			s.tokenRepo.ConsumeTwoFactorChallenge(ctx, token)
		}
		return nil, apperrors.ErrInvalidTwoFactorCode
	}

	s.logger.Info("two-factor login completed with recovery code", zap.String("user_id", strconv.FormatUint(user.ID, 10)))
	return s.completeChallenge(ctx, user, token)
}

// SetPreferredAuthMethod 设置用户的两步验证首选方式。
// method 取值为 "totp"（身份验证器应用）或 "passkey"（通行密钥）。
func (s *TwoFactorService) SetPreferredAuthMethod(ctx context.Context, userID, method string) *apperrors.AppError {
	if method != "totp" && method != "passkey" {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "invalid preferred auth method")
	}
	if err := s.userRepo.SetPreferredAuthMethod(ctx, userID, method); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to update preferred auth method", err)
	}
	return nil
}

// resolveChallengeUser 解析挑战令牌并返回对应活跃且已启用 2FA 的用户。
func (s *TwoFactorService) resolveChallengeUser(ctx context.Context, token string) (*model.User, *apperrors.AppError) {
	userID, err := s.tokenRepo.ResolveTwoFactorChallenge(ctx, token)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeSessionExpired, "failed to resolve 2fa challenge", err)
	}
	if userID == "" {
		return nil, apperrors.ErrSessionExpired
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return nil, apperrors.ErrAccountDisabled
	}
	if !user.TwoFactorEnabled {
		return nil, apperrors.ErrTwoFactorNotConfigured
	}
	return user, nil
}

// completeChallenge 在 2FA 校验通过后作废挑战令牌、清除失败计数并签发令牌。
func (s *TwoFactorService) completeChallenge(ctx context.Context, user *model.User, token string) (*LoginServiceResult, *apperrors.AppError) {
	consumedUserID, err := s.tokenRepo.ConsumeTwoFactorChallengeValue(ctx, token)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeSessionExpired, "failed to consume 2fa challenge", err)
	}
	if consumedUserID != strconv.FormatUint(user.ID, 10) {
		return nil, apperrors.ErrSessionExpired
	}
	_ = s.tokenRepo.ClearTwoFactorFailures(ctx, strconv.FormatUint(user.ID, 10))

	tokens, appErr := s.tokenService.IssueTokens(ctx, user)
	if appErr != nil {
		return nil, appErr
	}

	return &LoginServiceResult{
		Tokens:  tokens,
		User:    userToInfo(user),
		Actions: computeSecurityActions(ctx, s.configService, user),
	}, nil
}

// verifyEmailCode 校验发送到用户绑定邮箱的一次性验证码（用途为关闭 2FA）。
func (s *TwoFactorService) verifyEmailCode(ctx context.Context, user *model.User, code string) *apperrors.AppError {
	code = strings.TrimSpace(code)
	if code == "" {
		return apperrors.ErrInvalidCode
	}
	email, appErr := validation.NormalizeEmail(user.Email)
	if appErr != nil {
		return appErr
	}
	stored, err := s.tokenRepo.GetVerificationCode(ctx, email, twoFactorDisableEmailPurpose)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", err)
	}
	if stored == "" || stored != code {
		return apperrors.ErrInvalidCode
	}
	return nil
}

// validateTOTP 使用容忍度 window=1 校验动态码（允许前后各 1 个时间步长偏差）。
func validateTOTP(code, secret string) bool {
	opts := totp.ValidateOpts{
		Period:    30,
		Skew:      twoFactorTOTPSkew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(), opts)
	return err == nil && valid
}

// encodeQRCode 将 otpauth URI 渲染为二维码 PNG，并以 data URI 返回。
func encodeQRCode(key *otp.Key) (string, *apperrors.AppError) {
	img, err := key.Image(240, 240)
	if err != nil {
		return "", apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to render qr code", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to encode qr png", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// normalizeTwoFactorCode 去除空白并校验为 6 位数字；非法时返回空。
func normalizeTwoFactorCode(code string) string {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return ""
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return code
}

// generateRecoveryCodes 生成 n 个一次性恢复码及其 SHA-256 哈希。
// 明文格式为 XXXX-XXXX-XXXX-XXXX（4 组，每组 4 字符），便于人工抄写；
// 哈希基于规范化后的纯字母数字形式计算。
func generateRecoveryCodes(n int) (codes []string, hashes []string) {
	codes = make([]string, 0, n)
	hashes = make([]string, 0, n)
	for i := 0; i < n; i++ {
		plain := randomRecoveryCode()
		codes = append(codes, formatRecoveryCode(plain))
		hashes = append(hashes, sha256Hex(plain))
	}
	return codes, hashes
}

// randomRecoveryCode 生成 16 个随机字符的恢复码明文（已大写）。
func randomRecoveryCode() string {
	buf := make([]byte, 16)
	if _, err := cryptorand.Read(buf); err != nil {
		// crypto/rand 失败极少见；退化为时间种子以避免阻塞流程。
		// 此分支几乎不会触发，但保证函数总有返回值。
		for i := range buf {
			buf[i] = recoveryCodeCharset[int(time.Now().UnixNano())%len(recoveryCodeCharset)]
		}
		return string(buf)
	}
	out := make([]byte, 16)
	for i, b := range buf {
		out[i] = recoveryCodeCharset[int(b)%len(recoveryCodeCharset)]
	}
	return string(out)
}

// formatRecoveryCode 将 16 字符明文按 4-4-4-4 分组并用连字符分隔。
func formatRecoveryCode(plain string) string {
	var b strings.Builder
	b.Grow(19)
	for i, r := range plain {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// normalizeRecoveryCode 去除连字符与空白并统一为大写，校验长度；非法时返回空。
func normalizeRecoveryCode(code string) string {
	var b strings.Builder
	b.Grow(len(code))
	for _, r := range strings.ToUpper(strings.TrimSpace(code)) {
		switch {
		case r == '-' || r == ' ' || r == '\t':
			continue
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			return ""
		}
	}
	out := b.String()
	if len(out) != 16 {
		return ""
	}
	return out
}

// sha256Hex 返回输入的 SHA-256 十六进制摘要。
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
