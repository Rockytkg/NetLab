import { useEffect, useState } from 'react'
import { App, Form, Input } from 'antd'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { BeianSettings } from '@/types/settings'
import { useSettingsContext } from '../context'
import SettingsSubmitButton from '../components/SettingsSubmitButton'

export default function BeianPanel() {
  const { t } = useTranslation('settings')
  const { message } = App.useApp()
  const { settings, patchSettings } = useSettingsContext()
  const [form] = Form.useForm<BeianSettings>()
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    form.setFieldsValue(settings.beian)
  }, [form, settings.beian])

  const handleSave = async (values: BeianSettings) => {
    setSaving(true)
    try {
      await adminApi.updateBeian(values)
      patchSettings({ beian: values })
      message.success(t('saveSuccess'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Form form={form} layout="vertical" onFinish={handleSave} requiredMark={false}>
      <Form.Item name="icpBeian" label={t('beian.icp')} extra={t('beian.icpHelp')}>
        <Input placeholder={t('beian.icpPlaceholder')} allowClear maxLength={128} />
      </Form.Item>

      <Form.Item name="policeBeian" label={t('beian.police')} extra={t('beian.policeHelp')}>
        <Input placeholder={t('beian.policePlaceholder')} allowClear maxLength={128} />
      </Form.Item>

      <SettingsSubmitButton loading={saving} />
    </Form>
  )
}
