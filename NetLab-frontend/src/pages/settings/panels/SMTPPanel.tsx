import { useEffect, useState } from 'react'
import {
  App,
  Button,
  Col,
  Divider,
  Form,
  Input,
  InputNumber,
  Row,
  Space,
  Switch,
  Tag,
} from 'antd'
import { SendOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { SMTPSettings } from '@/types/settings'
import { SECRET_MASK } from '@/types/settings'
import Can from '@/components/auth/Can'
import { useSettingsContext } from '../context'
import SettingSwitchItem from '../components/SettingSwitchItem'
import SettingsSubmitButton from '../components/SettingsSubmitButton'

export default function SMTPPanel() {
  const { t } = useTranslation('settings')
  const { message } = App.useApp()
  const { settings, patchSettings } = useSettingsContext()
  const [form] = Form.useForm<SMTPSettings>()
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testTo, setTestTo] = useState('')
  const host = Form.useWatch('host', form)

  useEffect(() => {
    form.setFieldsValue({ ...settings.smtp, password: settings.smtp.password === SECRET_MASK ? '' : settings.smtp.password })
  }, [form, settings.smtp])

  const handleSave = async (values: SMTPSettings) => {
    setSaving(true)
    try {
      await adminApi.updateSMTP(values)
      patchSettings({ smtp: { ...values, password: SECRET_MASK } })
      message.success(t('saveSuccess'))
    } finally {
      setSaving(false)
    }
  }

  const handleTest = async () => {
    setTesting(true)
    try {
      await adminApi.testSMTP(testTo)
      setTestTo('')
      message.success(t('smtp.testSuccess'))
    } finally {
      setTesting(false)
    }
  }

  return (
    <Form form={form} layout="vertical" onFinish={handleSave} requiredMark={false}>
      <SettingSwitchItem
        name="enabled"
        label={
          <Space size="small">
            {t('smtp.enabled')}
            <Tag color={host ? 'success' : 'default'}>
              {host ? t('oauth.configured') : t('oauth.notConfigured')}
            </Tag>
          </Space>
        }
        help={t('smtp.enabledHelp')}
      />

      <Divider titlePlacement="start">{t('smtp.connection')}</Divider>

      <Row gutter={[24, 0]}>
        <Col xs={24} sm={16}>
          <Form.Item name="host" label={t('smtp.host')}>
            <Input placeholder={t('smtp.hostPlaceholder')} allowClear maxLength={255} />
          </Form.Item>
        </Col>
        <Col xs={24} sm={8}>
          <Form.Item name="port" label={t('smtp.port')}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={[24, 0]}>
        <Col xs={24} sm={16}>
          <Form.Item name="from" label={t('smtp.from')}>
            <Input placeholder={t('smtp.fromPlaceholder')} allowClear maxLength={255} />
          </Form.Item>
        </Col>
        <Col xs={24} sm={8}>
          <Form.Item name="useTls" label={t('smtp.useTls')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Col>
      </Row>

      <Divider />

      <Row gutter={[24, 0]}>
        <Col xs={24} sm={12}>
          <Form.Item name="username" label={t('smtp.username')}>
            <Input placeholder={t('smtp.usernamePlaceholder')} allowClear autoComplete="off" maxLength={255} />
          </Form.Item>
        </Col>
        <Col xs={24} sm={12}>
          <Form.Item name="password" label={t('smtp.password')} extra={t('smtp.passwordPlaceholder')}>
            <Input.Password autoComplete="new-password" maxLength={512} />
          </Form.Item>
        </Col>
      </Row>

      <Divider titlePlacement="start">{t('smtp.testTitle')}</Divider>

      <Form.Item label={t('smtp.testRecipient')}>
        <Space.Compact style={{ width: '100%' }}>
          <Input
            type="email"
            value={testTo}
            onChange={(event) => setTestTo(event.target.value)}
            placeholder={t('smtp.testRecipientPlaceholder')}
            allowClear
          />
          <Can permission="setting.update">
            <Button icon={<SendOutlined />} loading={testing} disabled={!testTo} onClick={handleTest}>
              {t('smtp.send')}
            </Button>
          </Can>
        </Space.Compact>
      </Form.Item>

      <SettingsSubmitButton loading={saving} />
    </Form>
  )
}
