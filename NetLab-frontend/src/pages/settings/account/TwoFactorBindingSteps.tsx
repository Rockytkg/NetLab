import { useEffect, useRef, useState } from 'react'
import { Alert, App, Button, Card, Checkbox, Col, Flex, Grid, Image, Input, Row, Skeleton, Space, Steps, Typography, theme } from 'antd'
import { SafetyOutlined, DownloadOutlined, CopyOutlined, LoadingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { accountApi } from '@/services/account'
import type { TwoFactorSetupResult } from '@/types/auth'

const { Text } = Typography
const { useBreakpoint } = Grid

interface TwoFactorBindingStepsProps {
  onComplete: () => void
}

/** 两步验证绑定流程：展示二维码 + 密钥，校验动态码后启用，并展示一次性恢复码。 */
export default function TwoFactorBindingSteps({ onComplete }: TwoFactorBindingStepsProps) {
  const { t } = useTranslation('settings')
  const { token } = theme.useToken()
  const screens = useBreakpoint()
  const isCompact = !screens.sm
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
        const result = await accountApi.beginTwoFactorSetup()
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
      const result = await accountApi.confirmTwoFactorSetup(value)
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

  const stepsItems = [
    { title: t('twoFactor.stepScan') },
    { title: t('twoFactor.stepVerify') },
    { title: t('twoFactor.stepRecovery') },
  ]

  if (step === 2) {
    return (
      <Flex vertical gap={token.marginLG}>
        <Steps size='small' current={2} items={stepsItems} />
        <Alert type='warning' showIcon title={t('twoFactor.recoveryNotice')} />
        <Card size='small' variant='borderless' style={{ background: token.colorFillAlter }}>
          <Row gutter={[token.marginSM, token.marginSM]}>
            {recoveryCodes.map((c) => (
              <Col key={c} xs={24} sm={12}>
                <Text code>{c}</Text>
              </Col>
            ))}
          </Row>
        </Card>
        <Space wrap>
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
      </Flex>
    )
  }

  return (
    <Flex vertical gap={token.marginLG}>
      <Steps size='small' current={step} items={stepsItems} />

      {loading || !setup ? (
        <Flex justify='center'>
          <Skeleton.Image active style={{ width: 200, height: 200 }} />
        </Flex>
      ) : (
        <>
          <Flex justify='center'>
            <Image
              src={setup.qrCode}
              alt={t('twoFactor.qrAlt')}
              width={184}
              height={184}
              preview={false}
              style={{
                // 二维码需白底以保证暗色模式下可扫描
                padding: token.paddingSM,
                background: '#ffffff',
                border: `1px solid ${token.colorBorderSecondary}`,
                borderRadius: token.borderRadiusLG,
              }}
            />
          </Flex>

          <Flex vertical gap={token.marginXXS}>
            <Text type='secondary'>{t('twoFactor.manualEntry')}</Text>
            <Text code copyable>
              {formattedSecret}
            </Text>
          </Flex>

          <Flex vertical align='center' gap={token.marginXS}>
            <Input.OTP
              size={isCompact ? 'middle' : 'large'}
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
            {submitting ? (
              <Text type='secondary'>
                <LoadingOutlined /> {t('twoFactor.verifying')}
              </Text>
            ) : errorStatus === 'error' ? (
              <Text type='danger'>{t('twoFactor.codeInvalid')}</Text>
            ) : (
              <Text type='secondary'>{t('twoFactor.autoVerifyHint')}</Text>
            )}
          </Flex>
        </>
      )}
    </Flex>
  )
}
