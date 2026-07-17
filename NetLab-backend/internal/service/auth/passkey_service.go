package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strconv"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
)

// PasskeyService 使用 go-webauthn 处理符合 WebAuthn 规范的注册与认证。
//
// 与此前的桩实现不同，本实现会对认证器返回的断言进行完整的密码学签名
// 校验、来源（origin）与 RP ID 校验，并通过签名计数器防止凭证克隆/重放。
type PasskeyService struct {
	passkeyRepo   *repository.PasskeyRepository
	userRepo      *repository.UserRepository
	tokenRepo     *repository.TokenRepository
	tokenService  *TokenService
	configService *sysconfig.Service
	redis         *redis.Client
	logger        *zap.Logger
	wa            *webauthn.WebAuthn
}

// NewPasskeyService 创建一个新的 PasskeyService。
//
// serverMode 用于推导 RP 配置：debug 模式下面向 localhost（http），
// 其它模式下使用 RP_ID / RP_ORIGIN 环境变量所指定的域名（https）。
func NewPasskeyService(
	passkeyRepo *repository.PasskeyRepository,
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	tokenService *TokenService,
	configService *sysconfig.Service,
	redis *redis.Client,
	logger *zap.Logger,
	serverMode string,
) *PasskeyService {
	rpID, rpOrigin := resolveRP(serverMode)

	wa, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "NetLab",
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		// 配置错误属于编程/部署问题——记录后仍返回一个 service，
		// 其方法会在 wa 为 nil 时优雅报错。
		logger.Error("failed to initialize webauthn", zap.Error(err))
	}

	return &PasskeyService{
		passkeyRepo:   passkeyRepo,
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		tokenService:  tokenService,
		configService: configService,
		redis:         redis,
		logger:        logger,
		wa:            wa,
	}
}

// PasskeyEmailCodePurpose 是 Passkey 增删操作二次邮箱校验所用的验证码用途。
const PasskeyEmailCodePurpose = "passkey"

// resolveRP 根据运行模式推导 RP ID 与 origin。
func resolveRP(serverMode string) (rpID, rpOrigin string) {
	if serverMode == "release" {
		// 生产环境应通过环境变量提供域名。
		rpID = getenvDefault("RP_ID", "localhost")
		rpOrigin = getenvDefault("RP_ORIGIN", "https://"+rpID)
		return rpID, rpOrigin
	}
	// 开发环境：前端默认运行于 http://localhost:5173，
	// RP ID 为主机名（不含端口与 scheme）。
	return "localhost", getenvDefault("RP_ORIGIN", "http://localhost:5173")
}

// ─── Redis 会话存储 ──────────────────────────────────────────────────

const (
	passkeyRegSessionPrefix  = "webauthn:reg:"
	passkeyAuthSessionPrefix = "webauthn:auth:"
	passkeySessionTTL        = 5 * time.Minute
)

func (s *PasskeyService) storeSession(ctx context.Context, key string, data *webauthn.SessionData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, key, raw, passkeySessionTTL).Err()
}

func (s *PasskeyService) loadSession(ctx context.Context, key string) (*webauthn.SessionData, error) {
	raw, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	_ = s.redis.Del(ctx, key) // 一次性使用
	return &data, nil
}

// ─── 注册流程 ────────────────────────────────────────────────────────

// BeginRegistration 生成 WebAuthn 注册质询，并把会话存入 Redis。
// 返回可直接序列化给前端的 CredentialCreation options。
func (s *PasskeyService) BeginRegistration(ctx context.Context, userID string) (*protocol.CredentialCreation, *apperrors.AppError) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, err
	}
	if s.wa == nil {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "webauthn not configured")
	}

	waUser, appErr := s.loadWebAuthnUser(ctx, userID)
	if appErr != nil {
		return nil, appErr
	}

	// 排除用户已注册的凭证，避免重复注册同一认证器。
	exclusions := make([]protocol.CredentialDescriptor, 0, len(waUser.creds))
	for _, c := range waUser.creds {
		exclusions = append(exclusions, c.Descriptor())
	}

	options, session, err := s.wa.BeginRegistration(waUser,
		webauthn.WithExclusions(exclusions),
	)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to begin registration", err)
	}

	if err := s.storeSession(ctx, passkeyRegSessionPrefix+userID, session); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to store session", err)
	}

	return options, nil
}

