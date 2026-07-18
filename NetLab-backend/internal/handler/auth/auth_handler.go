package auth

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"netlab-backend/internal/contextkeys"
	"netlab-backend/internal/dto/request"
	dtoresponse "netlab-backend/internal/dto/response"
	"netlab-backend/internal/middleware"
	"netlab-backend/internal/model"
	authsvc "netlab-backend/internal/service/auth"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/captcha"
	"netlab-backend/pkg/response"
)

// AuthHandler 处理认证相关端点的 HTTP 请求。
type AuthHandler struct {
	authService      *authsvc.AuthService
	verificationSvc  *authsvc.VerificationService
	passwordSvc      *authsvc.PasswordService
	passkeyService   *authsvc.PasskeyService
	tokenService     *authsvc.TokenService
	oauthService     *authsvc.OAuthService
	twoFactorService *authsvc.TwoFactorService
	captchaMgr       *captcha.Manager
	permLister       PermissionLister
	logger           *zap.Logger
}

// PermissionLister 提供通过角色名查询权限键列表的能力。
type PermissionLister interface {
	PermissionKeysForRoleID(roleID string) []string
	RoleNameForID(roleID string) string
	RoleNameForIdentifier(identifier string) string
}

// NewAuthHandler 创建一个新的 AuthHandler。
func NewAuthHandler(
	authService *authsvc.AuthService,
	verificationSvc *authsvc.VerificationService,
	passwordSvc *authsvc.PasswordService,
	passkeyService *authsvc.PasskeyService,
	tokenService *authsvc.TokenService,
	oauthService *authsvc.OAuthService,
	twoFactorService *authsvc.TwoFactorService,
	captchaMgr *captcha.Manager,
	permLister PermissionLister,
	logger *zap.Logger,
) *AuthHandler {
	return &AuthHandler{
		authService:      authService,
		verificationSvc:  verificationSvc,
		passwordSvc:      passwordSvc,
		passkeyService:   passkeyService,
		tokenService:     tokenService,
		oauthService:     oauthService,
		twoFactorService: twoFactorService,
		captchaMgr:       captchaMgr,
		permLister:       permLister,
		logger:           logger,
	}
}

// Login 处理 POST /api/auth/login
// @Summary      User login
// @Description  Authenticate with username/password and receive JWT tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.LoginParams  true  "Login credentials"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.LoginResult}
// @Failure      401   {object}  response.ApiResponse
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var params request.LoginParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}

	// 当系统配置中启用验证码时强制校验
	config, _ := h.authService.GetSystemConfig(c.Request.Context())
	if config != nil && config.CaptchaEnabled {
		if params.CaptchaID == "" || params.CaptchaCode == "" {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "captcha is required"))
			return
		}
		ok, err := h.captchaMgr.Verify(params.CaptchaID, params.CaptchaCode)
		if err != nil || !ok {
			response.Error(c, apperrors.ErrInvalidCode)
			return
		}
	}

	result, appErr := h.authService.Login(c.Request.Context(), params.Username, params.Password)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	h.applyRoleInfo(result.User)
	response.SuccessOK(c, loginResultToDTO(result))
}

// RefreshToken 处理 POST /api/auth/refresh
// @Summary      Refresh access token
// @Description  Exchange a refresh token for a new access token pair
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.RefreshTokenParams  true  "Refresh token"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.RefreshTokenResult}
// @Failure      401   {object}  response.ApiResponse
// @Router       /api/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var params request.RefreshTokenParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidRefreshToken, "invalid request parameters"))
		return
	}

	result, appErr := h.tokenService.RefreshTokens(c.Request.Context(), params.RefreshToken)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.SuccessOK(c, dtoresponse.RefreshTokenResult{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

// GetUserInfo 处理 GET /api/auth/userinfo
// @Summary      Get current user info
// @Description  Returns the authenticated user's profile information
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.UserInfo}
// @Failure      401  {object}  response.ApiResponse
// @Router       /api/auth/userinfo [get]
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	user, appErr := h.authService.GetUserInfo(c.Request.Context(), userID)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	result := userModelToResult(user)
	h.applyRoleInfo(result)
	if hasPasskey, err := h.passkeyService.HasPasskey(c.Request.Context(), userID); err == nil {
		result.HasPasskey = hasPasskey
	}
	response.SuccessOK(c, userInfoToDTO(result))
}

