import { Alert, App, Button, Card, Col, Form, Input, Row, Space, Steps, Typography, theme } from 'antd'
import { LockOutlined, MailOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'

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
      const user = await authApi.completeSecurityUpdate(values)
      useAuthStore.setState({ userInfo: user, securityActions: null })
      message.success(t('securityUpdateSuccess'))
      await useAuthStore.getState().logout({ callApi: false })
      navigate('/login', { replace: true })
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'grid',
        placeItems: 'center',
        padding: token.paddingLG,
        background: token.colorBgLayout,
      }}
    >
      <div
        style={{
          width: '100%',
          maxWidth: 960,
        }}
      >
        <Row gutter={[24, 24]} align='stretch'>
          <Col xs={24} md={10}>
            <Card style={{ height: '100%' }}>
              <Space orientation='vertical' size={24} style={{ width: '100%' }}>
                <div>
                  <SafetyCertificateOutlined style={{ fontSize: 40, color: token.colorPrimary }} />
                  <Title level={2} style={{ marginTop: 16, marginBottom: 8 }}>
                    {t('securityUpdateTitle')}
                  </Title>
                  <Text type='secondary'>{t('securityUpdateSubtitle', { username: userInfo?.username ?? '' })}</Text>
                </div>
                <Steps
                  orientation='vertical'
                  current={0}
                  items={[
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
                />
              </Space>
            </Card>
          </Col>

          <Col xs={24} md={14}>
            <Card>
              <Space orientation='vertical' size={24} style={{ width: '100%' }}>
                <div>
                  <Title level={4} style={{ marginBottom: 8 }}>
                    {needEmail ? t('securityUpdateFormTitleFull') : t('securityUpdateFormTitle')}
                  </Title>
                  <Text type='secondary'>{t('securityUpdateNotice')}</Text>
                </div>

                <Alert
                  type='warning'
                  showIcon
                  title={
                    needEmail
                      ? t(isDefaultAdmin ? 'defaultAdminNotice' : 'firstLoginNotice')
                      : t(actions?.reason === 'password_expired' ? 'passwordExpiredNotice' : 'passwordResetNotice')
                  }
                />

                <Form form={form} layout='vertical' requiredMark={false} onFinish={submit}>
                  <Form.Item
                    name='newPassword'
                    label={t('newPassword')}
                    rules={[
                      { required: true, message: t('newPasswordRequired') },
                      { min: 8, message: t('passwordMinLength') },
                    ]}
                  >
                    <Input.Password prefix={<LockOutlined />} autoComplete='new-password' maxLength={128} />
                  </Form.Item>

                  <Form.Item
                    name='confirmPassword'
                    label={t('confirmPassword')}
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
                    <Input.Password prefix={<LockOutlined />} autoComplete='new-password' maxLength={128} />
                  </Form.Item>

                  {needEmail && (
                    <>
                      <Form.Item
                        name='newEmail'
                        label={t('newEmail')}
                        rules={[{ required: true, type: 'email', message: t('emailInvalid') }]}
                      >
                        <Input prefix={<MailOutlined />} autoComplete='email' maxLength={255} />
                      </Form.Item>

                      {needEmailCode && (
                        <Form.Item
                          name='verifyCode'
                          label={t('verifyCode')}
                          rules={[{ required: true, len: 6, message: t('verifyCodeRequired') }]}
                        >
                          <Space.Compact style={{ width: '100%' }}>
                            <Input maxLength={6} autoComplete='one-time-code' />
                            <Button disabled={cooldown > 0} onClick={sendCode}>
                              {cooldown > 0 ? t('sendCodeRetry', { seconds: cooldown }) : t('sendCode')}
                            </Button>
                          </Space.Compact>
                        </Form.Item>
                      )}
                    </>
                  )}

                  <Button type='primary' htmlType='submit' block loading={submitting}>
                    {t('securityUpdateSubmit')}
                  </Button>
                </Form>
              </Space>
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  )
}
