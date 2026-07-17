import { useEffect, useState } from 'react'
import { Alert, Button, Card, Form, Input, Space, App, theme } from 'antd'
import { MailOutlined, SaveOutlined, SendOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import type { ChangeEmailParams } from '@/types/auth'

export default function ChangeEmailPanel() {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const [form] = Form.useForm<ChangeEmailParams>()
  const [cooldown, setCooldown] = useState(0)
  const [sending, setSending] = useState(false)
  const [saving, setSaving] = useState(false)
  const currentEmail = useAuthStore((s) => s.userInfo?.email)

  useEffect(() => {
    if (cooldown <= 0) return
    const timer = window.setInterval(() => setCooldown((value) => (value <= 1 ? 0 : value - 1)), 1000)
    return () => window.clearInterval(timer)
  }, [cooldown])

  const sendCode = async () => {
    try {
      const { newEmail } = await form.validateFields(['newEmail'])
      setSending(true)
      const res = await authApi.sendChangeEmailCode(newEmail)
      setCooldown(res.cooldown > 0 ? res.cooldown : 60)
      message.success(t('settings:changeEmail.codeSent'))
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return
    } finally {
      setSending(false)
    }
  }

  const submit = async (values: ChangeEmailParams) => {
    setSaving(true)
    try {
      const user = await authApi.changeEmail(values)
      useAuthStore.getState().setUserInfo(user)
      form.resetFields()
      setCooldown(0)
      message.success(t('settings:changeEmail.success'))
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card
      title={t('settings:changeEmail.title')}
      variant="outlined"
      styles={{ body: { paddingBlock: token.paddingLG } }}
    >
      <Alert
        type="info"
        showIcon
        title={t('settings:changeEmail.notice', { email: currentEmail || '-' })}
        style={{ marginBottom: token.margin }}
      />
      <Form
        form={form}
        layout="vertical"
        onFinish={submit}
        requiredMark={false}
        style={{ maxWidth: 460 }}
      >
        <Form.Item
          name="newEmail"
          label={t('settings:changeEmail.newEmail')}
          rules={[
            { required: true, message: t('settings:changeEmail.emailRequired') },
            { type: 'email', message: t('settings:changeEmail.emailInvalid') },
          ]}
        >
          <Input prefix={<MailOutlined />} autoComplete="email" maxLength={255} />
        </Form.Item>
        <Form.Item label={t('settings:account.codeLabel')}>
          <Space.Compact style={{ width: '100%' }}>
            <Form.Item
              name="verifyCode"
              noStyle
              rules={[
                { required: true, message: t('settings:account.codeRequired') },
                { len: 6, message: t('settings:account.codeRequired') },
              ]}
            >
              <Input maxLength={6} />
            </Form.Item>
            <Button
              icon={<SendOutlined />}
              loading={sending}
              disabled={cooldown > 0}
              onClick={sendCode}
            >
              {cooldown > 0
                ? t('settings:account.codeResend', { seconds: cooldown })
                : t('settings:account.sendCode')}
            </Button>
          </Space.Compact>
        </Form.Item>
        <Form.Item style={{ marginBottom: 0 }}>
          <Button type="primary" htmlType="submit" loading={saving} icon={<SaveOutlined />}>
            {saving ? t('settings:saving') : t('settings:changeEmail.submit')}
          </Button>
        </Form.Item>
      </Form>
    </Card>
  )
}
