package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"strconv"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	sysconfig "netlab-backend/internal/service/config"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
)

const (
	oauthStateTTL         = 10 * time.Minute
	oauthStateNamespace   = "oauth:state"
	oauthPendingNamespace = "oauth:pending"
)

// OAuthService 处理第三方 OAuth 登录流程。
//
// 状态令牌（CSRF state）与待绑定会话（pending binding）均通过
// TokenRepository 的 OneTimeToken 基础设施存储——仅保存令牌哈希，
// 消费时原子删除，避免直接操作裸 Redis key。
type OAuthService struct {
	configService *sysconfig.Service
	userRepo      *repository.UserRepository
	bindingRepo   *repository.OAuthBindingRepository
	tokenRepo     *repository.TokenRepository
	tokenService  *TokenService
	logger        *zap.Logger
	httpClient    *http.Client
}

// NewOAuthService 创建一个新的 OAuthService。
func NewOAuthService(
	configService *sysconfig.Service,
	userRepo *repository.UserRepository,
	bindingRepo *repository.OAuthBindingRepository,
	tokenRepo *repository.TokenRepository,
	tokenService *TokenService,
	logger *zap.Logger,
) *OAuthService {
	return &OAuthService{
		configService: configService,
		userRepo:      userRepo,
		bindingRepo:   bindingRepo,
		tokenRepo:     tokenRepo,
		tokenService:  tokenService,
		logger:        logger,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}
}

// OAuthUserInfo 保存从 OAuth 提供方获取的用户信息。
type OAuthUserInfo struct {
	Provider       string
	ProviderUserID string
	Email          string
	Username       string
	AvatarURL      string
}

// PendingOAuthBindingResult 描述一个待完成的 OAuth 绑定会话。
type PendingOAuthBindingResult struct {
	Token    string `json:"token"`
	Provider string `json:"provider"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
}

// ─── state / pending binding ─────────────────────────────────────────

// oauthState 是存入 OneTimeToken 的 state 载荷。
type oauthState struct {
	Intent string `json:"intent"`           // login | bind
	UserID string `json:"userId,omitempty"` // bind 时为发起用户
}

// pendingOAuthBinding 是待绑定会话中暂存的第三方身份。
type pendingOAuthBinding struct {
	Provider       string `json:"provider"`
	ProviderUserID string `json:"providerUserId"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	AvatarURL      string `json:"avatarUrl"`
}

// GenerateState 创建一个加密随机 state 并携带 login 意图。
func (s *OAuthService) GenerateState(ctx context.Context) (string, *apperrors.AppError) {
	return s.generateState(ctx, oauthState{Intent: "login"})
}

// GenerateBindState 创建一个绑定意图的 state（记录发起用户）。
func (s *OAuthService) GenerateBindState(ctx context.Context, userID string) (string, *apperrors.AppError) {
	return s.generateState(ctx, oauthState{Intent: "bind", UserID: userID})
}

func (s *OAuthService) generateState(ctx context.Context, payload oauthState) (string, *apperrors.AppError) {
	data, _ := json.Marshal(payload)
	token, err := s.tokenRepo.StoreOneTimeToken(ctx, oauthStateNamespace, data, oauthStateTTL)
	if err != nil {
		return "", apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to generate state", err)
	}
	return token, nil
}

func (s *OAuthService) consumeState(ctx context.Context, state string) (oauthState, *apperrors.AppError) {
	raw, err := s.tokenRepo.ConsumeOneTimeToken(ctx, oauthStateNamespace, state)
	if err != nil {
		return oauthState{}, apperrors.Wrap(apperrors.ErrCodeTokenExpired, "invalid oauth state", err)
	}
	if len(raw) == 0 {
		return oauthState{}, apperrors.ErrTokenExpired
	}
	var st oauthState
	if err := json.Unmarshal(raw, &st); err != nil {
		return oauthState{}, apperrors.ErrTokenExpired
	}
	if st.Intent == "" {
		st.Intent = "login"
	}
	return st, nil
}

