/**
 * 系统设置（管理端）相关类型定义。
 *
 * 密钥字段（SMTP 密码、OAuth Client Secret）在后端以 AES-GCM 加密存储，
 * 通过接口返回时被掩码为占位符 `__UNCHANGED__`。前端提交更新时，若这些
 * 字段仍为掩码或留空，后端会保留原有密钥。
 */

/** 密钥掩码占位符，必须与后端 sysconfig.SecretMask 保持一致。 */
export const SECRET_MASK = '__UNCHANGED__'

/** 安全策略设置 */
export interface SecuritySettings {
  registrationEnabled: boolean
  captchaEnabled: boolean
  passkeyEnabled: boolean
  passwordResetEnabled: boolean
  twoFactorRequired: boolean
  passwordMaxAgeDays: number
}

/** 备案信息设置 */
export interface BeianSettings {
  icpBeian: string
  policeBeian: string
}

/** SMTP 邮件服务设置（密码字段可能为掩码） */
export interface SMTPSettings {
  enabled: boolean
  host: string
  port: number
	username: string
	nickname: string
	phone: string
  password: string
  from: string
  useTls: boolean
}

/** 单个 OAuth 提供商设置（clientSecret 可能为掩码） */
export interface OAuthProviderSettings {
  id: string
  name: string
  enabled: boolean
  clientId: string
  clientSecret: string
  redirectUrl: string
  configured: boolean
}

/** 完整系统设置快照 */
export interface AdminSettings {
  security: SecuritySettings
  beian: BeianSettings
  smtp: SMTPSettings
  oauth: OAuthProviderSettings[]
}

/** ── Passkey 管理 ── */

/** 已注册的 passkey 元数据 */
export interface PasskeyInfo {
  id: string
  name: string
  createdAt: string
  lastUsedAt?: string | null
}

/** ── 用户管理（管理端） ── */

/** 管理端用户视图 */
export interface AdminUserView {
  id: string
  username: string
  nickname: string
  phone: string
  email: string
  role: string
  roleName: string
  status: string
	 twoFactorEnabled: boolean
	createdAt: string
}

/** 分页用户列表 */
export interface UserListResult {
  items: AdminUserView[]
  total: number
  page: number
  size: number
}

/** 用户列表筛选条件 */
export interface UserListParams {
  page?: number
  size?: number
  keyword?: string
  status?: string
  role?: string
}

/** 单用户可编辑字段 */
export interface UpdateUserParams {
	nickname: string
	phone: string
	email: string
  role: string
  status: string
	disableTwoFactor?: boolean
}

/** 系统设置中管理的 RADIUS 基础监听配置。 */
export interface RadiusListenerSettings {
  enabled: boolean
  bindHost: string
  authPort: number
  acctPort: number
}

/** 认证计费页统一保存的 RADIUS 与 Portal 监听配置。 */
export interface BillingSettings {
	radius: RadiusListenerSettings
	portal: { enabled: boolean; notifyPort: number }
}

/** 单用户创建字段 */
export interface CreateUserParams {
	username: string
	nickname: string
	phone: string
  email: string
  role: string
  password: string
}

/** 批量导出用户参数（仅按勾选的用户 ID 导出） */
export interface ExportUsersParams {
  userIds: string[]
}

/** 后端返回的用户导出数据，表格文件由前端生成。 */
export interface AdminUserExportView {
  username: string
  nickname: string
  phone: string
  email: string
  roleId: string
  role: string
  roleName: string
  status: string
  createdAt: string
}

/** 批量导入用户记录 */
export interface ImportUserParams {
	username: string
	nickname: string
	phone: string
  email: string
	roleId: string
	role: string
  password: string
}

/** 导入结果汇总 */
export interface ImportSummary {
  created: number
  skipped: number
  errors: string[]
}

/** ── RBAC 权限管理 ── */

/** 通用消息响应 */
export interface MessageResponse {
  message: string
}

/** 角色视图 */
export interface RoleView {
  id: string
  role: string
  roleName: string
  description?: string
  type: 'builtin' | 'custom'
  managementLevel: number
  hidden: boolean
	permissions?: PermissionRef[]
  createdAt: string
  updatedAt: string
}

/** 权限简洁引用 */
export interface PermissionRef {
  code: string
  resource: string
  action: string
}

/** 权限详细视图 */
export interface PermissionView {
  code: string
  resource: string
  action: string
  description?: string
  createdAt: string
}
