import { useEffect, useState } from 'react'
import { App, Flex, Form, Input, InputNumber, Row, Col, Switch, Typography } from 'antd'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { BillingSettings } from '@/types/settings'
import SettingsSubmitButton from '../components/SettingsSubmitButton'

export default function BillingPanel() {
  const { t } = useTranslation('settings')
  const { message } = App.useApp()
  const [form] = Form.useForm<BillingSettings>()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    void adminApi.getBillingSettings()
      .then((settings) => form.setFieldsValue(settings))
      .finally(() => setLoading(false))
  }, [form])

  const handleSave = async (values: BillingSettings) => {
    setSaving(true)
    try {
      form.setFieldsValue(await adminApi.updateBillingSettings(values))
      message.success(t('saveSuccess'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
	    <Form form={form} layout="vertical" onFinish={handleSave} requiredMark={false} disabled={loading} initialValues={{ radius: { bindHost: '0.0.0.0' }, portal: { enabled: false, notifyPort: 50100 } }}>
      <Flex vertical gap={16} style={{ marginBottom: 24 }}>
        <Flex align="center" justify="space-between" gap={16}>
          <Flex vertical gap={4}><Typography.Text strong>{t('billing.enabled')}</Typography.Text><Typography.Text type="secondary">{t('billing.enabledHelp')}</Typography.Text></Flex>
          <Form.Item name={['radius', 'enabled']} valuePropName="checked" noStyle><Switch /></Form.Item>
        </Flex>
        <Flex align="center" justify="space-between" gap={16}>
          <Flex vertical gap={4}><Typography.Text strong>{t('billing.portalEnabled')}</Typography.Text><Typography.Text type="secondary">{t('billing.portalEnabledHelp')}</Typography.Text></Flex>
          <Form.Item name={['portal', 'enabled']} valuePropName="checked" noStyle><Switch /></Form.Item>
        </Flex>
      </Flex>
      <Row gutter={[16, 0]}>
        <Col xs={24} sm={24}>
	          <Form.Item name={['radius', 'bindHost']} label={t('billing.bindHost')} extra={t('billing.bindHostHelp')} rules={[{ required: true, message: t('billing.bindHostRequired') }]}>
            <Input maxLength={64} />
          </Form.Item>
        </Col>
        <Col xs={24} sm={12}>
	          <Form.Item name={['radius', 'authPort']} label={t('billing.authPort')} rules={[{ required: true, message: t('billing.portRequired') }]}>
            <InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col xs={24} sm={12}>
	          <Form.Item name={['radius', 'acctPort']} label={t('billing.acctPort')} rules={[{ required: true, message: t('billing.portRequired') }]}>
            <InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
      </Row>
	      <Row gutter={[16, 0]}><Col xs={24} sm={12}><Form.Item name={['portal', 'notifyPort']} label={t('billing.portalNotifyPort')} rules={[{ required: true, message: t('billing.portRequired') }]}><InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} /></Form.Item></Col></Row>
	      <SettingsSubmitButton loading={saving} />
	    </Form>
    </>
  )
}
