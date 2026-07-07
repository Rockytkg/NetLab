import { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import type { LoginParams, LoginResult } from '@/types/auth'

export function useAuth() {
  const navigate = useNavigate()
  const {
    userInfo,
    isAuthenticated,
    login: storeLogin,
    logout: storeLogout,
  } = useAuthStore()

  /** Login — returns full result so callers can handle 2FA, toast, navigation */
  const login = useCallback(
    async (params: LoginParams): Promise<LoginResult> => {
      return await storeLogin(params)
    },
    [storeLogin]
  )

  const logout = useCallback(async () => {
    await storeLogout()
    navigate('/login', { replace: true })
  }, [storeLogout, navigate])

  return {
    userInfo,
    isAuthenticated: isAuthenticated(),
    login,
    logout,
  }
}