// Logout 处理 POST /api/auth/logout
// @Summary      Logout
// @Description  Revoke current tokens and clear session
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      401  {object}  response.ApiResponse
// @Router       /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if err := h.authService.Logout(c.Request.Context(), userID); err != nil {
		h.logger.Error("logout failed", zap.Error(err))
	}

	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "logged out"})
}

// GetCaptcha 处理 GET /api/auth/captcha
// @Summary      Get captcha
// @Description  Generate and return a captcha image and ID
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.CaptchaResult}
// @Router       /api/auth/captcha [get]
func (h *AuthHandler) GetCaptcha(c *gin.Context) {
	result, err := h.captchaMgr.Generate()
	if err != nil {
		h.logger.Error("generate captcha failed", zap.Error(err))
		response.InternalError(c, "failed to generate captcha")
		return
	}

	response.SuccessOK(c, dtoresponse.CaptchaResult{
		CaptchaID:    result.CaptchaID,
		CaptchaImage: "data:image/png;base64," + result.CaptchaImage,
	})
}

// Register 处理 POST /api/auth/register
// @Summary      Register new user
// @Description  Create a new user account
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.RegisterParams  true  "Registration details"
// @Success      201   {object}  response.ApiResponse{data=dtoresponse.RegisterResult}
// @Failure      409   {object}  response.ApiResponse
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var params request.RegisterParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}

	// 检查是否需要验证码
	config, _ := h.authService.GetSystemConfig(c.Request.Context())
	if config != nil && config.CaptchaEnabled {
		if params.CaptchaID == "" || params.CaptchaCode == "" {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "captcha is required"))
			return
		}
		ok, err := h.captchaMgr.Verify(params.CaptchaID, params.CaptchaCode)
		if err != nil || !ok {
			response.Error(c, apperrors.ErrInvalidCode)
			return
		}
	}

	if err := h.authService.Register(c.Request.Context(), params.Username, params.Nickname, params.Phone, params.Email, params.Password, params.VerifyCode); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessCreated(c, dtoresponse.RegisterResult{Message: "registration successful"})
}

// SendCode 处理 POST /api/auth/send-code
// @Summary      Send verification code
// @Description  Send a 6-digit verification code to the specified email
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.SendCodeParams  true  "Email and purpose"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.SendCodeResult}
// @Failure      429   {object}  response.ApiResponse
// @Router       /api/auth/send-code [post]
func (h *AuthHandler) SendCode(c *gin.Context) {
	var params request.SendCodeParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters"))
		return
	}

	locale := contextkeys.GetLocale(c)
	var cooldown int
	var appErr *apperrors.AppError
	if params.Purpose == "reset-password" {
		cooldown, appErr = h.passwordSvc.SendResetCode(c.Request.Context(), params.Email, locale, params.CaptchaID, params.CaptchaCode)
	} else {
		cooldown, appErr = h.verificationSvc.SendCode(c.Request.Context(), params.Email, params.Purpose, locale, params.CaptchaID, params.CaptchaCode)
	}
	if appErr != nil {
		if appErr.Code == apperrors.ErrCodeRateLimited {
			response.ErrorWithData(c, appErr, dtoresponse.SendCodeResult{Cooldown: cooldown})
			return
		}
		response.Error(c, appErr)
		return
	}

	response.SuccessOK(c, dtoresponse.SendCodeResult{
		Message:  "verification code sent",
		Cooldown: cooldown,
	})
}

// VerifyCode 处理 POST /api/auth/verify-code
// @Summary      Verify email code
// @Description  Validate a verification code sent to email before proceeding with the action
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.VerifyCodeParams  true  "Email, code, and purpose"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.VerifyCodeResult}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/verify-code [post]
func (h *AuthHandler) VerifyCode(c *gin.Context) {
	var params request.VerifyCodeParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters"))
		return
	}

	valid, appErr := h.verificationSvc.VerifyCode(c.Request.Context(), params.Email, params.Code, params.Purpose)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	if valid {
		response.SuccessOK(c, dtoresponse.VerifyCodeResult{
			Valid:   true,
			Message: "verification code is valid",
		})
	} else {
		response.Error(c, apperrors.ErrInvalidCode)
	}
}

