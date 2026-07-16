import { useCallback, useState } from 'react'
import { Button, Divider, Flex, Typography, Tooltip, App, theme, Modal, Tabs, Form, Input, Space } from 'antd'
import {
  GithubOutlined,
  GoogleOutlined,
  QqOutlined,
  WechatOutlined,
  MailOutlined,
  UserOutlined,
  LockOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import type { LoginResult, OAuthProvider, PendingOAuthBinding } from '@/types/auth'
import { authApi } from '@/services/auth'
import { completeLogin } from '@/utils/auth-flow'
import PasskeyButton from './PasskeyButton'
import LinuxDoIcon from '@/components/common/icons/LinuxDoIcon'

const { Text } = Typography

/** Slightly larger than Ant Design's default large circle (40px) for comfortable tap area */
const CIRCLE_BUTTON_SIZE = 44

const PROVIDER_ICONS: Record<string, React.ReactNode> = {
  github: <GithubOutlined />,
  google: <GoogleOutlined />,
  qq: <QqOutlined />,
  wechat: <WechatOutlined />,
  linuxdo: <LinuxDoIcon />,
}

interface OAuthSectionProps {
  providers: OAuthProvider[]
  passkeyEnabled?: boolean
}

export default function OAuthSection({ providers, passkeyEnabled = false }: OAuthSectionProps) {
  const { t } = useTranslation('login')
  const { message } = App.useApp()
  const { token } = theme.useToken()
  const navigate = useNavigate()
  const [pending, setPending] = useState<PendingOAuthBinding | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [codeCooldown, setCodeCooldown] = useState(0)
  const [activeTab, setActiveTab] = useState('existing')
  const [existingForm] = Form.useForm()
  const [createForm] = Form.useForm()

  const handleLoginResult = useCallback((result: LoginResult) => {
    if (result.pendingOAuthBinding) {
      setPending(result.pendingOAuthBinding)
      setActiveTab('existing')
      message.info(t('oauthBindingRequired'))
      return
    }
    if (completeLogin(result, navigate)) {
      message.success(t('loginSuccess'))
    }
  }, [message, navigate, t])

  const handleOAuthLogin = useCallback((provider: OAuthProvider) => {
    const width = 600; const height = 700
    const left = window.screenX + (window.outerWidth - width) / 2
    const top = window.screenY + (window.outerHeight - height) / 2

    const popup = window.open(
      provider.authUrl,
      `oauth-${provider.id}`,
      `width=${width},height=${height},left=${left},top=${top}`
    )

    if (!popup) {
      window.location.href = provider.authUrl
      return
    }

    const handleMessage = async (event: MessageEvent) => {
      if (event.data?.type !== 'oauth-callback' || !event.data?.code || !event.data?.state) return
      window.removeEventListener('message', handleMessage)
      popup?.close()

      try {
        const result = await authApi.oauthCallback({
          provider: provider.id,
          code: event.data.code,
          state: event.data.state,
        })
        handleLoginResult(result)
      } catch {
        message.error(t('loginFailed'))
      }
    }
    window.addEventListener('message', handleMessage)

    const checkClosed = setInterval(() => {
      if (popup.closed) { clearInterval(checkClosed); window.removeEventListener('message', handleMessage) }
    }, 500)
  }, [handleLoginResult, t, message])

  const requestEmailCode = async (purpose: 'register' | 'change-email') => {
    const form = purpose === 'register' ? createForm : existingForm
    const email = form.getFieldValue('email')
    if (!email) {
      message.warning(t('emailRequired'))
      return
    }
    try {
      const result = await authApi.sendCode({ email, purpose })
      setCodeCooldown(result.cooldown || 60)
      message.success(t('sendCodeSuccess'))
      const timer = window.setInterval(() => {
        setCodeCooldown((value) => {
          if (value <= 1) {
            window.clearInterval(timer)
            return 0
          }
          return value - 1
        })
      }, 1000)
    } catch {
      /* handled by interceptor */
    }
  }

  const bindExisting = async (values: { account: string; verifyCode: string }) => {
    if (!pending) return
    setSubmitting(true)
    try {
      const result = await authApi.oauthBindExisting({
        pendingToken: pending.token,
        account: values.account,
        verifyCode: values.verifyCode,
      })
      setPending(null)
      handleLoginResult(result)
    } finally {
      setSubmitting(false)
    }
  }

  const createAccount = async (values: {
    username: string
    email: string
    password: string
    confirmPassword: string
    verifyCode: string
  }) => {
    if (!pending) return
    setSubmitting(true)
    try {
      const result = await authApi.oauthCreateAccount({
        pendingToken: pending.token,
        ...values,
      })
      setPending(null)
      handleLoginResult(result)
    } finally {
      setSubmitting(false)
    }
  }

  const hasPasskey = passkeyEnabled
  const hasOAuth = providers.length > 0

  if (!hasPasskey && !hasOAuth) return null

  return (
    <>
      <Divider plain className="netlab-login-oauth-divider" style={{ marginBlock: token.marginLG, marginBottom: token.margin }}>
        <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>{t('orSignInWith')}</Text>
      </Divider>

      <Flex justify="center" gap={token.padding} className="netlab-login-oauth-buttons">
        {hasPasskey && <PasskeyButton variant="circle" />}
        {providers.map((provider) => (
          <Tooltip key={provider.id} title={t(`oauth_${provider.id}`, provider.name)}>
            <Button
              shape="circle"
              size="large"
              icon={PROVIDER_ICONS[provider.id]}
              onClick={() => handleOAuthLogin(provider)}
              aria-label={t(`oauth_${provider.id}`, provider.name)}
              style={{ width: CIRCLE_BUTTON_SIZE, height: CIRCLE_BUTTON_SIZE, fontSize: token.fontSizeLG }}
            />
          </Tooltip>
        ))}
      </Flex>

      <Modal
        title={t('oauthBindingTitle')}
        open={!!pending}
        onCancel={() => setPending(null)}
        footer={null}
        destroyOnHidden
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: token.margin }}>
          {t('oauthBindingSubtitle', { provider: pending?.provider ?? '' })}
        </Typography.Paragraph>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'existing',
              label: t('oauthBindExisting'),
              children: (
                <Form form={existingForm} layout="vertical" requiredMark={false} onFinish={bindExisting}>
                  <Form.Item name="account" label={t('oauthAccount')} rules={[{ required: true, message: t('oauthAccountRequired') }]}>
                    <Input prefix={<UserOutlined />} autoComplete="username" />
                  </Form.Item>
                  <Form.Item name="email" label={t('email')} rules={[{ required: true, type: 'email', message: t('emailInvalid') }]}>
                    <Input prefix={<MailOutlined />} autoComplete="email" />
                  </Form.Item>
                  <Form.Item name="verifyCode" label={t('verifyCode')} rules={[{ required: true, message: t('verifyCodeRequired') }]}>
                    <Space.Compact style={{ width: '100%' }}>
                      <Input maxLength={6} autoComplete="one-time-code" />
                      <Button disabled={codeCooldown > 0} onClick={() => requestEmailCode('change-email')}>
                        {codeCooldown > 0 ? t('sendCodeRetry', { seconds: codeCooldown }) : t('sendCode')}
                      </Button>
                    </Space.Compact>
                  </Form.Item>
                  <Button type="primary" htmlType="submit" block loading={submitting}>
                    {t('oauthBindAndLogin')}
                  </Button>
                </Form>
              ),
            },
            {
              key: 'create',
              label: t('oauthCreateAccount'),
              children: (
                <Form form={createForm} layout="vertical" requiredMark={false} onFinish={createAccount}>
                  <Form.Item name="username" label={t('username')} rules={[{ required: true, message: t('usernameRequired') }]}>
                    <Input prefix={<UserOutlined />} autoComplete="username" />
                  </Form.Item>
                  <Form.Item name="email" label={t('email')} rules={[{ required: true, type: 'email', message: t('emailInvalid') }]}>
                    <Input prefix={<MailOutlined />} autoComplete="email" />
                  </Form.Item>
                  <Form.Item name="password" label={t('password')} rules={[{ required: true, message: t('passwordRequired') }, { min: 8, message: t('passwordMinLength') }]}>
                    <Input.Password prefix={<LockOutlined />} autoComplete="new-password" />
                  </Form.Item>
                  <Form.Item
                    name="confirmPassword"
                    label={t('confirmPassword')}
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
                    <Input.Password prefix={<LockOutlined />} autoComplete="new-password" />
                  </Form.Item>
                  <Form.Item name="verifyCode" label={t('verifyCode')} rules={[{ required: true, message: t('verifyCodeRequired') }]}>
                    <Space.Compact style={{ width: '100%' }}>
                      <Input maxLength={6} autoComplete="one-time-code" />
                      <Button disabled={codeCooldown > 0} onClick={() => requestEmailCode('register')}>
                        {codeCooldown > 0 ? t('sendCodeRetry', { seconds: codeCooldown }) : t('sendCode')}
                      </Button>
                    </Space.Compact>
                  </Form.Item>
                  <Button type="primary" htmlType="submit" block loading={submitting}>
                    {t('oauthCreateAndLogin')}
                  </Button>
                </Form>
              ),
            },
          ]}
        />
      </Modal>
    </>
  )
}
