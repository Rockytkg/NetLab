import { useCallback, useEffect, useRef, useState } from 'react'
import { App, Button, Card, Flex, Grid, Input, Result, Segmented, Typography, theme } from 'antd'
import { KeyOutlined, LockOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import { navigateAfterLogin } from '@/utils/auth-flow'
import type { LoginResult } from '@/types/auth'

const { Title, Text } = Typography
const { useBreakpoint } = Grid

type VerifyMode = 'totp' | 'recovery'

interface LocationState {
  twoFactorToken?: string
  username?: string
}

/** 登录第二步：校验 TOTP 动态码或一次性恢复码并换取访问令牌。 */
export default function TwoFactorVerifyPage() {
  const { t } = useTranslation('login')
  const { token } = theme.useToken()
  const { message } = App.useApp()
  const navigate = useNavigate()
  const location = useLocation()
  const screens = useBreakpoint()
  const isCompact = !screens.md
  const state = (location.state ?? {}) as LocationState

  const [mode, setMode] = useState<VerifyMode>('totp')
  const [code, setCode] = useState('')
  const [recoveryCode, setRecoveryCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [errorStatus, setErrorStatus] = useState<'' | 'error'>('')
  const submittingRef = useRef(false)

  const finishLogin = useCallback(
    (result: LoginResult) => {
      if (!result.accessToken || !result.refreshToken || !result.user) {
        message.error(t('twoFactorFailed'))
        return
      }
      useAuthStore.setState({
        accessToken: result.accessToken,
        refreshToken: result.refreshToken,
        userInfo: result.user,
        securityActions: result.securityActions,
      })
      message.success(t('loginSuccess'))
      navigateAfterLogin(result, navigate)
    },
    [message, t, navigate],
  )

  const submitTotp = useCallback(
    async (value: string) => {
      if (!state.twoFactorToken || value.length !== 6 || submittingRef.current) return
      submittingRef.current = true
      setErrorStatus('')
      setLoading(true)
      try {
        const result = await authApi.verifyTwoFactorLogin({
          twoFactorToken: state.twoFactorToken,
          code: value,
        })
        finishLogin(result)
      } catch {
        // 校验失败：清空输入并置为错误状态
        setErrorStatus('error')
        setCode('')
      } finally {
        setLoading(false)
        submittingRef.current = false
      }
    },
    [state.twoFactorToken, finishLogin],
  )

  const submitRecovery = useCallback(async () => {
    if (!state.twoFactorToken) return
    const rc = recoveryCode.trim()
    if (!rc || loading) return
    setLoading(true)
    setErrorStatus('')
    try {
      const result = await authApi.verifyRecoveryLogin({
        twoFactorToken: state.twoFactorToken,
        recoveryCode: rc,
      })
      finishLogin(result)
    } catch {
      setErrorStatus('error')
      setRecoveryCode('')
    } finally {
      setLoading(false)
    }
  }, [state.twoFactorToken, recoveryCode, loading, finishLogin])

  // 输入满 6 位后自动提交，无需手动点击
  useEffect(() => {
    if (mode === 'totp' && code.length === 6) {
      void submitTotp(code)
    }
  }, [code, mode, submitTotp])

  const switchMode = (next: VerifyMode) => {
    setMode(next)
    setErrorStatus('')
    setCode('')
    setRecoveryCode('')
  }

  const cardStyle = {
    width: '100%',
    maxWidth: 400,
    background: token.colorBgElevated,
    boxShadow: token.boxShadowTertiary,
  }

  const pageStyle = {
    minHeight: '100dvh',
    padding: isCompact ? token.padding : token.paddingLG,
    background: token.colorBgLayout,
  }

  if (!state.twoFactorToken) {
    return (
      <Flex justify='center' align='center' style={pageStyle}>
        <Card variant='borderless' style={cardStyle}>
          <Result
            status='warning'
            title={t('twoFactorSessionExpired')}
            style={{ padding: `${token.paddingLG}px 0` }}
            extra={
              <Button type='primary' onClick={() => navigate('/login', { replace: true })}>
                {t('twoFactorBackToLogin')}
              </Button>
            }
          />
        </Card>
      </Flex>
    )
  }

  return (
    <Flex justify='center' align='center' style={pageStyle}>
      <Card
        variant='borderless'
        styles={{ body: { padding: isCompact ? token.paddingLG : token.paddingXL } }}
        style={cardStyle}
      >
        <Flex vertical gap={token.marginLG}>
          <Flex vertical gap={token.marginXXS}>
            <Flex align='center' gap={token.marginXS}>
              <img
                src='/logo-mark.svg'
                alt=''
                aria-hidden
                width={28}
                height={28}
                style={{ display: 'block', borderRadius: token.borderRadiusSM }}
              />
              <Title level={3} style={{ margin: 0, fontSize: 24, lineHeight: 1.3 }}>
                {t('twoFactorTitle')}
              </Title>
            </Flex>
            <Text type='secondary'>{t('twoFactorHello', { username: state.username ?? '' })}</Text>
          </Flex>

          <Segmented
            block
            size='large'
            value={mode}
            onChange={(value) => switchMode(value as VerifyMode)}
            options={[
              { label: t('twoFactorUseAuthenticator'), value: 'totp' },
              { label: t('twoFactorUseRecoveryCode'), value: 'recovery' },
            ]}
          />

          {mode === 'totp' ? (
            <Flex vertical align='center' gap={token.marginSM}>
              <Text type='secondary'>{t('twoFactorSubtitle')}</Text>
              <Input.OTP
                size={isCompact ? 'middle' : 'large'}
                length={6}
                value={code}
                status={errorStatus || undefined}
                disabled={loading}
                onChange={(val) => {
                  setErrorStatus('')
                  setCode(val)
                }}
              />
              {errorStatus === 'error' && <Text type='danger'>{t('twoFactorCodeInvalid')}</Text>}
            </Flex>
          ) : (
            <Flex vertical gap={token.marginSM}>
              <Flex vertical gap={token.marginXXS}>
                <Input
                  size='large'
                  prefix={<KeyOutlined style={{ color: token.colorTextQuaternary }} />}
                  status={errorStatus || undefined}
                  value={recoveryCode}
                  onChange={(e) => {
                    setErrorStatus('')
                    setRecoveryCode(e.target.value)
                  }}
                  placeholder={t('twoFactorRecoveryPlaceholder')}
                  onPressEnter={submitRecovery}
                  autoComplete='off'
                />
                {errorStatus === 'error' ? (
                  <Text type='danger'>{t('twoFactorRecoveryInvalid')}</Text>
                ) : (
                  <Text type='secondary'>{t('twoFactorRecoveryHint')}</Text>
                )}
              </Flex>
              <Button
                type='primary'
                block
                loading={loading}
                onClick={submitRecovery}
                style={{ height: 44, fontSize: 15, fontWeight: 500 }}
              >
                {t('twoFactorRecoverySubmit')}
              </Button>
            </Flex>
          )}

          <Flex vertical align='center' gap={token.margin}>
            <Button
              type='link'
              size='small'
              onClick={() => navigate('/login', { replace: true })}
              style={{ padding: 0, fontSize: 13 }}
            >
              {t('twoFactorBackToLogin')}
            </Button>
            <Flex justify='center' align='center' gap={token.marginXXS}>
              <LockOutlined style={{ color: token.colorTextQuaternary, fontSize: 11 }} />
              <Text type='secondary' style={{ fontSize: 11 }}>
                {t('secureTip')}
              </Text>
            </Flex>
          </Flex>
        </Flex>
      </Card>
    </Flex>
  )
}
