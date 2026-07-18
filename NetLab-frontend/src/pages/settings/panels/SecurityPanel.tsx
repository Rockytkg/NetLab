import { useEffect, useState } from 'react'
import { App, Divider, Form, InputNumber, List, Switch } from 'antd'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { SecuritySettings } from '@/types/settings'
import { useSettingsContext } from '../context'
import SettingsSubmitButton from '../components/SettingsSubmitButton'

interface SwitchItem {
  name: keyof SecuritySettings
  label: string
  help: string
}

export default function SecurityPanel() {
  const { t } = useTranslation('settings')
  const { message } = App.useApp()
  const { settings, patchSettings } = useSettingsContext()
  const [form] = Form.useForm<SecuritySettings>()
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    form.setFieldsValue(settings.security)
  }, [form, settings.security])

  const switches: SwitchItem[] = [
    { name: 'registrationEnabled', label: t('security.registration'), help: t('security.registrationHelp') },
    { name: 'captchaEnabled', label: t('security.captcha'), help: t('security.captchaHelp') },
    { name: 'passwordResetEnabled', label: t('security.passwordReset'), help: t('security.passwordResetHelp') },
    { name: 'passkeyEnabled', label: t('security.passkey'), help: t('security.passkeyHelp') },
    { name: 'twoFactorRequired', label: t('security.twoFactorRequired'), help: t('security.twoFactorRequiredHelp') },
  ]

  const handleSave = async (values: SecuritySettings) => {
    setSaving(true)
    try {
      await adminApi.updateSecurity(values)
      patchSettings({ security: values })
      message.success(t('saveSuccess'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Form form={form} layout="vertical" onFinish={handleSave} requiredMark={false}>
      {/* 开关项列表：名称与说明在左，开关在右，行间分隔线 */}
      <List
        itemLayout="horizontal"
        dataSource={switches}
        renderItem={(item) => (
          <List.Item
            actions={[
              <Form.Item key="switch" name={item.name} valuePropName="checked" noStyle>
                <Switch />
              </Form.Item>,
            ]}
          >
            <List.Item.Meta title={item.label} description={item.help} />
          </List.Item>
        )}
      />

      <Divider />

      <Form.Item
        name="passwordMaxAgeDays"
        label={t('security.passwordMaxAge')}
        extra={t('security.passwordMaxAgeHelp')}
        rules={[{ type: 'number', min: 0, max: 3650, message: t('security.passwordMaxAgeInvalid') }]}
      >
        <InputNumber min={0} max={3650} precision={0} style={{ width: 160 }} />
      </Form.Item>

      <SettingsSubmitButton loading={saving} />
    </Form>
  )
}
