package response

import "time"

// UserInfo 是 GET /auth/userinfo 的响应。
type UserInfo struct {
	ID                  string   `json:"id"`
	Username            string   `json:"username"`
	Nickname            string   `json:"nickname"`
	Phone               string   `json:"phone"`
	Avatar              string   `json:"avatar,omitempty"`
	Email               string   `json:"email,omitempty"`
	Role                string   `json:"role"`
	RoleName            string   `json:"roleName"`
	RoleID              string   `json:"roleId,omitempty"`
	Permissions         []string `json:"permissions"`
	TwoFactorEnabled    bool     `json:"twoFactorEnabled"`
	PreferredAuthMethod string   `json:"preferredAuthMethod,omitempty"`
	HasPasskey          bool     `json:"hasPasskey"`
}

// SecurityActions 描述登录后要求用户完成的强制安全动作。
type SecurityActions struct {
	RequirePasswordChange bool   `json:"requirePasswordChange"`
	RequireEmailChange    bool   `json:"requireEmailChange"`
	RequireTwoFactorSetup bool   `json:"requireTwoFactorSetup"`
	Reason                string `json:"reason,omitempty"`
}

// PendingOAuthBinding 是 OAuth 登录后待绑定本地账号的会话信息。
type PendingOAuthBinding struct {
	Token    string `json:"token"`
	Provider string `json:"provider"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
}

// LoginResult 是 POST /auth/login 及 passkey/OAuth 认证的响应。
type LoginResult struct {
	AccessToken         string               `json:"accessToken,omitempty"`
	RefreshToken        string               `json:"refreshToken,omitempty"`
	User                UserInfo             `json:"user,omitempty"`
	RequiresTwoFactor   bool                 `json:"requiresTwoFactor,omitempty"`
	TwoFactorToken      string               `json:"twoFactorToken,omitempty"`
	SecurityActions     SecurityActions      `json:"securityActions"`
	PendingOAuthBinding *PendingOAuthBinding `json:"pendingOAuthBinding,omitempty"`
}

// RefreshTokenResult 是 POST /auth/refresh 的响应。
type RefreshTokenResult struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// RegisterResult 是 POST /auth/register 的响应。
type RegisterResult struct {
	Message string `json:"message"`
}

// CaptchaResult 是 GET /auth/captcha 的响应。
type CaptchaResult struct {
	CaptchaID    string `json:"captchaId"`
	CaptchaImage string `json:"captchaImage"`
}

// SendCodeResult 是 POST /auth/send-code 的响应。
type SendCodeResult struct {
	Message  string `json:"message"`
	Cooldown int    `json:"cooldown"` // 秒
}

// Passkey 端点直接透传 go-webauthn 的 protocol 类型。

// SystemConfig 是 GET /auth/config 的响应。
type SystemConfig struct {
	RegistrationEnabled  bool            `json:"registrationEnabled"`
	CaptchaEnabled       bool            `json:"captchaEnabled"`
	PasskeyEnabled       bool            `json:"passkeyEnabled"`
	PasswordResetEnabled bool            `json:"passwordResetEnabled"`
	TwoFactorRequired    bool            `json:"twoFactorRequired"`
	OAuthProviders       []OAuthProvider `json:"oauthProviders"`
	// ICPBeian 是显示在登录页页脚的 ICP 备案号。
	// 前端根据固定模板构建链接；不返回 URL。
	ICPBeian string `json:"icpBeian"`
	// PoliceBeian 是公安备案号。
	// 前端根据固定模板构建链接；不返回 URL。
	PoliceBeian string `json:"policeBeian"`
}

// OAuthProvider 表示系统配置中已配置的 OAuth 提供商。
type OAuthProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Color   string `json:"color"`
	AuthURL string `json:"authUrl"`
}

// OAuthAuthorizeResult 是 GET /auth/oauth/authorize 的响应。
type OAuthAuthorizeResult struct {
	AuthURL string `json:"authUrl"`
	State   string `json:"state"`
}

// VerifyCodeResult 是 POST /auth/verify-code 的响应。
type VerifyCodeResult struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// TwoFactorSetupResult 是 POST /auth/2fa/setup 的响应。
type TwoFactorSetupResult struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauthUrl"`
	QRCode     string `json:"qrCode"` // data:image/png;base64
}

// TwoFactorEnableResult 是 POST /auth/2fa/enable 的响应。
// RecoveryCodes 为一次性恢复码明文，仅在此刻返回一次，后端只保存其哈希。
type TwoFactorEnableResult struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

// MessageResponse 是通用的消息响应。
type MessageResponse struct {
	Message string `json:"message"`
}

// OAuthBinding 是返回给账户中心的单条第三方绑定信息。
type OAuthBinding struct {
	Provider  string    `json:"provider"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
