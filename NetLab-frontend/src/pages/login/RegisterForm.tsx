import { useState, useRef, useCallback, useEffect } from 'react'
import { Form, Input, Button, App, Typography, theme, Image, Tooltip } from 'antd'
import {
  UserOutlined,
  LockOutlined,
  MailOutlined,
  SafetyCertificateOutlined,
  ReloadOutlined,
  NumberOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { authApi } from '@/services/auth'
import { isAuthSecurityError } from '@/services/authSecurity'
import { createPasswordStrengthRule } from '@/utils/password-strength'

const { Text } = Typography

interface RegisterFormProps {
  onBack: () => void
}

export default function RegisterForm({ onBack }: RegisterFormProps) {
  const { t } = useTranslation('login')
  const { message } = App.useApp()
  const { token: themeToken } = theme.useToken()
  const [loading, setLoading] = useState(false)
  const [cooldown, setCooldown] = useState(0)
  const [captchaId, setCaptchaId] = useState<string | null>(null)
  const [captchaImage, setCaptchaImage] = useState<string | null>(null)
  const [captchaLoading, setCaptchaLoading] = useState(false)
  const timerRef = useRef<ReturnType<typeof setInterval>>()
  const [form] = Form.useForm()

  const fetchCaptcha = useCallback(async () => {
    setCaptchaLoading(true)
    try {
      const result = await authApi.getCaptcha()
      setCaptchaId(result.captchaId)
      setCaptchaImage(result.captchaImage)
    } catch { /* handled by interceptor */ }
    finally { setCaptchaLoading(false) }
  }, [])

  useEffect(() => {
    fetchCaptcha()
  }, [fetchCaptcha])

  const handleSendCode = useCallback(async () => {
    if (cooldown > 0) return
    const email = form.getFieldValue('email')
    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      message.warning(t('emailInvalid'))
      return
    }
    let captchaCode: string
    try {
      captchaCode = await form.validateFields(['captchaCode']).then((values) => values.captchaCode)
    } catch {
      return
    }
    try {
      await authApi.sendCode({ email, purpose: 'register', captchaId: captchaId ?? undefined, captchaCode })
      message.success(t('sendCodeSuccess'))
      const cd = 60
      setCooldown(cd)
      timerRef.current = setInterval(() => {
        setCooldown((prev) => {
          if (prev <= 1) { if (timerRef.current) clearInterval(timerRef.current); return 0 }
          return prev - 1
        })
      }, 1000)
    } catch { /* handled by interceptor */ }
  }, [cooldown, form, captchaId, t, message])

  const onFinish = useCallback(async (values: {
    username: string; email: string; password: string
    confirmPassword: string; verifyCode: string
  }) => {
    setLoading(true)
    try {
      await authApi.register(values)
      message.success(t('registerSuccess'))
      onBack()
    } catch (error) {
      if (isAuthSecurityError(error)) message.error(error.message)
    } finally { setLoading(false) }
  }, [t, message, onBack])

  const iconStyle = { color: themeToken.colorTextQuaternary }

  return (
    <>
      <Form form={form} name="register" size="large" layout="vertical" requiredMark={false} onFinish={onFinish} className="netlab-login-form netlab-login-form-register">
        <Form.Item name="username" rules={[{ required: true, message: t('usernameRequired') }]}>
          <Input prefix={<UserOutlined style={iconStyle} />} placeholder={t('usernamePlaceholder')} autoComplete="username" />
        </Form.Item>

        <Form.Item name="email" rules={[{ required: true, message: t('emailRequired') }, { type: 'email', message: t('emailInvalid') }]}>
          <Input prefix={<MailOutlined style={iconStyle} />} placeholder={t('emailPlaceholder')} autoComplete="email" />
        </Form.Item>

        <Form.Item
          name="captchaCode"
          rules={[{ required: true, message: t('captchaRequired') }]}
        >
          <Input
            className="netlab-login-captcha-input"
            prefix={<SafetyCertificateOutlined style={iconStyle} />}
            placeholder={t('captchaPlaceholder')}
            autoComplete="off"
            suffix={
              captchaImage ? (
                <Tooltip title={t('clickToRefresh')}>
                  <Image
                    src={captchaImage}
                    alt="captcha"
                    height={32}
                    style={{ objectFit: 'contain', cursor: 'pointer' }}
                    preview={{
                      open: false,
                      cover: <ReloadOutlined spin={captchaLoading} />,
                      onOpenChange: () => fetchCaptcha(),
                    }}
                  />
                </Tooltip>
              ) : (
                <Button type="link" size="small" loading={captchaLoading} onClick={fetchCaptcha} style={{ padding: 0, fontSize: 12 }}>
                  {t('clickToRefresh')}
                </Button>
              )
            }
          />
        </Form.Item>

        <Form.Item
          name="verifyCode"
          rules={[
            { required: true, message: t('verifyCodeRequired') },
            { len: 6, message: t('verifyCodeInvalid') },
            { pattern: /^\d{6}$/, message: t('verifyCodeInvalid') },
          ]}
        >
          <Input
            prefix={<NumberOutlined style={iconStyle} />}
            placeholder={t('verifyCodePlaceholder')}
            maxLength={6}
            autoComplete="one-time-code"
            suffix={
              <Button type="link" size="small" disabled={cooldown > 0} onClick={handleSendCode} style={{ padding: 0, fontSize: 12 }}>
                {cooldown > 0 ? t('sendCodeRetry', { seconds: cooldown }) : t('sendCode')}
              </Button>
            }
          />
        </Form.Item>

        <Form.Item
          name="password"
          rules={[
            { required: true, message: t('passwordRequired') },
            createPasswordStrengthRule({
              t,
            }),
          ]}
        >
          <Input.Password prefix={<LockOutlined style={iconStyle} />} placeholder={t('passwordPlaceholder')} autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          dependencies={['password']}
          rules={[
            { required: true, message: t('confirmPasswordRequired') },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('password') === value) return Promise.resolve()
                return Promise.reject(new Error(t('passwordMismatch')))
              },
            }),
          ]}
        >
          <Input.Password prefix={<LockOutlined style={iconStyle} />} placeholder={t('confirmPasswordPlaceholder')} autoComplete="new-password" />
        </Form.Item>

        <Form.Item style={{ marginBottom: 4 }}>
          <Button type="primary" htmlType="submit" block loading={loading} style={{ height: 44, fontSize: 15, fontWeight: 500 }}>
            {t('registerTitle')}
          </Button>
        </Form.Item>

        <div style={{ textAlign: 'center' }}>
          <Text type="secondary" style={{ fontSize: 13 }}>
            {t('hasAccount')}{' '}
            <Button type="link" size="small" onClick={onBack} style={{ fontSize: 13, padding: 0 }}>
              {t('backToLogin')}
            </Button>
          </Text>
        </div>
      </Form>
    </>
  )
}
