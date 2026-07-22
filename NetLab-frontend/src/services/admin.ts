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
  AdminUserExportView,
	BillingSettings,
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

function saveWorkbook(rows: unknown[][], headers: string[], filename: string, widths: number[]) {
  const worksheet = XLSX.utils.aoa_to_sheet([headers, ...rows])
  worksheet['!cols'] = widths.map((wch) => ({ wch }))
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, worksheet, 'Users')
  const bytes = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' })
  saveBlob(new Blob([bytes], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }), filename)
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

	/** 获取认证计费页的统一监听配置。 */
	getBillingSettings(): Promise<BillingSettings> { return request.get('/settings/billing') },
	/** 统一保存并热应用 RADIUS 与 Portal 监听配置。 */
	updateBillingSettings(params: BillingSettings): Promise<BillingSettings> { return request.put('/settings/billing', params) },

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

  /** 获取导出数据并由浏览器生成 Excel 文件。 */
  async exportUsers(params: ExportUsersParams, headers: string[]): Promise<void> {
    // request 的响应拦截器已解包 API 信封，但其 Axios 类型声明保留了原始响应形状。
    const users = await request.post('/users/export', params) as unknown as AdminUserExportView[]
    const rows = users.map((user) => [
      user.username, user.nickname, user.phone, user.email, user.roleId,
      user.role, user.roleName, user.status, user.createdAt,
    ])
    saveWorkbook(rows, headers, `netlab-users-${new Date().toISOString().slice(0, 10)}.xlsx`, [20, 20, 16, 32, 12, 18, 20, 12, 24])
  },

  /** 在浏览器生成用户导入模板。 */
  downloadImportTemplate(headers: string[]): void {
    saveWorkbook([
      ['alice', 'Alice', '13800000001', 'alice@example.com', '', 'viewer', 'Vermilion-Otter-42'],
      ['bob', 'Bob', '13800000002', 'bob@example.com', '', 'viewer', 'Harbor-Piano-Sunset-9'],
    ], headers, 'netlab-users-template.xlsx', [20, 20, 16, 32, 12, 18, 24])
  },

  /** 在浏览器解析 xlsx/xls/csv，再提交后端约定的 JSON。 */
  async importUsers(file: File): Promise<ImportSummary> {
    const workbook = XLSX.read(await file.arrayBuffer(), { type: 'array' })
    const sheet = workbook.Sheets[workbook.SheetNames[0]]
    if (!sheet) throw new Error('empty import file')

    const rows = XLSX.utils.sheet_to_json<unknown[]>(sheet, { header: 1, defval: '', raw: false })
    if (rows.length < 2) throw new Error('import file has no data rows')
    const users: ImportUserParams[] = rows.slice(1).map((row) => ({
      username: cell(row, 0),
      nickname: cell(row, 1),
      phone: cell(row, 2),
      email: cell(row, 3),
      roleId: cell(row, 4),
      role: cell(row, 5),
      password: cell(row, 6),
    })).filter((user) => user.username || user.email || user.roleId || user.role || user.password)
    if (users.length === 0) throw new Error('import file has no data rows')
    return request.post('/users/import', { users })
  },
}

function cell(row: unknown[], index: number) {
  return index >= 0 ? String(row[index] ?? '').trim() : ''
}
