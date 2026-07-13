import { useTranslation } from 'react-i18next'
import {
  AutoComplete,
  Avatar,
  Badge,
  Button,
  Dropdown,
  Input,
  Space,
  theme,
  Typography,
} from 'antd'
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
  LogoutOutlined,
  SettingOutlined,
  GlobalOutlined,
  BellOutlined,
  SearchOutlined,
  BulbOutlined,
  CheckOutlined,
  SunOutlined,
  MoonOutlined,
  DesktopOutlined,
} from '@ant-design/icons'
import { useAppStore } from '@/stores/appStore'
import type { ThemeMode } from '@/stores/appStore'
import { useAuthStore } from '@/stores/authStore'
import { useI18n } from '@/hooks/useI18n'
import { LOCALE_OPTIONS } from '@/types/i18n'
import type { SupportedLocale } from '@/types/i18n'
import { useNavigate, useLocation } from 'react-router-dom'
import { useMemo } from 'react'

interface HeaderBarProps {
  onOpenMobileMenu: () => void
}

/** 路由 → i18n 页面标题映射 */
const PAGE_TITLES: Record<string, { titleNs: string; titleKey: string }> = {
  '/dashboard': { titleNs: 'menu', titleKey: 'dashboard' },
  '/labs': { titleNs: 'menu', titleKey: 'myLabs' },
  '/lab': { titleNs: 'topology', titleKey: 'title' },
  '/device-library': { titleNs: 'menu', titleKey: 'deviceLibrary' },
  '/templates': { titleNs: 'menu', titleKey: 'templateMarket' },
  '/settings/users': { titleNs: 'menu', titleKey: 'userManagement' },
  '/settings/profile': { titleNs: 'menu', titleKey: 'profile' },
  '/settings': { titleNs: 'menu', titleKey: 'settings' },
  '/help': { titleNs: 'menu', titleKey: 'help' },
  '/monitor': { titleNs: 'menu', titleKey: 'runMonitor' },
}

export default function HeaderBar({ onOpenMobileMenu }: HeaderBarProps) {
  const { t } = useTranslation(['common', 'menu', 'topology'])
  const { switchLanguage } = useI18n()
  const sidebarCollapsed = useAppStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useAppStore((s) => s.toggleSidebar)
  const locale = useAppStore((s) => s.locale)
  const userInfo = useAuthStore((s) => s.userInfo)
  const logout = useAuthStore((s) => s.logout)
  const themeMode = useAppStore((s) => s.themeMode)
  const setThemeMode = useAppStore((s) => s.setThemeMode)
  const navigate = useNavigate()
  const location = useLocation()
  const { token } = theme.useToken()

  // 根据当前路由派生出的页面标题
  const pageTitle = useMemo(() => {
    const match = Object.entries(PAGE_TITLES).find(([path]) =>
      location.pathname.startsWith(path)
    )
    if (match) {
      const { titleNs, titleKey } = match[1]
      return t(`${titleNs}:${titleKey}` as never)
    }
    return t('menu:home')
  }, [location.pathname, t])

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  const userMenuItems = {
    items: [
      {
        key: 'profile',
        icon: <UserOutlined />,
        label: t('common:profile'),
      },
      {
        key: 'settings',
        icon: <SettingOutlined />,
        label: t('common:settings'),
      },
      {
        key: 'language',
        icon: <GlobalOutlined />,
        label: t('common:language'),
        children: LOCALE_OPTIONS.map((opt) => ({
          key: `lang-${opt.value}`,
          label: opt.label,
          extra: locale === opt.value ? <CheckOutlined /> : undefined,
        })),
      },
      {
        key: 'theme',
        icon: <BulbOutlined />,
        label: t('common:themeMode'),
        children: [
          {
            key: 'theme-light',
            label: t('common:themeLight'),
            icon: <SunOutlined />,
            extra: themeMode === 'light' ? <CheckOutlined /> : undefined,
          },
          {
            key: 'theme-dark',
            label: t('common:themeDark'),
            icon: <MoonOutlined />,
            extra: themeMode === 'dark' ? <CheckOutlined /> : undefined,
          },
          {
            key: 'theme-system',
            label: t('common:themeSystem'),
            icon: <DesktopOutlined />,
            extra: themeMode === 'system' ? <CheckOutlined /> : undefined,
          },
        ],
      },
      { type: 'divider' as const },
      {
        key: 'logout',
        icon: <LogoutOutlined />,
        label: t('common:logout'),
        danger: true,
      },
    ],
    onClick: ({ key }: { key: string }) => {
      if (key === 'logout') handleLogout()
      else if (key === 'profile') navigate('/settings/profile')
      else if (key === 'settings') navigate('/settings')
      else if (key.startsWith('lang-')) {
        const lang = key.replace('lang-', '') as SupportedLocale
        switchLanguage(lang)
      }
      else if (key.startsWith('theme-')) {
        const mode = key.replace('theme-', '') as ThemeMode
        setThemeMode(mode)
      }
    },
  }

  // 全局搜索选项（占位 —— 将在第三阶段接入真实数据）
  const searchOptions = useMemo(
    () => [
      { value: 'search-labs', label: t('menu:myLabs'), category: t('menu:labs') },
      { value: 'search-devices', label: t('menu:deviceLibrary'), category: t('menu:labs') },
      { value: 'search-templates', label: t('menu:templateMarket'), category: t('menu:templateMarket') },
    ],
    [t]
  )

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '0 24px',
        height: 64,
        flexShrink: 0,
        background: token.colorBgContainer,
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
        position: 'sticky',
        top: 0,
        zIndex: 10,
      }}
    >
      {/* ── 左侧：折叠按钮 + 页面标题 ── */}
      <Space size={16}>
        <Button
          className="netlab-mobile-only netlab-icon-button"
          type="text"
          icon={<MenuUnfoldOutlined />}
          onClick={onOpenMobileMenu}
          aria-label={t('common:openNavigation')}
        />
        <Button
          className="netlab-icon-button netlab-desktop-only"
          type="text"
          onClick={toggleSidebar}
          icon={sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          aria-label={t('common:toggleSidebar')}
        />
        <div>
          <Typography.Text strong style={{ display: 'block', fontSize: 16 }}>
            {pageTitle}
          </Typography.Text>
        </div>
      </Space>

      {/* ── 右侧：搜索 + 通知 + 用户 ── */}
      <Space size="middle">
        <AutoComplete
          className="netlab-header-search"
          style={{ width: 240 }}
          options={searchOptions}
          placeholder={t('common:search')}
          onSelect={(value) => {
            if (value === 'search-labs') navigate('/labs')
            else if (value === 'search-devices') navigate('/device-library')
            else if (value === 'search-templates') navigate('/templates')
          }}
        >
          <Input prefix={<SearchOutlined />} allowClear />
        </AutoComplete>

        <Badge dot offset={[-4, 4]}>
          <Button
            className="netlab-icon-button"
            type="text"
            icon={<BellOutlined />}
            aria-label={t('common:notifications')}
          />
        </Badge>

        <Dropdown menu={userMenuItems} placement="bottomRight">
          <Space style={{ cursor: 'pointer', minWidth: 40 }}>
            <Avatar
              size="small"
              icon={<UserOutlined />}
              src={userInfo?.avatar}
            />
            <span className="netlab-desktop-only">
              {userInfo?.username || t('common:guestUser')}
            </span>
          </Space>
        </Dropdown>
      </Space>
    </div>
  )
}
