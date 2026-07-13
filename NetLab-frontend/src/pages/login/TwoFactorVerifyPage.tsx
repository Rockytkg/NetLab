import { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, App, Button, Card, Input, Space, Typography, theme } from 'antd'
import { SafetyOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import { navigateAfterLogin } from '@/utils/auth-flow'
import type { LoginResult } from '@/types/auth'

const { Title, Text } = Typography

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
  const state = (location.state ?? {}) as LocationState

  const [mode, setMode] = useState<'totp' | 'recovery'>('totp')
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

  const switchMode = (next: 'totp' | 'recovery') => {
    setMode(next)
    setErrorStatus('')
    setCode('')
    setRecoveryCode('')
  }

  if (!state.twoFactorToken) {
    return (
      <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: token.paddingLG }}>
      <div style={{ width: '100%', maxWidth: 360, textAlign: 'center' }}>
          <Alert type='warning' showIcon title={t('twoFactorSessionExpired')} style={{ marginBottom: token.marginLG }} />
          <Button type='primary' onClick={() => navigate('/login', { replace: true })}>
            {t('twoFactorBackToLogin')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: token.paddingLG }}>
      <div style={{ width: '100%', maxWidth: 420 }}>
        <Card>
          <Space orientation='vertical' size={24} style={{ width: '100%' }}>
            <div style={{ textAlign: 'center' }}>
              <SafetyOutlined style={{ fontSize: 36, color: token.colorPrimary }} />
              <Title level={3} style={{ marginTop: token.marginSM, marginBottom: token.marginXS }}>
                {t('twoFactorTitle')}
              </Title>
              <Text type='secondary'>{t('twoFactorHello', { username: state.username ?? '' })}</Text>
              <div>
                <Text type='secondary' style={{ fontSize: 13 }}>
                  {mode === 'totp' ? t('twoFactorSubtitle') : t('twoFactorRecoveryHint')}
                </Text>
              </div>
            </div>

          {mode === 'totp' ? (
            <>
              <div>
                <div style={{ display: 'flex', justifyContent: 'center', marginTop: token.marginSM }}>
                  <Input.OTP
                    length={6}
                    value={code}
                    status={errorStatus || undefined}
                    disabled={loading}
                    onChange={(val) => {
                      setErrorStatus('')
                      setCode(val)
                    }}
                  />
                </div>
                {errorStatus === 'error' && (
                  <div style={{ textAlign: 'center', marginTop: token.marginXS }}>
                    <Text type='danger' style={{ fontSize: 12 }}>
                      {t('twoFactorCodeInvalid')}
                    </Text>
                  </div>
                )}
              </div>
              <Button type='link' block onClick={() => switchMode('recovery')}>
                {t('twoFactorUseRecoveryCode')}
              </Button>
            </>
          ) : (
            <>
              <div>
                <Text strong>{t('twoFactorRecoveryLabel')}</Text>
                <Input
                  style={{ marginTop: token.marginSM }}
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
                {errorStatus === 'error' && (
                  <div style={{ marginTop: token.marginXS }}>
                    <Text type='danger' style={{ fontSize: 12 }}>
                      {t('twoFactorRecoveryInvalid')}
                    </Text>
                  </div>
                )}
              </div>
              <Button type='primary' block size='large' loading={loading} onClick={submitRecovery}>
                {t('twoFactorRecoverySubmit')}
              </Button>
              <Button type='link' block onClick={() => switchMode('totp')}>
                {t('twoFactorUseAuthenticator')}
              </Button>
            </>
          )}

          <Button type='link' block onClick={() => navigate('/login', { replace: true })}>
            {t('twoFactorBackToLogin')}
          </Button>
          </Space>
        </Card>
      </div>
    </div>
  )
}
