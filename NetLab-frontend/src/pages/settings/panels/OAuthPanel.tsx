import { useEffect, useState } from 'react'
import { App, Collapse, Empty, Form, Input, Space, Tag, Typography } from 'antd'
import { GithubOutlined, GoogleOutlined, QqOutlined, WechatOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { OAuthProviderSettings } from '@/types/settings'
import { SECRET_MASK } from '@/types/settings'
import LinuxDoIcon from '@/components/common/icons/LinuxDoIcon'
import { useSettingsContext } from '../context'
import SettingSwitchItem from '../components/SettingSwitchItem'
import SettingsSubmitButton from '../components/SettingsSubmitButton'

const { Text } = Typography

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

interface ProviderFormProps {
  provider: OAuthProviderSettings
}

function ProviderForm({ provider }: ProviderFormProps) {
  const { t } = useTranslation('settings')
  const { message } = App.useApp()
  const { patchOAuthProvider } = useSettingsContext()
  const [form] = Form.useForm<ProviderFormValues>()
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    form.setFieldsValue({
      enabled: provider.enabled,
      clientId: provider.clientId,
      clientSecret: provider.clientSecret === SECRET_MASK ? '' : provider.clientSecret,
      redirectUrl: provider.redirectUrl,
    })
  }, [form, provider])

  const handleSave = async (values: ProviderFormValues) => {
    setSaving(true)
    try {
      await adminApi.updateOAuthProvider(provider.id, values)
      patchOAuthProvider({ ...provider, ...values, clientSecret: SECRET_MASK, configured: Boolean(values.clientId) })
      message.success(t('saveSuccess'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Form form={form} layout="vertical" onFinish={handleSave} requiredMark={false}>
      <SettingSwitchItem name="enabled" label={t('oauth.enabled')} />

      <Form.Item name="clientId" label={t('oauth.clientId')}>
        <Input allowClear autoComplete="off" maxLength={255} />
      </Form.Item>

      <Form.Item name="clientSecret" label={t('oauth.clientSecret')}>
        <Input.Password
          placeholder={t('oauth.clientSecretPlaceholder')}
          autoComplete="new-password"
          maxLength={512}
        />
      </Form.Item>

      <Form.Item name="redirectUrl" label={t('oauth.redirectUrl')}>
        <Input placeholder={t('oauth.redirectUrlPlaceholder')} allowClear maxLength={512} />
      </Form.Item>

      <SettingsSubmitButton loading={saving}>
        {t('oauth.save', { provider: provider.name })}
      </SettingsSubmitButton>
    </Form>
  )
}

export default function OAuthPanel() {
  const { t } = useTranslation('settings')
  const { settings } = useSettingsContext()
  const providers = settings.oauth

  if (providers.length === 0) {
    return <Empty description={t('oauthBindings.noProviders')} />
  }

  return (
    <Collapse
      defaultActiveKey={[providers[0].id]}
      items={providers.map((provider) => ({
        key: provider.id,
        label: (
          <Space>
            {PROVIDER_ICONS[provider.id]}
            <Text strong>{provider.name}</Text>
          </Space>
        ),
        extra: (
          <Tag color={provider.configured ? 'success' : 'default'}>
            {provider.configured ? t('oauth.configured') : t('oauth.notConfigured')}
          </Tag>
        ),
        children: <ProviderForm provider={provider} />,
      }))}
    />
  )
}