// ForgotPassword 处理 POST /api/auth/forgot-password
// @Summary      Forgot password
// @Description  Initiate password reset flow by sending verification code to email
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.ForgotPasswordParams  true  "Email address"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var params request.ForgotPasswordParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters"))
		return
	}

	locale := contextkeys.GetLocale(c)
	if err := h.passwordSvc.ForgotPassword(c.Request.Context(), params.Email, locale, "", ""); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "if the email is registered, a verification code has been sent"})
}

// ResetPassword 处理 POST /api/auth/reset-password
// @Summary      Reset password
// @Description  Reset password using email verification code
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.ResetPasswordParams  true  "Reset password details"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var params request.ResetPasswordParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}

	if err := h.passwordSvc.ResetPassword(c.Request.Context(), params.Email, params.VerifyCode, params.NewPassword); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "password reset successful"})
}

// GetPasskeyRegisterOptions 处理 GET /api/auth/account/passkeys/register-options
// @Summary      Get passkey registration options
// @Description  Generate WebAuthn registration challenge for adding a passkey
// @Tags         Passkey
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse
// @Failure      401  {object}  response.ApiResponse
// @Router       /api/auth/account/passkeys/register-options [get]
func (h *AuthHandler) GetPasskeyRegisterOptions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	options, err := h.passkeyService.BeginRegistration(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回 WebAuthn 规范的 PublicKeyCredentialCreationOptions（内层对象），
	// 前端据此调用 navigator.credentials.create()。
	response.SuccessOK(c, gin.H{"publicKey": options.Response})
}

// VerifyPasskeyRegistration 处理 POST /api/auth/account/passkeys
// @Summary      Verify passkey registration
// @Description  Submit WebAuthn attestation to complete passkey registration
// @Tags         Passkey
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "WebAuthn credential"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/account/passkeys [post]
func (h *AuthHandler) VerifyPasskeyRegistration(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	// 请求体形如 { name?: string, verifyCode: string, credential: <WebAuthn attestation> }。
	var body struct {
		Name       string          `json:"name"`
		VerifyCode string          `json:"verifyCode"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Credential) == 0 {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid credential data"))
		return
	}

	if err := h.passkeyService.FinishRegistration(c.Request.Context(), userID, body.Name, body.VerifyCode, body.Credential); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "passkey registered successfully"})
}

// GetPasskeyAuthOptions 处理 GET /api/auth/passkey/auth-options
// @Summary      Get passkey authentication options
// @Description  Generate WebAuthn assertion challenge for passkey login
// @Tags         Passkey
// @Produce      json
// @Success      200  {object}  response.ApiResponse
// @Router       /api/auth/passkey/auth-options [get]
func (h *AuthHandler) GetPasskeyAuthOptions(c *gin.Context) {
	options, sessionID, err := h.passkeyService.BeginLogin(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	// 将 go-webauthn 的断言选项与会话 ID 一起返回；前端在 verify 时回传
	// 会话 ID，以便服务端定位对应的质询会话。
	response.SuccessOK(c, gin.H{
		"sessionId": sessionID,
		"publicKey": options.Response,
	})
}

// VerifyPasskeyAuth 处理 POST /api/auth/passkey/verify
// @Summary      Verify passkey authentication
// @Description  Submit WebAuthn assertion for passkey login
// @Tags         Passkey
// @Accept       json
// @Produce      json
// @Param        body  body      object  true  "WebAuthn assertion"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.LoginResult}
// @Failure      401   {object}  response.ApiResponse
// @Router       /api/auth/passkey/verify [post]
func (h *AuthHandler) VerifyPasskeyAuth(c *gin.Context) {
	// 请求体形如 { sessionId: string, credential: <WebAuthn assertion> }。
	var body struct {
		SessionID  string          `json:"sessionId"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.SessionID == "" || len(body.Credential) == 0 {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid assertion data"))
		return
	}

	result, appErr := h.passkeyService.FinishLogin(c.Request.Context(), body.SessionID, body.Credential)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	h.applyRoleInfo(result.User)
	response.SuccessOK(c, loginResultToDTO(result))
}

// ListPasskeys 处理 GET /api/auth/account/passkeys
// @Summary      List passkeys
// @Description  Return the authenticated user's registered passkeys
// @Tags         Passkey
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse
// @Router       /api/auth/account/passkeys [get]
func (h *AuthHandler) ListPasskeys(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	list, err := h.passkeyService.ListForUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, gin.H{"passkeys": list})
}

