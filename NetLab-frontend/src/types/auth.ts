/** 用户角色 */
export type UserRole = 'admin' | 'editor' | 'viewer'

/** 用户信息 */
export interface UserInfo {
  id: string
  username: string
  avatar?: string
  email?: string
  roles: UserRole[]
}

/** ── 登录 ── */

export interface LoginParams {
  username: string
  password: string
  captchaId?: string
  captchaCode?: string
}

export interface LoginResult {
  accessToken: string
  refreshToken: string
  user: UserInfo
  /** 会话签名密钥（hex，256 位）—— 用于 HMAC 请求签名 */
  signingKey?: string
}

/** ── Token ── */

export interface RefreshTokenResult {
  accessToken: string
  refreshToken: string
  /** 轮换后的签名密钥（若服务端轮换了密钥） */
  signingKey?: string
}

/** ── 图形验证码 ── */

export interface CaptchaResult {
  captchaId: string
  captchaImage: string // data:image/png;base64 算术验证码
}

/** ── 注册 ── */

export interface RegisterParams {
  username: string
  email: string
  password: string
  confirmPassword: string
  verifyCode: string
}

export interface RegisterResult {
  message: string
}

/** ── 验证码校验 ── */

export interface VerifyCodeParams {
  email: string
  code: string
  purpose: 'register' | 'reset-password'
}

export interface VerifyCodeResult {
  valid: boolean
  message: string
}

/** ── 忘记密码 ── */

export interface ForgotPasswordParams {
  email: string
}

export interface ResetPasswordParams {
  email: string
  verifyCode: string
  newPassword: string
  confirmPassword: string
}

/** ── 发送验证码 ── */

export interface SendCodeParams {
  email: string
  purpose: 'register' | 'reset-password'
}

export interface SendCodeResult {
  message: string
  cooldown: number
}

/** ── WebAuthn / Passkey ── */

/** 服务端下发的 passkey 注册（凭据创建）challenge */
export interface PasskeyRegisterOptions {
  challenge: string       // base64url
  rp: {
    name: string
    id: string
  }
  user: {
    id: string            // base64url
    name: string
    displayName: string
  }
  pubKeyCredParams: Array<{ type: string; alg: number }>
  timeout: number
  attestation: string
  authenticatorSelection: {
    authenticatorAttachment?: 'platform' | 'cross-platform'
    residentKey: string
    userVerification: string
  }
}

/** 服务端下发的 passkey 认证（凭据断言）challenge */
export interface PasskeyAuthOptions {
  challenge: string       // base64url
  rpId: string
  timeout: number
  userVerification: string
  allowCredentials?: Array<{
    id: string            // base64url
    type: string
    transports?: string[]
  }>
}

/** ── OAuth / 第三方登录 ── */

/** OAuth 提供方元数据 —— 由 /auth/config 返回 */
export interface OAuthProvider {
  /** 唯一提供方 ID: 'github' | 'google' | 'qq' | 'wechat' | ... */
  id: string
  /** 显示名称 */
  name: string
  /** 图标的简单 SVG path 字符串或内置图标键 */
  icon: string
  /** 登录按钮的品牌色 */
  color: string
  /** 启动 OAuth 重定向流程的后端接口 URL */
  authUrl: string
}

export interface OAuthCallbackParams {
  provider: string
  code: string
  state: string
}

/** ── 系统配置 ── */

export interface SystemConfig {
  /** 是否开放注册 */
  registrationEnabled: boolean
  /** 登录是否需要图形验证码 */
  captchaEnabled: boolean
  /** 是否启用 Passkey (WebAuthn) 登录 */
  passkeyEnabled: boolean
  /** 启用的第三方 OAuth 登录方式 */
  oauthProviders: OAuthProvider[]
  /** ICP 备案号（如 "京ICP备12345678号-1"），为空则不显示。跳转链接由前端固定模板拼接 */
  icpBeian?: string
  /** 公安备案号，为空则不显示。跳转链接由前端固定模板拼接 */
  policeBeian?: string
}
