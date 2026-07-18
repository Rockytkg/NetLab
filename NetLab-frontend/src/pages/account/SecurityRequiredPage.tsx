import { Alert, App, Button, Flex, Form, Input, Space, Typography, theme } from 'antd'
import { LockOutlined, MailOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import SecurityFlowLayout from '@/components/auth/SecurityFlowLayout'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import { createPasswordStrengthRule } from '@/utils/password-strength'
import '@/assets/css/login.css'

const { Title, Text } = Typography

export default function SecurityRequiredPage() {
  const { t } = useTranslation('login')
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const actions = useAuthStore((s) => s.securityActions)
  const userInfo = useAuthStore((s) => s.userInfo)
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)
  const [cooldown, setCooldown] = useState(0)

  const needPassword = !!actions?.requirePasswordChange
  const needEmail = !!actions?.requireEmailChange
  const isDefaultAdmin = actions?.reason === 'default_admin_bootstrap'
  const needEmailCode = needEmail && !isDefaultAdmin

  useEffect(() => {
    if (!needPassword && !needEmail) {
      navigate('/dashboard', { replace: true })
    }
  }, [needPassword, needEmail, navigate])

  const sendCode = async () => {
    try {
      await form.validateFields(['newEmail'])
      const newEmail = form.getFieldValue('newEmail')
      const result = await authApi.sendChangeEmailCode(newEmail)
      setCooldown(result.cooldown || 60)
      message.success(t('sendCodeSuccess'))
      const timer = window.setInterval(() => {
        setCooldown((value) => {
          if (value <= 1) {
            window.clearInterval(timer)
            return 0
          }
          return value - 1
        })
      }, 1000)
    } catch {
      /* form validation or interceptor handled */
    }
  }

  const submit = async (values: {
    newPassword: string
    confirmPassword: string
    newEmail?: string
    verifyCode?: string
  }) => {
    setSubmitting(true)
    try {
      await authApi.completeSecurityUpdate(values)
      message.success(t('securityUpdateSuccess'))
      // 完成后强制重新登录：服务端已撤销会话，logout 会一并清空本地
      // token / userInfo / securityActions（无需先写入随即被清空的用户信息）。
      await useAuthStore.getState().logout({ callApi: false })
      navigate('/login', { replace: true })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <SecurityFlowLayout
      title={t('securityUpdateTitle')}
      subtitle={t('securityUpdateSubtitle', { username: userInfo?.username ?? '' })}
      steps={[
        {
          title: needEmail ? t('securityUpdateStepAccount') : t('securityUpdateStepPassword'),
          content: needEmail
            ? t(isDefaultAdmin ? 'securityUpdateAdminEmailHint' : 'securityUpdateFirstLoginHint')
            : t('securityUpdatePasswordOnly'),
        },
        {
          title: t('securityUpdateStepRelogin'),
          content: t('securityUpdateReloginHint'),
        },
      ]}
    >
      <Flex vertical gap={token.marginLG}>
        <Flex vertical gap={token.marginXXS}>
          <Title level={4} style={{ margin: 0 }}>
            {needEmail ? t('securityUpdateFormTitleFull') : t('securityUpdateFormTitle')}
          </Title>
          <Text type='secondary'>{t('securityUpdateNotice')}</Text>
        </Flex>

        <Alert
          type='warning'
          showIcon
          title={
            needEmail
              ? t(isDefaultAdmin ? 'defaultAdminNotice' : 'firstLoginNotice')
              : t(actions?.reason === 'password_expired' ? 'passwordExpiredNotice' : 'passwordResetNotice')
          }
        />

        <Form
          form={form}
          size='large'
          layout='vertical'
          requiredMark={false}
          onFinish={submit}
          className='netlab-login-form'
          style={{ width: '100%' }}
        >
          <Form.Item
            name='newPassword'
            rules={[
              { required: true, message: t('newPasswordRequired') },
              createPasswordStrengthRule({
                t,
              }),
            ]}
          >
            <Input.Password
              prefix={<LockOutlined style={{ color: token.colorTextQuaternary }} />}
              placeholder={t('newPasswordPlaceholder')}
              autoComplete='new-password'
              maxLength={72}
            />
          </Form.Item>
          <Form.Item
            name='confirmPassword'
            dependencies={['newPassword']}
            rules={[
              { required: true, message: t('confirmPasswordRequired') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) return Promise.resolve()
                  return Promise.reject(new Error(t('passwordMismatch')))
                },
              }),
            ]}
          >
            <Input.Password
              prefix={<LockOutlined style={{ color: token.colorTextQuaternary }} />}
              placeholder={t('confirmPasswordPlaceholder')}
              autoComplete='new-password'
              maxLength={72}
            />
          </Form.Item>

          {needEmail && (
            <>
              <Form.Item
                name='newEmail'
                rules={[{ required: true, type: 'email', message: t('emailInvalid') }]}
              >
                <Input
                  prefix={<MailOutlined style={{ color: token.colorTextQuaternary }} />}
                  placeholder={t('emailPlaceholder')}
                  autoComplete='email'
                  maxLength={255}
                />
              </Form.Item>

              {needEmailCode && (
                <Form.Item
                  name='verifyCode'
                  rules={[{ required: true, len: 6, message: t('verifyCodeRequired') }]}
                >
                  <Space.Compact style={{ width: '100%' }}>
                    <Input
                      prefix={<SafetyCertificateOutlined style={{ color: token.colorTextQuaternary }} />}
                      placeholder={t('verifyCodePlaceholder')}
                      maxLength={6}
                      autoComplete='one-time-code'
                    />
                    <Button disabled={cooldown > 0} onClick={sendCode}>
                      {cooldown > 0 ? t('sendCodeRetry', { seconds: cooldown }) : t('sendCode')}
                    </Button>
                  </Space.Compact>
                </Form.Item>
              )}
            </>
          )}

          <Form.Item style={{ marginBottom: 0 }}>
            <Button
              type='primary'
              htmlType='submit'
              block
              loading={submitting}
              style={{ height: 44, fontSize: 15, fontWeight: 500 }}
            >
              {t('securityUpdateSubmit')}
            </Button>
          </Form.Item>
        </Form>

        <Flex justify='center' align='center' gap={token.marginXXS}>
          <LockOutlined style={{ color: token.colorTextQuaternary, fontSize: 11 }} />
          <Text type='secondary' style={{ fontSize: 11 }}>
            {t('secureTip')}
          </Text>
        </Flex>
      </Flex>
    </SecurityFlowLayout>
  )
}
