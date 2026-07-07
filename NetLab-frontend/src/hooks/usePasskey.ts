import { useCallback } from 'react'
import { App } from 'antd'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import { tokenStorage } from '@/utils/token'

/**
 * WebAuthn Passkey hook
 *
 * 封装浏览器原生的 Web Authentication API
 * （navigator.credentials），用于通过指纹、面部识别、
 * Windows Hello 或安全密钥实现免密登录。
 */

/** 将 base64url 转换为 ArrayBuffer（符合 WebAuthn 规范） */
function base64urlToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/')
  const padLen = (4 - (base64.length % 4)) % 4
  const padded = base64 + '='.repeat(padLen)
  const raw = atob(padded)
  const buffer = new ArrayBuffer(raw.length)
  const bytes = new Uint8Array(buffer)
  for (let i = 0; i < raw.length; i++) {
    bytes[i] = raw.charCodeAt(i)
  }
  return buffer
}

/** 将 ArrayBuffer 转换为 base64url */
function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

/** 序列化 PublicKeyCredential 以发送给服务器 */
function serializeCredential(cred: PublicKeyCredential): Record<string, unknown> {
  const response = cred.response as AuthenticatorAssertionResponse | AuthenticatorAttestationResponse
  const serialized: Record<string, unknown> = {
    id: cred.id,
    rawId: bufferToBase64url(cred.rawId),
    type: cred.type,
  }

  if ('authenticatorData' in response) {
    serialized.response = {
      authenticatorData: bufferToBase64url(response.authenticatorData),
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      signature: bufferToBase64url(response.signature),
      userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : null,
    }
  }
  if ('attestationObject' in response) {
    serialized.response = {
      ...((serialized.response as object) || {}),
      attestationObject: bufferToBase64url(response.attestationObject),
    }
  }
  return serialized
}

export function usePasskey() {
  const { message } = App.useApp()
  const navigate = useNavigate()

  /** 检查当前浏览器是否支持 WebAuthn */
  const isSupported = useCallback((): boolean => {
    return !!window.PublicKeyCredential
  }, [])

  /** 检查平台认证器（生物识别）是否可用 */
  const isPlatformAuthAvailable = useCallback(async (): Promise<boolean> => {
    if (!window.PublicKeyCredential) return false
    try {
      const available = await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()
      return available
    } catch {
      return false
    }
  }, [])

  /**
   * 注册新的 passkey（凭证创建）
   * 从用户设置页面调用，为现有账户添加 passkey
   */
  const register = useCallback(async (): Promise<boolean> => {
    try {
      const options = await authApi.getPasskeyRegisterOptions()

      const publicKey: PublicKeyCredentialCreationOptions = {
        challenge: base64urlToBuffer(options.challenge),
        rp: options.rp,
        user: {
          ...options.user,
          id: base64urlToBuffer(options.user.id),
        },
        pubKeyCredParams: options.pubKeyCredParams,
        timeout: options.timeout,
        attestation: options.attestation as AttestationConveyancePreference,
        authenticatorSelection: {
          ...options.authenticatorSelection,
          authenticatorAttachment: options.authenticatorSelection.authenticatorAttachment as AuthenticatorAttachment,
          residentKey: options.authenticatorSelection.residentKey as ResidentKeyRequirement,
          userVerification: options.authenticatorSelection.userVerification as UserVerificationRequirement,
        },
      }

      const credential = await navigator.credentials.create({ publicKey })
      if (!(credential instanceof PublicKeyCredential)) {
        throw new Error('Failed to create passkey')
      }

      const serialized = serializeCredential(credential)
      await authApi.verifyPasskeyRegistration(serialized)
      message.success('Passkey registered successfully')
      return true
    } catch (err) {
      if ((err as Error).name !== 'NotAllowedError') {
        message.error('Passkey registration failed')
      }
      return false
    }
  }, [message])

  /**
   * 使用现有 passkey 登录（凭证断言）
   * 触发浏览器原生的生物识别/安全密钥提示
   */
  const login = useCallback(async (): Promise<boolean> => {
    try {
      // 1. 从服务器获取 challenge
      const options = await authApi.getPasskeyAuthOptions()

      // 2. 构建 WebAuthn 断言请求
      const publicKey: PublicKeyCredentialRequestOptions = {
        challenge: base64urlToBuffer(options.challenge),
        rpId: options.rpId,
        timeout: options.timeout,
        userVerification: options.userVerification as UserVerificationRequirement,
      }

      if (options.allowCredentials?.length) {
        publicKey.allowCredentials = options.allowCredentials.map((cred) => ({
          ...cred,
          id: base64urlToBuffer(cred.id),
        }))
      }

      // 3. 触发浏览器生物识别/安全密钥提示
      const credential = await navigator.credentials.get({ publicKey })
      if (!(credential instanceof PublicKeyCredential)) {
        throw new Error('Failed to authenticate with passkey')
      }

      // 4. 将断言发送给服务器进行验证
      const serialized = serializeCredential(credential)
      const result = await authApi.verifyPasskeyAuth(serialized)

      // 5. 存储 token、会话密钥并重定向
      tokenStorage.setAccessToken(result.accessToken)
      tokenStorage.setRefreshToken(result.refreshToken)
      useAuthStore.setState({
        accessToken: result.accessToken,
        refreshToken: result.refreshToken,
        userInfo: result.user,
        signingKey: result.signingKey ?? null,
      })

      message.success('Welcome back!')
      navigate('/dashboard', { replace: true })
      return true
    } catch (err) {
      const e = err as Error
      if (e.name === 'NotAllowedError') {
        // 用户取消了生物识别提示 —— 这不是错误，不显示提示
      } else {
        message.error('Passkey authentication failed')
      }
      return false
    }
  }, [message, navigate])

  return { isSupported, isPlatformAuthAvailable, register, login }
}
