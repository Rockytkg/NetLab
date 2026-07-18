import { useEffect, useState } from 'react'
import { App, Button, Card, Divider, Flex, Form, InputNumber, Switch, Typography, theme } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { SecuritySettings } from '@/types/settings'
import SettingsSection from './SettingsSection'
import Can from '@/components/auth/Can'

const { Text } = Typography

interface SecurityPanelProps {
  value: SecuritySettings
  onSaved: (next: SecuritySettings) => void
}

/** 安全策略配置面板 */
export default function SecurityPanel({ value, onSaved }: SecurityPanelProps) {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const [form] = Form.useForm<SecuritySettings>()
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    form.setFieldsValue(value)
  }, [value, form])

  const handleSave = async (values: SecuritySettings) => {
    setSaving(true)
    try {
      await adminApi.updateSecurity(values)
      message.success(t('settings:saveSuccess'))
      onSaved(values)
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  const switches: Array<{ name: keyof SecuritySettings; label: string; help: string }> = [
    { name: 'registrationEnabled', label: t('settings:security.registration'), help: t('settings:security.registrationHelp') },
    { name: 'captchaEnabled', label: t('settings:security.captcha'), help: t('settings:security.captchaHelp') },
    { name: 'passwordResetEnabled', label: t('settings:security.passwordReset'), help: t('settings:security.passwordResetHelp') },
    { name: 'passkeyEnabled', label: t('settings:security.passkey'), help: t('settings:security.passkeyHelp') },
    { name: 'twoFactorRequired', label: t('settings:security.twoFactorRequired'), help: t('settings:security.twoFactorRequiredHelp') },
  ]

  return (
    <SettingsSection>
      <Form form={form} layout="vertical" onFinish={handleSave}>
        <Card size="small" variant="outlined" styles={{ body: { padding: 0 } }}>
          {switches.map((s, index) => (
            <div key={s.name}>
              <Flex
                align="center"
                justify="space-between"
                gap={token.marginLG}
                style={{ padding: `${token.paddingSM}px ${token.padding}px` }}
              >
                <Flex vertical gap={token.marginXXS} style={{ minWidth: 0 }}>
                  <Text strong>{s.label}</Text>
                  <Text type="secondary">{s.help}</Text>
                </Flex>
                <Form.Item name={s.name} valuePropName="checked" style={{ marginBottom: 0 }}>
                  <Switch checkedChildren="ON" unCheckedChildren="OFF" />
                </Form.Item>
              </Flex>
              {index < switches.length - 1 && <Divider style={{ margin: 0 }} />}
            </div>
          ))}
        </Card>
        <Divider style={{ marginBlock: token.marginLG }} />
        <Form.Item
          name="passwordMaxAgeDays"
          label={t('settings:security.passwordMaxAge')}
          extra={t('settings:security.passwordMaxAgeHelp')}
          rules={[{ type: 'number', min: 0, max: 3650, message: t('settings:security.passwordMaxAgeInvalid') }]}
        >
          <InputNumber min={0} max={3650} precision={0} style={{ width: '100%' }} />
        </Form.Item>
        <Divider style={{ marginBlock: token.marginLG }} />
        <Form.Item style={{ marginBottom: 0 }}>
          <Can permission="setting.update"><Button size="middle" type="primary" htmlType="submit" loading={saving} icon={<SaveOutlined />}>
            {saving ? t('settings:saving') : t('settings:save')}
          </Button></Can>
        </Form.Item>
      </Form>
    </SettingsSection>
  )
}
