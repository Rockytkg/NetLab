package request

// UpdateSecurityParams 是 PUT /api/admin/settings/security 的请求体。
type UpdateSecurityParams struct {
	RegistrationEnabled  bool `json:"registrationEnabled"`
	CaptchaEnabled       bool `json:"captchaEnabled"`
	PasskeyEnabled       bool `json:"passkeyEnabled"`
	PasswordResetEnabled bool `json:"passwordResetEnabled"`
	TwoFactorRequired    bool `json:"twoFactorRequired"`
	PasswordMaxAgeDays   int  `json:"passwordMaxAgeDays" binding:"min=0,max=3650"`
}

// UpdateBeianParams 是 PUT /api/admin/settings/beian 的请求体。
type UpdateBeianParams struct {
	ICPBeian    string `json:"icpBeian" binding:"max=128"`
	PoliceBeian string `json:"policeBeian" binding:"max=128"`
}

// UpdateSMTPParams 是 PUT /api/admin/settings/smtp 的请求体。
// Password 留空或为掩码占位符时，后端保留既有密钥。
type UpdateSMTPParams struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host" binding:"max=255"`
	Port     int    `json:"port" binding:"min=0,max=65535"`
	Username string `json:"username" binding:"max=255"`
	Password string `json:"password" binding:"max=512"`
	From     string `json:"from" binding:"max=255"`
	UseTLS   bool   `json:"useTls"`
}

// TestSMTPParams 是 POST /api/admin/settings/smtp/test 的请求体。
type TestSMTPParams struct {
	To string `json:"to" binding:"required,email,max=255"`
}

// UpdateOAuthParams 是 PUT /api/admin/settings/oauth/:provider 的请求体。
// ClientSecret 留空或为掩码占位符时，后端保留既有密钥。
type UpdateOAuthParams struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"clientId" binding:"max=255"`
	ClientSecret string `json:"clientSecret" binding:"max=512"`
	RedirectURL  string `json:"redirectUrl" binding:"max=512"`
}

// BatchUpdateRoleParams 是 PUT /api/admin/users/role 的请求体。
type BatchUpdateRoleParams struct {
	UserIDs []string `json:"userIds" binding:"required,min=1,dive,uuid"`
	Role    string   `json:"role" binding:"required,oneof=admin editor viewer"`
}

// UpdateUserParams 是 PUT /api/admin/users/:id 的请求体。
type UpdateUserParams struct {
	Email  string `json:"email" binding:"required,email,max=255"`
	Role   string `json:"role" binding:"required,oneof=admin editor viewer"`
	Status string `json:"status" binding:"required,oneof=active disabled locked"`
}

// CreateUserParams 是 POST /api/admin/users 的请求体。
type CreateUserParams struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Role     string `json:"role" binding:"required,oneof=admin editor viewer"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

// BatchDeleteUsersParams 是 DELETE /api/admin/users 的请求体。
type BatchDeleteUsersParams struct {
	UserIDs []string `json:"userIds" binding:"required,min=1,dive,uuid"`
}

// BatchResetPasswordParams 是 PUT /api/admin/users/reset-password 的请求体。
type BatchResetPasswordParams struct {
	UserIDs     []string `json:"userIds" binding:"required,min=1,dive,uuid"`
	NewPassword string   `json:"newPassword" binding:"required,min=8,max=128"`
}
