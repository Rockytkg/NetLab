import { useEffect, useState } from 'react'
import {
  Button,
  ConfigProvider,
  Flex,
  Grid,
  Layout,
  Menu,
  Result,
  Skeleton,
  Tabs,
  theme,
  Typography,
} from 'antd'
import {
  SafetyOutlined,
  MailOutlined,
  ApiOutlined,
  IdcardOutlined,
  EllipsisOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import { usePermission } from '@/hooks/usePermission'
import type { MenuProps, TabsProps } from 'antd'
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
import '@/assets/css/settings.css'

const { useBreakpoint } = Grid
const { Sider, Content } = Layout
const { Text, Title } = Typography

export default function SettingsPage() {
  const { t } = useTranslation('settings')
  const { token } = theme.useToken()
  const screens = useBreakpoint()
  const { can } = usePermission()
	const canReadSettings = can('setting.read')

  const [settings, setSettings] = useState<AdminSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadFailed, setLoadFailed] = useState(false)
  const [activeKey, setActiveKey] = useState('beian')

  const loadSettings = async (alive?: () => boolean) => {
    setLoading(true)
    setLoadFailed(false)
    try {
      const data = await adminApi.getSettings()
      if (!alive || alive()) setSettings(data)
    } catch {
      if (!alive || alive()) setLoadFailed(true)
    } finally {
      if (!alive || alive()) setLoading(false)
    }
  }

  const fetchSettings = () => loadSettings()

  useEffect(() => {
    if (!canReadSettings) {
      setLoading(false)
      return
    }
    let alive = true
    void loadSettings(() => alive)
    return () => {
      alive = false
    }
  }, [canReadSettings])

  // 局部更新已加载的设置快照，避免保存后整页刷新。
  const patch = (partial: Partial<AdminSettings>) =>
    setSettings((prev) => (prev ? { ...prev, ...partial } : prev))

  const patchOAuthProvider = (next: OAuthProviderSettings) =>
    setSettings((prev) =>
      prev ? { ...prev, oauth: prev.oauth.map((p) => (p.id === next.id ? next : p)) } : prev,
    )

  if (!canReadSettings) {
    return <Result status="403" title="403" subTitle={t('permissionDenied')} />
  }

  const settingsItems =
    settings &&
    [
      {
        key: 'beian',
        label: t('tabs.beian'),
        icon: <IdcardOutlined />,
        children: (
          <BeianPanel value={settings.beian} onSaved={(beian: BeianSettings) => patch({ beian })} />
        ),
      },
      {
        key: 'security',
        label: t('tabs.security'),
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
        label: t('tabs.smtp'),
        icon: <MailOutlined />,
        children: <SMTPPanel value={settings.smtp} onSaved={(smtp: SMTPSettings) => patch({ smtp })} />,
      },
      {
        key: 'oauth',
        label: t('tabs.oauth'),
        icon: <ApiOutlined />,
        children: <OAuthPanel value={settings.oauth} onSaved={patchOAuthProvider} />,
      },
    ]
  const activeItem = settingsItems?.find((item) => item.key === activeKey)
  const navigationItems: MenuProps['items'] | undefined = settingsItems?.map(
    ({ key, label, icon }) => ({
      key,
      label,
      icon,
    }),
  )
  const tabItems: TabsProps['items'] | undefined = settingsItems?.map(
    ({ key, label, icon }) => ({
      key,
      label,
      icon,
    }),
  )

  return (
    <ConfigProvider
      theme={{
        components: {
          Card: {
            bodyPadding: token.paddingLG,
          },
          Button: {
            controlHeight: 32,
          },
          Form: {
            itemMarginBottom: 12,
          },
          Tabs: {
            horizontalItemGutter: token.marginLG,
            horizontalItemPadding: `${token.paddingSM}px 0`,
            verticalItemPadding: `${token.paddingSM}px ${token.paddingLG}px`,
          },
        },
      }}
    >
      <div className="netlab-settings-shell">
        {loading ? (
          <Skeleton active paragraph={{ rows: 8 }} style={{ padding: token.paddingLG }} />
        ) : loadFailed || !settingsItems || !activeItem ? (
          <Result
            status="warning"
            title={t('loadFailedTitle')}
            subTitle={t('loadFailedDescription')}
            extra={
              <Button type="primary" icon={<ReloadOutlined />} onClick={fetchSettings}>
                {t('retry')}
              </Button>
            }
          />
        ) : screens.lg ? (
          <Layout className="netlab-settings-layout">
            <Sider
              width={260}
              theme="light"
              className="netlab-settings-sider"
              style={{ background: token.colorBgContainer }}
            >
              <Flex vertical gap={token.marginXXS} className="netlab-settings-nav-header">
                <Title level={5} style={{ margin: 0 }}>
                  {t('title')}
                </Title>
                <Text type="secondary">{t('pageDescription')}</Text>
              </Flex>
              <Menu
                mode="inline"
                selectedKeys={[activeKey]}
                items={navigationItems}
                onClick={({ key }) => setActiveKey(key)}
                style={{ borderInlineEnd: 'none' }}
              />
            </Sider>
            <Content className="netlab-settings-content">{activeItem?.children}</Content>
          </Layout>
        ) : (
          <div className="netlab-settings-mobile">
            <Flex vertical gap={token.marginXXS} className="netlab-settings-mobile-header">
              <Title level={5} style={{ margin: 0 }}>
                {t('title')}
              </Title>
              <Text type="secondary">{t('pageDescription')}</Text>
            </Flex>
            <Tabs
              activeKey={activeKey}
              onChange={setActiveKey}
              items={tabItems}
              className="netlab-settings-mobile-tabs"
              more={{ icon: <EllipsisOutlined />, trigger: 'click' }}
              tabBarGutter={token.marginLG}
              tabBarStyle={{
                flexShrink: 0,
                marginBottom: 0,
                paddingInline: token.paddingLG,
                paddingBlockStart: token.paddingSM,
              }}
              animated={{ inkBar: true, tabPane: false }}
            />
            <div className="netlab-settings-content">{activeItem?.children}</div>
          </div>
        )}
      </div>
    </ConfigProvider>
  )
}