// FinishRegistration 校验认证器的 attestation 响应并持久化凭证。
// rawResponse 为前端提交的原始 WebAuthn JSON。
// verifyCode 为发送到用户邮箱的一次性验证码，用于二次校验。
func (s *PasskeyService) FinishRegistration(ctx context.Context, userID, name, verifyCode string, rawResponse []byte) *apperrors.AppError {
	if s.wa == nil {
		return apperrors.New(apperrors.ErrCodeOperationDenied, "webauthn not configured")
	}

	if appErr := s.verifyEmailCode(ctx, userID, verifyCode); appErr != nil {
		return appErr
	}

	session, err := s.loadSession(ctx, passkeyRegSessionPrefix+userID)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "registration session expired")
	}

	waUser, appErr := s.loadWebAuthnUser(ctx, userID)
	if appErr != nil {
		return appErr
	}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(rawResponse)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidRequest, "invalid attestation response", err)
	}

	credential, err := s.wa.CreateCredential(waUser, *session, parsed)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidRequest, "attestation verification failed", err)
	}

	credJSON, err := json.Marshal(credential)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to serialize credential", err)
	}

	uid, _ := strconv.ParseUint(userID, 10, 64)
	credID := base64.RawURLEncoding.EncodeToString(credential.ID)

	if err := s.passkeyRepo.Save(ctx, uid, credID, string(credJSON), strings.TrimSpace(name), credential.Authenticator.SignCount); err != nil {
		// 仅当确为唯一键冲突（同一 credential_id 已注册）时才提示“数据重复”，
		// 其它数据库错误（如约束/连接问题）按操作失败处理，避免误导用户。
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "passkey already registered", err)
		}
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to save credential", err)
	}

	s.logger.Info("passkey registered",
		zap.String("user_id", userID),
		zap.String("credential_id", credID),
	)
	return nil
}

// ─── 认证流程（Discoverable / Passkey 登录）─────────────────────────

// BeginLogin 生成一次可发现凭证（passkey）登录质询。
func (s *PasskeyService) BeginLogin(ctx context.Context) (*protocol.CredentialAssertion, string, *apperrors.AppError) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, "", err
	}
	if s.wa == nil {
		return nil, "", apperrors.New(apperrors.ErrCodeOperationDenied, "webauthn not configured")
	}

	options, session, err := s.wa.BeginDiscoverableLogin()
	if err != nil {
		return nil, "", apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to begin login", err)
	}

	// 用随机会话 ID 关联质询，返回给前端并在 Finish 时回传。
	sessionID := base64.RawURLEncoding.EncodeToString([]byte(session.Challenge))
	if err := s.storeSession(ctx, passkeyAuthSessionPrefix+sessionID, session); err != nil {
		return nil, "", apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to store session", err)
	}

	return options, sessionID, nil
}

// FinishLogin 校验 passkey 登录断言，成功时更新签名计数器并签发 token。
func (s *PasskeyService) FinishLogin(ctx context.Context, sessionID string, rawResponse []byte) (*LoginServiceResult, *apperrors.AppError) {
	if s.wa == nil {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "webauthn not configured")
	}

	session, err := s.loadSession(ctx, passkeyAuthSessionPrefix+sessionID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeInvalidRequest, "login session expired")
	}

	parsed, err := protocol.ParseCredentialRequestResponseBytes(rawResponse)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidRequest, "invalid assertion response", err)
	}

	// 用于定位凭证所属用户的回调；同时把匹配到的记录带出以便更新计数器。
	var matchedUser *model.User
	var matchedCredID string
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		credID := base64.RawURLEncoding.EncodeToString(rawID)
		credUser, err := s.passkeyRepo.FindByCredentialID(ctx, credID)
		if err != nil {
			return nil, err
		}
		if credUser == nil {
			return nil, errors.New("credential not found")
		}
		matchedUser = credUser
		matchedCredID = credID
		return s.buildWebAuthnUser(ctx, strconv.FormatUint(credUser.ID, 10))
	}

	credential, err := s.wa.ValidateDiscoverableLogin(handler, *session, parsed)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "assertion verification failed", err)
	}
	if matchedUser == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// 防克隆：若认证器返回的计数器未增长（且非 0），视为可疑。
	if credential.Authenticator.CloneWarning {
		s.logger.Warn("passkey clone warning — sign count did not increase",
			zap.String("credential_id", matchedCredID),
		)
		return nil, apperrors.New(apperrors.ErrCodeInvalidCredentials, "credential may be cloned")
	}

	user, appErr := s.getActiveUser(ctx, strconv.FormatUint(matchedUser.ID, 10))
	if appErr != nil {
		return nil, appErr
	}

	// 持久化更新后的签名计数器与凭证状态。
	if credJSON, err := json.Marshal(credential); err == nil {
		_ = s.passkeyRepo.UpdateSignCount(ctx, matchedCredID, credential.Authenticator.SignCount, string(credJSON), time.Now())
	}

	tokens, appErr := s.tokenService.IssueTokens(ctx, user)
	if appErr != nil {
		return nil, appErr
	}

	s.logger.Info("passkey authentication successful", zap.String("user_id", strconv.FormatUint(user.ID, 10)))

	return &LoginServiceResult{
		Tokens:  tokens,
		User:    userToInfo(user),
		Actions: computeSecurityActions(ctx, s.configService, user),
	}, nil
}

// ─── 管理 ────────────────────────────────────────────────────────────

// PasskeyInfo 是返回给用户的 passkey 元数据。
type PasskeyInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
}

