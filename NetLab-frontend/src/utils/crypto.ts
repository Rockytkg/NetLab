/**
 * Web Crypto API 工具 —— HMAC 签名与 JWT 辅助函数。
 *
 * 注意：此处不做任何 payload 加密。传输机密性由 HTTPS/TLS 保证；
 * 打包进前端 bundle 的对称密钥会变为公开，无法带来任何机密性。
 */

// ── Buffer ↔ Hex ──────────────────────────────────────────

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

// ── SHA-256 ────────────────────────────────────────────────────────

/** 计算字符串的 SHA-256 哈希 → hex */
export async function sha256(message: string): Promise<string> {
  const data = new TextEncoder().encode(message)
  const hash = await crypto.subtle.digest('SHA-256', data)
  return bufferToHex(hash)
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
