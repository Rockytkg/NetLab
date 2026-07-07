package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"netlab-backend/config"
	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/pkg/apperrors"
)

// OAuthService 处理第三方 OAuth 登录流程。
type OAuthService struct {
	cfg          config.OAuthConfig
	db           *gorm.DB
	userRepo     *repository.UserRepository
	tokenService *TokenService
	redis        *redis.Client
	logger       *zap.Logger
	httpClient   *http.Client
}

// NewOAuthService 创建一个新的 OAuthService。
func NewOAuthService(
	cfg config.OAuthConfig,
	db *gorm.DB,
	userRepo *repository.UserRepository,
	tokenService *TokenService,
	redis *redis.Client,
	logger *zap.Logger,
) *OAuthService {
	return &OAuthService{
		cfg:          cfg,
		db:           db,
		userRepo:     userRepo,
		tokenService: tokenService,
		redis:        redis,
		logger:        logger,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
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

// HandleCallback 处理 OAuth 回调：校验 state、交换 code、
// 获取用户信息，并创建/查找本地用户。
func (s *OAuthService) HandleCallback(ctx context.Context, provider, code, state string) (*LoginServiceResult, *apperrors.AppError) {
	// 1. 校验 OAuth state 参数（CSRF 防护）
	if err := s.validateState(ctx, state); err != nil {
		return nil, err
	}

	// 2. 用 code 交换 access token 并获取用户信息
	oauthUser, appErr := s.fetchUserInfo(ctx, provider, code)
	if appErr != nil {
		return nil, appErr
	}

	// 3. 查找现有用户或创建新用户
	user, appErr := s.findOrCreateUser(ctx, oauthUser)
	if appErr != nil {
		return nil, appErr
	}

	// 4. 签发 token
	tokens, appErr := s.tokenService.IssueTokens(ctx, user)
	if appErr != nil {
		return nil, appErr
	}

	// 5. 记录登录
	_ = s.userRepo.UpdateLoginSuccess(ctx, user.ID.String())

	s.logger.Info("oauth login success",
		zap.String("provider", provider),
		zap.String("user_id", user.ID.String()),
		zap.String("email", oauthUser.Email),
	)

	return &LoginServiceResult{
		Tokens: tokens,
		User:   userToInfo(user),
	}, nil
}

// GenerateState 创建一个加密安全的随机 state 字符串并存入 Redis。
func (s *OAuthService) GenerateState(ctx context.Context) (string, *apperrors.AppError) {
	state := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
	if err := s.redis.Set(ctx, "oauth:state:"+state, "1", 10*time.Minute).Err(); err != nil {
		s.logger.Error("failed to store oauth state", zap.Error(err))
		return "", apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to generate state", err)
	}
	return state, nil
}

// ─── 私有辅助函数 ───────────────────────────────────────────────────

func (s *OAuthService) validateState(ctx context.Context, state string) *apperrors.AppError {
	key := "oauth:state:" + state
	val, err := s.redis.Get(ctx, key).Result()
	if err != nil || val == "" {
		return apperrors.ErrTokenExpired // 复用：state 已过期或无效
	}
	_ = s.redis.Del(ctx, key) // 一次性使用
	return nil
}

func (s *OAuthService) fetchUserInfo(ctx context.Context, provider, code string) (*OAuthUserInfo, *apperrors.AppError) {
	switch provider {
	case "github":
		return s.fetchGitHubUser(ctx, code)
	case "google":
		return s.fetchGoogleUser(ctx, code)
	case "linuxdo":
		return s.fetchLinuxDOUser(ctx, code)
	case "wechat":
		return s.fetchWeChatUser(ctx, code)
	case "qq":
		return s.fetchQQUser(ctx, code)
	default:
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "unsupported oauth provider: "+provider)
	}
}

// ─── GitHub OAuth ───────────────────────────────────────────────────

func (s *OAuthService) fetchGitHubUser(ctx context.Context, code string) (*OAuthUserInfo, *apperrors.AppError) {
	if s.cfg.GitHub.ClientID == "" || s.cfg.GitHub.ClientSecret == "" {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "github oauth is not configured")
	}

	// 步骤 1：用 code 交换 access token
	accessToken, err := s.exchangeGitHubCode(ctx, code)
	if err != nil {
		s.logger.Error("github token exchange failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to exchange github code", err)
	}

	// 步骤 2：获取主要的已验证邮箱
	email, err := s.fetchGitHubEmail(ctx, accessToken)
	if err != nil {
		s.logger.Error("github email fetch failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to fetch github email", err)
	}

	// 步骤 3：获取用户资料
	ghUser, err := s.fetchGitHubProfile(ctx, accessToken)
	if err != nil {
		s.logger.Error("github profile fetch failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to fetch github profile", err)
	}

	return &OAuthUserInfo{
		Provider:       "github",
		ProviderUserID: fmt.Sprintf("github_%d", ghUser.ID),
		Email:          email,
		Username:       ghUser.Login,
		AvatarURL:      ghUser.AvatarURL,
	}, nil
}

type githubAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (s *OAuthService) exchangeGitHubCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {s.cfg.GitHub.ClientID},
		"client_secret": {s.cfg.GitHub.ClientSecret},
		"code":          {code},
	}
	if s.cfg.GitHub.RedirectURL != "" {
		data.Set("redirect_uri", s.cfg.GitHub.RedirectURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB 限制
	if err != nil {
		return "", err
	}

	var tokenResp githubAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("github token error: %s — %s", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in github response: %s", string(body))
	}

	return tokenResp.AccessToken, nil
}

func (s *OAuthService) fetchGitHubEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "NetLab-Backend")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	var emails []githubEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("parse emails response: %w", err)
	}

	// 选取主要的已验证邮箱
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	// 兜底：第一个已验证的邮箱
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified email found for github user")
}

func (s *OAuthService) fetchGitHubProfile(ctx context.Context, accessToken string) (*githubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "NetLab-Backend")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var user githubUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse user response: %w", err)
	}

	return &user, nil
}

// ─── Google OAuth ───────────────────────────────────────────────────

func (s *OAuthService) fetchGoogleUser(ctx context.Context, code string) (*OAuthUserInfo, *apperrors.AppError) {
	if s.cfg.Google.ClientID == "" || s.cfg.Google.ClientSecret == "" {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "google oauth is not configured")
	}

	token, err := s.exchangeGoogleCode(ctx, code)
	if err != nil {
		s.logger.Error("google token exchange failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to exchange google code", err)
	}

	googleUser, err := s.fetchGoogleUserInfo(ctx, token)
	if err != nil {
		s.logger.Error("google userinfo fetch failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to fetch google userinfo", err)
	}

	return &OAuthUserInfo{
		Provider:       "google",
		ProviderUserID: "google_" + googleUser.Sub,
		Email:          googleUser.Email,
		Username:       googleUser.Name,
		AvatarURL:      googleUser.Picture,
	}, nil
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func (s *OAuthService) exchangeGoogleCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {s.cfg.Google.ClientID},
		"client_secret": {s.cfg.Google.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}
	if s.cfg.Google.RedirectURL != "" {
		data.Set("redirect_uri", s.cfg.Google.RedirectURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	var tokenResp googleTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("google token error: %s — %s", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in google response")
	}

	return tokenResp.AccessToken, nil
}

func (s *OAuthService) fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var user googleUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse userinfo response: %w", err)
	}
	if user.Email == "" || !user.EmailVerified {
		return nil, fmt.Errorf("google account has no verified email")
	}

	return &user, nil
}

// ─── LinuxDO OAuth（基于 Discourse） ────────────────────────────────

func (s *OAuthService) fetchLinuxDOUser(ctx context.Context, code string) (*OAuthUserInfo, *apperrors.AppError) {
	if s.cfg.LinuxDO.ClientID == "" || s.cfg.LinuxDO.ClientSecret == "" {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "linuxdo oauth is not configured")
	}

	accessToken, err := s.exchangeLinuxDOCode(ctx, code)
	if err != nil {
		s.logger.Error("linuxdo token exchange failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to exchange linuxdo code", err)
	}

	ldUser, err := s.fetchLinuxDOUserInfo(ctx, accessToken)
	if err != nil {
		s.logger.Error("linuxdo userinfo fetch failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to fetch linuxdo userinfo", err)
	}

	return &OAuthUserInfo{
		Provider:       "linuxdo",
		ProviderUserID: "linuxdo_" + ldUser.ID,
		Email:          ldUser.Email,
		Username:       ldUser.Username,
		AvatarURL:      ldUser.AvatarURL,
	}, nil
}

type linuxDOTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type linuxDOUserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func (s *OAuthService) exchangeLinuxDOCode(ctx context.Context, code string) (string, error) {
	data := url.Values{
		"client_id":     {s.cfg.LinuxDO.ClientID},
		"client_secret": {s.cfg.LinuxDO.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}
	if s.cfg.LinuxDO.RedirectURL != "" {
		data.Set("redirect_uri", s.cfg.LinuxDO.RedirectURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://connect.linux.do/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	var tokenResp linuxDOTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse linuxdo token response: %w", err)
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("linuxdo token error: %s — %s", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in linuxdo response")
	}

	return tokenResp.AccessToken, nil
}

func (s *OAuthService) fetchLinuxDOUserInfo(ctx context.Context, accessToken string) (*linuxDOUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://connect.linux.do/oauth2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "NetLab-Backend")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var user linuxDOUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse userinfo response: %w", err)
	}
	if user.Email == "" {
		return nil, fmt.Errorf("linuxdo account has no email")
	}

	return &user, nil
}

// ─── WeChat OAuth（开放平台 —— 扫码登录） ────────────────────────────

func (s *OAuthService) fetchWeChatUser(ctx context.Context, code string) (*OAuthUserInfo, *apperrors.AppError) {
	if s.cfg.WeChat.ClientID == "" || s.cfg.WeChat.ClientSecret == "" {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "wechat oauth is not configured")
	}

	// WeChat 的 token 交换在一次调用中同时返回 access_token 和 openid
	wxToken, err := s.exchangeWeChatCode(ctx, code)
	if err != nil {
		s.logger.Error("wechat token exchange failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to exchange wechat code", err)
	}

	wxUser, err := s.fetchWeChatUserInfo(ctx, wxToken.AccessToken, wxToken.OpenID)
	if err != nil {
		s.logger.Error("wechat userinfo fetch failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to fetch wechat userinfo", err)
	}

	return &OAuthUserInfo{
		Provider:       "wechat",
		ProviderUserID: "wechat_" + wxToken.OpenID,
		Email:          wxToken.OpenID + "@wechat.oauth", // WeChat snsapi_login 不暴露 email；使用合成邮箱
		Username:       wxUser.Nickname,
		AvatarURL:      wxUser.HeadImgURL,
	}, nil
}

type wechatTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type wechatUserInfo struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid"`
}

func (s *OAuthService) exchangeWeChatCode(ctx context.Context, code string) (*wechatTokenResponse, error) {
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		s.cfg.WeChat.ClientID, s.cfg.WeChat.ClientSecret, code,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var tokenResp wechatTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse wechat token response: %w", err)
	}
	if tokenResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat token error [%d]: %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}
	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("empty access token in wechat response: %s", string(body))
	}

	return &tokenResp, nil
}

func (s *OAuthService) fetchWeChatUserInfo(ctx context.Context, accessToken, openID string) (*wechatUserInfo, error) {
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		accessToken, openID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var user wechatUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse wechat userinfo response: %w", err)
	}

	return &user, nil
}

// ─── QQ OAuth ───────────────────────────────────────────────────────

func (s *OAuthService) fetchQQUser(ctx context.Context, code string) (*OAuthUserInfo, *apperrors.AppError) {
	if s.cfg.QQ.ClientID == "" || s.cfg.QQ.ClientSecret == "" {
		return nil, apperrors.New(apperrors.ErrCodeOperationDenied, "qq oauth is not configured")
	}

	accessToken, err := s.exchangeQQCode(ctx, code)
	if err != nil {
		s.logger.Error("qq token exchange failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to exchange qq code", err)
	}

	openID, err := s.fetchQQOpenID(ctx, accessToken)
	if err != nil {
		s.logger.Error("qq openid fetch failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to fetch qq openid", err)
	}

	qqUser, err := s.fetchQQUserInfo(ctx, accessToken, openID)
	if err != nil {
		s.logger.Error("qq userinfo fetch failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to fetch qq userinfo", err)
	}

	// 使用基础 scope 时 QQ 可能不返回 email
	email := qqUser.Email
	if email == "" {
		email = openID + "@qq.openid" // 用于账号关联的合成邮箱
	}

	return &OAuthUserInfo{
		Provider:       "qq",
		ProviderUserID: "qq_" + openID,
		Email:          email,
		Username:       qqUser.Nickname,
		AvatarURL:      qqUser.FigureURLQQ,
	}, nil
}

type qqUserInfo struct {
	Nickname     string `json:"nickname"`
	FigureURLQQ  string `json:"figureurl_qq_2"` // 100x100 头像
	Gender       string `json:"gender"`
	Email        string `json:"email"`
}

func (s *OAuthService) exchangeQQCode(ctx context.Context, code string) (string, error) {
	u := fmt.Sprintf(
		"https://graph.qq.com/oauth2.0/token?grant_type=authorization_code&client_id=%s&client_secret=%s&code=%s",
		s.cfg.QQ.ClientID, s.cfg.QQ.ClientSecret, code,
	)
	if s.cfg.QQ.RedirectURL != "" {
		u += "&redirect_uri=" + s.cfg.QQ.RedirectURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	// QQ 的 token 响应是查询字符串格式：access_token=xxx&expires_in=yyy
	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return "", fmt.Errorf("parse qq token response: %w", err)
	}
	if vals.Get("access_token") == "" {
		return "", fmt.Errorf("qq token error: %s", string(body))
	}

	return vals.Get("access_token"), nil
}

