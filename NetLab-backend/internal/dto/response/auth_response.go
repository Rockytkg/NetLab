package response



// UserInfo 是 GET /auth/userinfo 的响应。
type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Avatar   string   `json:"avatar,omitempty"`
	Email    string   `json:"email,omitempty"`
	Roles    []string `json:"roles"`
}

// LoginResult 是 POST /auth/login 及 passkey/OAuth 认证的响应。
type LoginResult struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserInfo `json:"user"`
	SigningKey   string   `json:"signing_key,omitempty"`
}

// RefreshTokenResult 是 POST /auth/refresh 的响应。
type RefreshTokenResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	SigningKey   string `json:"signing_key,omitempty"`
}

// RegisterResult 是 POST /auth/register 的响应。
type RegisterResult struct {
	Message string `json:"message"`
}

// CaptchaResult 是 GET /auth/captcha 的响应。
type CaptchaResult struct {
	CaptchaID    string `json:"captcha_id"`
	CaptchaImage string `json:"captcha_image"`
}

// SendCodeResult 是 POST /auth/send-code 的响应。
type SendCodeResult struct {
	Message  string `json:"message"`
	Cooldown int    `json:"cooldown"` // 秒
}

// PasskeyRegisterOptions 是 GET /auth/passkey/register-options 的响应。
type PasskeyRegisterOptions struct {
	Challenge              string                   `json:"challenge"`
	RP                     PasskeyRP                `json:"rp"`
	User                   PasskeyUser              `json:"user"`
	PubKeyCredParams       []PasskeyCredParam       `json:"pub_key_cred_params"`
	Timeout                int                      `json:"timeout"`
	Attestation            string                   `json:"attestation"`
	AuthenticatorSelection AuthenticatorSelection   `json:"authenticator_selection"`
}

// PasskeyRP 表示依赖方（Relying Party）信息。
type PasskeyRP struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// PasskeyUser 表示 WebAuthn 用户实体。
type PasskeyUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// PasskeyCredParam 表示公钥凭据参数。
type PasskeyCredParam struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

// AuthenticatorSelection 配置认证器要求。
type AuthenticatorSelection struct {
	AuthenticatorAttachment string `json:"authenticator_attachment,omitempty"`
	ResidentKey             string `json:"resident_key"`
	UserVerification        string `json:"user_verification"`
}

// PasskeyAuthOptions 是 GET /auth/passkey/auth-options 的响应。
type PasskeyAuthOptions struct {
	Challenge        string              `json:"challenge"`
	RPID             string              `json:"rp_id"`
	Timeout          int                 `json:"timeout"`
	UserVerification string              `json:"user_verification"`
	AllowCredentials []AllowCredential   `json:"allow_credentials,omitempty"`
}

// AllowCredential 表示用于认证的允许凭据。
type AllowCredential struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Transports []string `json:"transports,omitempty"`
}

// SystemConfig 是 GET /auth/config 的响应。
type SystemConfig struct {
	RegistrationEnabled bool            `json:"registration_enabled"`
	CaptchaEnabled      bool            `json:"captcha_enabled"`
	PasskeyEnabled      bool            `json:"passkey_enabled"`
	OAuthProviders      []OAuthProvider `json:"oauth_providers"`
	// ICPBeian 是显示在登录页页脚的 ICP 备案号。
	// 前端根据固定模板构建链接；不返回 URL。
	ICPBeian string `json:"icp_beian"`
	// PoliceBeian 是公安备案号。
	// 前端根据固定模板构建链接；不返回 URL。
	PoliceBeian string `json:"police_beian"`
}

// OAuthProvider 表示系统配置中已配置的 OAuth 提供商。
type OAuthProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Color   string `json:"color"`
	AuthURL string `json:"auth_url"`
}

// OAuthAuthorizeResult 是 GET /auth/oauth/authorize 的响应。
type OAuthAuthorizeResult struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// VerifyCodeResult 是 POST /auth/verify-code 的响应。
type VerifyCodeResult struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// MessageResponse 是通用的消息响应。
type MessageResponse struct {
	Message string `json:"message"`
}
