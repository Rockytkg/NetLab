import { useEffect, useState } from 'react'
import { Alert, Flex, Typography, theme } from 'antd'
import { KeyOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import SecurityFlowLayout from '@/components/auth/SecurityFlowLayout'
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
    <SecurityFlowLayout
      title={t('settings:twoFactor.title')}
      subtitle={t('login:twoFactorForceSubtitle', { username: userInfo?.username ?? '' })}
      steps={[
        { title: t('settings:twoFactor.forceStepAuthenticator'), icon: <KeyOutlined /> },
        { title: t('settings:twoFactor.forceStepAutoVerify'), icon: <ThunderboltOutlined /> },
      ]}
    >
      <Flex vertical gap={token.marginLG}>
        <Flex vertical gap={token.marginXXS}>
          <Title level={4} style={{ margin: 0 }}>
            {t('settings:twoFactor.setupTitle')}
          </Title>
          <Text type='secondary'>{t('settings:twoFactor.forcePanelHint')}</Text>
        </Flex>
        <Alert type='warning' showIcon title={t('settings:twoFactor.forceNotice')} />
        <TwoFactorBindingSteps onComplete={handleComplete} />
      </Flex>
    </SecurityFlowLayout>
  )
}
