import request from './request'
import type {
  RoleView,
  PermissionView,
  PermissionRef,
  MessageResponse,
} from '@/types/settings'

/**
 * 角色管理接口，权限编码使用稳定的 resource.action 格式。
 */
export const rbacApi = {
  /** 获取所有角色列表 */
  listRoles(): Promise<RoleView[]> {
    return request.get('/rbac/roles')
  },

  /** 获取单个角色详情（含权限列表） */
  getRole(id: string): Promise<RoleView> {
    return request.get(`/rbac/roles/${id}`)
  },

  /** 创建自定义角色 */
  createRole(role: string, roleName: string, description?: string, permissions: string[] = []): Promise<RoleView> {
    return request.post('/rbac/roles', { role, roleName, description, permissions })
  },

  /** 更新角色名称和描述 */
  updateRole(id: string, roleName: string, description?: string): Promise<MessageResponse> {
    return request.put(`/rbac/roles/${id}`, { roleName, description })
  },

  /** 删除角色 */
  deleteRole(id: string): Promise<MessageResponse> {
    return request.delete(`/rbac/roles/${id}`)
  },

  /** 获取角色的权限 ID 列表 */
  getRolePermissions(id: string): Promise<PermissionRef[]> {
    return request.get(`/rbac/roles/${id}/permissions`)
  },

  /** 覆盖设置角色的权限集 */
  setRolePermissions(id: string, permissions: string[]): Promise<MessageResponse> {
    return request.put(`/rbac/roles/${id}/permissions`, { permissions })
  },

  /** 获取所有可用权限目录 */
  listPermissions(): Promise<PermissionView[]> {
    return request.get('/rbac/permissions')
  },
}