// ListForUser 返回某用户的所有 passkey 元数据。
func (s *PasskeyService) ListForUser(ctx context.Context, userID string) ([]PasskeyInfo, *apperrors.AppError) {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeInternal, "invalid user id")
	}
	info, err := s.passkeyRepo.List(ctx, uid)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to list passkeys", err)
	}
	out := make([]PasskeyInfo, 0, len(info))
	for _, item := range info {
		name := item.Name
		if name == "" {
			name = "Passkey"
		}
		out = append(out, PasskeyInfo{
			ID:         item.ID,
			Name:       name,
			CreatedAt:  item.CreatedAt,
			LastUsedAt: item.LastUsedAt,
		})
	}
	return out, nil
}

// DeleteForUser 删除属于该用户的一个 passkey。
// verifyCode 为发送到用户邮箱的一次性验证码，用于二次校验。
func (s *PasskeyService) DeleteForUser(ctx context.Context, userID, passkeyID, verifyCode string) *apperrors.AppError {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeInternal, "invalid user id")
	}
	if appErr := s.verifyEmailCode(ctx, userID, verifyCode); appErr != nil {
		return appErr
	}
	affected, err := s.passkeyRepo.DeleteByCredentialID(ctx, uid, passkeyID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to delete passkey", err)
	}
	if affected == 0 {
		return apperrors.New(apperrors.ErrCodeInvalidRequest, "passkey not found")
	}
	return nil
}

// ─── 内部辅助 ────────────────────────────────────────────────────────

// verifyEmailCode 校验发送到用户邮箱的一次性验证码（用途为 passkey）。
// 验证码由已认证的 /auth/account/email-code 端点发放。
func (s *PasskeyService) verifyEmailCode(ctx context.Context, userID, code string) *apperrors.AppError {
	if code == "" {
		return apperrors.ErrInvalidCode
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return apperrors.ErrUserNotFound
	}
	email, appErr := validation.NormalizeEmail(user.Email)
	if appErr != nil {
		return appErr
	}
	stored, err := s.tokenRepo.GetVerificationCode(ctx, email, PasskeyEmailCodePurpose)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", err)
	}
	if stored == "" || stored != code {
		return apperrors.ErrInvalidCode
	}
	return nil
}

// HasPasskey 返回用户是否已注册至少一个通行密钥。
func (s *PasskeyService) HasPasskey(ctx context.Context, userID string) (bool, error) {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return false, nil
	}
	return s.passkeyRepo.HasPasskey(ctx, uid)
}

func (s *PasskeyService) ensureEnabled(ctx context.Context) *apperrors.AppError {
	sec, err := s.configService.Security(ctx)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to load config", err)
	}
	if !sec.PasskeyEnabled {
		return apperrors.New(apperrors.ErrCodeOperationDenied, "passkey authentication is disabled")
	}
	return nil
}

func (s *PasskeyService) getActiveUser(ctx context.Context, userID string) (*model.User, *apperrors.AppError) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return nil, apperrors.ErrAccountDisabled
	}
	return user, nil
}

// loadWebAuthnUser 构建带有已注册凭证的 webauthn 用户适配器。
func (s *PasskeyService) loadWebAuthnUser(ctx context.Context, userID string) (*webauthnUser, *apperrors.AppError) {
	u, appErr := s.getActiveUser(ctx, userID)
	if appErr != nil {
		return nil, appErr
	}
	wu, err := s.assembleUser(ctx, u)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to load credentials", err)
	}
	return wu, nil
}

// buildWebAuthnUser 是 loadWebAuthnUser 的接口版本，供 DiscoverableUserHandler 使用。
func (s *PasskeyService) buildWebAuthnUser(ctx context.Context, userID string) (webauthn.User, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || u == nil {
		return nil, errors.New("user not found")
	}
	return s.assembleUser(ctx, u)
}

func (s *PasskeyService) assembleUser(ctx context.Context, u *model.User) (*webauthnUser, error) {
	credentials, err := s.passkeyRepo.GetCredentials(ctx, u.ID)
	if err != nil {
		s.logger.Warn("failed to read passkey credential",
			zap.String("user_id", strconv.FormatUint(u.ID, 10)), zap.Error(err))
	}
	waCreds := make([]webauthn.Credential, 0, len(credentials))
	for _, cred := range credentials {
		if cred.Credential == "" {
			continue
		}
		var wc webauthn.Credential
		if err := json.Unmarshal([]byte(cred.Credential), &wc); err != nil {
			s.logger.Warn("skipping malformed passkey credential",
				zap.String("credential_id", cred.CredentialID), zap.Error(err))
		} else {
			waCreds = append(waCreds, wc)
		}
	}
	return &webauthnUser{user: u, creds: waCreds}, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ─── webauthn.User 适配器 ────────────────────────────────────────────

// webauthnUser 让 model.User 满足 webauthn.User 接口。
type webauthnUser struct {
	user  *model.User
	creds []webauthn.Credential
}

func (w *webauthnUser) WebAuthnID() []byte {
	id := w.user.ID // uuid.UUID 底层是 [16]byte
	return []byte(strconv.FormatUint(id, 10))
}
func (w *webauthnUser) WebAuthnName() string                       { return w.user.Username }
func (w *webauthnUser) WebAuthnDisplayName() string                { return w.user.Username }
func (w *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return w.creds }

// 确保接口实现（编译期校验）。
var _ webauthn.User = (*webauthnUser)(nil)