// DeletePasskey 处理 DELETE /api/auth/account/passkeys/:id
// @Summary      Delete passkey
// @Description  Remove one of the authenticated user's passkeys (requires email verification code)
// @Tags         Passkey
// @Produce      json
// @Security     BearerAuth
// @Param        id          path      string  true  "Passkey ID"
// @Param        verifyCode  query     string  true  "Email verification code"
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/auth/account/passkeys/{id} [delete]
func (h *AuthHandler) DeletePasskey(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	verifyCode := c.Query("verifyCode")
	if err := h.passkeyService.DeleteForUser(c.Request.Context(), userID, c.Param("id"), verifyCode); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "passkey deleted"})
}

// ChangePassword 处理 POST /api/auth/account/change-password
// @Summary      Change password
// @Description  Change the authenticated user's password after verifying the current one
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.ChangePasswordParams  true  "Password change details"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      401   {object}  response.ApiResponse
// @Router       /api/auth/account/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	var params request.ChangePasswordParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}

	if err := h.passwordSvc.ChangePassword(c.Request.Context(), userID, params.CurrentPassword, params.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "password changed"})
}

// CompleteSecurityUpdate 处理 POST /api/auth/account/security-update
// @Summary      完成安全更新
// @Description  完成强制安全更新（修改密码/邮箱），通常由管理员重置密码后首次登录触发
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.CompleteSecurityUpdateParams  true  "安全更新参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.UserInfo}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/account/security-update [post]
func (h *AuthHandler) CompleteSecurityUpdate(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	var params request.CompleteSecurityUpdateParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}
	user, appErr := h.authService.CompleteRequiredSecurityUpdate(
		c.Request.Context(),
		userID,
		params.NewPassword,
		params.NewEmail,
		params.VerifyCode,
	)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	info := userModelToResult(user)
	h.applyRoleInfo(info)
	response.SuccessOK(c, userInfoToDTO(info))
}

// SendAccountEmailCode 处理 POST /api/auth/account/email-code
// @Summary      Send account email verification code
// @Description  Send a verification code to the authenticated user's own email for sensitive operations
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.AccountEmailCodeParams  true  "Purpose"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.SendCodeResult}
// @Router       /api/auth/account/email-code [post]
func (h *AuthHandler) SendAccountEmailCode(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	var params request.AccountEmailCodeParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters"))
		return
	}

	locale := contextkeys.GetLocale(c)
	cooldown, appErr := h.authService.SendAccountEmailCode(c.Request.Context(), userID, params.Purpose, locale)
	if appErr != nil {
		if appErr.Code == apperrors.ErrCodeRateLimited {
			response.ErrorWithData(c, appErr, dtoresponse.SendCodeResult{Cooldown: cooldown})
			return
		}
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, dtoresponse.SendCodeResult{Message: "verification code sent", Cooldown: cooldown})
}

// SendChangeEmailCode 处理 POST /api/auth/account/email-change-code
// @Summary      Send change-email verification code
// @Description  Send a 5-minute verification code to the requested new email address
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.ChangeEmailCodeParams  true  "New email"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.SendCodeResult}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/account/email-change-code [post]
func (h *AuthHandler) SendChangeEmailCode(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	var params request.ChangeEmailCodeParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}

	locale := contextkeys.GetLocale(c)
	cooldown, appErr := h.authService.SendChangeEmailCode(c.Request.Context(), userID, params.NewEmail, locale)
	if appErr != nil {
		if appErr.Code == apperrors.ErrCodeRateLimited {
			response.ErrorWithData(c, appErr, dtoresponse.SendCodeResult{Cooldown: cooldown})
			return
		}
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, dtoresponse.SendCodeResult{Message: "verification code sent", Cooldown: cooldown})
}

// ChangeEmail 处理 PUT /api/auth/account/email
// @Summary      Change account email
// @Description  Change the authenticated user's email using a verification code sent to the new email
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.ChangeEmailParams  true  "New email and verification code"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.UserInfo}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/account/email [put]
func (h *AuthHandler) ChangeEmail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	var params request.ChangeEmailParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}

	if err := h.authService.ChangeEmail(c.Request.Context(), userID, params.NewEmail, params.VerifyCode); err != nil {
		response.Error(c, err)
		return
	}
	user, appErr := h.authService.GetUserInfo(c.Request.Context(), userID)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	info := userModelToResult(user)
	h.applyRoleInfo(info)
	response.SuccessOK(c, userInfoToDTO(info))
}

