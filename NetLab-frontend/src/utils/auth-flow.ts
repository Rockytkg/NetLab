import type { NavigateFunction } from 'react-router-dom'
import type { LoginResult } from '@/types/auth'
import { useAuthStore } from '@/stores/authStore'

export function hasRequiredSecurityAction(result: LoginResult): boolean {
  return !!(
    result.securityActions?.requirePasswordChange ||
    result.securityActions?.requireEmailChange ||
    result.securityActions?.requireTwoFactorSetup
  )
}

export function persistLoginResult(result: LoginResult): boolean {
  if (!result.accessToken || !result.refreshToken || !result.user) return false
  useAuthStore.setState({
    accessToken: result.accessToken,
    refreshToken: result.refreshToken,
    userInfo: result.user,
    securityActions: result.securityActions,
  })
  return true
}

export function navigateAfterLogin(result: LoginResult, navigate: NavigateFunction) {
  if (result.securityActions?.requirePasswordChange || result.securityActions?.requireEmailChange) {
    navigate('/account/security-required', { replace: true })
    return
  }
  if (result.securityActions?.requireTwoFactorSetup) {
    navigate('/account/2fa-setup', { replace: true })
    return
  }
  navigate('/dashboard', { replace: true })
}

export function completeLogin(result: LoginResult, navigate: NavigateFunction): boolean {
  if (!persistLoginResult(result)) return false
  navigateAfterLogin(result, navigate)
  return true
}
