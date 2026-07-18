import request from './request'
import type {
  AccountCodePurpose,
  ChangeEmailParams,
  ChangePasswordParams,
  OAuthAuthorizeResult,
  OAuthBinding,
  OAuthCallbackParams,
  PasskeyRegisterOptions,
  SendCodeResult,
  SystemConfig,
  TwoFactorEnableResult,
  TwoFactorSetupResult,
  UpdateProfileParams,
  UserInfo,
} from '@/types/auth'
import type { PasskeyInfo } from '@/types/settings'

/** 个人中心所需的聚合数据，页面只依赖这个领域模型。 */
export interface AccountCenterSnapshot {
  user: UserInfo
  system: SystemConfig
  passkeys: PasskeyInfo[]
  bindings: OAuthBinding[]
}

/**
 * 账户域 API。
 * 页面不再直接拼接 auth 资源请求；所有个人中心读写都从这里进入，
 * 后续可无感切换到后端聚合接口。
 */
export const accountApi = {
  async getSnapshot(): Promise<AccountCenterSnapshot> {
    const [user, system, passkeyResult, bindingResult] = await Promise.all([
      request.get('/auth/userinfo') as unknown as Promise<UserInfo>,
      request.get('/auth/config', { requireAuth: false, skipAuthRefresh: true }) as unknown as Promise<SystemConfig>,
      request.get('/auth/account/passkeys') as unknown as Promise<{ passkeys: PasskeyInfo[] }>,
      request.get('/auth/oauth/bindings') as unknown as Promise<{ bindings: OAuthBinding[] }>,
    ])

    return {
      user,
      system,
      passkeys: passkeyResult.passkeys ?? [],
      bindings: bindingResult.bindings ?? [],
    }
  },

  updateProfile(params: UpdateProfileParams): Promise<UserInfo> {
    return request.put('/auth/account/profile', params)
  },

  changePassword(params: ChangePasswordParams): Promise<{ message: string }> {
    return request.post('/auth/account/change-password', params)
  },

  sendAccountEmailCode(purpose: AccountCodePurpose): Promise<SendCodeResult> {
    return request.post('/auth/account/email-code', { purpose })
  },

  beginTwoFactorSetup(): Promise<TwoFactorSetupResult> {
    return request.post('/auth/2fa/setup', {})
  },

  confirmTwoFactorSetup(code: string): Promise<TwoFactorEnableResult> {
    return request.post('/auth/2fa/enable', { code })
  },

  sendChangeEmailCode(newEmail: string): Promise<SendCodeResult> {
    return request.post('/auth/account/email-change-code', { newEmail })
  },

  changeEmail(params: ChangeEmailParams): Promise<UserInfo> {
    return request.put('/auth/account/email', params)
  },

  getPasskeyRegisterOptions(): Promise<PasskeyRegisterOptions> {
    return request.get('/auth/account/passkeys/register-options')
  },

  verifyPasskeyRegistration(payload: {
    name: string
    verifyCode: string
    credential: Record<string, unknown>
  }): Promise<{ message: string }> {
    return request.post('/auth/account/passkeys', payload)
  },

  listPasskeys(): Promise<{ passkeys: PasskeyInfo[] }> {
    return request.get('/auth/account/passkeys')
  },

  deletePasskey(id: string, verifyCode: string): Promise<{ message: string }> {
    return request.delete(`/auth/account/passkeys/${id}`, { params: { verifyCode } })
  },

  listOAuthBindings(): Promise<{ bindings: OAuthBinding[] }> {
    return request.get('/auth/oauth/bindings')
  },

  getOAuthBindURL(provider: string): Promise<OAuthAuthorizeResult> {
    return request.get('/auth/oauth/bind-url', { params: { provider } })
  },

  bindOAuth(params: OAuthCallbackParams): Promise<{ message: string }> {
    return request.post('/auth/oauth/bind', params)
  },

  unbindOAuth(provider: string): Promise<{ message: string }> {
    return request.delete(`/auth/oauth/bindings/${provider}`)
  },

  disableTwoFactor(verifyCode: string): Promise<{ message: string }> {
    return request.post('/auth/2fa/disable', { verifyCode })
  },

  setPreferredAuthMethod(method: 'totp' | 'passkey'): Promise<{ message: string }> {
    return request.put('/auth/account/preferred-auth-method', { method })
  },
}
