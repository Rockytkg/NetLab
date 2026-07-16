import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Menu, theme } from 'antd'
import {
  AppstoreOutlined,
  DashboardOutlined,
  DesktopOutlined,
  CloudOutlined,
  CloudDownloadOutlined,
  ControlOutlined,
  SettingOutlined,
  QuestionCircleOutlined,
  RadarChartOutlined,
  TeamOutlined,
  ClusterOutlined,
  DatabaseOutlined,
  UploadOutlined,
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
  const rootSubmenuKeys = ['workspace', 'infrastructure', 'administration']

  const menuItems = useMemo<MenuItem[]>(
    () => [
      {
        key: 'workspace',
        icon: <DashboardOutlined />,
        label: t('workspace'),
        children: [
          {
            key: '/dashboard',
            icon: <DashboardOutlined />,
            label: t('dashboard'),
          },
          {
            key: '/device-groups',
            icon: <ClusterOutlined />,
            label: t('deviceGroups'),
          },
          {
            key: '/observability',
            icon: <RadarChartOutlined />,
            label: t('observability'),
          },
        ],
      },
      {
        key: 'infrastructure',
        icon: <DatabaseOutlined />,
        label: t('infrastructure'),
        children: [
          {
            key: '/device-library',
            icon: <DatabaseOutlined />,
            label: t('deviceLibrary'),
          },
          {
            key: 'operations-templates',
            icon: <CloudOutlined />,
            label: t('templateMarket'),
            children: [
              {
                key: '/operations-templates',
                icon: <AppstoreOutlined />,
                label: t('browseTemplates'),
              },
              {
                key: '/operations-templates/upload',
                icon: <UploadOutlined />,
                label: t('myUploads'),
              },
              {
                key: '/operations-templates/installed',
                icon: <CloudDownloadOutlined />,
                label: t('installed'),
              },
            ],
          },
          {
            key: '/devices/demo/topology',
            icon: <DesktopOutlined />,
            label: t('deviceTopology'),
          },
        ],
      },
      ...(isAdmin
        ? [
            {
              key: 'administration',
              icon: <SettingOutlined />,
              label: t('administration'),
              children: [
                {
                  key: '/settings',
                  icon: <ControlOutlined />,
                  label: t('settings'),
                },
                {
                  key: '/settings/users',
                  icon: <TeamOutlined />,
                  label: t('userManagement'),
                },
              ],
            } as MenuItem,
          ]
        : []),
      {
        key: '/help',
        icon: <QuestionCircleOutlined />,
        label: t('help'),
      },
    ],
    [isAdmin, t],
  )

  const leafKeys = useMemo(() => {
    const keys: string[] = []
    const collect = (items: MenuItem[] = []) => {
      items.forEach((item) => {
        if (!item || item.type === 'divider' || item.type === 'group') return
        if ('children' in item && item.children) {
          collect(item.children as MenuItem[])
          return
        }
        if ('key' in item && typeof item.key === 'string' && item.key.startsWith('/')) {
          keys.push(item.key)
        }
      })
    }

    collect(menuItems)
    return keys
  }, [menuItems])

  const selectedKey = useMemo(() => {
    if (location.pathname.startsWith('/devices/') && location.pathname.endsWith('/topology')) {
      return '/devices/demo/topology'
    }

    return (
      leafKeys
        .filter((key) => location.pathname === key || location.pathname.startsWith(`${key}/`))
        .sort((a, b) => b.length - a.length)[0] ?? location.pathname
    )
  }, [leafKeys, location.pathname])

  const selectedKeys = [selectedKey]

  const selectedAncestorKeys = useMemo(() => {
    const findPath = (items: MenuItem[] = [], target: string, parents: string[] = []): string[] => {
      for (const item of items) {
        if (!item || item.type === 'divider' || item.type === 'group') continue
        const key = 'key' in item && typeof item.key === 'string' ? item.key : undefined
        const nextParents = key && !key.startsWith('/') ? [...parents, key] : parents

        if (key === target) return parents
        if ('children' in item && item.children) {
          const path = findPath(item.children as MenuItem[], target, nextParents)
          if (path.length) return path
        }
      }
      return []
    }

    return findPath(menuItems, selectedKey)
  }, [menuItems, selectedKey])

  const [openKeys, setOpenKeys] = useState<string[]>(selectedAncestorKeys)

  useEffect(() => {
    if (!collapsed) setOpenKeys(selectedAncestorKeys)
  }, [collapsed, selectedAncestorKeys])

  const handleOpenChange: MenuProps['onOpenChange'] = (keys) => {
    const latestOpenKey = keys.find((key) => !openKeys.includes(key))

    if (latestOpenKey && rootSubmenuKeys.includes(latestOpenKey)) {
      setOpenKeys(keys.filter((key) => key === latestOpenKey || !rootSubmenuKeys.includes(key)))
      return
    }

    setOpenKeys(keys)
  }

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (leafKeys.includes(key)) navigate(key)
  }

  const menuProps: MenuProps = collapsed
    ? {}
    : {
        openKeys,
        onOpenChange: handleOpenChange,
      }

  return (
    <div style={{ padding: collapsed ? '12px 8px' : '12px 10px' }}>
      <Menu
        mode="inline"
        selectedKeys={selectedKeys}
        inlineCollapsed={collapsed}
        items={menuItems}
        onClick={handleMenuClick}
        {...menuProps}
        style={{
          borderInlineEnd: 'none',
          background: token.colorBgContainer,
        }}
      />
    </div>
  )
}
