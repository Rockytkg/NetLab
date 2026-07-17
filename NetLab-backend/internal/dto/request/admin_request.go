package request

// UpdateSecurityParams 是 PUT /api/settings/security 的请求体。
type UpdateSecurityParams struct {
	RegistrationEnabled  bool `json:"registrationEnabled"`
	CaptchaEnabled       bool `json:"captchaEnabled"`
	PasskeyEnabled       bool `json:"passkeyEnabled"`
	PasswordResetEnabled bool `json:"passwordResetEnabled"`
	TwoFactorRequired    bool `json:"twoFactorRequired"`
	PasswordMaxAgeDays   int  `json:"passwordMaxAgeDays" binding:"min=0,max=3650"`
}

// UpdateBeianParams 是 PUT /api/settings/beian 的请求体。
type UpdateBeianParams struct {
	ICPBeian    string `json:"icpBeian" binding:"max=128"`
	PoliceBeian string `json:"policeBeian" binding:"max=128"`
}

// UpdateSMTPParams 是 PUT /api/settings/smtp 的请求体。
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

// TestSMTPParams 是 POST /api/settings/smtp/test 的请求体。
type TestSMTPParams struct {
	To string `json:"to" binding:"required,email,max=255"`
}

// UpdateOAuthParams 是 PUT /api/settings/oauth/:provider 的请求体。
// ClientSecret 留空或为掩码占位符时，后端保留既有密钥。
type UpdateOAuthParams struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"clientId" binding:"max=255"`
	ClientSecret string `json:"clientSecret" binding:"max=512"`
	RedirectURL  string `json:"redirectUrl" binding:"max=512"`
}

// BatchUpdateRoleParams 是 PUT /api/users/role 的请求体。
type BatchUpdateRoleParams struct {
	UserIDs []string `json:"userIds" binding:"required,min=1,dive,numeric"`
	Role    string   `json:"role" binding:"required,oneof=admin editor viewer"`
}

// UpdateUserParams 是 PUT /api/users/:id 的请求体。
type UpdateUserParams struct {
	Nickname         string `json:"nickname" binding:"required,max=64"`
	Phone            string `json:"phone" binding:"required"`
	Email            string `json:"email" binding:"required,email,max=255"`
	Role             string `json:"role" binding:"required,oneof=admin editor viewer"`
	Status           string `json:"status" binding:"required,oneof=active disabled locked"`
	DisableTwoFactor bool   `json:"disableTwoFactor"`
}

// CreateUserParams 是 POST /api/users 的请求体。
type CreateUserParams struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Nickname string `json:"nickname" binding:"required,max=64"`
	Phone    string `json:"phone" binding:"required"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Role     string `json:"role" binding:"required,oneof=admin editor viewer"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// BatchDeleteUsersParams 是 DELETE /api/users 的请求体。
type BatchDeleteUsersParams struct {
	UserIDs []string `json:"userIds" binding:"required,min=1,dive,numeric"`
}

// BatchResetPasswordParams 是 PUT /api/users/reset-password 的请求体。
type BatchResetPasswordParams struct {
	UserIDs     []string `json:"userIds" binding:"required,min=1,dive,uuid"`
	NewPassword string   `json:"newPassword" binding:"required,min=8,max=72"`
}

// ExportUsersParams 是 POST /api/users/export 的请求体。
// 仅按显式勾选的用户 ID 导出；上限防止超大 IN 查询与内存占用。
type ExportUsersParams struct {
	UserIDs []string `json:"userIds" binding:"required,min=1,max=1000,dive,numeric"`
}

// ImportUsersParams 是 POST /api/users/import 的 JSON 请求体。
type ImportUsersParams struct {
	Users []ImportUserParams `json:"users" binding:"required,min=1,max=1000,dive"`
}

// ImportUserParams 是单条待导入用户数据。表格文件由前端解析后传入。
type ImportUserParams struct {
	Username string `json:"username" binding:"required,max=64"`
	Nickname string `json:"nickname" binding:"required,max=64"`
	Phone    string `json:"phone" binding:"required,max=20"`
	Email    string `json:"email" binding:"required,max=255"`
	Role     string `json:"role" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=72"`
}
