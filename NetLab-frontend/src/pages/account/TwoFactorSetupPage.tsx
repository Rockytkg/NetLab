import { useEffect, useState } from 'react'
import { Alert, Card, Col, Row, Space, Steps, Typography, theme } from 'antd'
import { KeyOutlined, SafetyOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import TwoFactorBindingSteps from '@/pages/settings/account/TwoFactorBindingSteps'

const { Title, Text } = Typography

/** 强制两步验证绑定引导页：系统要求开启 2FA 时拦截未绑定用户。 */
export default function TwoFactorSetupPage() {
  const { t } = useTranslation(['settings', 'login'])
  const { token } = theme.useToken()
  const navigate = useNavigate()
  const userInfo = useAuthStore((s) => s.userInfo)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    let alive = true
    ;(async () => {
      try {
        const user = await authApi.getUserInfo()
        useAuthStore.getState().setUserInfo(user)
        if (user.twoFactorEnabled) {
          useAuthStore.setState({ securityActions: null })
          navigate('/dashboard', { replace: true })
          return
        }
      } catch {
        // ignore; show binding UI
      }
      if (alive) setReady(true)
    })()
    return () => {
      alive = false
    }
  }, [navigate])

  const handleComplete = async () => {
    await authApi.getUserInfo().then((user) => useAuthStore.setState({ userInfo: user })).catch(() => undefined)
    useAuthStore.setState({ securityActions: null })
    navigate('/dashboard', { replace: true })
  }

  if (!ready) return null

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
          maxWidth: 980,
        }}
      >
        <Row gutter={[24, 24]} align='top'>
          <Col xs={24} md={10}>
            <Card style={{ height: '100%' }}>
              <Space orientation='vertical' size={24} style={{ width: '100%' }}>
                <div>
                  <SafetyOutlined style={{ fontSize: 42, color: token.colorPrimary }} />
                  <Title level={2} style={{ marginTop: 16, marginBottom: 8 }}>
                    {t('settings:twoFactor.title')}
                  </Title>
                  <Text type='secondary'>{t('login:twoFactorForceSubtitle', { username: userInfo?.username ?? '' })}</Text>
                </div>
                <Steps
                  orientation='vertical'
                  current={0}
                  items={[
                    {
                      title: t('settings:twoFactor.forceStepAuthenticator'),
                      icon: <KeyOutlined />,
                    },
                    {
                      title: t('settings:twoFactor.forceStepAutoVerify'),
                      icon: <ThunderboltOutlined />,
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
                    {t('settings:twoFactor.setupTitle')}
                  </Title>
                  <Text type='secondary'>{t('settings:twoFactor.forcePanelHint')}</Text>
                </div>
                <Alert type='warning' showIcon title={t('settings:twoFactor.forceNotice')} />
                <TwoFactorBindingSteps onComplete={handleComplete} />
              </Space>
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  )
}
