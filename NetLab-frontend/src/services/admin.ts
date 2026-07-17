import request from './request'
import * as XLSX from 'xlsx'
import type {
  AdminSettings,
  SecuritySettings,
  BeianSettings,
  SMTPSettings,
  OAuthProviderSettings,
  AdminUserView,
  UserListResult,
  ImportSummary,
  ImportUserParams,
  ExportUsersParams,
  UserListParams,
  UpdateUserParams,
  CreateUserParams,
} from '@/types/settings'

/** 触发浏览器将 Blob 保存为本地文件。 */
function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/**
 * 系统设置与用户资源接口，访问控制由后端 RBAC 权限决定。
 *
 * 所有写操作在后端热生效（无需重启）。密钥字段（SMTP 密码、OAuth
 * Client Secret）在返回时被掩码；提交时若仍为掩码或留空，则保留原值。
 */
export const adminApi = {
  /** 获取完整系统设置（密钥已掩码） */
  getSettings(): Promise<AdminSettings> {
    return request.get('/settings')
  },

  /** 更新安全策略 */
  updateSecurity(params: SecuritySettings): Promise<{ message: string }> {
    return request.put('/settings/security', params)
  },

  /** 更新备案信息 */
  updateBeian(params: BeianSettings): Promise<{ message: string }> {
    return request.put('/settings/beian', params)
  },

  /** 更新 SMTP 邮件服务配置 */
  updateSMTP(params: SMTPSettings): Promise<{ message: string }> {
    return request.put('/settings/smtp', params)
  },

  /** 使用当前 SMTP 配置发送测试邮件 */
  testSMTP(to: string): Promise<{ message: string }> {
    return request.post('/settings/smtp/test', { to })
  },

  /** 更新指定 OAuth 提供商配置 */
  updateOAuthProvider(
    provider: string,
    params: Pick<OAuthProviderSettings, 'enabled' | 'clientId' | 'clientSecret' | 'redirectUrl'>,
  ): Promise<{ message: string }> {
    return request.put(`/settings/oauth/${provider}`, params)
  },

  /** ── 用户管理 ── */

  /** 分页获取用户列表 */
  listUsers(params: UserListParams): Promise<UserListResult> {
    return request.get('/users', { params })
  },

  /** 更新单个用户资料、角色和状态 */
  updateUser(id: string, params: UpdateUserParams): Promise<{ message: string }> {
    return request.put(`/users/${id}`, params)
  },

  /** 创建单个用户 */
  createUser(params: CreateUserParams): Promise<AdminUserView> {
    return request.post('/users', params)
  },

  /** 批量删除用户 */
  batchDeleteUsers(userIds: string[]): Promise<{ message: string }> {
    return request.delete('/users', { data: { userIds } })
  },

  /** 批量重置密码为统一新密码 */
  batchResetPassword(userIds: string[], newPassword: string): Promise<{ message: string }> {
    return request.put('/users/reset-password', { userIds, newPassword })
  },

  /** 导出勾选的用户为 Excel 文件（触发浏览器下载） */
  async exportUsers(params: ExportUsersParams): Promise<void> {
    const response = await request.post('/users/export', params, {
      responseType: 'blob',
    })
    saveBlob(response as unknown as Blob, `netlab-users-${new Date().toISOString().slice(0, 10)}.xlsx`)
  },

  /** 下载本地化表头的用户导入模板（xlsx） */
  async downloadImportTemplate(): Promise<void> {
    const response = await request.get('/users/import-template', {
      responseType: 'blob',
    })
    saveBlob(response as unknown as Blob, 'netlab-users-template.xlsx')
  },

  /** 在浏览器解析 xlsx/xls/csv，再提交后端约定的 JSON。 */
  async importUsers(file: File): Promise<ImportSummary> {
    const workbook = XLSX.read(await file.arrayBuffer(), { type: 'array' })
    const sheet = workbook.Sheets[workbook.SheetNames[0]]
    if (!sheet) throw new Error('empty import file')

    const rows = XLSX.utils.sheet_to_json<unknown[]>(sheet, { header: 1, defval: '', raw: false })
    if (rows.length < 2) throw new Error('import file has no data rows')
    const headers = (rows[0] ?? []).map((value) => normalizeImportHeader(String(value)))
    const indexes = {
      username: headers.indexOf('username'),
      nickname: headers.indexOf('nickname'),
      phone: headers.indexOf('phone'),
      email: headers.indexOf('email'),
      role: headers.indexOf('role'),
      password: headers.indexOf('password'),
    }
    if (indexes.username < 0 || indexes.nickname < 0 || indexes.phone < 0 || indexes.email < 0) {
      throw new Error('username, nickname, phone and email columns are required')
    }

    const users: ImportUserParams[] = rows.slice(1).map((row) => ({
      username: cell(row, indexes.username),
      nickname: cell(row, indexes.nickname),
      phone: cell(row, indexes.phone),
      email: cell(row, indexes.email),
      role: cell(row, indexes.role),
      password: cell(row, indexes.password),
    })).filter((user) => user.username || user.email || user.role || user.password)
    if (users.length === 0) throw new Error('import file has no data rows')
    return request.post('/users/import', { users })
  },
}

function normalizeImportHeader(value: string) {
  let header = value.replace(/^\uFEFF/, '').trim().toLowerCase()
  const suffixIndex = header.search(/[ \u3000(（]/)
  if (suffixIndex > 0) header = header.slice(0, suffixIndex).trim()
  return ({
    username: 'username',
    '\u7528\u6237\u540d': 'username',
    nickname: 'nickname',
    '\u6635\u79f0': 'nickname',
    phone: 'phone',
    '\u624b\u673a\u53f7': 'phone',
    email: 'email',
    '\u90ae\u7bb1': 'email',
    role: 'role',
    '\u89d2\u8272': 'role',
    password: 'password',
    '\u5bc6\u7801': 'password',
  } as Record<string, string>)[header] ?? header
}

function cell(row: unknown[], index: number) {
  return index >= 0 ? String(row[index] ?? '').trim() : ''
}
