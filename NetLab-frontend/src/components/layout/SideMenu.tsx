import { useNavigate, useLocation } from 'react-router-dom'
import { Menu, theme } from 'antd'
import {
  DashboardOutlined,
  ExperimentOutlined,
  DesktopOutlined,
  CloudOutlined,
  SettingOutlined,
  QuestionCircleOutlined,
  MonitorOutlined,
  TeamOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import type { MenuProps } from 'antd'
import { useAuthStore } from '@/stores/authStore'

interface SideMenuProps {
  collapsed: boolean
}

export default function SideMenu({ collapsed }: SideMenuProps) {
  const { t } = useTranslation('menu')
  const navigate = useNavigate()
  const location = useLocation()
  const { token } = theme.useToken()
  const role = useAuthStore((s) => s.userInfo?.role)
  const isAdmin = role === 'admin' || role === 'super_admin'

  type MenuItem = Required<MenuProps>['items'][number]

  const menuItems: MenuItem[] = [
    {
      type: 'group',
      label: collapsed ? '' : t('workspace'),
      children: [
        {
          key: '/dashboard',
          icon: <DashboardOutlined />,
          label: t('dashboard'),
        },
        {
          key: '/labs',
          icon: <ExperimentOutlined />,
          label: t('myLabs'),
        },
        {
          key: '/monitor',
          icon: <MonitorOutlined />,
          label: t('runMonitor'),
        },
      ],
    },
    {
      type: 'group',
      label: collapsed ? '' : t('labs'),
      children: [
        {
          key: '/device-library',
          icon: <DesktopOutlined />,
          label: t('deviceLibrary'),
        },
        {
          key: '/templates',
          icon: <CloudOutlined />,
          label: t('templateMarket'),
        },
      ],
    },
    {
      type: 'divider',
    },
    // 系统设置仅对管理员可见。
    ...(isAdmin
      ? [
          {
            key: '/settings/users',
            icon: <TeamOutlined />,
            label: t('userManagement'),
          } as MenuItem,
          {
            key: '/settings',
            icon: <SettingOutlined />,
            label: t('settings'),
          } as MenuItem,
        ]
      : []),
    {
      key: '/help',
      icon: <QuestionCircleOutlined />,
      label: t('help'),
    },
  ]

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (key.startsWith('/')) navigate(key)
  }

  // 根据当前路径计算选中的 key
  const selectedKeys = [location.pathname]

  // 计算包含当前路径的分组的展开 key
  const allGroupKeys = menuItems
    .filter((item) => item && 'children' in item)
    .map((item) => item!.key as string)

  const openKeys = allGroupKeys.filter((groupKey) => {
    const group = menuItems.find((item) => item!.key === groupKey)
    if (group && 'children' in group) {
      return (group as { children: { key: string }[] }).children.some(
        (child) => location.pathname.startsWith(child.key)
      )
    }
    return false
  })

  return (
    <div style={{ padding: collapsed ? '12px 8px' : '12px 12px' }}>
      <Menu
        mode="inline"
        selectedKeys={selectedKeys}
        defaultOpenKeys={openKeys}
        inlineCollapsed={collapsed}
        items={menuItems}
        onClick={handleMenuClick}
        style={{
          borderInlineEnd: 'none',
          background: token.colorBgContainer,
        }}
      />
    </div>
  )
}
