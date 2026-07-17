/**
 * auth-normalize.ts — UserInfo 边界归一化工具。
 *
 * 确保所有写入 authStore 的 userInfo 具有稳定的 permissions 字段引用，
 * 防止 Zustand v5 + React 19 因 selector 返回新数组引用导致无限重渲染。
 *
 * 使用方式：authStore.setUserInfo(normalizeUserInfo(raw))
 */

import type { UserInfo } from '@/types/auth'

/**
 * 归一化 UserInfo，确保 permissions 字段始终为稳定数组引用。
 * - null / undefined → null
 * - permissions 缺失 / null / 非数组 → 补为 []
 * - 不修改入参对象，返回新对象
 * - 其他字段原样透传
 */
export function normalizeUserInfo(
  user: UserInfo | null | undefined,
): UserInfo | null {
  if (!user) return null

  return {
    ...user,
    permissions: Array.isArray(user.permissions) ? user.permissions : [],
  }
}
