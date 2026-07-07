/**
 * Web Crypto API 工具 —— HMAC 签名辅助函数。
 *
 * 会话签名密钥在认证期间建立并存储于 authStore 中，
 * 以便请求拦截器能够同步访问它。
 *
 * ── 签名 (HMAC-SHA256) ──
 * 签名内容: `${method}\n${path}\n${timestamp}\n${bodyHash}`
 * 服务端使用共享密钥进行校验。时间戳用于防重放。
 *
 * 注意：此处不做任何 payload 加密。传输机密性由 HTTPS/TLS 保证；
 * 打包进前端 bundle 的对称密钥会变为公开，无法带来任何机密性。
 */

// ── Buffer ↔ Hex / Base64 ──────────────────────────────────────────

/** ArrayBuffer → hex 字符串 */
export function bufferToHex(buf: ArrayBuffer): string {
  return Array.from(new Uint8Array(buf))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

/** Hex 字符串 → ArrayBuffer。输入为奇数长度或非 hex 时抛出异常。 */
export function hexToBuffer(hex: string): ArrayBuffer {
  const clean = hex.trim()
  if (clean.length % 2 !== 0 || /[^0-9a-fA-F]/.test(clean)) {
    throw new Error('invalid hex string')
  }
  const buf = new Uint8Array(clean.length / 2)
  for (let i = 0; i < buf.length; i++) {
    buf[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16)
  }
  return buf.buffer
}

/** Base64 → ArrayBuffer */
export function base64ToBuffer(b64: string): ArrayBuffer {
  const binary = atob(b64)
  const buf = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    buf[i] = binary.charCodeAt(i)
  }
  return buf.buffer
}

/** ArrayBuffer → base64（标准，非 URL-safe） */
export function bufferToBase64(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf)
  let binary = ''
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary)
}

// ── SHA-256 ────────────────────────────────────────────────────────

/** 计算字符串的 SHA-256 哈希 → hex */
export async function sha256(message: string): Promise<string> {
  const data = new TextEncoder().encode(message)
  const hash = await crypto.subtle.digest('SHA-256', data)
  return bufferToHex(hash)
}

// ── HMAC 签名 ────────────────────────────────────────────────────

/**
 * 导入原始 HMAC 密钥。secretKey 是一个 hex 编码的 256 位密钥
 * （64 个字符），在认证期间建立并由后端返回。
 */
export async function importSigningKey(secretKeyHex: string): Promise<CryptoKey> {
  const keyData = hexToBuffer(secretKeyHex)
  return crypto.subtle.importKey(
    'raw',
    keyData,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
}

/**
 * 创建 HMAC-SHA256 请求签名。
 *
 * 签名 payload（换行分隔）:
 *   {HTTP_METHOD}\n{path}\n{ISO-timestamp}\n{SHA256(body)}
 *
 * 服务端重建相同的 payload 并比对 HMAC。
 * 时间戳在服务端校验（±5 分钟窗口）以防重放。
 */
export async function createRequestSignature(
  signingKey: CryptoKey,
  method: string,
  path: string,
  timestamp: string,
  body: unknown,
): Promise<string> {
  const encoder = new TextEncoder()

  // 对请求体做哈希（GET/HEAD 等为空字符串）
  const bodyStr = body != null ? JSON.stringify(body) : ''
  const bodyHash = await sha256(bodyStr)

  const signPayload = `${method.toUpperCase()}\n${path}\n${timestamp}\n${bodyHash}`

  const signature = await crypto.subtle.sign('HMAC', signingKey, encoder.encode(signPayload))
  return bufferToHex(signature)
}

// ── JWT 工具 ───────────────────────────────────────────────────

interface JwtPayload {
  exp?: number
  iat?: number
  sub?: string
  [key: string]: unknown
}

/** 解码 JWT payload（不做验证）。失败时返回 null。 */
export function decodeJwtPayload(token: string): JwtPayload | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    // 解码 payload（中间部分）—— base64url → base64 → JSON
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const json = atob(base64)
    return JSON.parse(json) as JwtPayload
  } catch {
    return null
  }
}