func (s *OAuthService) storePendingBinding(ctx context.Context, u *OAuthUserInfo) (*PendingOAuthBindingResult, *apperrors.AppError) {
	data, _ := json.Marshal(pendingOAuthBinding{
		Provider:       u.Provider,
		ProviderUserID: u.ProviderUserID,
		Email:          u.Email,
		Username:       u.Username,
		AvatarURL:      u.AvatarURL,
	})
	token, err := s.tokenRepo.StoreOneTimeToken(ctx, oauthPendingNamespace, data, oauthStateTTL)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to store pending binding", err)
	}
	return &PendingOAuthBindingResult{
		Token:    token,
		Provider: u.Provider,
		Email:    u.Email,
		Username: u.Username,
		Avatar:   u.AvatarURL,
	}, nil
}

func (s *OAuthService) consumePendingBinding(ctx context.Context, token string) (*OAuthUserInfo, *apperrors.AppError) {
	raw, err := s.tokenRepo.ConsumeOneTimeToken(ctx, oauthPendingNamespace, token)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeSessionExpired, "invalid pending binding", err)
	}
	if len(raw) == 0 {
		return nil, apperrors.ErrSessionExpired
	}
	var p pendingOAuthBinding
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, apperrors.ErrSessionExpired
	}
	return &OAuthUserInfo{
		Provider:       p.Provider,
		ProviderUserID: p.ProviderUserID,
		Email:          p.Email,
		Username:       p.Username,
		AvatarURL:      p.AvatarURL,
	}, nil
}

// ─── 登录回调 ─────────────────────────────────────────────────────────

// HandleCallback 处理 OAuth 登录回调：校验 state、交换 code、获取用户信息。
// 若第三方身份尚未绑定本地账号，返回 pending 绑定会话，交由前端引导用户
// 绑定已有账号或创建新账号。
func (s *OAuthService) HandleCallback(ctx context.Context, provider, code, state string) (*LoginServiceResult, *apperrors.AppError) {
	st, err := s.consumeState(ctx, state)
	if err != nil {
		return nil, err
	}
	if st.Intent != "login" {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "state intent mismatch")
	}

	oauthUser, appErr := s.fetchUserInfo(ctx, provider, code)
	if appErr != nil {
		return nil, appErr
	}

	user, pending, appErr := s.resolveLoginUser(ctx, oauthUser)
	if appErr != nil {
		return nil, appErr
	}
	if pending != nil {
		return &LoginServiceResult{PendingOAuthBinding: pending}, nil
	}

	s.logger.Info("oauth login success",
		zap.String("provider", provider),
		zap.String("user_id", strconv.FormatUint(user.ID, 10)),
		zap.String("email", oauthUser.Email),
	)
	return s.issueOAuthLogin(ctx, user, provider)
}

// HandleBindCallback 处理已登录用户的"绑定第三方账号"回调：校验 state
// 归属当前用户、交换 code 获取第三方身份，并写入绑定关系。
func (s *OAuthService) HandleBindCallback(ctx context.Context, userID, provider, code, state string) *apperrors.AppError {
	st, err := s.consumeState(ctx, state)
	if err != nil {
		return err
	}
	if st.Intent != "bind" || st.UserID != userID {
		return apperrors.New(apperrors.ErrCodeOperationDenied, "state intent mismatch")
	}

	oauthUser, appErr := s.fetchUserInfo(ctx, provider, code)
	if appErr != nil {
		return appErr
	}

	uid, parseErr := strconv.ParseUint(userID, 10, 64)
	if parseErr != nil {
		return apperrors.New(apperrors.ErrCodeInternal, "invalid user id")
	}

	// 该第三方身份是否已被其他账号绑定？
	existingUser, dbErr := s.bindingRepo.FindByProviderUID(ctx, oauthUser.Provider, oauthUser.ProviderUserID)
	if dbErr != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "database error", dbErr)
	}
	if existingUser != nil {
		if existingUser.ID == uid {
			return nil // 幂等：已绑定到本账号
		}
		return apperrors.New(apperrors.ErrCodeDuplicateEntry, "this third-party account is already linked to another user")
	}

	// 互斥绑定：已绑定其他提供商则拒绝。
	if has, dbErr := s.bindingRepo.HasAny(ctx, uid); dbErr != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "database error", dbErr)
	} else if has {
		return apperrors.New(apperrors.ErrCodeDuplicateEntry, "provider already linked to this account")
	}

	if err := s.bindingRepo.Bind(ctx, uid, oauthUser.Provider, oauthUser.ProviderUserID, oauthUser.Email); err != nil {
		s.logger.Error("save oauth binding failed", zap.Error(err))
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to bind provider", err)
	}
	s.logger.Info("oauth provider bound",
		zap.String("provider", oauthUser.Provider),
		zap.String("user_id", userID),
	)
	return nil
}

