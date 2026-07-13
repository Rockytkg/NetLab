package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	sysconfig "netlab-backend/internal/service/config"
)

// maxOAuthResponseBytes 限制单次 OAuth HTTP 响应读取量（1 MiB），
// 防止恶意/异常提供方返回超大响应耗尽内存。
const maxOAuthResponseBytes = 1 << 20

// oauthFetcher 是单个 OAuth 提供商的获取函数：交换授权码并取得第三方身份。
type oauthFetcher func(ctx context.Context, c *http.Client, cfg sysconfig.ProviderSettings, code string) (*OAuthUserInfo, error)

// oauthFetchers 是受支持提供商的注册表。
// 新增提供商只需实现一个 oauthFetcher 并在此注册。
var oauthFetchers = map[string]oauthFetcher{
	"github":  fetchGitHubUser,
	"google":  fetchGoogleUser,
	"linuxdo": fetchLinuxDOUser,
	"wechat":  fetchWeChatUser,
	"qq":      fetchQQUser,
}

// ─── 共享 HTTP 辅助 ───────────────────────────────────────────────────

// oauthRequest 执行一次 HTTP 请求，返回限制大小后的响应体。
func oauthRequest(ctx context.Context, c *http.Client, method, rawURL string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponseBytes))
}

// oauthExchangeCode 以 POST 表单方式交换授权码，解析 JSON token 响应。
// github / google / linuxdo 共享此模式。
func oauthExchangeCode(ctx context.Context, c *http.Client, tokenURL string, cfg sysconfig.ProviderSettings, code string, extra url.Values) (string, error) {
	form := url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
	}
	for k, vs := range extra {
		form[k] = vs
	}
	if cfg.RedirectURL != "" {
		form.Set("redirect_uri", cfg.RedirectURL)
	}
	body, err := oauthRequest(ctx, c, http.MethodPost, tokenURL, strings.NewReader(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Accept":       "application/json",
	})
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	var resp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error,omitempty"`
		ErrorDesc   string `json:"error_description,omitempty"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("token error: %s — %s", resp.Error, resp.ErrorDesc)
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response: %s", string(body))
	}
	return resp.AccessToken, nil
}

// oauthGetBearer 以 Bearer token 发起 GET 并返回响应体。
func oauthGetBearer(ctx context.Context, c *http.Client, rawURL, token string, headers map[string]string) ([]byte, error) {
	h := map[string]string{"Authorization": "Bearer " + token}
	for k, v := range headers {
		h[k] = v
	}
	return oauthRequest(ctx, c, http.MethodGet, rawURL, nil, h)
}

// ─── GitHub ───────────────────────────────────────────────────────────

func fetchGitHubUser(ctx context.Context, c *http.Client, cfg sysconfig.ProviderSettings, code string) (*OAuthUserInfo, error) {
	token, err := oauthExchangeCode(ctx, c, "https://github.com/login/oauth/access_token", cfg, code, nil)
	if err != nil {
		return nil, err
	}
	hdrs := map[string]string{"Accept": "application/vnd.github+json", "User-Agent": "NetLab-Backend"}

	emailsBody, err := oauthGetBearer(ctx, c, "https://api.github.com/user/emails", token, hdrs)
	if err != nil {
		return nil, fmt.Errorf("fetch emails: %w", err)
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(emailsBody, &emails); err != nil {
		return nil, fmt.Errorf("parse emails: %w", err)
	}
	email := pickVerifiedEmail(emails)
	if email == "" {
		return nil, fmt.Errorf("no verified email found for github user")
	}

	profileBody, err := oauthGetBearer(ctx, c, "https://api.github.com/user", token, hdrs)
	if err != nil {
		return nil, fmt.Errorf("fetch profile: %w", err)
	}
	var profile struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(profileBody, &profile); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	return &OAuthUserInfo{
		Provider:       "github",
		ProviderUserID: fmt.Sprintf("github_%d", profile.ID),
		Email:          email,
		Username:       profile.Login,
		AvatarURL:      profile.AvatarURL,
	}, nil
}

// pickVerifiedEmail 优先返回主要且已验证的邮箱，其次返回任一已验证邮箱。
func pickVerifiedEmail(emails []struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}) string {
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email
		}
	}
	return ""
}

// ─── Google ───────────────────────────────────────────────────────────

func fetchGoogleUser(ctx context.Context, c *http.Client, cfg sysconfig.ProviderSettings, code string) (*OAuthUserInfo, error) {
	token, err := oauthExchangeCode(ctx, c, "https://oauth2.googleapis.com/token", cfg, code, url.Values{"grant_type": {"authorization_code"}})
	if err != nil {
		return nil, err
	}
	body, err := oauthGetBearer(ctx, c, "https://www.googleapis.com/oauth2/v3/userinfo", token, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	var u struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("parse userinfo: %w", err)
	}
	if u.Email == "" || !u.EmailVerified {
		return nil, fmt.Errorf("google account has no verified email")
	}
	return &OAuthUserInfo{
		Provider:       "google",
		ProviderUserID: "google_" + u.Sub,
		Email:          u.Email,
		Username:       u.Name,
		AvatarURL:      u.Picture,
	}, nil
}

// ─── LinuxDO（基于 Discourse） ────────────────────────────────────────

func fetchLinuxDOUser(ctx context.Context, c *http.Client, cfg sysconfig.ProviderSettings, code string) (*OAuthUserInfo, error) {
	token, err := oauthExchangeCode(ctx, c, "https://connect.linux.do/oauth2/token", cfg, code, url.Values{"grant_type": {"authorization_code"}})
	if err != nil {
		return nil, err
	}
	body, err := oauthGetBearer(ctx, c, "https://connect.linux.do/oauth2/userinfo", token, map[string]string{
		"Accept":     "application/json",
		"User-Agent": "NetLab-Backend",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	var u struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("parse userinfo: %w", err)
	}
	return &OAuthUserInfo{
		Provider:       "linuxdo",
		ProviderUserID: "linuxdo_" + u.ID,
		Email:          u.Email,
		Username:       u.Username,
		AvatarURL:      u.AvatarURL,
	}, nil
}

// ─── WeChat（snsapi_login，不暴露邮箱） ────────────────────────────────

func fetchWeChatUser(ctx context.Context, c *http.Client, cfg sysconfig.ProviderSettings, code string) (*OAuthUserInfo, error) {
	u := "https://api.weixin.qq.com/sns/oauth2/access_token?appid=" + url.QueryEscape(cfg.ClientID) +
		"&secret=" + url.QueryEscape(cfg.ClientSecret) +
		"&code=" + url.QueryEscape(code) +
		"&grant_type=authorization_code"
	body, err := oauthRequest(ctx, c, http.MethodGet, u, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	var token struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("wechat token error [%d]: %s", token.ErrCode, token.ErrMsg)
	}

	u = "https://api.weixin.qq.com/sns/userinfo?access_token=" + url.QueryEscape(token.AccessToken) +
		"&openid=" + url.QueryEscape(token.OpenID)
	body, err = oauthRequest(ctx, c, http.MethodGet, u, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	var info struct {
		OpenID     string `json:"openid"`
		Nickname   string `json:"nickname"`
		HeadImgURL string `json:"headimgurl"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse userinfo: %w", err)
	}
	return &OAuthUserInfo{
		Provider:       "wechat",
		ProviderUserID: "wechat_" + info.OpenID,
		Email:          info.OpenID + "@wechat.oauth", // WeChat snsapi_login 不暴露 email；使用合成邮箱
		Username:       info.Nickname,
		AvatarURL:      info.HeadImgURL,
	}, nil
}

