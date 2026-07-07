/**
 * 公开 auth 接口的预共享密钥请求签名。
 *
 * 公开 auth 接口（login、register、reset-password）在会话签名密钥
 * 建立之前运行。为了给后端提供一个轻量的防篡改 / 防重放机制，
 * 我们使用预共享签名密钥以 HMAC-SHA256 对请求签名，与服务端的
 * Signature 中间件相匹配。
 *
 * 注意：此处刻意不做请求体加密。传输机密性由 HTTPS/TLS 保证。
 * 密码在 TLS 隧道内以明文传输，后端使用 bcrypt 进行校验。
 * 将预共享 AES 密钥打包进前端 bundle 会使其变为公开，
 * 因此无法带来任何机密性——故已移除。
 *
 * ── 签名 ──
 * Payload: X-Request-Id + salt + X-Timestamp + SHA-256(body JSON)
 * 算法: HMAC-SHA256，hex 编码
 * 时间戳用于防重放（服务端 ±5 分钟窗口）。
 *
 * 所有加密操作均使用 Web Crypto API (crypto.subtle)。
 */
import { bufferToHex, hexToBuffer, sha256 } from '@/utils/crypto'
import { getI18nT } from '@/utils/i18n-bridge'

/** 在抛出时解析本地化的 auth-security 错误消息。 */
function authSecurityErrorMessage(): string {
  return getI18nT()('common:authSecurityUnavailable')
}

// 派生出的 HMAC 密钥缓存
let cachedHmacKey: CryptoKey | null = null
let keySource: string | null = null

export class AuthSecurityError extends Error {
  constructor(message?: string) {
    super(message ?? authSecurityErrorMessage())
    this.name = 'AuthSecurityError'
  }
}

export function isAuthSecurityError(error: unknown): error is AuthSecurityError {
  return error instanceof AuthSecurityError
}

// ── 环境变量校验 ──────────────────────────────────────────

function getRequiredEnv(_name: string, value: string | undefined): string {
  if (!value?.trim()) {
    throw new AuthSecurityError()
  }
  return value.trim()
}

// ── 密钥派生 ──────────────────────────────────────────────────

/**
 * 从签名密钥环境变量派生 HMAC-SHA256 密钥。
 * VITE_AUTH_SIGNATURE_KEY 的原始值（hex 编码）会被解码为字节，
 * 并直接用作 HMAC 密钥。
 */
async function deriveHmacKey(): Promise<CryptoKey> {
  const source = getRequiredEnv(
    'VITE_AUTH_SIGNATURE_KEY',
    import.meta.env.VITE_AUTH_SIGNATURE_KEY,
  )

  if (cachedHmacKey && keySource === source) return cachedHmacKey

  // VITE_AUTH_SIGNATURE_KEY 是 hex 编码的原始密钥（与后端 AUTH_SIGNATURE_KEY 相匹配）
  const keyBytes = hexToBuffer(source)
  cachedHmacKey = await crypto.subtle.importKey(
    'raw',
    keyBytes,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
  keySource = source
  return cachedHmacKey
}

/**
 * 使缓存的 HMAC 密钥失效（例如开发期间环境变量发生变化时）。
 */
export function invalidateAuthSecurityKeys(): void {
  cachedHmacKey = null
  keySource = null
}

// ── HMAC-SHA256 签名 ────────────────────────────────────────────

/**
 * 生成随机请求 ID（16 字节，hex 编码）。
 */
function generateRequestId(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  return bufferToHex(bytes.buffer)
}

export interface AuthSignatureHeaders {
  'X-Request-Id': string
  'X-Signature': string
  'X-Timestamp': string
}

/**
 * 为预共享密钥 auth 请求创建签名头。
 *
 * Payload: X-Request-Id + salt + X-Timestamp + SHA-256(body JSON)
 * body JSON 是后端将读取并校验的、精确序列化后的（明文）请求体。
 * payload 中的时间戳用于防重放攻击（服务端强制 ±5 分钟）。
 *
 * @param bodyText - 将作为请求体发送的精确 JSON 字符串。
 */
export async function createAuthSignatureHeaders(
  bodyText: string,
): Promise<AuthSignatureHeaders> {
  const hmacKey = await deriveHmacKey()
  const requestId = generateRequestId()
  const salt = getRequiredEnv(
    'VITE_AUTH_SIGNATURE_SALT',
    import.meta.env.VITE_AUTH_SIGNATURE_SALT,
  )
  const timestamp = new Date().toISOString()
  const bodyHash = await sha256(bodyText)

  const signPayload = `${requestId}${salt}${timestamp}${bodyHash}`

  const encoder = new TextEncoder()
  const signature = await crypto.subtle.sign('HMAC', hmacKey, encoder.encode(signPayload))

  return {
    'X-Request-Id': requestId,
    'X-Signature': bufferToHex(signature),
    'X-Timestamp': timestamp,
  }
}
