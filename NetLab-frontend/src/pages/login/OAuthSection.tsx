import { useCallback } from 'react'
import { Button, Divider, Flex, Typography, Tooltip, App, theme } from 'antd'
import {
  GithubOutlined,
  GoogleOutlined,
  QqOutlined,
  WechatOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import type { OAuthProvider } from '@/types/auth'
import { authApi } from '@/services/auth'
import { useAuthStore } from '@/stores/authStore'
import { tokenStorage } from '@/utils/token'
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
        tokenStorage.setAccessToken(result.accessToken)
        tokenStorage.setRefreshToken(result.refreshToken)
        useAuthStore.setState({
          accessToken: result.accessToken,
          refreshToken: result.refreshToken,
          userInfo: result.user,
        })
        message.success(t('loginSuccess'))
        navigate('/dashboard', { replace: true })
      } catch {
        message.error(t('loginFailed'))
      }
    }
    window.addEventListener('message', handleMessage)

    const checkClosed = setInterval(() => {
      if (popup.closed) { clearInterval(checkClosed); window.removeEventListener('message', handleMessage) }
    }, 500)
  }, [t, message, navigate])

  const hasPasskey = passkeyEnabled
  const hasOAuth = providers.length > 0

  if (!hasPasskey && !hasOAuth) return null

  return (
    <>
      <Divider plain style={{ marginBlock: token.marginLG, marginBottom: token.margin }}>
        <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>{t('orSignInWith')}</Text>
      </Divider>

      <Flex justify="center" gap={token.padding}>
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
    </>
  )
}
