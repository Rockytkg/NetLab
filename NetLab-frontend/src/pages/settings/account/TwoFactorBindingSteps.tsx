import { useEffect, useRef, useState } from 'react'
import { Alert, App, Button, Checkbox, Image, Input, Skeleton, Space, Steps, Typography, theme } from 'antd'
import { SafetyOutlined, DownloadOutlined, CopyOutlined, LoadingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { authApi } from '@/services/auth'
import type { TwoFactorSetupResult } from '@/types/auth'

const { Text, Paragraph } = Typography

interface TwoFactorBindingStepsProps {
  onComplete: () => void
}

/** 两步验证绑定流程：展示二维码 + 密钥，校验动态码后启用，并展示一次性恢复码。 */
export default function TwoFactorBindingSteps({ onComplete }: TwoFactorBindingStepsProps) {
  const { t } = useTranslation('settings')
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const [setup, setSetup] = useState<TwoFactorSetupResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [code, setCode] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [errorStatus, setErrorStatus] = useState<'' | 'error'>('')
  const [step, setStep] = useState(0)
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const [saved, setSaved] = useState(false)
  const submittingRef = useRef(false)

  useEffect(() => {
    let alive = true
    ;(async () => {
      try {
        const result = await authApi.beginTwoFactorSetup()
        if (alive) setSetup(result)
      } catch {
        // interceptor
      } finally {
        if (alive) setLoading(false)
      }
    })()
    return () => {
      alive = false
    }
  }, [])

  const submit = async (value: string) => {
    if (value.length !== 6 || submittingRef.current) return
    submittingRef.current = true
    setErrorStatus('')
    setSubmitting(true)
    try {
      const result = await authApi.confirmTwoFactorSetup(value)
      message.success(t('twoFactor.enableSuccess'))
      setRecoveryCodes(result.recoveryCodes ?? [])
      setStep(2)
    } catch {
      // 校验失败：清空输入并置为错误状态
      setErrorStatus('error')
      setCode('')
    } finally {
      setSubmitting(false)
      submittingRef.current = false
    }
  }

  // 输入满 6 位后自动提交
  useEffect(() => {
    if (step === 1 && code.length === 6) {
      void submit(code)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code, step])

  const handleDownload = () => {
    const text = recoveryCodes.map((c, i) => `${i + 1}. ${c}`).join('\n')
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'netlab-recovery-codes.txt'
    a.click()
    URL.revokeObjectURL(url)
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(recoveryCodes.join('\n'))
      message.success(t('twoFactor.recoveryCopied'))
    } catch {
      // ignore
    }
  }

  const formattedSecret = setup ? setup.secret.replace(/(.{4})/g, '$1 ').trim() : ''

  if (step === 2) {
    return (
      <Space orientation='vertical' size={token.marginLG} style={{ width: '100%' }}>
        <Steps
          size='small'
          current={2}
          items={[
            { title: t('twoFactor.stepScan') },
            { title: t('twoFactor.stepVerify') },
            { title: t('twoFactor.stepRecovery') },
          ]}
        />
        <Alert type='warning' showIcon title={t('twoFactor.recoveryNotice')} />
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))',
            gap: token.marginSM,
            background: token.colorFillQuaternary,
            padding: token.padding,
            borderRadius: token.borderRadius,
            fontFamily: 'monospace',
          }}
        >
          {recoveryCodes.map((c) => (
            <Text key={c} code style={{ fontSize: 13 }}>
              {c}
            </Text>
          ))}
        </div>
        <Space>
          <Button icon={<DownloadOutlined />} onClick={handleDownload}>
            {t('twoFactor.recoveryDownload')}
          </Button>
          <Button icon={<CopyOutlined />} onClick={handleCopy}>
            {t('twoFactor.recoveryCopy')}
          </Button>
        </Space>
        <Checkbox checked={saved} onChange={(e) => setSaved(e.target.checked)}>
          {t('twoFactor.recoveryConfirm')}
        </Checkbox>
        <Button type='primary' block icon={<SafetyOutlined />} disabled={!saved} onClick={onComplete}>
          {t('twoFactor.recoveryComplete')}
        </Button>
      </Space>
    )
  }

  return (
    <Space orientation='vertical' size={token.marginLG} style={{ width: '100%' }}>
      <Steps
        size='small'
        current={step}
        items={[
          { title: t('twoFactor.stepScan') },
          { title: t('twoFactor.stepVerify') },
          { title: t('twoFactor.stepRecovery') },
        ]}
      />

      {loading || !setup ? (
        <div style={{ display: 'flex', justifyContent: 'center' }}>
          <Skeleton.Image active style={{ width: 200, height: 200 }} />
        </div>
      ) : (
        <>
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <Image src={setup.qrCode} alt={t('twoFactor.qrAlt')} width={200} height={200} preview={false} />
          </div>

          <div>
            <Text type='secondary'>{t('twoFactor.manualEntry')}</Text>
            <Paragraph style={{ marginTop: token.marginXS, marginBottom: 0 }}>
              <Text code copyable>
                {formattedSecret}
              </Text>
            </Paragraph>
          </div>

          <div>
            <div style={{ marginTop: token.marginXS, display: 'flex', justifyContent: 'center' }}>
              <Input.OTP
                length={6}
                value={code}
                status={errorStatus || undefined}
                disabled={submitting}
                onChange={(val) => {
                  setStep(1)
                  setErrorStatus('')
                  setCode(val)
                }}
              />
            </div>
            {submitting ? (
              <div style={{ textAlign: 'center', marginTop: token.marginXS }}>
                <Text type='secondary' style={{ fontSize: 12 }}>
                  <LoadingOutlined style={{ marginRight: token.marginXS }} />
                  {t('twoFactor.verifying')}
                </Text>
              </div>
            ) : errorStatus === 'error' ? (
              <div style={{ textAlign: 'center', marginTop: token.marginXS }}>
                <Text type='danger' style={{ fontSize: 12 }}>
                  {t('twoFactor.codeInvalid')}
                </Text>
              </div>
            ) : (
              <div style={{ textAlign: 'center', marginTop: token.marginXS }}>
                <Text type='secondary' style={{ fontSize: 12 }}>
                  {t('twoFactor.autoVerifyHint')}
                </Text>
              </div>
            )}
          </div>
        </>
      )}
    </Space>
  )
}
