import { useEffect, useState } from 'react'
import { Card, ConfigProvider, Result, Skeleton, Tabs, theme } from 'antd'
import {
  SafetyOutlined,
  MailOutlined,
  ApiOutlined,
  IdcardOutlined,
  EllipsisOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import { useAuthStore } from '@/stores/authStore'
import type {
  AdminSettings,
  SecuritySettings,
  BeianSettings,
  SMTPSettings,
  OAuthProviderSettings,
} from '@/types/settings'
import BeianPanel from './BeianPanel'
import SecurityPanel from './SecurityPanel'
import SMTPPanel from './SMTPPanel'
import OAuthPanel from './OAuthPanel'

export default function SettingsPage() {
  const { t } = useTranslation('settings')
  const { token } = theme.useToken()
  const role = useAuthStore((s) => s.userInfo?.role)
  const isAdmin = role === 'admin' || role === 'super_admin'

  const [settings, setSettings] = useState<AdminSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeKey, setActiveKey] = useState('beian')

  useEffect(() => {
    if (!isAdmin) {
      setLoading(false)
      return
    }
    let alive = true
    ;(async () => {
      try {
        const data = await adminApi.getSettings()
        if (alive) setSettings(data)
      } catch {
        // 拦截器已提示错误
      } finally {
        if (alive) setLoading(false)
      }
    })()
    return () => {
      alive = false
    }
  }, [isAdmin])

  // 局部更新已加载的设置快照，避免保存后整页刷新。
  const patch = (partial: Partial<AdminSettings>) =>
    setSettings((prev) => (prev ? { ...prev, ...partial } : prev))

  const patchOAuthProvider = (next: OAuthProviderSettings) =>
    setSettings((prev) =>
      prev ? { ...prev, oauth: prev.oauth.map((p) => (p.id === next.id ? next : p)) } : prev,
    )

  if (!isAdmin) {
    return <Result status="403" title="403" subTitle={t('settings:adminOnly')} />
  }

  const tabItems =
    settings &&
    [
      {
        key: 'beian',
        label: t('settings:tabs.beian'),
        icon: <IdcardOutlined />,
        children: (
          <BeianPanel value={settings.beian} onSaved={(beian: BeianSettings) => patch({ beian })} />
        ),
      },
      {
        key: 'security',
        label: t('settings:tabs.security'),
        icon: <SafetyOutlined />,
        children: (
          <SecurityPanel
            value={settings.security}
            onSaved={(security: SecuritySettings) => patch({ security })}
          />
        ),
      },
      {
        key: 'smtp',
        label: t('settings:tabs.smtp'),
        icon: <MailOutlined />,
        children: <SMTPPanel value={settings.smtp} onSaved={(smtp: SMTPSettings) => patch({ smtp })} />,
      },
      {
        key: 'oauth',
        label: t('settings:tabs.oauth'),
        icon: <ApiOutlined />,
        children: <OAuthPanel value={settings.oauth} onSaved={patchOAuthProvider} />,
      },
    ]

  return (
    <ConfigProvider
      theme={{
        components: {
          Card: {
            bodyPadding: token.paddingLG,
            headerHeight: 48,
          },
          List: {
            itemPadding: `${token.padding}px ${token.paddingLG}px`,
          },
          Tabs: {
            horizontalItemGutter: token.marginLG,
            horizontalItemPadding: `${token.paddingSM}px 0`,
          },
        },
      }}
    >
      <Card
        className="netlab-settings-shell"
        style={{ height: '100%', minHeight: 0, overflow: 'hidden' }}
        styles={{
          body: {
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            minHeight: 0,
            minWidth: 0,
            overflow: 'hidden',
          },
        }}
      >
        {loading || !tabItems ? (
          <Skeleton active paragraph={{ rows: 8 }} />
        ) : (
          <Tabs
            activeKey={activeKey}
            onChange={setActiveKey}
            items={tabItems}
            className="netlab-settings-tabs"
            more={{ icon: <EllipsisOutlined />, trigger: 'click' }}
            tabBarGutter={token.marginLG}
            tabBarStyle={{ flexShrink: 0, marginBottom: 0, minWidth: 0 }}
            animated={{ inkBar: true, tabPane: false }}
            destroyOnHidden={false}
            style={{
              display: 'flex',
              flex: 1,
              flexDirection: 'column',
              minHeight: 0,
              minWidth: 0,
              overflow: 'hidden',
            }}
          />
        )}
      </Card>
    </ConfigProvider>
  )
}
