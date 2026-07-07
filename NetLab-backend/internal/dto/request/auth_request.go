package request

// LoginParams 是 POST /auth/login 的请求体。
// 登录时仅检查凭据是否存在——密码强度策略
//（最小长度等）在注册/重置时强制执行，而非在此处。若在登录时强制
// 最小长度，会在密码比对之前就拒绝有效的现有账户（例如已初始化的
// 默认 admin）。
type LoginParams struct {
	Username    string `json:"username" binding:"required,min=1,max=64"`
	Password    string `json:"password" binding:"required,max=128"`
	CaptchaID   string `json:"captcha_id,omitempty"`
	CaptchaCode string `json:"captcha_code,omitempty"`
}

// RegisterParams 是 POST /auth/register 的请求体。
type RegisterParams struct {
	Username        string `json:"username" binding:"required,min=3,max=64"`
	Email           string `json:"email" binding:"required,email,max=255"`
	Password        string `json:"password" binding:"required,min=8,max=128"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
	VerifyCode      string `json:"verify_code" binding:"required,len=6"`
	CaptchaID       string `json:"captcha_id,omitempty"`
	CaptchaCode     string `json:"captcha_code,omitempty"`
}

// RefreshTokenParams 是 POST /auth/refresh 的请求体。
type RefreshTokenParams struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// SendCodeParams 是 POST /auth/send-code 的请求体。
type SendCodeParams struct {
	Email   string `json:"email" binding:"required,email,max=255"`
	Purpose string `json:"purpose" binding:"required,oneof=register reset-password"`
}

// ForgotPasswordParams 是 POST /auth/forgot-password 的请求体。
type ForgotPasswordParams struct {
	Email string `json:"email" binding:"required,email,max=255"`
}

// ResetPasswordParams 是 POST /auth/reset-password 的请求体。
type ResetPasswordParams struct {
	Email           string `json:"email" binding:"required,email,max=255"`
	VerifyCode      string `json:"verify_code" binding:"required,len=6"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

// OAuthCallbackParams 是 POST /auth/oauth/callback 的请求体。
type OAuthCallbackParams struct {
	Provider string `json:"provider" binding:"required"`
	Code     string `json:"code" binding:"required"`
	State    string `json:"state" binding:"required"`
}

// VerifyCodeParams 是 POST /auth/verify-code 的请求体。
type VerifyCodeParams struct {
	Email   string `json:"email" binding:"required,email,max=255"`
	Code    string `json:"code" binding:"required,len=6"`
	Purpose string `json:"purpose" binding:"required,oneof=register reset-password"`
}

// PasskeyVerificationParams 是 passkey 验证端点的请求体。
type PasskeyVerificationParams struct {
	Data map[string]interface{} `json:"-"`
}
