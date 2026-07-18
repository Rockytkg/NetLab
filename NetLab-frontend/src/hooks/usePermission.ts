import { useMemo } from 'react'
import { useAuthStore } from '@/stores/authStore'

export type Permission = string

/** 模块级稳定空数组引用 —— 避免 Zustand v5 每次 select 返回新引用触发无限重渲染。 */
const EMPTY_PERMISSIONS: string[] = []

export function usePermission() {
  const permissions = useAuthStore((s) => s.userInfo?.permissions ?? EMPTY_PERMISSIONS)
  const permissionSet = useMemo(() => new Set(permissions), [permissions])

  return {
    can: (permission: string) => permissionSet.has(permission),
  }
}
