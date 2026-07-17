// ── Axios 自定义配置扩展 ────────────────────────────────

declare module 'axios' {
  interface AxiosRequestConfig {
    /**
     * 是否需要 JWT 认证头 (default: true)。
     */
    requireAuth?: boolean

    /**
     * 是否对公开 auth 接口做预共享密钥签名 (default: false)。
     */
    authSign?: boolean

    /**
     * 跳过 401 自动 token 刷新 (default: false)。
     */
    skipAuthRefresh?: boolean

    /**
     * 内部标记：该请求已因 401 刷新并重试过一次。
     * 防止"刷新→重试→再 401"无限循环。
     */
    _retry?: boolean

  }

  interface InternalAxiosRequestConfig {
    /**
     * 是否需要 JWT 认证头 (default: true)。
     * 设为 false 可跳过 Bearer token 注入（如登录、刷新 token 等公开接口）。
     */
    requireAuth?: boolean

    /**
     * 是否对公开 auth 接口做预共享密钥签名 (default: false)。
     * 启用后使用 VITE_AUTH_SIGNATURE_KEY 对（明文）请求体做 HMAC 签名，
     * 添加 X-Request-Id / X-Signature / X-Timestamp 头。
     * 请求体不加密——机密性由 HTTPS 保证。
     */
    authSign?: boolean

    /**
     * 跳过 401 自动 token 刷新 (default: false)。
     * auth 相关接口（如 login、refresh）本身应设为 true。
     */
    skipAuthRefresh?: boolean

    /**
     * 内部标记：该请求已因 401 刷新并重试过一次。
     * 防止"刷新→重试→再 401"无限循环。
     */
    _retry?: boolean

  }
}

// ── 错误码常量 ────────────────────────────────────────────

/** HTTP 状态码 → i18n key 映射 */
export const HTTP_ERROR_I18N_MAP: Record<number, string> = {
  400: 'common:error400',
  401: 'common:error401',
  403: 'common:forbidden',
  404: 'common:error404',
  405: 'common:error405',
  408: 'common:error408',
  409: 'common:error409',
  422: 'common:error422',
  429: 'common:error429',
  500: 'common:serverError',
  502: 'common:error502',
  503: 'common:error503',
  504: 'common:error504',
}

/** 业务错误码 → i18n key 映射 */
export const BUSINESS_ERROR_I18N_MAP: Record<number, string> = {
  1001: 'common:errInvalidCredentials',
  1002: 'common:errAccountLocked',
  1003: 'common:errAccountDisabled',
  1004: 'common:errTokenExpired',
  1005: 'common:errInvalidRefreshToken',
  1006: 'common:errUserNotFound',
  1007: 'common:errEmailExists',
  1008: 'common:errUsernameExists',
  1009: 'common:errInvalidCode',
  1010: 'common:errWeakPassword',
  1011: 'common:errRateLimited',
  1012: 'common:errSessionExpired',
  1013: 'common:errDuplicateEntry',
  1014: 'common:errOperationDenied',
  1015: 'common:errResourceInUse',
  1016: 'common:errEmailNotConfigured',
  1017: 'common:errEmailSendFailed',
  1018: 'common:errPasswordResetDisabled',
  1019: 'common:errUnauthorized',
  1020: 'common:errInvalidTwoFactorCode',
  1021: 'common:errTwoFactorNotConfigured',
  1022: 'common:errInternalError',
  1023: 'common:errInvalidSignature',
  1024: 'common:errInvalidRequest',
  1025: 'common:errInvalidFile',
}
