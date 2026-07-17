import request from './request'
import type {
  RoleView,
  PermissionView,
  PermissionRef,
  MessageResponse,
} from '@/types/settings'

/**
 * RBAC 权限管理接口，访问控制由 rbac:read / rbac:write 决定。
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
  createRole(name: string, description?: string): Promise<RoleView> {
    return request.post('/rbac/roles', { name, description })
  },

  /** 更新角色名称和描述 */
  updateRole(id: string, name: string, description?: string): Promise<MessageResponse> {
    return request.put(`/rbac/roles/${id}`, { name, description })
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
  setRolePermissions(id: string, permissionIds: string[]): Promise<MessageResponse> {
    return request.put(`/rbac/roles/${id}/permissions`, { permissionIds })
  },

  /** 获取所有可用权限目录 */
  listPermissions(): Promise<PermissionView[]> {
    return request.get('/rbac/permissions')
  },
}
