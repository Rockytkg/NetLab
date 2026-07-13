import { Navigate, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuthStore } from '@/stores/authStore'
import Loading from '@/components/common/Loading'

interface AuthGuardProps {
  children: ReactNode
}

export default function AuthGuard({ children }: AuthGuardProps) {
  const location = useLocation()
  const token = useAuthStore((s) => s.accessToken)
  const securityActions = useAuthStore((s) => s.securityActions)
  const loading = useAuthStore((s) => s.loading)

  if (loading) {
    return <Loading tip="Verifying..." />
  }

  if (!token) {
    return (
      <Navigate
        to={`/login?redirect=${encodeURIComponent(location.pathname)}`}
        replace
      />
    )
  }

  if (
    location.pathname !== '/account/security-required' &&
    (securityActions?.requirePasswordChange || securityActions?.requireEmailChange)
  ) {
    return <Navigate to="/account/security-required" replace />
  }

  if (
    location.pathname !== '/account/2fa-setup' &&
    securityActions?.requireTwoFactorSetup
  ) {
    return <Navigate to="/account/2fa-setup" replace />
  }

  return <>{children}</>
}
