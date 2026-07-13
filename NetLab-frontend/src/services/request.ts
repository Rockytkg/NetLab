import axios, {
  type AxiosError,
  type InternalAxiosRequestConfig,
  type AxiosResponse,
} from 'axios'
import { API_BASE_URL, REQUEST_TIMEOUT, TOKEN_REFRESH_BUFFER } from '@/utils/constants'
import { useAuthStore } from '@/stores/authStore'
import { useAppStore } from '@/stores/appStore'
import { getI18nT } from '@/utils/i18n-bridge'
import { getMessageApi } from '@/utils/message-bridge'
import {
  decodeJwtPayload,
} from '@/utils/crypto'
import { createAuthSignatureHeaders } from './authSecurity'
import {
  HTTP_ERROR_I18N_MAP,
  BUSINESS_ERROR_I18N_MAP,
} from '@/types/api'

// ── Axios 实例 ──────────────────────────────────────────────────

/**
 * 上下文感知的 message API。通过 message bridge 懒解析，使本模块
 * 保持在 React 树之外，同时仍能使用 App 作用域内的实例
 * （避免 "Static function can not consume context" 警告）。
 */
const message = {
  error: (content: string) => getMessageApi().error(content),
}

const request = axios.create({
  baseURL: API_BASE_URL,
  timeout: REQUEST_TIMEOUT,
  headers: {
    'Content-Type': 'application/json',
  },
})

// ── Token 刷新状态（防止并发刷新） ───────────────

let isRefreshing = false
let pendingQueue: Array<(token: string | null) => void> = []

function resolvePendingQueue(token: string | null) {
  pendingQueue.forEach((cb) => cb(token))
  pendingQueue = []
}

// ── JWT 过期辅助函数 ───────────────────────────────────────────────

/**
 * 检查 access token 是否即将过期。
 * 若 token 的 `exp` 落在 TOKEN_REFRESH_BUFFER 之内则返回 true。
 */
function isTokenExpiringSoon(token: string): boolean {
  const payload = decodeJwtPayload(token)
  if (!payload?.exp) return false
  const expiresInMs = payload.exp * 1000 - Date.now()
  return expiresInMs < TOKEN_REFRESH_BUFFER
}

/**
 * 尝试主动刷新 token。
 * 使用刷新队列模式——如果已有刷新正在进行，
 * 调用方会在队列上等待，而不是发起第二次刷新。
 */
async function tryProactiveRefresh(): Promise<string | null> {
  const store = useAuthStore.getState()
  if (!store.refreshToken) return null

  if (isRefreshing) {
    // 将该调用方入队——它会在进行中的刷新完成后被解析
    return new Promise((resolve) => {
      pendingQueue.push((token) => resolve(token))
    })
  }

  isRefreshing = true
  try {
    const newToken = await store.refreshAccessToken()
    resolvePendingQueue(newToken)
    return newToken
  } catch {
    resolvePendingQueue(null)
    return null
  } finally {
    isRefreshing = false
  }
}

// ══════════════════════════════════════════════════════════════════════
// 请求拦截器
// ══════════════════════════════════════════════════════════════════════

