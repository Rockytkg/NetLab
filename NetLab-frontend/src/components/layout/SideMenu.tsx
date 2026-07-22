import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Menu, theme } from 'antd'
import {
  AccountBookOutlined,
  ControlOutlined,
  DashboardOutlined,
  GlobalOutlined,
  HistoryOutlined,
  ProfileOutlined,
  SafetyCertificateOutlined,
  SecurityScanOutlined,
  SettingOutlined,
  TeamOutlined,
  UnlockOutlined,
  WifiOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import type { MenuProps } from 'antd'
import { usePermission } from '@/hooks/usePermission'

interface SideMenuProps {
  collapsed: boolean
}

export default function SideMenu({ collapsed }: SideMenuProps) {
  const { t } = useTranslation('menu')
  const navigate = useNavigate()
  const location = useLocation()
  const { token } = theme.useToken()
  const { can } = usePermission()
	const canReadSettings = can('setting.read')
	const canReadUsers = can('user.read')
	const canReadRbac = can('rbac.read')
	const canReadLogs = can('log.read')
		const canReadRadius = can('radius.read')
		const canReadPortal = can('portal.read')

  type MenuItem = Required<MenuProps>['items'][number]
  const rootSubmenuKeys = ['administration', 'billing']

  const menuItems = useMemo<MenuItem[]>(
    () => [
      {
        key: '/dashboard',
        icon: <DashboardOutlined />,
        label: t('dashboard'),
      },
	      ...(canReadRadius || canReadPortal
        ? [
            {
              key: 'billing',
              icon: <AccountBookOutlined />,
              label: t('billing'),
              children: [
                {
                  key: 'billing-business',
                  label: t('radiusGroupBusiness'),
                  children: [
                    { key: '/billing/nas', icon: <GlobalOutlined />, label: t('radiusNas') },
                    { key: '/billing/users', icon: <TeamOutlined />, label: t('radiusUsers') },
                    { key: '/billing/profiles', icon: <ProfileOutlined />, label: t('radiusProfiles') },
                    { key: '/billing/sessions', icon: <WifiOutlined />, label: t('radiusSessions') },
                    { key: '/billing/accounting', icon: <AccountBookOutlined />, label: t('radiusAccounting') },
                    { key: '/billing/auth-logs', icon: <HistoryOutlined />, label: t('radiusAuthLogs') },
                  ],
                },
                {
                  key: 'billing-auth',
                  label: t('radiusGroupAuth'),
                  children: [
	            { key: '/billing/dot1x', icon: <SecurityScanOutlined />, label: t('radiusDot1x') },
	                    { key: '/billing/bypass', icon: <UnlockOutlined />, label: t('radiusBypass') },
	                    ...(canReadPortal ? [{ key: '/billing/portal', icon: <WifiOutlined />, label: t('portal') }] : []),
                  ],
                },
                {
                  key: 'billing-service',
                  label: t('radiusGroupService'),
                  children: [
                    { key: '/billing/certs', icon: <SafetyCertificateOutlined />, label: t('radiusCerts') },
                    { key: '/billing/settings', icon: <ControlOutlined />, label: t('radiusSettings') },
                  ],
                },
              ],
            } as MenuItem,
          ]
        : []),
      ...(canReadSettings || canReadUsers || canReadRbac || canReadLogs
        ? [
            {
              key: 'administration',
              icon: <SettingOutlined />,
              label: t('administration'),
              children: [
                ...(canReadSettings
                  ? [
                      {
                        key: '/settings',
                        icon: <ControlOutlined />,
                        label: t('settings'),
                      },
                    ]
                  : []),
                ...(canReadUsers
                  ? [
                      {
                        key: '/settings/users',
                        icon: <TeamOutlined />,
                        label: t('userManagement'),
                      },
                    ]
                  : []),
                ...(canReadRbac
                  ? [
                      {
                        key: '/settings/roles',
                        icon: <SafetyCertificateOutlined />,
                        label: t('roleManagement'),
                      },
                    ]
                  : []),
                ...(canReadLogs
                  ? [
                      {
                        key: '/settings/login-logs',
                        icon: <HistoryOutlined />,
                        label: t('loginLogs'),
                      },
                    ]
                  : []),
              ],
            } as MenuItem,
          ]
        : []),
    ],
	    [canReadSettings, canReadUsers, canReadRbac, canReadLogs, canReadRadius, canReadPortal, t],
  )

  const leafKeys = useMemo(() => {
    const keys: string[] = []
    const collect = (items: MenuItem[] = []) => {
      items.forEach((item) => {
        if (!item || item.type === 'divider') return
        // group 只是标题分组，其子项仍是可达叶子，必须递归收集
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
    // 用户中心不属于侧边栏菜单，避免被 /settings 的前缀匹配误激活。
    if (location.pathname === '/settings/profile' || location.pathname.startsWith('/settings/profile/')) {
      return ''
    }

    return (
      leafKeys
        .filter((key) => location.pathname === key || location.pathname.startsWith(`${key}/`))
        .sort((a, b) => b.length - a.length)[0] ?? location.pathname
    )
  }, [leafKeys, location.pathname])

  const selectedKeys = selectedKey ? [selectedKey] : []

  const selectedAncestorKeys = useMemo(() => {
    const findPath = (items: MenuItem[] = [], target: string, parents: string[] = []): string[] => {
      for (const item of items) {
        if (!item || item.type === 'divider') continue
        const key = 'key' in item && typeof item.key === 'string' ? item.key : undefined
        // group 不是可展开的 submenu，其 key 不参与父路径收集，仅递归其子项
        const nextParents =
          item.type !== 'group' && key && !key.startsWith('/') ? [...parents, key] : parents

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
