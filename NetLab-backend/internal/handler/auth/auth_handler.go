package auth

import (
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
	authService    *authsvc.AuthService
	passkeyService *authsvc.PasskeyService
	tokenService   *authsvc.TokenService
	oauthService   *authsvc.OAuthService
	captchaMgr     *captcha.Manager
	logger         *zap.Logger
}

// NewAuthHandler 创建一个新的 AuthHandler。
func NewAuthHandler(
	authService *authsvc.AuthService,
	passkeyService *authsvc.PasskeyService,
	tokenService *authsvc.TokenService,
	oauthService *authsvc.OAuthService,
	captchaMgr *captcha.Manager,
	logger *zap.Logger,
) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		passkeyService: passkeyService,
		tokenService:   tokenService,
		oauthService:   oauthService,
		captchaMgr:     captchaMgr,
		logger:         logger,
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
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid request parameters: "+err.Error()))
		return
	}

	// 当系统配置中启用验证码时强制校验
	config, _ := h.authService.GetSystemConfig(c.Request.Context())
	if config != nil && config.CaptchaEnabled {
		if params.CaptchaID == "" || params.CaptchaCode == "" {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "captcha is required"))
			return
		}
		ok, err := h.captchaMgr.Verify(params.CaptchaID, params.CaptchaCode)
		if err != nil || !ok {
			response.Error(c, apperrors.ErrInvalidCode)
			return
		}
	}

	result, appErr := h.authService.Login(c.Request.Context(), params.Username, params.Password, params.CaptchaID, params.CaptchaCode)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.SuccessOK(c, dtoresponse.LoginResult{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		User:         userInfoToDTO(result.User),
		SigningKey:   result.Tokens.SigningKey,
	})
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
		SigningKey:   result.SigningKey,
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

	response.SuccessOK(c, userInfoToDTO(userModelToResult(user)))
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
	jti := middleware.GetJTI(c)
	tokenExp := middleware.GetTokenExp(c)

	if err := h.authService.Logout(c.Request.Context(), userID, jti, tokenExp); err != nil {
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
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid request parameters: "+err.Error()))
		return
	}

	// 检查是否需要验证码
	config, _ := h.authService.GetSystemConfig(c.Request.Context())
	if config != nil && config.CaptchaEnabled {
		if params.CaptchaID == "" || params.CaptchaCode == "" {
			response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "captcha is required"))
			return
		}
		ok, err := h.captchaMgr.Verify(params.CaptchaID, params.CaptchaCode)
		if err != nil || !ok {
			response.Error(c, apperrors.ErrInvalidCode)
			return
		}
	}

	if err := h.authService.Register(c.Request.Context(), params.Username, params.Email, params.Password, params.VerifyCode); err != nil {
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
	cooldown, appErr := h.authService.SendVerificationCode(c.Request.Context(), params.Email, params.Purpose, locale)
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

	valid, appErr := h.authService.VerifyCode(c.Request.Context(), params.Email, params.Code, params.Purpose)
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
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid request parameters"))
		return
	}

	locale := contextkeys.GetLocale(c)
	if err := h.authService.ForgotPassword(c.Request.Context(), params.Email, locale); err != nil {
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
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid request parameters: "+err.Error()))
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), params.Email, params.VerifyCode, params.NewPassword); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessOK(c, dtoresponse.MessageResponse{Message: "password reset successful"})
}

// GetPasskeyRegisterOptions 处理 GET /api/auth/passkey/register-options
// @Summary      Get passkey registration options
// @Description  Generate WebAuthn registration challenge for adding a passkey
// @Tags         Passkey
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.PasskeyRegisterOptions}
// @Failure      401  {object}  response.ApiResponse
// @Router       /api/auth/passkey/register-options [get]
func (h *AuthHandler) GetPasskeyRegisterOptions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	options, err := h.passkeyService.GetRegisterOptions(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessOK(c, dtoresponse.PasskeyRegisterOptions{
		Challenge: options.Challenge,
		RP: dtoresponse.PasskeyRP{
			Name: options.RP.Name,
			ID:   options.RP.ID,
		},
		User: dtoresponse.PasskeyUser{
			ID:          options.User.ID,
			Name:        options.User.Name,
			DisplayName: options.User.DisplayName,
		},
		PubKeyCredParams: toPasskeyCredParams(options.PubKeyCredParams),
		Timeout:          options.Timeout,
		Attestation:      options.Attestation,
		AuthenticatorSelection: dtoresponse.AuthenticatorSelection{
			AuthenticatorAttachment: options.AuthenticatorSelection.AuthenticatorAttachment,
			ResidentKey:             options.AuthenticatorSelection.ResidentKey,
			UserVerification:        options.AuthenticatorSelection.UserVerification,
		},
	})
}

