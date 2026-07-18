/** 用户角色 */
export type UserRole = 'super_admin' | 'admin' | 'editor' | 'viewer'

/** 用户信息 */
export interface UserInfo {
  id: string
  username: string
  nickname: string
  phone: string
  avatar?: string
  email?: string
  role: string
  roleIdentifier: UserRole
  roleId?: string
  permissions: string[]
  twoFactorEnabled?: boolean
  /** 首选验证方式：'totp'（身份验证器应用）或 'passkey'（通行密钥） */
  preferredAuthMethod?: string
  /** 当前用户是否已注册至少一个通行密钥 */
  hasPasskey?: boolean
}

export interface SecurityActions {
  requirePasswordChange: boolean
  requireEmailChange: boolean
  requireTwoFactorSetup: boolean
  reason?: 'default_admin_bootstrap' | 'first_login' | 'password_reset' | 'password_expired' | string
}

/** ── 登录 ── */

export interface LoginParams {
  username: string
  password: string
  captchaId?: string
  captchaCode?: string
}

export interface LoginResult {
  accessToken?: string
  refreshToken?: string
  user?: UserInfo
  requiresTwoFactor?: boolean
  twoFactorToken?: string
  securityActions: SecurityActions
  pendingOAuthBinding?: PendingOAuthBinding
}

/** ── Token ── */

export interface RefreshTokenResult {
  accessToken: string
  refreshToken: string
}

/** ── 图形验证码 ── */

export interface CaptchaResult {
  captchaId: string
  captchaImage: string // data:image/png;base64 算术验证码
}

/** ── 注册 ── */

export interface RegisterParams {
  username: string
  nickname: string
  phone: string
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
  purpose: 'register' | 'reset-password' | 'change-email'
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
  purpose: 'register' | 'reset-password' | 'change-email'
  captchaId?: string
  captchaCode?: string
}

export interface SendCodeResult {
  message: string
  cooldown: number
}

/** ── WebAuthn / Passkey ── */

/** 服务端下发的 passkey 注册（凭据创建）challenge。
 *  由于该负载跳过 snake↔camel 转换，字段保持后端原始命名。
 *  publicKey 为 WebAuthn 规范的 PublicKeyCredentialCreationOptions
 *  （challenge / user.id 为 base64url 字符串）。 */
export interface PasskeyRegisterOptions {
  publicKey: PublicKeyCredentialCreationOptionsJSON
}

/** 服务端下发的 passkey 认证（凭据断言）challenge 及会话 ID。
 *  该负载跳过 snake↔camel 转换，字段保持后端原始命名。 */
export interface PasskeyAuthOptions {
  sessionId: string
  publicKey: PublicKeyCredentialRequestOptionsJSON
}

/** WebAuthn 创建选项的 JSON 形态（base64url 编码的二进制字段）。 */
export interface PublicKeyCredentialCreationOptionsJSON {
  challenge: string
  rp: { name: string; id: string }
  user: { id: string; name: string; displayName: string }
  pubKeyCredParams: Array<{ type: string; alg: number }>
  timeout?: number
  attestation?: string
  excludeCredentials?: Array<{ id: string; type: string; transports?: string[] }>
  authenticatorSelection?: {
    authenticatorAttachment?: 'platform' | 'cross-platform'
    residentKey?: string
    requireResidentKey?: boolean
    userVerification?: string
  }
}

/** WebAuthn 断言选项的 JSON 形态（base64url 编码的二进制字段）。 */
export interface PublicKeyCredentialRequestOptionsJSON {
  challenge: string
  rpId?: string
  timeout?: number
  userVerification?: string
  allowCredentials?: Array<{ id: string; type: string; transports?: string[] }>
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

export interface PendingOAuthBinding {
  token: string
  provider: string
  email?: string
  username?: string
  avatar?: string
}

export interface OAuthBindExistingParams {
  pendingToken: string
  account: string
  verifyCode: string
}

export interface OAuthCreateAccountParams {
  pendingToken: string
  username: string
  email: string
  password: string
  confirmPassword: string
  verifyCode: string
}

/** 已登录用户的第三方账号绑定关系 */
export interface OAuthBinding {
  provider: string
  email?: string
  createdAt: string
}

/** OAuth 绑定授权 URL 响应 */
export interface OAuthAuthorizeResult {
  authUrl: string
  state: string
}

/** ── 账户自助 ── */

export interface ChangePasswordParams {
  currentPassword: string
  newPassword: string
  confirmPassword: string
}

export interface CompleteSecurityUpdateParams {
  newPassword: string
  confirmPassword: string
  newEmail?: string
  verifyCode?: string
}

export interface ChangeEmailCodeParams {
  newEmail: string
}

export interface ChangeEmailParams {
  newEmail: string
  verifyCode: string
}

/** 账户内取验证码用途 */
export type AccountCodePurpose = 'passkey' | 'disable-2fa'

/** ── 系统配置 ── */

export interface SystemConfig {
  /** 是否开放注册 */
  registrationEnabled: boolean
  /** 登录是否需要图形验证码 */
  captchaEnabled: boolean
  /** 是否启用 Passkey (WebAuthn) 登录 */
  passkeyEnabled: boolean
  /** 是否启用密码重置（忘记密码）功能 */
  passwordResetEnabled?: boolean
  /** 系统是否强制开启两步验证 */
  twoFactorRequired?: boolean
  /** 启用的第三方 OAuth 登录方式 */
  oauthProviders: OAuthProvider[]
  /** ICP 备案号（如 "京ICP备12345678号-1"），为空则不显示。跳转链接由前端固定模板拼接 */
  icpBeian?: string
  /** 公安备案号，为空则不显示。跳转链接由前端固定模板拼接 */
  policeBeian?: string
}

/** 两步验证 (TOTP) */

/** POST /auth/2fa/setup 的响应：密钥、otpauth URI 与二维码 data URI */
export interface TwoFactorSetupResult {
  secret: string
  otpauthUrl: string
  qrCode: string
}

/** POST /auth/login/2fa 的请求体 */
export interface VerifyTwoFactorParams {
  twoFactorToken: string
  code: string
}

/** POST /auth/2fa/enable 的响应：一次性恢复码（仅此刻返回一次） */
export interface TwoFactorEnableResult {
  recoveryCodes: string[]
}

/** POST /auth/login/recovery 的请求体 */
export interface RecoveryLoginParams {
  twoFactorToken: string
  recoveryCode: string
}


