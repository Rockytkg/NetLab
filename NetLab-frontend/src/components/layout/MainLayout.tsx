import { Suspense, useState } from 'react'
import { Outlet } from 'react-router-dom'
import { Drawer, Grid, Layout, theme, Typography } from 'antd'
import { useAppStore } from '@/stores/appStore'
import { useTranslation } from 'react-i18next'
import SideMenu from './SideMenu'
import HeaderBar from './HeaderBar'
import Loading from '@/components/common/Loading'
import { useResolvedTheme } from '@/hooks/useResolvedTheme'

const { Sider, Content, Footer } = Layout
const { useBreakpoint } = Grid

export default function MainLayout() {
  const collapsed = useAppStore((s) => s.sidebarCollapsed)
  const themeMode = useAppStore((s) => s.themeMode)
  const resolvedTheme = useResolvedTheme(themeMode)
  const { token } = theme.useToken()
  const screens = useBreakpoint()
  const { t } = useTranslation(['common', 'menu'])
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  // 响应式断点标志
  const isDesktop = !!screens.lg // ≥ 992px
  const isTablet = !!screens.md  // ≥ 768px
  const isDark = resolvedTheme === 'dark'

  // 内容内边距 —— 响应式
  const contentPadding = screens.xxl ? 32 : screens.xl ? 24 : 16

  // 共享的品牌标识区块（桌面端 Sider 和移动端 Drawer 均使用）
  const brandingBlock = (
    <div
      style={{
        height: 64,
        display: 'flex',
        alignItems: 'center',
        justifyContent: collapsed && isDesktop ? 'center' : 'flex-start',
        gap: 12,
        paddingInline: collapsed && isDesktop ? 0 : 20,
        borderBlockEnd: `1px solid ${token.colorBorderSecondary}`,
        transition: 'padding 0.2s ease',
      }}
    >
      <img
        src="/logo-mark.svg"
        alt=""
        aria-hidden
        width={32}
        height={32}
        style={{
          display: 'block',
          flexShrink: 0,
          borderRadius: token.borderRadius,
        }}
      />

      {/* 品牌文字 —— 桌面端折叠时隐藏 */}
      {(!collapsed || !isDesktop) && (
        <div style={{ minWidth: 0 }}>
          <Typography.Text
            strong
            style={{
              fontSize: 16,
              lineHeight: '24px',
              display: 'block',
              color: token.colorText,
            }}
          >
            {t('common:appName')}
          </Typography.Text>
          <Typography.Text
            type="secondary"
            style={{ fontSize: 12, lineHeight: '20px' }}
          >
            {t('common:appSubtitle')}
          </Typography.Text>
        </div>
      )}
    </div>
  )

  const siderContent = (
    <>
      {brandingBlock}
      <SideMenu collapsed={Boolean(collapsed && isDesktop)} />
    </>
  )

  return (
    <Layout hasSider={isDesktop} style={{ minHeight: '100vh' }}>
      {/* ── 桌面端侧边栏 ── */}
      {isDesktop && (
        <Sider
          className="netlab-shell-sider"
          trigger={null}
          collapsible
          collapsed={collapsed}
          width={232}
          collapsedWidth={80}
          breakpoint="lg"
          theme={isDark ? 'dark' : 'light'}
          style={{
            background: token.colorBgContainer,
            borderInlineEnd: `1px solid ${token.colorBorderSecondary}`,
            position: 'sticky',
            top: 0,
            height: '100vh',
            zIndex: 11,
            overflow: 'auto',
          }}
        >
          {siderContent}
        </Sider>
      )}

      {/* ── 主区域 —— 固定视口高度，仅内容区滚动 ── */}
      <Layout style={{ height: '100vh' }}>
        {/* 头部 */}
        <HeaderBar onOpenMobileMenu={() => setMobileMenuOpen(true)} />

        {/* 内容 */}
        <Content
          className="netlab-main-content"
          style={{
            padding: contentPadding,
            minHeight: 280,
            overflow: 'auto',
            background: token.colorBgLayout,
          }}
        >
          <Suspense fallback={<Loading fullScreen={false} />}>
            <Outlet />
          </Suspense>
        </Content>

        {/* 底部 —— 状态栏 */}
        {isTablet && (
          <Footer className="netlab-layout-footer">
            <span>NetLab v0.1.0</span>
            <span>{t('menu:dashboard')}</span>
          </Footer>
        )}
      </Layout>

      {/* ── 移动端抽屉 ── */}
      <Drawer
        title={t('common:appName')}
        placement="left"
        size={280}
        open={!isDesktop && mobileMenuOpen}
        onClose={() => setMobileMenuOpen(false)}
        styles={{ body: { padding: 0 } }}
      >
        {siderContent}
      </Drawer>
    </Layout>
  )
}