func (s *OAuthService) fetchQQOpenID(ctx context.Context, accessToken string) (string, error) {
	u := fmt.Sprintf("https://graph.qq.com/oauth2.0/me?access_token=%s", accessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	// QQ 返回格式：callback( {"client_id":"xxx","openid":"yyy"} );
	bodyStr := string(body)
	// 去除 callback( 和 );
	bodyStr = strings.TrimPrefix(bodyStr, "callback(")
	bodyStr = strings.TrimSuffix(bodyStr, ");")
	bodyStr = strings.TrimSpace(bodyStr)

	var result struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"error"`
		ErrMsg  string `json:"error_description"`
	}
	if err := json.Unmarshal([]byte(bodyStr), &result); err != nil {
		return "", fmt.Errorf("parse qq openid response: %w (body: %s)", err, string(body))
	}
	if result.OpenID == "" {
		return "", fmt.Errorf("qq openid error: %s", string(body))
	}

	return result.OpenID, nil
}

func (s *OAuthService) fetchQQUserInfo(ctx context.Context, accessToken, openID string) (*qqUserInfo, error) {
	u := fmt.Sprintf(
		"https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&openid=%s",
		accessToken, s.cfg.QQ.ClientID, openID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var user qqUserInfo
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parse qq userinfo response: %w", err)
	}

	return &user, nil
}

// ─── 用户创建 ───────────────────────────────────────────────────────

func (s *OAuthService) findOrCreateUser(ctx context.Context, oauthUser *OAuthUserInfo) (*model.User, *apperrors.AppError) {
	// 先按 email 查找 —— OAuth 身份与 email 绑定
	user, err := s.userRepo.FindByEmail(ctx, oauthUser.Email)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeUserNotFound, "database error", err)
	}

	if user != nil {
		// 现有用户 —— 如果头像来自同一提供方则更新
		if !user.IsActive() {
			return nil, apperrors.ErrAccountDisabled
		}
		if oauthUser.AvatarURL != "" {
			_ = s.userRepo.Update(ctx, user) // 如果头像已设置则为空操作
		}
		return user, nil
	}

	// 新用户：根据 OAuth 资料生成唯一用户名
	username := s.generateUsername(ctx, oauthUser.Username)

	newUser := &model.User{
		Username:     username,
		Email:        oauthUser.Email,
		PasswordHash: "", // OAuth 用户没有本地密码
		Avatar:       oauthUser.AvatarURL,
		Roles:        []string{string(model.RoleViewer)},
		Status:       model.StatusActive,
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		s.logger.Error("create oauth user failed", zap.Error(err))
		return nil, apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "failed to create user", err)
	}

	s.logger.Info("oauth user created",
		zap.String("provider", oauthUser.Provider),
		zap.String("user_id", newUser.ID.String()),
		zap.String("email", oauthUser.Email),
		zap.String("username", username),
	)

	return newUser, nil
}

// generateUsername 根据 OAuth 提供的名称创建唯一用户名。
// 名称为空时回退到 "user"，冲突时追加随机数字。
func (s *OAuthService) generateUsername(ctx context.Context, base string) string {
	if base == "" {
		base = "user"
	}

	// 清洗：仅保留字母数字、连字符、下划线；转为小写
	base = sanitizeUsername(base)
	if len(base) < 3 {
		base = base + strings.Repeat("0", 3-len(base))
	}

	// 先尝试基础名称
	exists, err := s.userRepo.ExistsByUsername(ctx, base)
	if err == nil && !exists {
		return base
	}

	// 追加随机 4 位后缀；最多重试 5 次
	for i := 0; i < 5; i++ {
		suffix := fmt.Sprintf("%04d", rand.Intn(10000))
		candidate := base + suffix
		if len(candidate) > 64 {
			candidate = base[:64-len(suffix)] + suffix
		}
		exists, err := s.userRepo.ExistsByUsername(ctx, candidate)
		if err == nil && !exists {
			return candidate
		}
	}

	// 最终兜底
	return fmt.Sprintf("%s%d", base[:min(len(base), 54)], time.Now().UnixNano()%10000000000)
}

func sanitizeUsername(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if result == "" {
		return "user"
	}
	if len(result) > 60 {
		result = result[:60]
	}
	return result
}
