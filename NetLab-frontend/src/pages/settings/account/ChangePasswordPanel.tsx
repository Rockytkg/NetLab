import { useState } from 'react'
import { Alert, Button, Card, Form, Input, App, theme } from 'antd'
import { LockOutlined, SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import { createPasswordStrengthRule } from '@/utils/password-strength'
import type { ChangePasswordParams } from '@/types/auth'

/**
 * 修改密码面板。校验当前密码后更新，成功即视为全部会话失效，
 * 前端主动登出并跳转登录页。
 */
export default function ChangePasswordPanel() {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const logout = useAuthStore((s) => s.logout)
  const [form] = Form.useForm<ChangePasswordParams>()
  const [saving, setSaving] = useState(false)

  const handleSubmit = async (values: ChangePasswordParams) => {
    setSaving(true)
    try {
      await authApi.changePassword(values)
      message.success(t('settings:changePassword.success'))
      form.resetFields()
      // 改密后会话已在服务端失效，直接清理本地会话并跳转登录页
      // （无需再调用 logout 接口，其 token 已被吊销）。
      await logout({ callApi: false })
      navigate('/login', { replace: true })
    } catch {
      // 拦截器已提示错误
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card
      title={t('settings:changePassword.title')}
      variant="outlined"
      styles={{ body: { paddingBlock: token.paddingLG } }}
    >
      <Alert
        type="info"
        showIcon
        title={t('settings:changePassword.notice')}
        style={{ marginBottom: token.marginLG }}
      />
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        requiredMark={false}
        style={{ maxWidth: 420 }}
      >
        <Form.Item
          name="currentPassword"
          label={t('settings:changePassword.current')}
          rules={[{ required: true, message: t('settings:changePassword.currentRequired') }]}
        >
          <Input.Password prefix={<LockOutlined />} autoComplete="current-password" maxLength={128} />
        </Form.Item>
        <Form.Item
          name="newPassword"
          label={t('settings:changePassword.new')}
          rules={[
            { required: true, message: t('settings:changePassword.newRequired') },
            createPasswordStrengthRule({
              t,
            }),
          ]}
        >
          <Input.Password prefix={<LockOutlined />} autoComplete="new-password" maxLength={72} />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          label={t('settings:changePassword.confirm')}
          dependencies={['newPassword']}
          rules={[
            { required: true, message: t('settings:changePassword.confirmRequired') },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('newPassword') === value) return Promise.resolve()
                return Promise.reject(new Error(t('settings:changePassword.mismatch')))
              },
            }),
          ]}
        >
          <Input.Password prefix={<LockOutlined />} autoComplete="new-password" maxLength={128} />
        </Form.Item>
        <Form.Item style={{ marginBottom: 0 }}>
          <Button type="primary" htmlType="submit" loading={saving} icon={<SaveOutlined />}>
            {saving ? t('settings:saving') : t('settings:changePassword.submit')}
          </Button>
        </Form.Item>
      </Form>
    </Card>
  )
}