// UnbindProvider 解除当前用户与某提供商的绑定。
// 为防止账号被锁死：若用户没有本地密码，且解绑后将不再有任何绑定，则拒绝。
func (s *OAuthService) UnbindProvider(ctx context.Context, userID, provider string) *apperrors.AppError {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeInternal, "invalid user id")
	}
	user, dbErr := s.userRepo.FindByID(ctx, userID)
	if dbErr != nil || user == nil {
		return apperrors.ErrUserNotFound
	}
	if user.PasswordHash == "" {
		hasAny, cErr := s.bindingRepo.HasAny(ctx, uid)
		if cErr != nil {
			return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "database error", cErr)
		}
		if hasAny {
			return apperrors.New(apperrors.ErrCodeOperationDenied, "cannot unbind the only login method; set a password first")
		}
	}
	dbErr = s.bindingRepo.Unbind(ctx, uid, provider)
	if dbErr != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to unbind provider", dbErr)
	}
	s.logger.Info("oauth provider unbound",
		zap.String("provider", provider),
		zap.String("user_id", userID),
	)
	return nil
}

// ListBindings 返回当前用户已绑定的提供商列表。
func (s *OAuthService) ListBindings(ctx context.Context, userID string) ([]model.OAuthProviderInfo, *apperrors.AppError) {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeInternal, "invalid user id")
	}
	list, dbErr := s.bindingRepo.ListByUser(ctx, uid)
	if dbErr != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to list bindings", dbErr)
	}
	return list, nil
}

// ─── pending -> 绑定 / 创建 ──────────────────────────────────────────

// BindPendingToExisting 将 pending 第三方身份绑定到已有账号。
// account 可为用户名或邮箱；需提供该账号邮箱收到的验证码以证明归属。
func (s *OAuthService) BindPendingToExisting(ctx context.Context, pendingToken, account, verifyCode string) (*LoginServiceResult, *apperrors.AppError) {
	oauthUser, appErr := s.consumePendingBinding(ctx, pendingToken)
	if appErr != nil {
		return nil, appErr
	}
	account = strings.TrimSpace(account)
	user, appErr := s.findUserByIdentifier(ctx, account)
	if appErr != nil {
		return nil, appErr
	}
	if !user.IsActive() {
		return nil, apperrors.ErrAccountDisabled
	}
	code, appErr := validation.NormalizeVerifyCode(verifyCode)
	if appErr != nil {
		return nil, appErr
	}
	if appErr := s.verifyEmailCode(ctx, user.Email, "change-email", code); appErr != nil {
		return nil, appErr
	}
	if appErr := s.createBindingStrict(ctx, user.ID, oauthUser); appErr != nil {
		return nil, appErr
	}
	return s.issueOAuthLogin(ctx, user, oauthUser.Provider)
}

// CreateAccountForPending 为 pending 第三方身份创建新账号并完成登录。
func (s *OAuthService) CreateAccountForPending(ctx context.Context, pendingToken, username, email, password, verifyCode string) (*LoginServiceResult, *apperrors.AppError) {
	oauthUser, appErr := s.consumePendingBinding(ctx, pendingToken)
	if appErr != nil {
		return nil, appErr
	}
	username, appErr = validation.NormalizeUsername(username)
	if appErr != nil {
		return nil, appErr
	}
	email, appErr = validation.NormalizeEmail(email)
	if appErr != nil {
		return nil, appErr
	}
	verifyCode, appErr = validation.NormalizeVerifyCode(verifyCode)
	if appErr != nil {
		return nil, appErr
	}
	if appErr := validation.ValidatePassword(password); appErr != nil {
		return nil, appErr
	}
	if exists, err := s.userRepo.ExistsByUsername(ctx, username); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	} else if exists {
		return nil, apperrors.ErrUsernameExists
	}
	if exists, err := s.userRepo.ExistsByEmail(ctx, email); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "database error", err)
	} else if exists {
		return nil, apperrors.ErrEmailExists
	}
	if appErr := s.verifyEmailCode(ctx, email, "register", verifyCode); appErr != nil {
		return nil, appErr
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeWeakPassword, "failed to hash password", err)
	}
	now := time.Now()
	user := &model.User{
		Username:          username,
		Email:             email,
		PasswordHash:      hash,
		Avatar:            oauthUser.AvatarURL,
		Role:              model.RoleViewer,
		Status:            model.StatusActive,
		PasswordChangedAt: &now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "failed to create user", err)
	}
	if appErr := s.createBindingStrict(ctx, user.ID, oauthUser); appErr != nil {
		return nil, appErr
	}
	return s.issueOAuthLogin(ctx, user, oauthUser.Provider)
}

