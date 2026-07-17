import { useEffect, useState } from 'react'
import { Button, Card, Divider, Form, Input, Space, Switch, Tabs, Tag, App, theme } from 'antd'
import {
  GithubOutlined,
  GoogleOutlined,
  QqOutlined,
  WechatOutlined,
  SaveOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { OAuthProviderSettings } from '@/types/settings'
import { SECRET_MASK } from '@/types/settings'
import LinuxDoIcon from '@/components/common/icons/LinuxDoIcon'
import SettingsSection from './SettingsSection'
import Can from '@/components/auth/Can'

const PROVIDER_ICONS: Record<string, React.ReactNode> = {
  github: <GithubOutlined />,
  google: <GoogleOutlined />,
  qq: <QqOutlined />,
  wechat: <WechatOutlined />,
  linuxdo: <LinuxDoIcon />,
}

interface ProviderFormValues {
  enabled: boolean
  clientId: string
  clientSecret: string
  redirectUrl: string
}

interface ProviderCardProps {
  provider: OAuthProviderSettings
  onSaved: (next: OAuthProviderSettings) => void
}

/** 单个 OAuth 提供商配置卡片 */
function ProviderCard({ provider, onSaved }: ProviderCardProps) {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const [form] = Form.useForm<ProviderFormValues>()
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    form.setFieldsValue({
      enabled: provider.enabled,
      clientId: provider.clientId,
      clientSecret: provider.clientSecret === SECRET_MASK ? '' : provider.clientSecret,
      redirectUrl: provider.redirectUrl,
    })
  }, [provider, form])

  const handleSave = async (values: ProviderFormValues) => {
    setSaving(true)
    try {
      await adminApi.updateOAuthProvider(provider.id, values)
      message.success(t('settings:saveSuccess'))
      onSaved({
        ...provider,
        ...values,
        clientSecret: SECRET_MASK,
        configured: !!values.clientId,
      })
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  return (
    <Form form={form} layout="vertical" onFinish={handleSave} style={{ height: '100%' }}>
      <Card
        className="netlab-settings-panel-card"
        size="small"
        variant="outlined"
        style={{ height: '100%' }}
        title={
          <Space size={token.marginSM} wrap>
            {PROVIDER_ICONS[provider.id]}
            <span>{provider.name}</span>
            <Tag color={provider.configured ? 'success' : 'default'} style={{ marginInlineStart: 0 }}>
              {provider.configured ? t('settings:oauth.configured') : t('settings:oauth.notConfigured')}
            </Tag>
          </Space>
        }
        extra={
          <Form.Item name="enabled" valuePropName="checked" style={{ marginBottom: 0 }}>
            <Switch size="small" />
          </Form.Item>
        }
        styles={{
          header: {
            minHeight: 48,
            paddingInline: token.padding,
          },
          body: { padding: token.padding },
        }}
      >
        <Form.Item name="clientId" label={t('settings:oauth.clientId')}>
          <Input allowClear autoComplete="off" maxLength={255} />
        </Form.Item>
        <Form.Item name="clientSecret" label={t('settings:oauth.clientSecret')}>
          <Input.Password placeholder={t('settings:oauth.clientSecretPlaceholder')} autoComplete="new-password" maxLength={512} />
        </Form.Item>
        <Form.Item name="redirectUrl" label={t('settings:oauth.redirectUrl')}>
          <Input placeholder={t('settings:oauth.redirectUrlPlaceholder')} allowClear maxLength={512} />
        </Form.Item>
        <Divider style={{ marginBlock: token.margin }} />
        <Form.Item style={{ marginBottom: 0 }}>
          <Can resource="setting" action="update"><Button size="middle" type="primary" htmlType="submit" loading={saving} icon={<SaveOutlined />}>
            {saving ? t('settings:saving') : t('settings:oauth.save', { provider: provider.name })}
          </Button></Can>
        </Form.Item>
      </Card>
    </Form>
  )
}

interface OAuthPanelProps {
  value: OAuthProviderSettings[]
  onSaved: (next: OAuthProviderSettings) => void
}

/** 第三方登录配置面板 */
export default function OAuthPanel({ value, onSaved }: OAuthPanelProps) {
  return (
    <SettingsSection>
      <Tabs
        className="netlab-settings-provider-tabs"
        destroyOnHidden={false}
        items={value.map((provider) => ({
          key: provider.id,
          label: provider.name,
          icon: PROVIDER_ICONS[provider.id],
          children: <ProviderCard provider={provider} onSaved={onSaved} />,
        }))}
      />
    </SettingsSection>
  )
}
