import { Suspense, lazy } from 'react'
import {
  createBrowserRouter,
  Navigate,
  RouterProvider,
} from 'react-router-dom'
import AuthGuard from '@/components/auth/AuthGuard'
import MainLayout from '@/components/layout/MainLayout'
import Loading from '@/components/common/Loading'

// ── 懒加载页面 ──
const LoginPage = lazy(() => import('@/pages/login'))
const OAuthCallbackPage = lazy(() => import('@/pages/login/OAuthCallbackPage'))
const SecurityRequiredPage = lazy(() => import('@/pages/account/SecurityRequiredPage'))
const TwoFactorSetupPage = lazy(() => import('@/pages/account/TwoFactorSetupPage'))
const TwoFactorVerifyPage = lazy(() => import('@/pages/login/TwoFactorVerifyPage'))
const ForbiddenPage = lazy(() => import('@/pages/error/403'))
const NotFoundPage = lazy(() => import('@/pages/error/404'))
const DashboardPage = lazy(() => import('@/pages/dashboard'))
const SettingsLayout = lazy(() => import('@/pages/settings'))
const BeianPanel = lazy(() => import('@/pages/settings/panels/BeianPanel'))
const SecurityPanel = lazy(() => import('@/pages/settings/panels/SecurityPanel'))
const SMTPPanel = lazy(() => import('@/pages/settings/panels/SMTPPanel'))
const OAuthPanel = lazy(() => import('@/pages/settings/panels/OAuthPanel'))
const SettingsProfilePage = lazy(() => import('@/pages/settings/profile'))
const UsersPage = lazy(() => import('@/pages/settings/users'))
const RolesPage = lazy(() => import('@/pages/settings/roles'))
const LoginLogsPage = lazy(() => import('@/pages/settings/login-logs'))

const router = createBrowserRouter([
  // ── 公开路由 ──
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/login/2fa',
    element: <TwoFactorVerifyPage />,
  },
  {
    path: '/oauth/callback',
    element: <OAuthCallbackPage />,
  },
  {
    path: '/403',
    element: <ForbiddenPage />,
  },
  {
    path: '/account/security-required',
    element: (
      <AuthGuard>
        <SecurityRequiredPage />
      </AuthGuard>
    ),
  },
  {
    path: '/account/2fa-setup',
    element: (
      <AuthGuard>
        <TwoFactorSetupPage />
      </AuthGuard>
    ),
  },
  {
    path: '*',
    element: <NotFoundPage />,
  },
  // 根路由重定向
  {
    path: '/',
    element: <Navigate to="/dashboard" replace />,
  },

  // ── 受保护路由 (AuthGuard + MainLayout) ──
  {
    element: (
      <AuthGuard>
        <MainLayout />
      </AuthGuard>
    ),
    children: [
      // 运维工作台
      { path: '/dashboard', element: <DashboardPage /> },

      // 设置（左侧菜单 + 子路由）
      {
        path: '/settings',
        element: <SettingsLayout />,
        children: [
          { index: true, element: <Navigate to="beian" replace /> },
          { path: 'beian', element: <BeianPanel /> },
          { path: 'security', element: <SecurityPanel /> },
          { path: 'smtp', element: <SMTPPanel /> },
          { path: 'oauth', element: <OAuthPanel /> },
        ],
      },
      { path: '/settings/profile', element: <SettingsProfilePage /> },
      { path: '/settings/users', element: <UsersPage /> },
      { path: '/settings/roles', element: <RolesPage /> },
      { path: '/settings/login-logs', element: <LoginLogsPage /> },
    ],
  },
])

export default function AppRouter() {
  return (
    <Suspense fallback={<Loading />}>
      <RouterProvider router={router} />
    </Suspense>
  )
}
