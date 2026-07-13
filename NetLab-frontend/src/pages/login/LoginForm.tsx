import { useState, useCallback, useEffect } from 'react'
import { Form, Input, Button, Checkbox, App, theme, Image, Typography, Tooltip } from 'antd'
import { UserOutlined, LockOutlined, SafetyCertificateOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { authApi } from '@/services/auth'
import { isAuthSecurityError } from '@/services/authSecurity'
import type { LoginParams, SystemConfig } from '@/types/auth'
import { navigateAfterLogin } from '@/utils/auth-flow'

const { Text } = Typography

interface LoginFormProps {
  config: SystemConfig | null
  onForgotPassword: () => void
  onRegister: () => void
}

export default function LoginForm({ config, onForgotPassword, onRegister }: LoginFormProps) {
  const { t } = useTranslation('login')
  const { login } = useAuth()
  const navigate = useNavigate()
  const { message } = App.useApp()
  const { token: themeToken } = theme.useToken()
  const [loading, setLoading] = useState(false)

  // captcha 状态
  const [captchaImage, setCaptchaImage] = useState<string | null>(null)
  const [captchaId, setCaptchaId] = useState<string | null>(null)
  const [captchaLoading, setCaptchaLoading] = useState(false)

  const captchaEnabled = config?.captchaEnabled ?? false
  const registrationEnabled = config?.registrationEnabled ?? false
  const passwordResetEnabled = config?.passwordResetEnabled ?? true

  const fetchCaptcha = useCallback(async () => {
    if (!captchaEnabled) return
    setCaptchaLoading(true)
    try {
      const result = await authApi.getCaptcha()
      setCaptchaId(result.captchaId)
      setCaptchaImage(result.captchaImage)
    } catch { /* handled by interceptor */ }
    finally { setCaptchaLoading(false) }
  }, [captchaEnabled])

  // 如果启用则在挂载时获取 captcha
  useEffect(() => {
    if (captchaEnabled) fetchCaptcha()
  }, [captchaEnabled, fetchCaptcha])

  const onFinish = async (values: LoginParams & { captchaCode?: string }) => {
    setLoading(true)
    try {
      const result = await login({
        username: values.username,
        password: values.password,
        captchaId: captchaId ?? undefined,
        captchaCode: values.captchaCode,
      })
      if (result.requiresTwoFactor) {
        navigate('/login/2fa', {
          state: { twoFactorToken: result.twoFactorToken, username: result.user?.username },
        })
        return
      }
      message.success(t('loginSuccess'))
      navigateAfterLogin(result, navigate)
    } catch (error) {
      if (isAuthSecurityError(error)) message.error(error.message)
      if (captchaEnabled) fetchCaptcha()
    } finally {
      setLoading(false)
    }
  }

  return (
    <Form
      name="login"
      size="large"
      layout="vertical"
      requiredMark={false}
      onFinish={onFinish}
      initialValues={{ remember: true }}
      style={{ width: '100%' }}
    >
      <Form.Item
        name="username"
        rules={[{ required: true, message: t('usernameRequired') }]}
      >
        <Input
          prefix={<UserOutlined style={{ color: themeToken.colorTextQuaternary }} />}
          placeholder={t('usernamePlaceholder')}
          autoComplete="username"
          autoFocus
        />
      </Form.Item>

      <Form.Item
        name="password"
        rules={[{ required: true, message: t('passwordRequired') }]}
      >
        <Input.Password
          prefix={<LockOutlined style={{ color: themeToken.colorTextQuaternary }} />}
          placeholder={t('passwordPlaceholder')}
          autoComplete="current-password"
        />
      </Form.Item>

      {captchaEnabled && (
        <Form.Item
          name="captchaCode"
          rules={[{ required: true, message: t('captchaRequired') }]}
        >
          <Input
            className="netlab-login-captcha-input"
            prefix={<SafetyCertificateOutlined style={{ color: themeToken.colorTextQuaternary }} />}
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
      )}

      <Form.Item style={{ marginBottom: 8 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
          <Form.Item name="remember" valuePropName="checked" noStyle>
            <Checkbox style={{ fontSize: 13 }}>{t('remember')}</Checkbox>
          </Form.Item>
          {passwordResetEnabled && (
            <Button type="link" size="small" onClick={onForgotPassword} style={{ fontSize: 13, padding: 0 }}>
              {t('forgotPassword')}
            </Button>
          )}
        </div>
      </Form.Item>

      <Form.Item style={{ marginBottom: registrationEnabled ? 4 : 0 }}>
        <Button type="primary" htmlType="submit" block loading={loading} style={{ height: 44, fontSize: 15, fontWeight: 500 }}>
          {t('submit')}
        </Button>
      </Form.Item>

      {registrationEnabled && (
        <div style={{ textAlign: 'center' }}>
          <Text type="secondary" style={{ fontSize: 13 }}>
            {t('noAccount')}{' '}
            <Button type="link" size="small" onClick={onRegister} style={{ fontSize: 13, padding: 0 }}>
              {t('registerNow')}
            </Button>
          </Text>
        </div>
      )}
    </Form>
  )
}