// OAuthAuthorize 处理 GET /api/auth/oauth/authorize
// @Summary      Get OAuth authorize URL
// @Description  Generate an OAuth authorize URL with a CSRF state token for the specified provider
// @Tags         Auth
// @Produce      json
// @Param        provider  query     string  true  "OAuth provider ID (github, google)"
// @Success      200       {object}  response.ApiResponse{data=dtoresponse.OAuthAuthorizeResult}
// @Failure      400       {object}  response.ApiResponse
// @Router       /api/auth/oauth/authorize [get]
func (h *AuthHandler) OAuthAuthorize(c *gin.Context) {
	provider := c.Query("provider")
	if provider == "" {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "provider is required"))
		return
	}

	state, err := h.oauthService.GenerateState(c.Request.Context())
	if err != nil {
		h.logger.Error("generate oauth state failed", zap.Error(err))
		response.Error(c, err)
		return
	}

	// 从配置仓储中获取该 provider 的授权 URL
	config, configErr := h.authService.GetSystemConfig(c.Request.Context())
	if configErr != nil {
		h.logger.Error("get system config failed", zap.Error(configErr))
		response.Error(c, configErr)
		return
	}

	var authURL string
	for _, p := range config.OAuthProviders {
		if p.ID == provider {
			authURL = p.AuthURL
			break
		}
	}

	if authURL == "" {
		response.Error(c, apperrors.New(apperrors.ErrCodeOperationDenied, "provider not configured: "+provider))
		return
	}

	// 追加 state 参数以进行 CSRF 防护
	sep := "&"
	if !containsQueryParam(authURL) {
		sep = "?"
	}
	authURL += sep + "state=" + state

	response.SuccessOK(c, dtoresponse.OAuthAuthorizeResult{
		AuthURL: authURL,
		State:   state,
	})
}

// OAuthCallback 处理 POST /api/auth/oauth/callback
// @Summary      OAuth callback
// @Description  Handle OAuth provider callback and exchange code for tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.OAuthCallbackParams  true  "OAuth callback data"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.LoginResult}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/oauth/callback [post]
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	var params request.OAuthCallbackParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid oauth parameters"))
		return
	}

	result, appErr := h.oauthService.HandleCallback(c.Request.Context(), params.Provider, params.Code, params.State)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	h.applyRoleInfo(result.User)
	response.SuccessOK(c, loginResultToDTO(result))
}

// OAuthBindExisting 处理 POST /api/auth/oauth/bind-existing
// @Summary      绑定已有账号
// @Description  将待绑定的第三方 OAuth 身份绑定到已有本地账号
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.OAuthBindExistingParams  true  "绑定参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.LoginResult}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/oauth/bind-existing [post]
func (h *AuthHandler) OAuthBindExisting(c *gin.Context) {
	var params request.OAuthBindExistingParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}
	result, appErr := h.oauthService.BindPendingToExisting(c.Request.Context(), params.PendingToken, params.Account, params.VerifyCode)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	h.applyRoleInfo(result.User)
	response.SuccessOK(c, loginResultToDTO(result))
}

// OAuthCreateAccount 处理 POST /api/auth/oauth/create-account
// @Summary      创建新账号并绑定
// @Description  为待绑定的第三方 OAuth 身份创建新的本地账号并完成登录
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.OAuthCreateAccountParams  true  "创建账号参数"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.LoginResult}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/oauth/create-account [post]
func (h *AuthHandler) OAuthCreateAccount(c *gin.Context) {
	var params request.OAuthCreateAccountParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid request parameters: "+err.Error()))
		return
	}
	result, appErr := h.oauthService.CreateAccountForPending(c.Request.Context(), params.PendingToken, params.Username, params.Nickname, params.Phone, params.Email, params.Password, params.VerifyCode)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	h.applyRoleInfo(result.User)
	response.SuccessOK(c, loginResultToDTO(result))
}

