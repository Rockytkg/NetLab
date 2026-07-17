import request from './request'
import type {
  LoginParams,
  LoginResult,
  RefreshTokenResult,
  UserInfo,
  CaptchaResult,
  RegisterParams,
  RegisterResult,
  VerifyCodeParams,
  VerifyCodeResult,
  ForgotPasswordParams,
  ResetPasswordParams,
  SendCodeParams,
  SendCodeResult,
  PasskeyRegisterOptions,
  PasskeyAuthOptions,
  OAuthCallbackParams,
  OAuthBindExistingParams,
  OAuthCreateAccountParams,
  OAuthBinding,
  OAuthAuthorizeResult,
  ChangePasswordParams,
  CompleteSecurityUpdateParams,
  ChangeEmailParams,
  AccountCodePurpose,
  SystemConfig,
  TwoFactorSetupResult,
  TwoFactorEnableResult,
  VerifyTwoFactorParams,
  RecoveryLoginParams,
} from '@/types/auth'
import type { PasskeyInfo } from '@/types/settings'

// 公开（未认证）接口的 Axios 配置。
// 避免不必要的 JWT 注入和主动刷新尝试。
const PUBLIC_CONFIG = {
  requireAuth: false,
  skipAuthRefresh: true,
} as const

// 携带敏感字段（如密码）的公开接口配置。
// 请求体以明文发送（由 HTTPS 保护），并使用预共享 HMAC 密钥签名，
// 以便后端 Signature 中间件校验完整性并拒绝重放攻击。
// 请求体不加密——详见 services/authSecurity.ts。
const SIGNED_PUBLIC_CONFIG = {
  requireAuth: false,
  skipAuthRefresh: true,
  authSign: true,
} as const

