/** API 基础路径 */
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

/** 请求超时时间 (ms) */
export const REQUEST_TIMEOUT = 15_000

/** Token 过期刷新缓冲时间 (ms)：在 Token 过期前 5 分钟刷新 */
export const TOKEN_REFRESH_BUFFER = 5 * 60 * 1000
