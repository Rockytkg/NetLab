import { useEffect, useState } from 'react'
import { Button, Col, Divider, Form, Input, Row, Space, App, theme } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { adminApi } from '@/services/admin'
import type { BeianSettings } from '@/types/settings'
import SettingsSection from './SettingsSection'
import Can from '@/components/auth/Can'

interface BeianPanelProps {
  value: BeianSettings
  onSaved: (next: BeianSettings) => void
}

/** 备案信息配置面板 */
export default function BeianPanel({ value, onSaved }: BeianPanelProps) {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const [form] = Form.useForm<BeianSettings>()
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    form.setFieldsValue(value)
  }, [value, form])

  const handleSave = async (values: BeianSettings) => {
    setSaving(true)
    try {
      await adminApi.updateBeian(values)
      message.success(t('settings:saveSuccess'))
      onSaved(values)
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  return (
    <SettingsSection>
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSave}
        requiredMark={false}
      >
        <Row gutter={[token.marginLG, token.marginSM]} style={{ marginInline: 0 }}>
          <Col xs={24} md={12}>
            <Form.Item
              name="icpBeian"
              label={t('settings:beian.icp')}
              extra={t('settings:beian.icpHelp')}
            >
              <Input placeholder={t('settings:beian.icpPlaceholder')} allowClear maxLength={128} />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              name="policeBeian"
              label={t('settings:beian.police')}
              extra={t('settings:beian.policeHelp')}
            >
              <Input placeholder={t('settings:beian.policePlaceholder')} allowClear maxLength={128} />
            </Form.Item>
          </Col>
        </Row>
        <Divider style={{ marginBlock: token.marginLG }} />
        <Form.Item style={{ marginBottom: 0 }}>
          <Space>
            <Can permission="setting.update"><Button size="middle" type="primary" htmlType="submit" loading={saving} icon={<SaveOutlined />}>
              {saving ? t('settings:saving') : t('settings:save')}
            </Button></Can>
          </Space>
        </Form.Item>
      </Form>
    </SettingsSection>
  )
}
