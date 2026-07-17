package request

// LoginParams 是 POST /auth/login 的请求体。
// 登录时仅检查凭据是否存在——密码强度策略
// （最小长度等）在注册/重置时强制执行，而非在此处。若在登录时强制
// 最小长度，会在密码比对之前就拒绝有效的现有账户（例如已初始化的
// 默认 admin）。
type LoginParams struct {
	Username    string `json:"username" binding:"required,min=1,max=64"`
	Password    string `json:"password" binding:"required,max=72"`
	CaptchaID   string `json:"captchaId,omitempty"`
	CaptchaCode string `json:"captchaCode,omitempty"`
}

// RegisterParams 是 POST /auth/register 的请求体。
type RegisterParams struct {
	Username        string `json:"username" binding:"required,min=3,max=64"`
	Nickname        string `json:"nickname" binding:"required,max=64"`
	Phone           string `json:"phone" binding:"required"`
	Email           string `json:"email" binding:"required,email,max=255"`
	Password        string `json:"password" binding:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,eqfield=Password"`
	VerifyCode      string `json:"verifyCode" binding:"required,len=6"`
	CaptchaID       string `json:"captchaId,omitempty"`
	CaptchaCode     string `json:"captchaCode,omitempty"`
}

// RefreshTokenParams 是 POST /auth/refresh 的请求体。
type RefreshTokenParams struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// SendCodeParams 是 POST /auth/send-code 的请求体。
type SendCodeParams struct {
	CaptchaID   string `json:"captchaId,omitempty"`
	CaptchaCode string `json:"captchaCode,omitempty"`
	Email       string `json:"email" binding:"required,email,max=255"`
	Purpose     string `json:"purpose" binding:"required,oneof=register reset-password change-email"`
}

// ForgotPasswordParams 是 POST /auth/forgot-password 的请求体。
type ForgotPasswordParams struct {
	Email string `json:"email" binding:"required,email,max=255"`
}

// ResetPasswordParams 是 POST /auth/reset-password 的请求体。
type ResetPasswordParams struct {
	Email           string `json:"email" binding:"required,email,max=255"`
	VerifyCode      string `json:"verifyCode" binding:"required,len=6"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,eqfield=NewPassword"`
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
	Purpose string `json:"purpose" binding:"required,oneof=register reset-password change-email"`
}

// PasskeyVerificationParams 是 passkey 验证端点的请求体。
type PasskeyVerificationParams struct {
	Data map[string]interface{} `json:"-"`
}

// ChangePasswordParams 是 POST /auth/account/change-password 的请求体（已登录用户）。
type ChangePasswordParams struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=72"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,eqfield=NewPassword"`
}

type CompleteSecurityUpdateParams struct {
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,eqfield=NewPassword"`
	NewEmail        string `json:"newEmail" binding:"omitempty,email,max=255"`
	VerifyCode      string `json:"verifyCode" binding:"omitempty,len=6"`
}

// AccountEmailCodeParams 是 POST /auth/account/email-code 的请求体。
// 向当前登录用户自己的邮箱发送验证码，用于敏感操作的二次校验。
type AccountEmailCodeParams struct {
	Purpose string `json:"purpose" binding:"required,oneof=passkey disable-2fa"`
}

// ChangeEmailCodeParams 是 POST /auth/account/email-change-code 的请求体。
type ChangeEmailCodeParams struct {
	NewEmail string `json:"newEmail" binding:"required,email,max=255"`
}

// ChangeEmailParams 是 PUT /auth/account/email 的请求体。
type ChangeEmailParams struct {
	NewEmail   string `json:"newEmail" binding:"required,email,max=255"`
	VerifyCode string `json:"verifyCode" binding:"required,len=6"`
}

type OAuthBindExistingParams struct {
	PendingToken string `json:"pendingToken" binding:"required"`
	Account      string `json:"account" binding:"required,max=255"`
	VerifyCode   string `json:"verifyCode" binding:"required,len=6"`
}

type OAuthCreateAccountParams struct {
	PendingToken    string `json:"pendingToken" binding:"required"`
	Username        string `json:"username" binding:"required,min=3,max=64"`
	Nickname        string `json:"nickname" binding:"required,max=64"`
	Phone           string `json:"phone" binding:"required"`
	Email           string `json:"email" binding:"required,email,max=255"`
	Password        string `json:"password" binding:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,eqfield=Password"`
	VerifyCode      string `json:"verifyCode" binding:"required,len=6"`
}

// EnableTwoFactorParams is the request body of POST /auth/2fa/enable.
type EnableTwoFactorParams struct {
	Code string `json:"code" binding:"required,len=6"`
}

// DisableTwoFactorParams is the request body of POST /auth/2fa/disable.
type DisableTwoFactorParams struct {
	VerifyCode string `json:"verifyCode" binding:"required,len=6"`
}

// VerifyTwoFactorParams is the request body of POST /auth/login/2fa.
type VerifyTwoFactorParams struct {
	TwoFactorToken string `json:"twoFactorToken" binding:"required"`
	Code           string `json:"code" binding:"required,len=6"`
}

// RecoveryLoginParams is the request body of POST /auth/login/recovery.
type RecoveryLoginParams struct {
	TwoFactorToken string `json:"twoFactorToken" binding:"required"`
	RecoveryCode   string `json:"recoveryCode" binding:"required"`
}

// PreferredAuthMethodParams is the request body of PUT /auth/account/preferred-auth-method.
type PreferredAuthMethodParams struct {
	Method string `json:"method" binding:"required,oneof=totp passkey"`
}