request.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    // ── 1. JWT 认证 ──────────────────────────────────
    const requireAuth = config.requireAuth !== false
    const skipRefresh = config.skipAuthRefresh === true

    if (requireAuth && config.headers) {
      let token = useAuthStore.getState().accessToken

      // 主动刷新：若 token 即将过期，且当前不是 auth 接口
      // （否则会导致无限递归）。
      if (token && !skipRefresh && isTokenExpiringSoon(token)) {
        const newToken = await tryProactiveRefresh()
        if (newToken) token = newToken
      }

      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
    }

    // ── 2. 语言头 ───────────────────────────────────────
    if (config.headers) {
      const locale = useAppStore.getState().locale
      config.headers['Accept-Language'] = locale
      // X-User-Language 携带用户的明确偏好，在后端优先级
      // 高于浏览器注入的 Accept-Language。
      config.headers['X-User-Language'] = locale
    }

    // ── 3. 请求追踪 ID ──────────────────────────────────
    if (config.headers) {
      config.headers['X-Request-Id'] ??=
        `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
    }

    // ── 4. 预共享密钥 auth 签名 ───────────────────────
    // 用于携带敏感字段的公开 auth 接口（login/register/reset-password）。
    // 请求体以明文发送（由 HTTPS 保护）；
    // 我们仅附加 HMAC 签名，以便后端校验完整性并拒绝重放。
    // 对实际发送的、精确序列化后的请求体进行签名。
    if (config.authSign && config.headers) {
      const bodyText = config.data != null ? JSON.stringify(config.data) : ''
      const sigHeaders = await createAuthSignatureHeaders(bodyText)
      config.headers['X-Request-Id'] = sigHeaders['X-Request-Id']
      config.headers['X-Signature'] = sigHeaders['X-Signature']
      config.headers['X-Timestamp'] = sigHeaders['X-Timestamp']
    }

    return config
  },
  (error: AxiosError) => Promise.reject(error),
)

// ══════════════════════════════════════════════════════════════════════
// 响应拦截器 —— 成功
// ══════════════════════════════════════════════════════════════════════

request.interceptors.response.use(
  (response: AxiosResponse) => {
    const { data } = response
    const t = getI18nT()

    // ── 1. 解包 { code, data, message } 信封 ──────────────
    const payload = data
    if (payload && typeof payload === 'object' && 'code' in payload) {
      const apiResponse = payload as { code: number; data: unknown; message?: string }

      // 成功码
      if (apiResponse.code === 0 || apiResponse.code === 200) {
        return apiResponse.data
      }

      // ── 2. 业务错误码 → i18n 消息 ──────────────────────
      const bizCode = apiResponse.code
      const i18nKey = BUSINESS_ERROR_I18N_MAP[bizCode]
      const errorMsg =
        (i18nKey ? t(i18nKey) : null)
        ?? apiResponse.message
        ?? t('common:requestFailed')

      message.error(errorMsg)

      // 特殊处理：token/会话过期强制登出
      if (bizCode === 1004 || bizCode === 1005 || bizCode === 1012) {
        const authStore = useAuthStore.getState()
        authStore.logout({ callApi: false })
        // 重定向到登录页 —— 使用 window.location 以在 React 树之外生效
        window.location.href = `/login?redirect=${encodeURIComponent(window.location.pathname)}`
      }

      return Promise.reject(new Error(errorMsg))
    }

    // 非信封响应 —— 原样透传
    return payload
  },

  // ═══════════════════════════════════════════════════════════════
  // 响应拦截器 —— 错误
  // ═══════════════════════════════════════════════════════════════

  async (error: AxiosError) => {
    const { config, response } = error
    const t = getI18nT()

    // ── 1. 网络错误（无响应） ───────────────────────────
    if (!response) {
      message.error(t('common:networkError'))
      return Promise.reject(error)
    }

    const { status, data } = response

    // ── 2. 从响应体提取业务错误码 ────────
    const bizCode =
      data && typeof data === 'object' && 'code' in (data as Record<string, unknown>)
        ? (data as { code: number }).code
        : null

    // ── 3. 401 —— 尝试刷新 token（双 token 设计） ───────
    if (status === 401 && config && config.skipAuthRefresh !== true) {
      // 若已尝试过刷新但仍返回 401，则不再循环
      if (isRefreshing) {
        return new Promise((resolve) => {
          pendingQueue.push((token) => {
            if (token && config.headers) {
              config.headers.Authorization = `Bearer ${token}`
              resolve(request(config))
            } else {
              // 刷新失败 —— 以原始错误 reject
              resolve(Promise.reject(error))
            }
          })
        })
      }

      isRefreshing = true
      try {
        const newToken = await useAuthStore.getState().refreshAccessToken()
        resolvePendingQueue(newToken)

        if (newToken && config.headers) {
          config.headers.Authorization = `Bearer ${newToken}`
          return request(config)
        }

        // 刷新失败 —— 会话已失效
        message.error(t('common:sessionExpiredLoginAgain'))
        resolvePendingQueue(null)
        return Promise.reject(error)
      } catch {
        resolvePendingQueue(null)
        message.error(t('common:sessionExpiredLoginAgain'))
        return Promise.reject(error)
      } finally {
        isRefreshing = false
      }
    }

    // ── 4. 业务错误码（最具体 —— 优先检查） ──────
    if (bizCode && typeof bizCode === 'number' && BUSINESS_ERROR_I18N_MAP[bizCode]) {
      const bizKey = BUSINESS_ERROR_I18N_MAP[bizCode]
      const bizMsg =
        data && typeof data === 'object' && 'message' in (data as Record<string, unknown>)
          ? (data as { message: string }).message
          : undefined
      message.error(bizMsg || t(bizKey))
      return Promise.reject(new Error(bizMsg || t(bizKey)))
    }

    // ── 5. HTTP 状态码 → i18n 消息 ────────────────────────────
    const httpKey = HTTP_ERROR_I18N_MAP[status]
    if (httpKey) {
      message.error(t(httpKey, { status }))
      return Promise.reject(error)
    }

    // ── 6. 兜底（仅当 bizCode 与 httpKey 均未匹配时到达） ──
    const fallbackMsg =
      data && typeof data === 'object' && 'message' in (data as Record<string, unknown>)
        ? (data as { message: string }).message
        : t('common:errorUnknown', { code: status })

    // 401 已由上面的刷新流程处理；此处不重复提示
    if (status !== 401) {
      message.error(fallbackMsg)
    }

    return Promise.reject(error)
  },
)

export default request
