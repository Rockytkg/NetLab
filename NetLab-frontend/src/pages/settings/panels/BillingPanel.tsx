import { useEffect, useState } from 'react'
import { App, Form, Input, InputNumber, List, Row, Col, Switch } from 'antd'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { RadiusListenerSettings } from '@/types/settings'
import SettingsSubmitButton from '../components/SettingsSubmitButton'

export default function BillingPanel() {
  const { t } = useTranslation('settings')
  const { message } = App.useApp()
  const [form] = Form.useForm<RadiusListenerSettings>()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    void adminApi.getRadiusListenerSettings()
      .then((settings) => form.setFieldsValue(settings))
      .finally(() => setLoading(false))
  }, [form])

  const handleSave = async (values: RadiusListenerSettings) => {
    setSaving(true)
    try {
      await adminApi.updateRadiusListenerSettings(values)
      message.success(t('saveSuccess'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Form form={form} layout="vertical" onFinish={handleSave} requiredMark={false} disabled={loading}>
      <List
        itemLayout="horizontal"
        dataSource={[{ label: t('billing.enabled'), help: t('billing.enabledHelp') }]}
        renderItem={(item) => (
          <List.Item
            actions={[<Form.Item key="enabled" name="enabled" valuePropName="checked" noStyle><Switch /></Form.Item>]}
          >
            <List.Item.Meta title={item.label} description={item.help} />
          </List.Item>
        )}
      />
      <Row gutter={[16, 0]} style={{ marginTop: 24 }}>
        <Col xs={24} sm={24}>
          <Form.Item name="bindHost" label={t('billing.bindHost')} extra={t('billing.bindHostHelp')} rules={[{ required: true, message: t('billing.bindHostRequired') }]}>
            <Input maxLength={64} />
          </Form.Item>
        </Col>
        <Col xs={24} sm={12}>
          <Form.Item name="authPort" label={t('billing.authPort')} rules={[{ required: true, message: t('billing.portRequired') }]}>
            <InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} sm={12}>
          <Form.Item name="acctPort" label={t('billing.acctPort')} rules={[{ required: true, message: t('billing.portRequired') }]}>
            <InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
      </Row>
      <SettingsSubmitButton loading={saving} />
    </Form>
  )
}
