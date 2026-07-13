import request from './request'
import type {
  AdminSettings,
  SecuritySettings,
  BeianSettings,
  SMTPSettings,
  OAuthProviderSettings,
  AdminUserView,
  UserListResult,
  ImportSummary,
  UserListParams,
  UpdateUserParams,
  CreateUserParams,
} from '@/types/settings'

/**
 * 系统设置管理接口（仅 admin）。
 *
 * 所有写操作在后端热生效（无需重启）。密钥字段（SMTP 密码、OAuth
 * Client Secret）在返回时被掩码；提交时若仍为掩码或留空，则保留原值。
 */
export const adminApi = {
  /** 获取完整系统设置（密钥已掩码） */
  getSettings(): Promise<AdminSettings> {
    return request.get('/admin/settings')
  },

  /** 更新安全策略 */
  updateSecurity(params: SecuritySettings): Promise<{ message: string }> {
    return request.put('/admin/settings/security', params)
  },

  /** 更新备案信息 */
  updateBeian(params: BeianSettings): Promise<{ message: string }> {
    return request.put('/admin/settings/beian', params)
  },

  /** 更新 SMTP 邮件服务配置 */
  updateSMTP(params: SMTPSettings): Promise<{ message: string }> {
    return request.put('/admin/settings/smtp', params)
  },

  /** 使用当前 SMTP 配置发送测试邮件 */
  testSMTP(to: string): Promise<{ message: string }> {
    return request.post('/admin/settings/smtp/test', { to })
  },

  /** 更新指定 OAuth 提供商配置 */
  updateOAuthProvider(
    provider: string,
    params: Pick<OAuthProviderSettings, 'enabled' | 'clientId' | 'clientSecret' | 'redirectUrl'>,
  ): Promise<{ message: string }> {
    return request.put(`/admin/settings/oauth/${provider}`, params)
  },

  /** ── 用户管理 ── */

  /** 分页获取用户列表 */
  listUsers(params: UserListParams): Promise<UserListResult> {
    return request.get('/admin/users', { params })
  },

  /** 更新单个用户邮箱、角色和状态 */
  updateUser(id: string, params: UpdateUserParams): Promise<{ message: string }> {
    return request.put(`/admin/users/${id}`, params)
  },

  /** 创建单个用户 */
  createUser(params: CreateUserParams): Promise<AdminUserView> {
    return request.post('/admin/users', params)
  },

  /** 批量修改用户角色 */
  batchUpdateRole(userIds: string[], role: string): Promise<{ message: string }> {
    return request.put('/admin/users/role', { userIds, role })
  },

  /** 批量删除用户 */
  batchDeleteUsers(userIds: string[]): Promise<{ message: string }> {
    return request.delete('/admin/users', { data: { userIds } })
  },

  /** 批量重置密码为统一新密码 */
  batchResetPassword(userIds: string[], newPassword: string): Promise<{ message: string }> {
    return request.put('/admin/users/reset-password', { userIds, newPassword })
  },

  /** 通过 CSV 文件批量导入用户 */
  importUsers(file: File): Promise<ImportSummary> {
    const formData = new FormData()
    formData.append('file', file)
    // 不手动设置 Content-Type：axios 会为 FormData 自动附加带 boundary 的
    // multipart/form-data 头。
    return request.post('/admin/users/import', formData)
  },
}
