import request from './request'
import { saveWorkbook } from '@/utils/xlsx'
import type { LoginLogItem, LoginLogListParams, LoginLogListResult } from '@/types/log'

/** 导出时状态/登录方式 value → 本地化 label 的映射 */
export interface LoginLogExportLabels {
  status: Record<string, string>
  type: Record<string, string>
}

/**
 * 日志查询接口，访问控制由后端 RBAC 权限（log.read）决定。
 */
export const logApi = {
  /** 分页获取登录日志 */
  listLoginLogs(params: LoginLogListParams): Promise<LoginLogListResult> {
    return request.get('/logs/logins', { params })
  },

  /** 批量删除登录日志（log.delete） */
  deleteLoginLogs(ids: number[]): Promise<{ deleted: number }> {
    return request.delete('/logs/logins', { data: { ids } })
  },

  /** 将选中的登录日志在浏览器生成 Excel 并下载（内容已本地化）。 */
  exportLoginLogs(
    logs: LoginLogItem[],
    headers: string[],
    labels: LoginLogExportLabels,
    filename: string,
  ): void {
    const rows = logs.map((log) => [
      new Date(log.createdAt).toLocaleString(),
      log.username,
      labels.type[log.loginType] ?? log.loginType,
      labels.status[log.status] ?? log.status,
      log.ip,
      log.location,
      log.os,
      log.browser,
      log.fingerprint,
      log.userAgent,
    ])
    saveWorkbook(rows, headers, filename, [20, 16, 12, 10, 16, 14, 18, 14, 24, 40])
  },
}