// ListOAuthBindings 处理 GET /api/auth/oauth/bindings
// @Summary      List OAuth bindings
// @Description  Return the authenticated user's linked third-party accounts
// @Tags         Account
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse
// @Router       /api/auth/oauth/bindings [get]
func (h *AuthHandler) ListOAuthBindings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	list, err := h.oauthService.ListBindings(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	out := make([]dtoresponse.OAuthBinding, len(list))
	for i, b := range list {
		out[i] = dtoresponse.OAuthBinding{
			Provider:  b.Provider,
			Email:     b.Email,
			CreatedAt: b.CreatedAt,
		}
	}
	response.SuccessOK(c, gin.H{"bindings": out})
}

// GetOAuthBindURL 处理 GET /api/auth/oauth/bind-url?provider=
// @Summary      Get OAuth bind URL
// @Description  Generate an OAuth authorize URL with a bind-intent state for linking a provider
// @Tags         Account
// @Produce      json
// @Security     BearerAuth
// @Param        provider  query     string  true  "OAuth provider ID"
// @Success      200       {object}  response.ApiResponse{data=dtoresponse.OAuthAuthorizeResult}
// @Router       /api/auth/oauth/bind-url [get]
func (h *AuthHandler) GetOAuthBindURL(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	provider := c.Query("provider")
	if provider == "" {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "provider is required"))
		return
	}

	state, err := h.oauthService.GenerateBindState(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	config, configErr := h.authService.GetSystemConfig(c.Request.Context())
	if configErr != nil {
		response.Error(c, configErr)
		return
	}
	var authURL string
	for _, p := range config.OAuthProviders {
		if p.ID == provider {
			authURL = p.AuthURL
			break
		}
	}
	if authURL == "" {
		response.Error(c, apperrors.New(apperrors.ErrCodeOperationDenied, "provider not configured: "+provider))
		return
	}

	sep := "&"
	if !containsQueryParam(authURL) {
		sep = "?"
	}
	authURL += sep + "state=" + state

	response.SuccessOK(c, dtoresponse.OAuthAuthorizeResult{AuthURL: authURL, State: state})
}

// BindOAuth 处理 POST /api/auth/oauth/bind
// @Summary      Bind OAuth provider
// @Description  Complete linking a third-party account to the authenticated user
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.OAuthCallbackParams  true  "OAuth callback data"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/auth/oauth/bind [post]
func (h *AuthHandler) BindOAuth(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	var params request.OAuthCallbackParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid oauth parameters"))
		return
	}
	if err := h.oauthService.HandleBindCallback(c.Request.Context(), userID, params.Provider, params.Code, params.State); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "provider linked"})
}

// UnbindOAuth 处理 DELETE /api/auth/oauth/bindings/:provider
// @Summary      Unbind OAuth provider
// @Description  Remove a linked third-party account from the authenticated user
// @Tags         Account
// @Produce      json
// @Security     BearerAuth
// @Param        provider  path      string  true  "OAuth provider ID"
// @Success      200       {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/auth/oauth/bindings/{provider} [delete]
func (h *AuthHandler) UnbindOAuth(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	if err := h.oauthService.UnbindProvider(c.Request.Context(), userID, c.Param("provider")); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "provider unlinked"})
}

// GetSystemConfig 处理 GET /api/auth/config
// @Summary      获取系统公开配置
// @Description  Returns public system config (registration status, available OAuth providers, etc.)
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.SystemConfig}
// @Router       /api/auth/config [get]
func (h *AuthHandler) GetSystemConfig(c *gin.Context) {
	// 安全策略是运行时动态配置，禁止浏览器、代理和 Service Worker 复用旧响应。
	c.Header("Cache-Control", "no-store")
	config, err := h.authService.GetSystemConfig(c.Request.Context())
	if err != nil {
		h.logger.Error("get system config failed", zap.Error(err))
		response.Error(c, err)
		return
	}

	oauthProviders := make([]dtoresponse.OAuthProvider, len(config.OAuthProviders))
	for i, p := range config.OAuthProviders {
		oauthProviders[i] = dtoresponse.OAuthProvider{
			ID:      p.ID,
			Name:    p.Name,
			Icon:    p.Icon,
			Color:   p.Color,
			AuthURL: p.AuthURL,
		}
	}

	response.SuccessOK(c, dtoresponse.SystemConfig{
		RegistrationEnabled:  config.RegistrationEnabled,
		CaptchaEnabled:       config.CaptchaEnabled,
		PasskeyEnabled:       config.PasskeyEnabled,
		PasswordResetEnabled: config.PasswordResetEnabled,
		TwoFactorRequired:    config.TwoFactorRequired,
		OAuthProviders:       oauthProviders,
		ICPBeian:             config.ICPBeian,
		PoliceBeian:          config.PoliceBeian,
	})
}

