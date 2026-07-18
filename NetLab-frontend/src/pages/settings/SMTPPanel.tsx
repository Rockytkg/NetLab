import { useEffect, useState } from 'react'
import {
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  InputNumber,
  Row,
  Space,
  Switch,
  Tag,
  App,
  theme,
} from 'antd'
import { MailOutlined, SaveOutlined, SendOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { SMTPSettings } from '@/types/settings'
import { SECRET_MASK } from '@/types/settings'
import SettingsSection from './SettingsSection'
import Can from '@/components/auth/Can'

interface SMTPPanelProps {
  value: SMTPSettings
  onSaved: (next: SMTPSettings) => void
}

/** 邮件服务（SMTP）配置面板 */
export default function SMTPPanel({ value, onSaved }: SMTPPanelProps) {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const [form] = Form.useForm<SMTPSettings>()
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testTo, setTestTo] = useState('')
  const host = Form.useWatch('host', form)

  useEffect(() => {
    // 密码字段以掩码回填，提示用户“留空保持不变”。
    form.setFieldsValue({ ...value, password: value.password === SECRET_MASK ? '' : value.password })
  }, [value, form])

  const handleSave = async (values: SMTPSettings) => {
    setSaving(true)
    try {
      await adminApi.updateSMTP(values)
      message.success(t('settings:saveSuccess'))
      onSaved({ ...values, password: SECRET_MASK })
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  const handleTest = async () => {
    setTesting(true)
    try {
      await adminApi.testSMTP(testTo)
      message.success(t('settings:smtp.testSuccess'))
      setTestTo('')
    } catch {
      // 拦截器已提示错误
    } finally {
      setTesting(false)
    }
  }

  const actions = (
    <Space wrap>
      <Can permission="setting.update"><Button size="middle" type="primary" htmlType="submit" loading={saving} icon={<SaveOutlined />}>
        {saving ? t('settings:saving') : t('settings:save')}
      </Button></Can>
    </Space>
  )

  return (
    <>
      <SettingsSection>
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          requiredMark={false}
          style={{ height: '100%', minHeight: 0 }}
        >
          <Card
            className="netlab-settings-panel-card"
            size="small"
            variant="outlined"
            title={
              <Space size={token.marginSM} wrap>
                <MailOutlined />
                <span>{t('settings:tabs.smtp')}</span>
                <Tag color={host ? 'success' : 'default'} style={{ marginInlineStart: 0 }}>
                  {host ? t('settings:oauth.configured') : t('settings:oauth.notConfigured')}
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
            <Row gutter={[token.marginLG, token.margin]} style={{ marginInline: 0 }}>
              <Col xs={24} lg={14}>
                <Row gutter={[token.margin, token.marginSM]} style={{ marginInline: 0 }}>
                  <Col xs={24} md={16}>
                    <Form.Item name="host" label={t('settings:smtp.host')}>
                      <Input placeholder={t('settings:smtp.hostPlaceholder')} allowClear maxLength={255} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item name="port" label={t('settings:smtp.port')}>
                      <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={16}>
                    <Form.Item name="from" label={t('settings:smtp.from')}>
                      <Input placeholder={t('settings:smtp.fromPlaceholder')} allowClear maxLength={255} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item
                      name="useTls"
                      label={t('settings:smtp.useTls')}
                      valuePropName="checked"
                    >
                      <Switch />
                    </Form.Item>
                  </Col>
                </Row>
              </Col>
              <Col xs={24} lg={10}>
                <Row gutter={[token.margin, token.marginSM]} style={{ marginInline: 0 }}>
                  <Col xs={24} md={12} lg={24}>
                    <Form.Item name="username" label={t('settings:smtp.username')}>
                      <Input placeholder={t('settings:smtp.usernamePlaceholder')} allowClear autoComplete="off" maxLength={255} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12} lg={24}>
                    <Form.Item
                      name="password"
                      label={t('settings:smtp.password')}
                      extra={t('settings:smtp.passwordPlaceholder')}
                    >
                      <Input.Password autoComplete="new-password" maxLength={512} />
                    </Form.Item>
                  </Col>
                  <Col xs={24}>
                    <Form.Item label={t('settings:smtp.testRecipient')}>
                      <Space.Compact style={{ width: '100%' }}>
                        <Input
                          type="email"
                          value={testTo}
                          onChange={(e) => setTestTo(e.target.value)}
                          placeholder={t('settings:smtp.testRecipientPlaceholder')}
                          allowClear
                        />
                        <Can permission="setting.update"><Button
                          size="middle"
                          icon={<SendOutlined />}
                          loading={testing}
                          disabled={!testTo}
                          onClick={handleTest}
                        >
                          {t('settings:smtp.test')}
                        </Button></Can>
                      </Space.Compact>
                    </Form.Item>
                  </Col>
                </Row>
              </Col>
            </Row>
            <Divider style={{ marginBlock: token.margin }} />
            <Form.Item style={{ marginBottom: 0 }}>{actions}</Form.Item>
          </Card>
        </Form>
      </SettingsSection>
    </>
  )
}
