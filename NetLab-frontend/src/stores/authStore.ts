import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { UserInfo, LoginParams, LoginResult, SecurityActions } from '@/types/auth'
import { authApi } from '@/services/auth'

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  userInfo: UserInfo | null
  securityActions: SecurityActions | null
  loading: boolean

  /** 登录 —— 返回完整结果。调用方在存储 token 前需检查 requiresTwoFactor。 */
  login: (params: LoginParams) => Promise<LoginResult>
  /**
   * 登出。默认会在清除本地会话前尽力执行一次服务端登出
   * （token 黑名单）。对于 token 已失效的强制/过期会话场景，
   * 可传入 `{ callApi: false }`。
   */
  logout: (options?: { callApi?: boolean }) => Promise<void>
  refreshAccessToken: () => Promise<string | null>
  fetchUserInfo: () => Promise<void>
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,
      userInfo: null,
      securityActions: null,
      loading: false,

      login: async (params: LoginParams) => {
        const result = await authApi.login(params)
        if (!result.accessToken || !result.refreshToken || !result.user) return result
        set({
          accessToken: result.accessToken,
          refreshToken: result.refreshToken,
          userInfo: result.user,
          securityActions: result.securityActions,
        })
        return result
      },

      logout: async (options?: { callApi?: boolean }) => {
        const callApi = options?.callApi !== false
        // 尽力执行服务端登出（将当前 token 加入黑名单）。
        // 必须在清除本地 token 之前运行，因为该接口需要有效的 JWT。
        // 在强制/过期路径（callApi === false）下跳过。
        // 失败会被忽略 —— 无论如何都会清除本地会话。
        if (callApi && get().accessToken) {
          try {
            await authApi.logout()
          } catch {
            /* 忽略 —— 继续清除本地会话 */
          }
        }
        set({
          accessToken: null,
          refreshToken: null,
          userInfo: null,
          securityActions: null,
        })
      },

      refreshAccessToken: async () => {
        const { refreshToken } = get()
        if (!refreshToken) return null

        try {
          const result = await authApi.refreshToken(refreshToken)
          set({
            accessToken: result.accessToken,
            refreshToken: result.refreshToken,
          })
          return result.accessToken
        } catch {
          // 刷新失败 —— 强制登出（token 已失效，跳过 API 调用）
          get().logout({ callApi: false })
          return null
        }
      },

      fetchUserInfo: async () => {
        set({ loading: true })
        try {
          const user = await authApi.getUserInfo()
          set({ userInfo: user, loading: false })
        } catch {
          set({ loading: false })
          throw new Error('Failed to fetch user information')
        }
      },

      isAuthenticated: () => {
        return !!get().accessToken
      },
    }),
    {
      name: 'netlab-auth',
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        userInfo: state.userInfo,
        securityActions: state.securityActions,
      }),
    }
  )
)