// ─── 内部辅助 ─────────────────────────────────────────────────────────

// fetchUserInfo 依据 provider 分发到对应的获取函数，交换 code 并取得第三方身份。
func (s *OAuthService) fetchUserInfo(ctx context.Context, provider, code string) (*OAuthUserInfo, *apperrors.AppError) {
	fetch, ok := oauthFetchers[provider]
	if !ok {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "unsupported oauth provider: "+provider)
	}
	cfg, ok, err := s.configService.OAuthProvider(ctx, provider)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to load oauth config", err)
	}
	if !ok || !cfg.IsConfigured() {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, provider+" oauth is not configured")
	}
	user, err := fetch(ctx, s.httpClient, cfg, code)
	if err != nil {
		s.logger.Error("oauth fetch failed", zap.String("provider", provider), zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to fetch oauth user", err)
	}
	return user, nil
}

// resolveLoginUser 按 (provider, providerUserID) 绑定关系定位登录用户。
// 命中绑定 -> 直接登录；未命中 -> 返回 pending 绑定会话。
func (s *OAuthService) resolveLoginUser(ctx context.Context, oauthUser *OAuthUserInfo) (*model.User, *PendingOAuthBindingResult, *apperrors.AppError) {
	existingUser, err := s.bindingRepo.FindByProviderUID(ctx, oauthUser.Provider, oauthUser.ProviderUserID)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if existingUser != nil {
		if !existingUser.IsActive() {
			return nil, nil, apperrors.ErrAccountDisabled
		}
		return existingUser, nil, nil
	}
	pending, appErr := s.storePendingBinding(ctx, oauthUser)
	return nil, pending, appErr
}

// createBindingStrict 幂等地为用户建立一条 OAuth 绑定；若该第三方身份
// 已绑定到其他账号则返回错误。
func (s *OAuthService) createBindingStrict(ctx context.Context, userID uint64, oauthUser *OAuthUserInfo) *apperrors.AppError {
	existingUser, err := s.bindingRepo.FindByProviderUID(ctx, oauthUser.Provider, oauthUser.ProviderUserID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "database error", err)
	}
	if existingUser != nil {
		if existingUser.ID != userID {
			return apperrors.New(apperrors.ErrCodeDuplicateEntry, "this third-party account is already linked to another user")
		}
		return nil
	}
	if err := s.bindingRepo.Bind(ctx, userID, oauthUser.Provider, oauthUser.ProviderUserID, oauthUser.Email); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to bind provider", err)
	}
	return nil
}

// issueOAuthLogin 签发 token 并记录登录成功。
func (s *OAuthService) issueOAuthLogin(ctx context.Context, user *model.User, provider string) (*LoginServiceResult, *apperrors.AppError) {
	tokens, appErr := s.tokenService.IssueTokens(ctx, user)
	if appErr != nil {
		return nil, appErr
	}
	_ = s.userRepo.UpdateLoginSuccess(ctx, strconv.FormatUint(user.ID, 10))
	return &LoginServiceResult{
		Tokens:  tokens,
		User:    userToInfo(user),
		Actions: computeSecurityActions(ctx, s.configService, user),
	}, nil
}

// findUserByIdentifier 按用户名查找，未命中再按邮箱查找。
func (s *OAuthService) findUserByIdentifier(ctx context.Context, account string) (*model.User, *apperrors.AppError) {
	user, err := s.userRepo.FindByUsername(ctx, account)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}
	if user == nil {
		user, err = s.userRepo.FindByEmail(ctx, account)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
		}
	}
	if user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	return user, nil
}

// verifyEmailCode 校验并一次性消费邮箱验证码。
func (s *OAuthService) verifyEmailCode(ctx context.Context, email, purpose, code string) *apperrors.AppError {
	stored, err := s.tokenRepo.GetVerificationCode(ctx, email, purpose)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidCode, "failed to verify code", err)
	}
	if stored == "" || stored != code {
		return apperrors.ErrInvalidCode
	}
	return nil
}
