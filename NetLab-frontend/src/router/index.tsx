import { Suspense, lazy } from 'react'
import {
  createBrowserRouter,
  Navigate,
  RouterProvider,
  useParams,
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
const DeviceGroupsPage = lazy(() => import('@/pages/device-groups'))
const DeviceTopologyPage = lazy(() => import('@/pages/devices/topology'))
const ObservabilityPage = lazy(() => import('@/pages/observability'))
const DeviceLibraryPage = lazy(() => import('@/pages/device-library'))
const OperationsTemplatesPage = lazy(() => import('@/pages/operations-templates'))
const OperationsTemplateUploadPage = lazy(() => import('@/pages/operations-templates/upload'))
const InstalledOperationsTemplatesPage = lazy(() => import('@/pages/operations-templates/installed'))
const SettingsPage = lazy(() => import('@/pages/settings'))
const SettingsProfilePage = lazy(() => import('@/pages/settings/profile'))
const UsersPage = lazy(() => import('@/pages/settings/users'))
const HelpPage = lazy(() => import('@/pages/help'))

function LegacyDeviceTopologyRedirect() {
  const { deviceId } = useParams()
  return <Navigate to={`/devices/${deviceId ?? ''}/topology`} replace />
}

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
      // 工作台
      // 运维工作台
      { path: '/dashboard', element: <DashboardPage /> },
      { path: '/device-groups', element: <DeviceGroupsPage /> },
      { path: '/device-library', element: <DeviceLibraryPage /> },
      { path: '/devices/:deviceId/topology', element: <DeviceTopologyPage /> },
      { path: '/observability', element: <ObservabilityPage /> },

      // 运维模板
      { path: '/operations-templates', element: <OperationsTemplatesPage /> },
      { path: '/operations-templates/upload', element: <OperationsTemplateUploadPage /> },
      { path: '/operations-templates/installed', element: <InstalledOperationsTemplatesPage /> },

      // 旧信息架构兼容重定向
      { path: '/labs', element: <Navigate to="/device-groups" replace /> },
      { path: '/lab/:deviceId', element: <LegacyDeviceTopologyRedirect /> },
      { path: '/lab/:deviceId/monitor', element: <Navigate to="/observability" replace /> },
      { path: '/monitor', element: <Navigate to="/observability" replace /> },
      { path: '/templates', element: <Navigate to="/operations-templates" replace /> },
      { path: '/templates/upload', element: <Navigate to="/operations-templates/upload" replace /> },
      { path: '/templates/installed', element: <Navigate to="/operations-templates/installed" replace /> },

      // 设置
      { path: '/settings', element: <SettingsPage /> },
      { path: '/settings/profile', element: <SettingsProfilePage /> },
      { path: '/settings/users', element: <UsersPage /> },

      // 帮助
      { path: '/help', element: <HelpPage /> },
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