export const authApi = {
  /** ── 登录 ── */

  login(params: LoginParams): Promise<LoginResult> {
    return request.post('/auth/login', params, SIGNED_PUBLIC_CONFIG)
  },

  /** ── Token ── */

  refreshToken(refreshToken: string): Promise<RefreshTokenResult> {
    return request.post('/auth/refresh', { refreshToken }, PUBLIC_CONFIG)
  },

  getUserInfo(): Promise<UserInfo> {
    return request.get('/auth/userinfo')
  },

  logout(): Promise<void> {
    return request.post('/auth/logout')
  },

  /** ── 图形验证码 ── */

  getCaptcha(): Promise<CaptchaResult> {
    return request.get('/auth/captcha', PUBLIC_CONFIG)
  },

  /** ── 注册 ── */

  register(params: RegisterParams): Promise<RegisterResult> {
    return request.post('/auth/register', params, SIGNED_PUBLIC_CONFIG)
  },

  /** ── 验证码 ── */

  sendCode(params: SendCodeParams): Promise<SendCodeResult> {
    return request.post('/auth/send-code', params, PUBLIC_CONFIG)
  },

  /** ── 验证码校验 ── */

  verifyCode(params: VerifyCodeParams): Promise<VerifyCodeResult> {
    return request.post('/auth/verify-code', params, PUBLIC_CONFIG)
  },

  /** ── 忘记密码 ── */

  forgotPassword(params: ForgotPasswordParams): Promise<{ message: string }> {
    return request.post('/auth/forgot-password', params, PUBLIC_CONFIG)
  },

  resetPassword(params: ResetPasswordParams): Promise<{ message: string }> {
    return request.post('/auth/reset-password', params, SIGNED_PUBLIC_CONFIG)
  },

  /** ── WebAuthn / Passkey ── */

  /** 获取注册 Passkey 的 challenge */
  getPasskeyRegisterOptions(): Promise<PasskeyRegisterOptions> {
    return request.get('/auth/account/passkeys/register-options')
  },

  /** 验证并注册新的 Passkey（credential 为原始 WebAuthn attestation） */
  verifyPasskeyRegistration(payload: {
    name: string
    verifyCode: string
    credential: Record<string, unknown>
  }): Promise<{ message: string }> {
    return request.post('/auth/account/passkeys', payload)
  },

  /** 获取 Passkey 登录的 challenge 及会话 ID */
  getPasskeyAuthOptions(): Promise<PasskeyAuthOptions> {
    return request.get('/auth/passkey/auth-options', PUBLIC_CONFIG)
  },

  /** 验证 Passkey 登录断言 */
  verifyPasskeyAuth(payload: {
    sessionId: string
    credential: Record<string, unknown>
  }): Promise<LoginResult> {
    return request.post('/auth/passkey/verify', payload, PUBLIC_CONFIG)
  },

  /** 列出当前用户已注册的 passkey */
  listPasskeys(): Promise<{ passkeys: PasskeyInfo[] }> {
    return request.get('/auth/account/passkeys')
  },

  /** 删除一个 passkey（需邮箱验证码） */
  deletePasskey(id: string, verifyCode: string): Promise<{ message: string }> {
    return request.delete(`/auth/account/passkeys/${id}`, {
      params: { verifyCode },
    })
  },

  /** ── 账户自助 ── */

  /** 向当前用户邮箱发送验证码（敏感操作二次校验） */
  sendAccountEmailCode(purpose: AccountCodePurpose): Promise<SendCodeResult> {
    return request.post('/auth/account/email-code', { purpose })
  },

  /** 修改密码（校验当前密码） */
  changePassword(params: ChangePasswordParams): Promise<{ message: string }> {
    return request.post('/auth/account/change-password', params)
  },

  completeSecurityUpdate(params: CompleteSecurityUpdateParams): Promise<UserInfo> {
    return request.post('/auth/account/security-update', params)
  },

  /** 向新邮箱发送 5 分钟有效的验证码 */
  sendChangeEmailCode(
    newEmail: string,
    captcha?: Pick<SendCodeParams, 'captchaId' | 'captchaCode'>,
  ): Promise<SendCodeResult> {
    return request.post('/auth/account/email-change-code', { newEmail, ...captcha })
  },

  /** 修改当前账号邮箱 */
  changeEmail(params: ChangeEmailParams): Promise<UserInfo> {
    return request.put('/auth/account/email', params)
  },

  /** ── OAuth 绑定管理 ── */

  /** 列出当前用户的第三方账号绑定 */
  listOAuthBindings(): Promise<{ bindings: OAuthBinding[] }> {
    return request.get('/auth/oauth/bindings')
  },

  /** 获取绑定授权 URL（bind 意图） */
  getOAuthBindURL(provider: string): Promise<OAuthAuthorizeResult> {
    return request.get('/auth/oauth/bind-url', { params: { provider } })
  },

  /** 完成第三方账号绑定 */
  bindOAuth(params: OAuthCallbackParams): Promise<{ message: string }> {
    return request.post('/auth/oauth/bind', params)
  },

  /** 解绑第三方账号 */
  unbindOAuth(provider: string): Promise<{ message: string }> {
    return request.delete(`/auth/oauth/bindings/${provider}`)
  },

  /** ── OAuth / 第三方登录 ── */

  /** 用 OAuth 回调参数换取 Token */
  oauthCallback(params: OAuthCallbackParams): Promise<LoginResult> {
    return request.post('/auth/oauth/callback', params, PUBLIC_CONFIG)
  },

  oauthBindExisting(params: OAuthBindExistingParams): Promise<LoginResult> {
    return request.post('/auth/oauth/bind-existing', params, PUBLIC_CONFIG)
  },

  oauthCreateAccount(params: OAuthCreateAccountParams): Promise<LoginResult> {
    return request.post('/auth/oauth/create-account', params, SIGNED_PUBLIC_CONFIG)
  },

  /** ── 系统配置 ── */

  getSystemConfig(): Promise<SystemConfig> {
    return request.get('/auth/config', PUBLIC_CONFIG)
  },

  /** 两步验证 (TOTP) */

  /** 生成密钥与二维码，开始绑定流程 */
  beginTwoFactorSetup(): Promise<TwoFactorSetupResult> {
    return request.post('/auth/2fa/setup', {})
  },

  /** 校验动态码并启用两步验证，返回一次性恢复码（仅此刻返回） */
  confirmTwoFactorSetup(code: string): Promise<TwoFactorEnableResult> {
    return request.post('/auth/2fa/enable', { code })
  },

  /** 校验绑定邮箱的验证码并关闭两步验证 */
  disableTwoFactor(verifyCode: string): Promise<{ message: string }> {
    return request.post('/auth/2fa/disable', { verifyCode })
  },

  /** 用挑战令牌 + 动态码换取访问令牌 */
  verifyTwoFactorLogin(params: VerifyTwoFactorParams): Promise<LoginResult> {
    return request.post('/auth/login/2fa', params, PUBLIC_CONFIG)
  },

  /** 用挑战令牌 + 一次性恢复码换取访问令牌 */
  verifyRecoveryLogin(params: RecoveryLoginParams): Promise<LoginResult> {
    return request.post('/auth/login/recovery', params, PUBLIC_CONFIG)
  },

  /** 设置两步验证首选方式（totp / passkey） */
  setPreferredAuthMethod(method: 'totp' | 'passkey'): Promise<{ message: string }> {
    return request.put('/auth/account/preferred-auth-method', { method })
  },
}

