import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Card, Flex, Grid, Menu, Result, Skeleton, theme } from 'antd'
import {
  ApiOutlined,
  IdcardOutlined,
  MailOutlined,
  ReloadOutlined,
  SafetyOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import type { MenuProps } from 'antd'
import { adminApi } from '@/services/admin'
import { usePermission } from '@/hooks/usePermission'
import type { AdminSettings, OAuthProviderSettings } from '@/types/settings'
import type { SettingsOutletContext } from './context'

const { useBreakpoint } = Grid

export default function SettingsLayout() {
  const { t } = useTranslation('settings')
  const { can } = usePermission()
  const canReadSettings = can('setting.read')
  const [settings, setSettings] = useState<AdminSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadFailed, setLoadFailed] = useState(false)

  const navigate = useNavigate()
  const location = useLocation()
  const screens = useBreakpoint()
  const { token } = theme.useToken()
  const isDesktop = !!screens.md

  const loadSettings = useCallback(async () => {
    setLoading(true)
    setLoadFailed(false)
    try {
      setSettings(await adminApi.getSettings())
    } catch {
      setLoadFailed(true)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!canReadSettings) {
      setLoading(false)
      return
    }
    void loadSettings()
  }, [canReadSettings, loadSettings])

  const patchSettings = useCallback((partial: Partial<AdminSettings>) => {
    setSettings((current) => (current ? { ...current, ...partial } : current))
  }, [])

  const patchOAuthProvider = useCallback((next: OAuthProviderSettings) => {
    setSettings((current) =>
      current ? { ...current, oauth: current.oauth.map((item) => (item.id === next.id ? next : item)) } : current,
    )
  }, [])

  const menuItems = useMemo<MenuProps['items']>(
    () => [
      { key: 'beian', icon: <IdcardOutlined />, label: t('tabs.beian') },
      { key: 'security', icon: <SafetyOutlined />, label: t('tabs.security') },
      { key: 'smtp', icon: <MailOutlined />, label: t('tabs.smtp') },
      { key: 'oauth', icon: <ApiOutlined />, label: t('tabs.oauth') },
    ],
    [t],
  )

  const selectedKey = location.pathname.split('/').filter(Boolean)[1] ?? 'beian'

  const outletContext = useMemo<SettingsOutletContext | null>(
    () => (settings ? { settings, patchSettings, patchOAuthProvider } : null),
    [settings, patchSettings, patchOAuthProvider],
  )

  if (!canReadSettings) {
    return <Result status="403" title="403" subTitle={t('permissionDenied')} />
  }

  if (loadFailed) {
    return (
      <Result
        status="warning"
        title={t('loadFailedTitle')}
        subTitle={t('loadFailedDescription')}
        extra={
          <Button type="primary" icon={<ReloadOutlined />} onClick={() => void loadSettings()}>
            {t('retry')}
          </Button>
        }
      />
    )
  }

  return (
    <Card variant="outlined" styles={{ body: { padding: 0 } }}>
      <Flex vertical={!isDesktop}>
        {/* 设置分组导航：桌面端左侧竖排，移动端顶部横排 */}
        <div
          style={{
            flexShrink: 0,
            padding: isDesktop ? token.paddingSM : 0,
            borderInlineEnd: isDesktop ? `1px solid ${token.colorBorderSecondary}` : 'none',
            borderBottom: isDesktop ? 'none' : `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <Menu
            mode={isDesktop ? 'inline' : 'horizontal'}
            selectedKeys={[selectedKey]}
            items={menuItems}
            onClick={({ key }) => navigate(`/settings/${key}`)}
            style={{
              width: isDesktop ? 208 : '100%',
              borderInlineEnd: 'none',
              borderBottom: 'none',
              background: 'transparent',
            }}
          />
        </div>

        <div style={{ flex: 1, minWidth: 0, padding: isDesktop ? token.paddingLG : token.padding }}>
          {loading || !outletContext ? (
            <Skeleton active paragraph={{ rows: 8 }} />
          ) : (
            <Outlet context={outletContext} />
          )}
        </div>
      </Flex>
    </Card>
  )
}
