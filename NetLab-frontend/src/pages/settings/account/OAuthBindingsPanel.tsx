import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Card, Empty, List, Popconfirm, Spin, Tag, Typography, App, theme } from 'antd'
import { GithubOutlined, GoogleOutlined, QqOutlined, WechatOutlined, LinkOutlined, DisconnectOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { authApi } from '@/services/auth'
import LinuxDoIcon from '@/components/common/icons/LinuxDoIcon'
import type { OAuthBinding, OAuthProvider } from '@/types/auth'

const { Text } = Typography

const PROVIDER_ICONS: Record<string, React.ReactNode> = {
  github: <GithubOutlined />,
  google: <GoogleOutlined />,
  qq: <QqOutlined />,
  wechat: <WechatOutlined />,
  linuxdo: <LinuxDoIcon />,
}

interface OAuthBindingsPanelProps {
  /** /auth/config 返回的已启用第三方提供商 */
  providers: OAuthProvider[]
}

/**
 * 个人中心 · 第三方账号绑定面板。
 * 通过弹窗完成 OAuth 授权（bind 意图），回调后调用 /auth/oauth/bind 写入绑定。
 */
export default function OAuthBindingsPanel({ providers }: OAuthBindingsPanelProps) {
  const { t } = useTranslation(['settings', 'common', 'login'])
  const { token } = theme.useToken()
  const { message } = App.useApp()

  const [bindings, setBindings] = useState<OAuthBinding[]>([])
  const [loading, setLoading] = useState(false)
  const [busyProvider, setBusyProvider] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await authApi.listOAuthBindings()
      setBindings(res.bindings ?? [])
    } catch {
      // 拦截器已提示错误
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const boundMap = useMemo(() => {
    const m = new Map<string, OAuthBinding>()
    bindings.forEach((b) => m.set(b.provider, b))
    return m
  }, [bindings])

  const handleBind = useCallback(
    async (provider: OAuthProvider) => {
      setBusyProvider(provider.id)
      try {
        const { authUrl } = await authApi.getOAuthBindURL(provider.id)

        const width = 600
        const height = 700
        const left = window.screenX + (window.outerWidth - width) / 2
        const top = window.screenY + (window.outerHeight - height) / 2
        const popup = window.open(
          authUrl,
          `oauth-bind-${provider.id}`,
          `width=${width},height=${height},left=${left},top=${top}`,
        )
        if (!popup) {
          message.error(t('settings:oauthBindings.popupBlocked'))
          setBusyProvider(null)
          return
        }

        const handleMessage = async (event: MessageEvent) => {
          if (event.data?.type !== 'oauth-callback' || !event.data?.code || !event.data?.state) return
          window.removeEventListener('message', handleMessage)
          clearInterval(checkClosed)
          popup?.close()
          try {
            await authApi.bindOAuth({
              provider: provider.id,
              code: event.data.code,
              state: event.data.state,
            })
            message.success(t('settings:oauthBindings.bindSuccess'))
            await load()
          } catch {
            // 拦截器已提示错误
          } finally {
            setBusyProvider(null)
          }
        }
        window.addEventListener('message', handleMessage)

        const checkClosed = setInterval(() => {
          if (popup.closed) {
            clearInterval(checkClosed)
            window.removeEventListener('message', handleMessage)
            setBusyProvider(null)
          }
        }, 500)
      } catch {
        setBusyProvider(null)
      }
    },
    [t, message, load],
  )

  const handleUnbind = useCallback(
    async (provider: string) => {
      setBusyProvider(provider)
      try {
        await authApi.unbindOAuth(provider)
        message.success(t('settings:oauthBindings.unbindSuccess'))
        await load()
      } catch {
        // 拦截器已提示错误
      } finally {
        setBusyProvider(null)
      }
    },
    [t, message, load],
  )

  return (
    <Card
      title={t('settings:oauthBindings.title')}
      variant="outlined"
      styles={{ body: { paddingBlock: token.paddingLG } }}
    >
      <Text type="secondary">{t('settings:oauthBindings.description')}</Text>

      <div style={{ marginTop: token.marginLG }}>
        <Spin spinning={loading}>
          {providers.length === 0 ? (
            <Empty description={t('settings:oauthBindings.noProviders')} />
          ) : (
            <List
              itemLayout="horizontal"
              dataSource={providers}
              split
              renderItem={(provider) => {
                const bound = boundMap.get(provider.id)
                return (
                  <List.Item
                    actions={[
                      bound ? (
                        <Popconfirm
                          key="unbind"
                          title={t('settings:oauthBindings.unbindConfirm', { provider: provider.name })}
                          okText={t('common:confirm')}
                          cancelText={t('common:cancel')}
                          okButtonProps={{ danger: true }}
                          onConfirm={() => handleUnbind(provider.id)}
                        >
                          <Button
                            danger
                            icon={<DisconnectOutlined />}
                            loading={busyProvider === provider.id}
                          >
                            {t('settings:oauthBindings.unbind')}
                          </Button>
                        </Popconfirm>
                      ) : (
                        <Button
                          key="bind"
                          type="primary"
                          icon={<LinkOutlined />}
                          loading={busyProvider === provider.id}
                          onClick={() => handleBind(provider)}
                        >
                          {t('settings:oauthBindings.bind')}
                        </Button>
                      ),
                    ]}
                  >
                    <List.Item.Meta
                      avatar={
                        <span style={{ fontSize: token.fontSizeHeading3, color: provider.color }}>
                          {PROVIDER_ICONS[provider.id] ?? <LinkOutlined />}
                        </span>
                      }
                      title={
                        <span>
                          <Text strong>{provider.name}</Text>{' '}
                          <Tag color={bound ? 'success' : 'default'}>
                            {bound
                              ? t('settings:oauthBindings.bound')
                              : t('settings:oauthBindings.notBound')}
                          </Tag>
                        </span>
                      }
                      description={
                        bound?.email ? (
                          <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>
                            {bound.email}
                          </Text>
                        ) : null
                      }
                    />
                  </List.Item>
                )
              }}
            />
          )}
        </Spin>
      </div>
    </Card>
  )
}