// VerifyPasskeyRegistration 处理 POST /api/auth/passkey/register
// @Summary      Verify passkey registration
// @Description  Submit WebAuthn attestation to complete passkey registration
// @Tags         Passkey
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "WebAuthn credential"
// @Success      200   {object}  response.ApiResponse{data=dtoresponse.MessageResponse}
// @Failure      400   {object}  response.ApiResponse
// @Router       /api/auth/passkey/register [post]
func (h *AuthHandler) VerifyPasskeyRegistration(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		response.Error(c, apperrors.ErrTokenExpired)
		return
	}

	var credentialData map[string]interface{}
	if err := c.ShouldBindJSON(&credentialData); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid credential data"))
		return
	}

	if err := h.passkeyService.VerifyRegistration(c.Request.Context(), userID, credentialData); err != nil {
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
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.PasskeyAuthOptions}
// @Router       /api/auth/passkey/auth-options [get]
func (h *AuthHandler) GetPasskeyAuthOptions(c *gin.Context) {
	options, err := h.passkeyService.GetAuthOptions(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	allowCredentials := make([]dtoresponse.AllowCredential, len(options.AllowCredentials))
	for i, ac := range options.AllowCredentials {
		allowCredentials[i] = dtoresponse.AllowCredential{
			ID:         ac.ID,
			Type:       ac.Type,
			Transports: ac.Transports,
		}
	}

	response.SuccessOK(c, dtoresponse.PasskeyAuthOptions{
		Challenge:        options.Challenge,
		RPID:             options.RPID,
		Timeout:          options.Timeout,
		UserVerification: options.UserVerification,
		AllowCredentials: allowCredentials,
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
	var assertionData map[string]interface{}
	if err := c.ShouldBindJSON(&assertionData); err != nil {
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid assertion data"))
		return
	}

	result, appErr := h.passkeyService.VerifyAuth(c.Request.Context(), assertionData)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.SuccessOK(c, dtoresponse.LoginResult{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		User:         userInfoToDTO(result.User),
		SigningKey:   result.Tokens.SigningKey,
	})
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
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "provider is required"))
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
		response.Error(c, apperrors.New(apperrors.ErrCodeInvalidCredentials, "invalid oauth parameters"))
		return
	}

	result, appErr := h.oauthService.HandleCallback(c.Request.Context(), params.Provider, params.Code, params.State)
	if appErr != nil {
		response.Error(c, appErr)
		return
	}

	response.SuccessOK(c, dtoresponse.LoginResult{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		User:         userInfoToDTO(result.User),
		SigningKey:   result.Tokens.SigningKey,
	})
}

// GetSystemConfig 处理 GET /api/auth/config
// @Summary      Get system configuration
// @Description  Returns public system config (registration status, available OAuth providers, etc.)
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  response.ApiResponse{data=dtoresponse.SystemConfig}
// @Router       /api/auth/config [get]
func (h *AuthHandler) GetSystemConfig(c *gin.Context) {
	config, err := h.authService.GetSystemConfig(c.Request.Context())
	if err != nil {
		h.logger.Error("get system config failed", zap.Error(err))
		response.SuccessOK(c, dtoresponse.SystemConfig{
			RegistrationEnabled: true,
			PasskeyEnabled:      true,
			OAuthProviders:      []dtoresponse.OAuthProvider{},
		})
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
		RegistrationEnabled: config.RegistrationEnabled,
		CaptchaEnabled:      config.CaptchaEnabled,
		PasskeyEnabled:      config.PasskeyEnabled,
		OAuthProviders:      oauthProviders,
		ICPBeian:            config.ICPBeian,
		PoliceBeian:         config.PoliceBeian,
	})
}

// ─── 辅助函数 ────────────────────────────────────────────────────────

func userInfoToDTO(info *authsvc.UserInfoResult) dtoresponse.UserInfo {
	if info == nil {
		return dtoresponse.UserInfo{}
	}
	return dtoresponse.UserInfo{
		ID:       info.ID,
		Username: info.Username,
		Avatar:   info.Avatar,
		Email:    info.Email,
		Roles:    info.Roles,
	}
}

func userModelToResult(u *model.User) *authsvc.UserInfoResult {
	return &authsvc.UserInfoResult{
		ID:       u.ID.String(),
		Username: u.Username,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Roles:    u.Roles,
	}
}

func containsQueryParam(rawURL string) bool {
	for _, c := range rawURL {
		if c == '?' {
			return true
		}
	}
	return false
}

func toPasskeyCredParams(params []authsvc.PubKeyCredParam) []dtoresponse.PasskeyCredParam {
	result := make([]dtoresponse.PasskeyCredParam, len(params))
	for i, p := range params {
		result[i] = dtoresponse.PasskeyCredParam{
			Type: p.Type,
			Alg:  p.Alg,
		}
	}
	return result
}
