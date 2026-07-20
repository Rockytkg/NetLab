import { useCallback, useEffect, useState } from 'react'
import {
  App,
  Button,
  Col,
  Form,
  InputNumber,
  Row,
  Select,
  Spin,
  Switch,
  Typography,
} from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { radiusApi } from '@/services/radius'
import Can from '@/components/auth/Can'
import type { RadiusServerSettings } from '@/types/radius'

type MessageAuthMode = RadiusServerSettings['messageAuthMode']
export type PolicySection = 'authentication' | 'protection' | 'session'

interface AuthPolicyFormProps {
  section: PolicySection
}

/** 认证与会话策略表单：安全策略与会话/记账默认值。 */
export default function AuthPolicyForm({ section }: AuthPolicyFormProps) {
  const { t } = useTranslation(['radius', 'common'])
  const { message } = App.useApp()

  const [form] = Form.useForm<RadiusServerSettings>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [currentSettings, setCurrentSettings] = useState<RadiusServerSettings | null>(null)
  const messageAuthMode = Form.useWatch('messageAuthMode', form)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const settings = await radiusApi.getSettings()
      form.setFieldsValue(settings.server)
      setCurrentSettings(settings.server)
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [form])

  useEffect(() => {
    load()
  }, [load])

  const handleSave = async (values: Partial<RadiusServerSettings>) => {
    if (!currentSettings) return
    setSaving(true)
    try {
      const updated = await radiusApi.updateServerSettings({ ...currentSettings, ...values })
      form.setFieldsValue(updated)
      setCurrentSettings(updated)
      message.success(t('radius:common.saveSuccess'))
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  // 当前选中模式的说明文案
  const modeTips: Record<MessageAuthMode, string> = {
    disabled: t('radius:config.form.messageAuthModeDisabledTip'),
    warn: t('radius:config.form.messageAuthModeWarnTip'),
    enforce: t('radius:config.form.messageAuthModeEnforceTip'),
  }

  return (
    <Spin spinning={loading}>
      <Form form={form} layout="vertical" requiredMark={false} onFinish={handleSave}>
        {section === 'authentication' && <Row gutter={[24, 0]}>
          <Col xs={24} sm={12}>
            <Form.Item
              name="messageAuthMode"
              label={t('radius:config.form.messageAuthMode')}
              extra={messageAuthMode ? modeTips[messageAuthMode as MessageAuthMode] : undefined}
              rules={[{ required: true, message: t('radius:config.form.messageAuthModeRequired') }]}
            >
              <Select
                options={[
                  { value: 'disabled', label: t('radius:config.form.messageAuthModeDisabled') },
                  { value: 'warn', label: t('radius:config.form.messageAuthModeWarn') },
                  { value: 'enforce', label: t('radius:config.form.messageAuthModeEnforce') },
                ]}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="ignorePassword"
              label={t('radius:config.form.ignorePassword')}
              valuePropName="checked"
              extra={
                <Typography.Text type="warning">
                  {t('radius:config.form.ignorePasswordTip')}
                </Typography.Text>
              }
            >
              <Switch />
            </Form.Item>
          </Col>
        </Row>}
        {section === 'protection' && <Row gutter={[24, 0]}>
          <Col xs={24} sm={12}>
            <Form.Item
              name="rejectDelayMaxRejects"
              label={t('radius:config.form.rejectDelayMaxRejects')}
              tooltip={t('radius:config.form.rejectDelayMaxRejectsTip')}
              rules={[{ required: true, message: t('radius:config.form.numberRequired') }]}
            >
              <InputNumber min={1} max={1000} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="rejectDelayWindowSeconds"
              label={t('radius:config.form.rejectDelayWindowSeconds')}
              tooltip={t('radius:config.form.rejectDelayWindowSecondsTip')}
              rules={[{ required: true, message: t('radius:config.form.numberRequired') }]}
            >
              <InputNumber min={1} max={3600} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>}
        {section === 'session' && <Row gutter={[24, 0]}>
          <Col xs={24} sm={8}>
            <Form.Item
              name="sessionTimeout"
              label={t('radius:config.form.sessionTimeout')}
              extra={t('radius:config.form.sessionTimeoutTip')}
              rules={[{ required: true, message: t('radius:config.form.numberRequired') }]}
            >
              <InputNumber min={0} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="acctInterimInterval"
              label={t('radius:config.form.acctInterimInterval')}
              tooltip={t('radius:config.form.acctInterimIntervalTip')}
              rules={[{ required: true, message: t('radius:config.form.numberRequired') }]}
            >
              <InputNumber min={30} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="historyDays"
              label={t('radius:config.form.historyDays')}
              extra={t('radius:config.form.historyDaysTip')}
              rules={[{ required: true, message: t('radius:config.form.numberRequired') }]}
            >
              <InputNumber min={0} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>}

        <Can permission="radius.manage">
          <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving}>
            {t('common:save')}
          </Button>
        </Can>
      </Form>
    </Spin>
  )
}
