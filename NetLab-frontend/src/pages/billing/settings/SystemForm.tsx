import { useCallback, useEffect, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Col,
  Form,
  InputNumber,
  Row,
  Select,
  Spin,
  Switch,
  theme,
} from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import Can from '@/components/auth/Can'
import type { RadiusCertItem, RadiusSystemSettings } from '@/types/radius'

/** RadSec 设置表单：TLS 传输与证书绑定，保存后服务自动重载。 */
export default function SystemForm() {
  const { t } = useTranslation(['radius', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()

  const [form] = Form.useForm<RadiusSystemSettings>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [currentSettings, setCurrentSettings] = useState<RadiusSystemSettings | null>(null)
  const [serverCerts, setServerCerts] = useState<RadiusCertItem[]>([])
  const [caCerts, setCaCerts] = useState<RadiusCertItem[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [settings, servers, cas] = await Promise.all([
        radiusApi.getSettings(),
        radiusApi.listCerts({ size: 200, certType: 'server' }),
        radiusApi.listCerts({ size: 200, certType: 'ca' }),
      ])
      form.setFieldsValue(settings.system)
      setCurrentSettings(settings.system)
      setServerCerts(servers.items ?? [])
      setCaCerts(cas.items ?? [])
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [form])

  useEffect(() => {
    load()
  }, [load])

  const handleSave = async (values: Partial<RadiusSystemSettings>) => {
    if (!currentSettings) return
    setSaving(true)
    try {
      const updated = await radiusApi.updateSystemSettings({ ...currentSettings, ...values })
      form.setFieldsValue(updated)
      setCurrentSettings(updated)
      message.success(t('radius:common.saveSuccess'))
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  // 证书下拉：0 表示未选择
  const certOptions = (certs: RadiusCertItem[]) => [
    { value: 0, label: t('radius:settings.certNone') },
    ...certs.map((cert) => ({ value: cert.id, label: cert.name })),
  ]

  return (
    <Spin spinning={loading}>
      <Alert
        type="info"
        showIcon
        title={t('radius:settings.reloadTip')}
        style={{ marginBottom: token.margin }}
      />
      <Form form={form} layout="vertical" requiredMark={false} onFinish={handleSave}>
        <Row gutter={[24, 0]}>
          <Col xs={24} sm={8}>
            <Form.Item
              name="radsecEnabled"
              label={t('radius:settings.form.radsecEnabled')}
              tooltip={t('radius:settings.form.radsecEnabledTip')}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Col>
          <Col xs={24} sm={16}>
            <Form.Item
              name="radsecPort"
              label={t('radius:settings.form.radsecPort')}
              rules={[{ required: true, message: t('radius:settings.form.portRequired') }]}
            >
              <InputNumber min={1} max={65535} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={[24, 0]}>
          <Col xs={24} sm={12}>
            <Form.Item
              name="radsecCertId"
              label={t('radius:settings.form.radsecCertId')}
              tooltip={t('radius:settings.form.radsecCertIdTip')}
            >
              <Select showSearch={{ optionFilterProp: 'label' }} options={certOptions(serverCerts)} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="radsecCaCertId"
              label={t('radius:settings.form.radsecCaCertId')}
              tooltip={t('radius:settings.form.radsecCaCertIdTip')}
            >
              <Select showSearch={{ optionFilterProp: 'label' }} options={certOptions(caCerts)} />
            </Form.Item>
          </Col>
        </Row>

        <Can permission="radius.manage">
          <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving}>
            {t('common:save')}
          </Button>
        </Can>
      </Form>
    </Spin>
  )
}