// ─── QQ ───────────────────────────────────────────────────────────────

func fetchQQUser(ctx context.Context, c *http.Client, cfg sysconfig.ProviderSettings, code string) (*OAuthUserInfo, error) {
	u := "https://graph.qq.com/oauth2.0/token?grant_type=authorization_code&client_id=" + url.QueryEscape(cfg.ClientID) +
		"&client_secret=" + url.QueryEscape(cfg.ClientSecret) +
		"&code=" + url.QueryEscape(code)
	if cfg.RedirectURL != "" {
		u += "&redirect_uri=" + url.QueryEscape(cfg.RedirectURL)
	}
	body, err := oauthRequest(ctx, c, http.MethodGet, u, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	// QQ token 响应为查询字符串格式：access_token=xxx&expires_in=yyy
	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	accessToken := vals.Get("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("qq token error: %s", string(body))
	}

	openID, err := qqFetchOpenID(ctx, c, accessToken)
	if err != nil {
		return nil, err
	}

	u = "https://graph.qq.com/user/get_user_info?access_token=" + url.QueryEscape(accessToken) +
		"&oauth_consumer_key=" + url.QueryEscape(cfg.ClientID) +
		"&openid=" + url.QueryEscape(openID)
	body, err = oauthRequest(ctx, c, http.MethodGet, u, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	var info struct {
		Nickname  string `json:"nickname"`
		FigureURL string `json:"figureurl_qq_2"` // 100x100 头像
		Email     string `json:"email"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse userinfo: %w", err)
	}
	email := info.Email
	if email == "" {
		email = openID + "@qq.openid" // 用于账号关联的合成邮箱
	}
	return &OAuthUserInfo{
		Provider:       "qq",
		ProviderUserID: "qq_" + openID,
		Email:          email,
		Username:       info.Nickname,
		AvatarURL:      info.FigureURL,
	}, nil
}

// qqFetchOpenID 获取 QQ OpenID。QQ 返回 callback(JSON) 包裹格式。
func qqFetchOpenID(ctx context.Context, c *http.Client, accessToken string) (string, error) {
	u := "https://graph.qq.com/oauth2.0/me?access_token=" + url.QueryEscape(accessToken)
	body, err := oauthRequest(ctx, c, http.MethodGet, u, nil, nil)
	if err != nil {
		return "", fmt.Errorf("fetch openid: %w", err)
	}
	// 去除 callback( ... ); 包裹
	s := strings.TrimSpace(string(body))
	s = strings.TrimPrefix(s, "callback(")
	s = strings.TrimSuffix(s, ");")
	s = strings.TrimSpace(s)
	var result struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"error"`
		ErrMsg  string `json:"error_description"`
	}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return "", fmt.Errorf("parse openid response: %w (body: %s)", err, string(body))
	}
	if result.OpenID == "" {
		return "", fmt.Errorf("qq openid error: %s", string(body))
	}
	return result.OpenID, nil
}
