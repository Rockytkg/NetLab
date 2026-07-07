import { useState, useRef, useCallback } from 'react'
import {
  Modal,
  Form,
  Input,
  Button,
  Steps,
  Result,
  App,
  Typography,
  theme,
} from 'antd'
import { MailOutlined, LockOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { authApi } from '@/services/auth'
import { isAuthSecurityError } from '@/services/authSecurity'

const { Text } = Typography

interface ForgotPasswordModalProps {
  open: boolean
  onClose: () => void
}

type StepKey = 'verify' | 'reset' | 'done'

/** 第一步的内部组件 —— 需要访问父级 Form 实例 */
function Step1Form({ t, themeToken, loading, cooldown, onSendCode, onFinish }: {
  t: (key: string, opts?: Record<string, unknown>) => string
  themeToken: ReturnType<typeof theme.useToken>['token']
  loading: boolean
  cooldown: number
  onSendCode: (email: string) => Promise<void>
  onFinish: (values: { email: string; verifyCode: string }) => void
}) {
  const [form] = Form.useForm()

  const handleSendCodeLocal = useCallback(async () => {
    try {
      await form.validateFields(['email'])
    } catch {
      return
    }
    const emailValue = form.getFieldValue('email')
    await onSendCode(emailValue)
  }, [form, onSendCode])

  return (
    <Form form={form} size="large" layout="vertical" onFinish={onFinish}>
      <Form.Item
        name="email"
        rules={[
          { required: true, message: t('emailRequired') },
          { type: 'email', message: t('emailInvalid') },
        ]}
      >
        <Input
          prefix={<MailOutlined style={{ color: themeToken.colorTextQuaternary }} />}
          placeholder={t('emailPlaceholder')}
          autoComplete="email"
        />
      </Form.Item>

      <Form.Item
        name="verifyCode"
        rules={[{ required: true, message: t('verifyCodeRequired') }]}
      >
        <Input
          prefix={<SafetyCertificateOutlined style={{ color: themeToken.colorTextQuaternary }} />}
          placeholder={t('verifyCodePlaceholder')}
          suffix={
            <Button
              type="link"
              size="small"
              loading={loading}
              disabled={cooldown > 0}
              onClick={handleSendCodeLocal}
              style={{ padding: 0, fontSize: 12 }}
            >
              {cooldown > 0 ? t('sendCodeRetry', { seconds: cooldown }) : t('sendCode')}
            </Button>
          }
        />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <Button type="primary" htmlType="submit" block loading={loading}>
          {t('forgotPasswordStep1')}
        </Button>
      </Form.Item>
    </Form>
  )
}

export default function ForgotPasswordModal({ open, onClose }: ForgotPasswordModalProps) {
  const { t } = useTranslation('login')
  const { message } = App.useApp()
  const { token: themeToken } = theme.useToken()
  const [currentStep, setCurrentStep] = useState<StepKey>('verify')
  const [email, setEmail] = useState('')
  const [verifiedCode, setVerifiedCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [cooldown, setCooldown] = useState(0)
  const timerRef = useRef<ReturnType<typeof setInterval>>()

  // ── 打开/关闭时重置 ──
  const handleClose = useCallback(() => {
    setCurrentStep('verify')
    setEmail('')
    setVerifiedCode('')
    setCooldown(0)
    if (timerRef.current) clearInterval(timerRef.current)
    onClose()
  }, [onClose])

  // ── 发送验证码 ──
  const handleSendCode = useCallback(async (emailValue: string) => {
    if (cooldown > 0) return
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailValue)) {
      message.warning(t('emailInvalid'))
      return
    }
    setLoading(true)
    try {
      await authApi.sendCode({ email: emailValue, purpose: 'reset-password' })
      message.success(t('sendCodeSuccess'))
      const cd = 60
      setCooldown(cd)
      timerRef.current = setInterval(() => {
        setCooldown((prev) => {
          if (prev <= 1) {
            if (timerRef.current) clearInterval(timerRef.current)
            return 0
          }
          return prev - 1
        })
      }, 1000)
    } catch {
      // HTTP/业务错误已由 Axios 拦截器（request.ts）弹出提示
    } finally {
      setLoading(false)
    }
  }, [cooldown, t, message])

  // ── 第一步 → 第二步：向后端校验验证码 ──
  // 重要：此处不要对 HTTP/业务错误调用 message.error()。
  // Axios 响应拦截器（request.ts:349-369）已经为所有 HTTP 错误
  // （404、500 等）和业务错误（code !== 0）弹出提示。
  // 若在此再弹一次提示，会导致一次失败出现两条消息。
  // 只有 AuthSecurityError（客户端、HTTP 之前）需要手动弹出提示。
  const handleVerify = useCallback(async (values: { email: string; verifyCode: string }) => {
    setLoading(true)
    try {
      const result = await authApi.verifyCode({
        email: values.email,
        code: values.verifyCode,
        purpose: 'reset-password',
      })
      if (!result.valid) {
        // 业务逻辑拒绝 —— API 成功返回（HTTP 200）
        // 但 valid=false。这不是 HTTP 错误，因此拦截器
        // 不会弹出提示。我们必须在此弹出提示。
        message.error(result.message || t('verifyCodeInvalid'))
        return
      }
      setEmail(values.email)
      setVerifiedCode(values.verifyCode)
      setCurrentStep('reset')
    } catch (error) {
      // 仅处理客户端抛出的 AuthSecurityError（缺少环境变量）。
      // 响应信封中的 HTTP 错误和业务错误
      // （code !== 0/200）已由拦截器弹出提示。
      if (isAuthSecurityError(error)) {
        message.error(error.message)
      }
    } finally {
      setLoading(false)
    }
  }, [t, message])

  // ── 第二步 → 第三步：重置密码 ──
  const handleReset = useCallback(async (values: { newPassword: string }) => {
    setLoading(true)
    try {
      await authApi.resetPassword({
        email,
        verifyCode: verifiedCode,
        newPassword: values.newPassword,
        confirmPassword: values.newPassword,
      })
      setCurrentStep('done')
    } catch (error) {
      // 同样的规则：拦截器处理 HTTP/业务错误。
      // 只有 AuthSecurityError 需要手动弹出提示。
      if (isAuthSecurityError(error)) {
        message.error(error.message)
      }
    } finally {
      setLoading(false)
    }
  }, [email, verifiedCode, message])

  // ── 步骤指示器 ──
  const stepItems = [
    { title: t('forgotPasswordStep1') },
    { title: t('forgotPasswordStep2') },
    { title: t('forgotPasswordStep3') },
  ]
  const stepIndex = currentStep === 'verify' ? 0 : currentStep === 'reset' ? 1 : 2

  return (
    <Modal
      title={null}
      open={open}
      onCancel={handleClose}
      footer={null}
      width={440}
      centered
      destroyOnHidden
    >
      <div style={{ padding: '8px 0' }}>
        {/* 头部 */}
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Typography.Title level={4} style={{ marginBottom: 4 }}>
            {t('forgotPasswordTitle')}
          </Typography.Title>
          <Text type="secondary">{t('forgotPasswordSubtitle')}</Text>
        </div>

        <Steps
          current={stepIndex}
          items={stepItems}
          size="small"
          style={{ marginBottom: 32 }}
        />

        {/* ── 第一步：邮箱 + 发送验证码 ── */}
        {currentStep === 'verify' && (
          <Step1Form
            t={t}
            themeToken={themeToken}
            loading={loading}
            cooldown={cooldown}
            onSendCode={handleSendCode}
            onFinish={handleVerify}
          />
        )}

        {/* ── 第二步：重置密码（验证码已在第一步校验） ── */}
        {currentStep === 'reset' && (
          <Form size="large" layout="vertical" onFinish={handleReset}>
            <Form.Item
              name="newPassword"
              rules={[
                { required: true, message: t('newPasswordRequired') },
                { min: 8, message: t('passwordMinLength') },
              ]}
            >
              <Input.Password
                prefix={<LockOutlined style={{ color: themeToken.colorTextQuaternary }} />}
                placeholder={t('newPasswordPlaceholder')}
                autoComplete="new-password"
              />
            </Form.Item>

            <Form.Item style={{ marginBottom: 0 }}>
              <Button type="primary" htmlType="submit" block loading={loading}>
                {t('forgotPasswordStep2')}
              </Button>
            </Form.Item>
          </Form>
        )}

        {/* ── 第三步：完成 ── */}
        {currentStep === 'done' && (
          <Result
            status="success"
            title={t('resetPasswordSuccess')}
            extra={
              <Button type="primary" onClick={handleClose}>
                {t('backToLogin')}
              </Button>
            }
          />
        )}
      </div>
    </Modal>
  )
}
