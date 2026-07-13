import { useCallback } from 'react'
import { App } from 'antd'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/services/auth'
import { completeLogin } from '@/utils/auth-flow'
import type {
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
} from '@/types/auth'

/**
 * WebAuthn Passkey hook
 *
 * 封装浏览器原生的 Web Authentication API
 * （navigator.credentials），用于通过指纹、面部识别、
 * Windows Hello 或安全密钥实现免密登录。
 *
 * 后端使用 go-webauthn 进行符合规范的签名校验，因此本 hook 负责在
 * base64url（服务端 JSON）与 ArrayBuffer（浏览器 API）之间做转换。
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

/** 序列化 PublicKeyCredential 以发送给服务器（标准 WebAuthn JSON 形态） */
function serializeCredential(cred: PublicKeyCredential): Record<string, unknown> {
  const response = cred.response as AuthenticatorAssertionResponse & AuthenticatorAttestationResponse
  const serialized: Record<string, unknown> = {
    id: cred.id,
    rawId: bufferToBase64url(cred.rawId),
    type: cred.type,
  }

  const resp: Record<string, unknown> = {
    clientDataJSON: bufferToBase64url(response.clientDataJSON),
  }
  if ('attestationObject' in response && response.attestationObject) {
    resp.attestationObject = bufferToBase64url(response.attestationObject)
  }
  if ('authenticatorData' in response && response.authenticatorData) {
    resp.authenticatorData = bufferToBase64url(response.authenticatorData)
    resp.signature = bufferToBase64url(response.signature)
    resp.userHandle = response.userHandle ? bufferToBase64url(response.userHandle) : null
  }
  serialized.response = resp
  return serialized
}

/** 将服务端创建选项映射为浏览器 API 所需的结构 */
function toCreationOptions(
  opts: PublicKeyCredentialCreationOptionsJSON,
): PublicKeyCredentialCreationOptions {
  return {
    challenge: base64urlToBuffer(opts.challenge),
    rp: opts.rp,
    user: {
      id: base64urlToBuffer(opts.user.id),
      name: opts.user.name,
      displayName: opts.user.displayName,
    },
    pubKeyCredParams: opts.pubKeyCredParams as PublicKeyCredentialParameters[],
    timeout: opts.timeout,
    attestation: opts.attestation as AttestationConveyancePreference | undefined,
    excludeCredentials: opts.excludeCredentials?.map((c) => ({
      id: base64urlToBuffer(c.id),
      type: c.type as 'public-key',
      transports: c.transports as AuthenticatorTransport[] | undefined,
    })),
    authenticatorSelection: opts.authenticatorSelection
      ? {
          authenticatorAttachment: opts.authenticatorSelection
            .authenticatorAttachment as AuthenticatorAttachment | undefined,
          residentKey: opts.authenticatorSelection.residentKey as
            | ResidentKeyRequirement
            | undefined,
          requireResidentKey: opts.authenticatorSelection.requireResidentKey,
          userVerification: opts.authenticatorSelection.userVerification as
            | UserVerificationRequirement
            | undefined,
        }
      : undefined,
  }
}

/** 将服务端断言选项映射为浏览器 API 所需的结构 */
function toRequestOptions(
  opts: PublicKeyCredentialRequestOptionsJSON,
): PublicKeyCredentialRequestOptions {
  const publicKey: PublicKeyCredentialRequestOptions = {
    challenge: base64urlToBuffer(opts.challenge),
    rpId: opts.rpId,
    timeout: opts.timeout,
    userVerification: opts.userVerification as UserVerificationRequirement | undefined,
  }
  if (opts.allowCredentials?.length) {
    publicKey.allowCredentials = opts.allowCredentials.map((c) => ({
      id: base64urlToBuffer(c.id),
      type: c.type as 'public-key',
      transports: c.transports as AuthenticatorTransport[] | undefined,
    }))
  }
  return publicKey
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
      return await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()
    } catch {
      return false
    }
  }, [])

  /**
   * 注册新的 passkey（凭证创建）。
   * 从个人中心调用，为现有账户添加 passkey。
   * name 为用户为该 passkey 指定的可读名称；
   * verifyCode 为发送到用户邮箱的一次性验证码（后端二次校验）。
   */
  const register = useCallback(
    async (name: string, verifyCode: string): Promise<boolean> => {
      try {
        const options = await authApi.getPasskeyRegisterOptions()
        const publicKey = toCreationOptions(options.publicKey)

        const credential = await navigator.credentials.create({ publicKey })
        if (!(credential instanceof PublicKeyCredential)) {
          throw new Error('Failed to create passkey')
        }

        const serialized = serializeCredential(credential)
        await authApi.verifyPasskeyRegistration({ name, verifyCode, credential: serialized })
        return true
      } catch (err) {
        if ((err as Error).name !== 'NotAllowedError') {
          message.error((err as Error).message || 'Passkey registration failed')
        }
        return false
      }
    },
    [message],
  )

  /**
   * 使用现有 passkey 登录（凭证断言）。
   * 触发浏览器原生的生物识别/安全密钥提示。
   */
  const login = useCallback(async (): Promise<boolean> => {
    try {
      // 1. 从服务器获取 challenge 与会话 ID
      const options = await authApi.getPasskeyAuthOptions()

      // 2. 触发浏览器生物识别/安全密钥提示
      const publicKey = toRequestOptions(options.publicKey)
      const credential = await navigator.credentials.get({ publicKey })
      if (!(credential instanceof PublicKeyCredential)) {
        throw new Error('Failed to authenticate with passkey')
      }

      // 3. 将断言连同会话 ID 发送给服务器验证
      const serialized = serializeCredential(credential)
      const result = await authApi.verifyPasskeyAuth({
        sessionId: options.sessionId,
        credential: serialized,
      })

      completeLogin(result, navigate)
      return true
    } catch (err) {
      const e = err as Error
      if (e.name !== 'NotAllowedError') {
        message.error(e.message || 'Passkey authentication failed')
      }
      return false
    }
  }, [message, navigate])

  return { isSupported, isPlatformAuthAvailable, register, login }
}