// ─── 辅助函数 ────────────────────────────────────────────────────────

// userInfoToDTO 将 UserInfoResult 转换为 API 响应的 DTO。
func userInfoToDTO(info *authsvc.UserInfoResult) dtoresponse.UserInfo {
	if info == nil {
		return dtoresponse.UserInfo{}
	}
	perms := info.Permissions
	if perms == nil {
		perms = []string{}
	}
	return dtoresponse.UserInfo{
		ID:                  info.ID,
		Username:            info.Username,
		Nickname:            info.Nickname,
		Phone:               info.Phone,
		Avatar:              info.Avatar,
		Email:               info.Email,
		Role:                info.Role,
		RoleName:            info.RoleName,
		RoleID:              info.RoleID,
		Permissions:         perms,
		TwoFactorEnabled:    info.TwoFactorEnabled,
		PreferredAuthMethod: info.PreferredAuthMethod,
		HasPasskey:          info.HasPasskey,
	}
}

// applyRoleInfo 通过权限列表器补充用户的角色名称和权限键列表。
func (h *AuthHandler) applyRoleInfo(info *authsvc.UserInfoResult) {
	if info == nil || h.permLister == nil {
		return
	}
	roleID := info.RoleID
	info.Permissions = h.permLister.PermissionKeysForRoleID(roleID)
	info.RoleName = h.permLister.RoleNameForID(roleID)
}

// loginResultToDTO 将登录服务结果转换为 API 响应的 DTO。
func loginResultToDTO(result *authsvc.LoginServiceResult) dtoresponse.LoginResult {
	if result == nil {
		return dtoresponse.LoginResult{}
	}
	out := dtoresponse.LoginResult{
		RequiresTwoFactor: result.RequiresTwoFactor,
		TwoFactorToken:    result.TwoFactorToken,
		SecurityActions: dtoresponse.SecurityActions{
			RequirePasswordChange: result.Actions.RequirePasswordChange,
			RequireEmailChange:    result.Actions.RequireEmailChange,
			RequireTwoFactorSetup: result.Actions.RequireTwoFactorSetup,
			Reason:                result.Actions.Reason,
		},
	}
	if result.Tokens != nil {
		out.AccessToken = result.Tokens.AccessToken
		out.RefreshToken = result.Tokens.RefreshToken
	}
	if result.User != nil {
		out.User = userInfoToDTO(result.User)
	}
	if result.PendingOAuthBinding != nil {
		out.PendingOAuthBinding = &dtoresponse.PendingOAuthBinding{
			Token:    result.PendingOAuthBinding.Token,
			Provider: result.PendingOAuthBinding.Provider,
			Email:    result.PendingOAuthBinding.Email,
			Username: result.PendingOAuthBinding.Username,
			Avatar:   result.PendingOAuthBinding.Avatar,
		}
	}
	return out
}

// userModelToResult 将 User 模型转换为 UserInfoResult。
func userModelToResult(u *model.User) *authsvc.UserInfoResult {
	return &authsvc.UserInfoResult{
		ID:                  strconv.FormatUint(u.ID, 10),
		Username:            u.Username,
		Nickname:            u.Nickname,
		Phone:               u.Phone,
		Avatar:              u.Avatar,
		Email:               u.Email,
		Role:                string(u.Role),
		RoleName:            u.RoleName,
		RoleID:              strconv.FormatUint(u.RoleID, 10),
		TwoFactorEnabled:    u.TwoFactorEnabled,
		PreferredAuthMethod: u.PreferredAuthMethod,
	}
}

// containsQueryParam 检查 URL 中是否已包含查询参数。
func containsQueryParam(rawURL string) bool {
	for _, c := range rawURL {
		if c == '?' {
			return true
		}
	}
	return false
}

