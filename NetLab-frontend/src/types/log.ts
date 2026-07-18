/**
 * 日志相关类型定义。
 *
 * 与后端日志接口契约保持一致（camelCase）。
 */

/** 单条登录日志 */
export interface LoginLogItem {
  id: number
  username: string
  loginType: string
  status: string
  ip: string
  os: string
  browser: string
  userAgent: string
  fingerprint: string
  location: string
  createdAt: string
}

/** 登录日志列表筛选条件 */
export interface LoginLogListParams {
  page?: number
  size?: number
  keyword?: string
  status?: string
  loginType?: string
}

/** 分页登录日志列表 */
export interface LoginLogListResult {
  items: LoginLogItem[]
  total: number
  page: number
  size: number
}
