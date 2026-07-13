import { useEffect, useState } from 'react'
import { Button, Divider, Form, InputNumber, List, Switch, Typography, App, theme } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { SecuritySettings } from '@/types/settings'
import SettingsSection from './SettingsSection'

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
        <List
          bordered
          itemLayout="horizontal"
          dataSource={switches}
          renderItem={(s) => (
            <List.Item
              actions={[
                <Form.Item
                  key={s.name}
                  name={s.name}
                  valuePropName="checked"
                  style={{ marginBottom: 0 }}
                >
                  <Switch checkedChildren="ON" unCheckedChildren="OFF" />
                </Form.Item>,
              ]}
            >
              <List.Item.Meta
                title={<Text strong>{s.label}</Text>}
                description={<Text type="secondary">{s.help}</Text>}
              />
            </List.Item>
          )}
        />
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
          <Button type="primary" htmlType="submit" loading={saving} icon={<SaveOutlined />}>
            {saving ? t('settings:saving') : t('settings:save')}
          </Button>
        </Form.Item>
      </Form>
    </SettingsSection>
  )
}