// BeginTwoFactorSetup handles POST /api/auth/2fa/setup
// @Summary      Begin two-factor setup
// @Description  Generate a TOTP secret and QR code for binding an authenticator app
// @Tags         Account
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.TwoFactorSetupResult}
// @Router       /api/auth/2fa/setup [post]
func (h *AuthHandler) BeginTwoFactorSetup(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	result, appErr := h.twoFactorService.BeginSetup(c.Request.Context(), userID)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, dtoresponse.TwoFactorSetupResult{
		Secret:     result.Secret,
		OtpauthURL: result.OtpauthURL,
		QRCode:     result.QRCode,
	})
}

// ConfirmTwoFactorSetup handles POST /api/auth/2fa/enable
// @Summary      Confirm two-factor setup
// @Description  Verify a TOTP code and enable two-factor authentication
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.EnableTwoFactorParams  true  "TOTP code"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/auth/2fa/enable [post]
func (h *AuthHandler) ConfirmTwoFactorSetup(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	var params request.EnableTwoFactorParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	result, appErr := h.twoFactorService.ConfirmSetup(c.Request.Context(), userID, params.Code)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, dtoresponse.TwoFactorEnableResult{RecoveryCodes: result.RecoveryCodes})
}

// DisableTwoFactor handles POST /api/auth/2fa/disable
// @Summary      Disable two-factor authentication
// @Description  Disable 2FA after verifying the current password
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.DisableTwoFactorParams  true  "Current password"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/auth/2fa/disable [post]
func (h *AuthHandler) DisableTwoFactor(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	var params request.DisableTwoFactorParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	if appErr := h.twoFactorService.Disable(c.Request.Context(), userID, params.VerifyCode); appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "two-factor authentication disabled"})
}

// VerifyTwoFactorLogin handles POST /api/auth/login/2fa
// @Summary      Verify two-factor login
// @Description  Exchange a two-factor challenge token and TOTP code for access tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.VerifyTwoFactorParams  true  "Two-factor challenge and code"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.LoginResult}
// @Failure      401   {object}  response.ApiResponse
// @Router       /api/auth/login/2fa [post]
func (h *AuthHandler) VerifyTwoFactorLogin(c *gin.Context) {
	var params request.VerifyTwoFactorParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	result, appErr := h.twoFactorService.VerifyLogin(c.Request.Context(), params.TwoFactorToken, params.Code)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	h.applyRoleInfo(result.User)
	response.SuccessOK(c, loginResultToDTO(result))
}

// VerifyRecoveryLogin handles POST /api/auth/login/recovery
// @Summary      Verify two-factor login with recovery code
// @Description  Exchange a two-factor challenge token and a one-time recovery code for access tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.RecoveryLoginParams  true  "Two-factor challenge and recovery code"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.LoginResult}
// @Failure      401   {object}  response.ApiResponse
// @Router       /api/auth/login/recovery [post]
func (h *AuthHandler) VerifyRecoveryLogin(c *gin.Context) {
	var params request.RecoveryLoginParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	result, appErr := h.twoFactorService.VerifyLoginWithRecovery(c.Request.Context(), params.TwoFactorToken, params.RecoveryCode)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}
	h.applyRoleInfo(result.User)
	response.SuccessOK(c, loginResultToDTO(result))
}

// SetPreferredAuthMethod handles PUT /api/auth/account/preferred-auth-method
// @Summary      Set preferred two-factor method
// @Description  Set the preferred verification method (totp or passkey) for the current user
// @Tags         Account
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.PreferredAuthMethodParams  true  "Preferred method"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Router       /api/auth/account/preferred-auth-method [put]
func (h *AuthHandler) SetPreferredAuthMethod(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}
	var params request.PreferredAuthMethodParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid parameters: "+err.Error()))
		return
	}
	// 选择通行密钥作为首选方式时，要求用户已注册至少一个通行密钥。
	if params.Method == "passkey" {
		if hasPasskey, err := h.passkeyService.HasPasskey(c.Request.Context(), userID); err != nil || !hasPasskey {
			response.Error(c, apperrors.New(apperrors.ErrCodeOperationDenied, "no registered passkey"))
			return
		}
	}
	if appErr := h.twoFactorService.SetPreferredAuthMethod(c.Request.Context(), userID, params.Method); appErr != nil {
		response.Error(c, appErr)
		return
	}
	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "preferred auth method updated"})
}
