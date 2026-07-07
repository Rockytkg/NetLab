import { useState } from 'react'
import { Button, Flex, Tooltip, Typography, theme } from 'antd'
import { SafetyCertificateOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { usePasskey } from '@/hooks/usePasskey'

const { Text } = Typography

/** 略大于 Ant Design 默认的大号圆形按钮（40px），以获得舒适的点击区域 */
const CIRCLE_BUTTON_SIZE = 44

interface PasskeyButtonProps {
  variant?: 'default' | 'circle'
}

export default function PasskeyButton({ variant = 'default' }: PasskeyButtonProps) {
  const { t } = useTranslation('login')
  const { token } = theme.useToken()
  const { isPlatformAuthAvailable, login } = usePasskey()
  const [loading, setLoading] = useState(false)
  const [available, setAvailable] = useState<boolean | null>(null)

  useState(() => {
    isPlatformAuthAvailable().then(setAvailable)
  })

  const handlePasskeyLogin = async () => {
    setLoading(true)
    try { await login() } finally { setLoading(false) }
  }

  if (available === false) return null

  // 紧凑圆形变体 —— 与 OAuth 按钮内联并排使用
  if (variant === 'circle') {
    return (
      <Tooltip title={t('passkeyTooltip')}>
        <Button
          shape="circle"
          size="large"
          icon={<SafetyCertificateOutlined />}
          onClick={handlePasskeyLogin}
          loading={loading}
          aria-label={t('signInWithPasskey')}
          style={{ width: CIRCLE_BUTTON_SIZE, height: CIRCLE_BUTTON_SIZE, fontSize: token.fontSizeLG }}
        />
      </Tooltip>
    )
  }

  // 默认独立变体
  return (
    <Flex vertical align="center">
      <Tooltip title={t('passkeyTooltip')}>
        <Button
          size="middle"
          icon={<SafetyCertificateOutlined />}
          onClick={handlePasskeyLogin}
          loading={loading}
          style={{ fontSize: token.fontSize, fontWeight: token.fontWeightStrong }}
        >
          {t('signInWithPasskey')}
        </Button>
      </Tooltip>
      <Text type="secondary" style={{ fontSize: token.fontSizeSM, marginTop: token.marginXS }}>
        {t('passkeyHint')}
      </Text>
    </Flex>
  )
}
