import { useEffect, useState } from 'react'
import { Avatar, Card, Descriptions, Grid, Space, Tag, Tabs, Typography, theme } from 'antd'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/authStore'
import { authApi } from '@/services/auth'
import type { SystemConfig } from '@/types/auth'
import { getAvatarColor } from '@/utils/avatar'
import ChangePasswordPanel from './account/ChangePasswordPanel'
import ChangeEmailPanel from './account/ChangeEmailPanel'
import PasskeyPanel from './account/PasskeyPanel'
import OAuthBindingsPanel from './account/OAuthBindingsPanel'
import TwoFactorPanel from './account/TwoFactorPanel'

const { Title } = Typography

/**
 * 个人中心。
 * 聚合账户资料、修改密码、Passkey 管理与第三方账号绑定。
 * 是否展示 Passkey / 第三方绑定取决于系统安全策略（/auth/config）。
 */
export default function SettingsProfilePage() {
  const { t } = useTranslation(['settings', 'menu', 'common'])
  const { token } = theme.useToken()
  const userInfo = useAuthStore((s) => s.userInfo)
  const [config, setConfig] = useState<SystemConfig | null>(null)
  const screens = Grid.useBreakpoint()

  useEffect(() => {
    let alive = true
    ;(async () => {
      try {
        const data = await authApi.getSystemConfig()
        if (alive) setConfig(data)
      } catch {
        // 拦截器已提示错误
      }
    })()
    return () => {
      alive = false
    }
  }, [])

  const oauthEnabled = (config?.oauthProviders?.length ?? 0) > 0
  const passkeyEnabled = !!config?.passkeyEnabled
  const twoFactorRequired = !!config?.twoFactorRequired
  const securityTabs = [
    {
      key: 'email',
      label: t('settings:changeEmail.title'),
      children: <ChangeEmailPanel />,
    },
    {
      key: 'password',
      label: t('settings:changePassword.title'),
      children: <ChangePasswordPanel />,
    },
    {
      key: 'passkey',
      label: t('settings:passkey.title'),
      children: <PasskeyPanel enabled={passkeyEnabled} />,
    },
    {
      key: 'twofa',
      label: t('settings:twoFactor.title'),
      children: <TwoFactorPanel forceRequired={twoFactorRequired} />,
    },
    ...(oauthEnabled
      ? [
          {
            key: 'oauth',
            label: t('settings:oauthBindings.title'),
            children: <OAuthBindingsPanel providers={config!.oauthProviders} />,
          },
        ]
      : []),
  ]

  return (
    <div style={{ width: '100%', height: '100%', overflow: 'hidden' }}>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: screens.lg ? 'minmax(280px, 360px) minmax(0, 1fr)' : '1fr',
          gap: token.marginLG,
          alignItems: 'start',
        }}
      >
        {/* 账户资料 */}
        <Card variant="outlined" styles={{ body: { paddingBlock: token.paddingLG } }}>
          <Space size={token.marginLG} align="center" wrap>
            <Avatar
              size={72}
              src={userInfo?.avatar}
              style={{ backgroundColor: getAvatarColor(userInfo?.nickname) }}
            >
              {userInfo?.nickname?.charAt(0)}
            </Avatar>
            <div>
              <Title level={4} style={{ marginBottom: token.marginXS }}>
                {userInfo?.nickname || '-'}
              </Title>
              <Space size={token.marginXS} wrap>
                {userInfo?.role && (
                  <Tag color="blue">
                    {userInfo.role}
                  </Tag>
                )}
              </Space>
            </div>
          </Space>

          <Descriptions
            column={1}
            bordered
            size="middle"
            style={{ marginTop: token.marginLG }}
          >
            <Descriptions.Item label={t('settings:profile.username')}>
              {userInfo?.username || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('settings:profile.phone')}>
              {userInfo?.phone || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('settings:profile.email')}>
              {userInfo?.email || '-'}
            </Descriptions.Item>
          </Descriptions>
        </Card>

        <Tabs
          items={securityTabs}
          type="card"
          destroyOnHidden={false}
          style={{ minWidth: 0 }}
        />
      </div>
    </div>
  )
}
