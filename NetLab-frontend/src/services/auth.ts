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
  SystemConfig,
} from '@/types/auth'

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
    return request.get('/auth/passkey/register-options')
  },

  /** 验证并注册新的 Passkey */
  verifyPasskeyRegistration(credential: Record<string, unknown>): Promise<{ message: string }> {
    return request.post('/auth/passkey/register', credential)
  },

  /** 获取 Passkey 登录的 challenge */
  getPasskeyAuthOptions(): Promise<PasskeyAuthOptions> {
    return request.get('/auth/passkey/auth-options', PUBLIC_CONFIG)
  },

  /** 验证 Passkey 登录断言 */
  verifyPasskeyAuth(assertion: Record<string, unknown>): Promise<LoginResult> {
    return request.post('/auth/passkey/verify', assertion, PUBLIC_CONFIG)
  },

  /** ── OAuth / 第三方登录 ── */

  /** 用 OAuth 回调参数换取 Token */
  oauthCallback(params: OAuthCallbackParams): Promise<LoginResult> {
    return request.post('/auth/oauth/callback', params, PUBLIC_CONFIG)
  },

  /** ── 系统配置 ── */

  getSystemConfig(): Promise<SystemConfig> {
    return request.get('/auth/config', PUBLIC_CONFIG)
  },
}
