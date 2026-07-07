import { Button, Card, Result, Typography, theme } from 'antd'
import { HomeOutlined, ArrowLeftOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

const { Text } = Typography

export default function NotFoundPage() {
  const navigate = useNavigate()
  const { t } = useTranslation('common')
  const { token } = theme.useToken()

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
        background: token.colorBgLayout,
      }}
    >
      {/* Brand mark */}
      <div
        aria-hidden
        style={{
          width: 56,
          height: 56,
          borderRadius: 14,
          background: `linear-gradient(135deg, ${token.colorPrimary}, ${token.colorPrimaryActive})`,
          display: 'grid',
          placeItems: 'center',
          fontSize: 24,
          fontWeight: 700,
          color: '#fff',
          marginBottom: 32,
          boxShadow: `0 8px 24px ${token.colorPrimary}40`,
        }}
      >
        N
      </div>

      <Card
        style={{
          width: 'min(100%, 480px)',
          borderRadius: token.borderRadiusLG,
          border: `1px solid ${token.colorBorderSecondary}`,
        }}
        styles={{ body: { padding: '40px 32px' } }}
      >
        <Result
          status="404"
          title={
            <Text
              strong
              style={{ fontSize: 20, color: token.colorText }}
            >
              404
            </Text>
          }
          subTitle={
            <Text type="secondary" style={{ fontSize: 14 }}>
              {t('noData')}
            </Text>
          }
          extra={
            <div style={{ display: 'flex', gap: 12, justifyContent: 'center' }}>
              <Button
                icon={<ArrowLeftOutlined />}
                onClick={() => navigate(-1)}
              >
                {t('back')}
              </Button>
              <Button
                type="primary"
                icon={<HomeOutlined />}
                onClick={() => navigate('/', { replace: true })}
              >
                {t('profile')}
              </Button>
            </div>
          }
        />
      </Card>

      <Text type="secondary" style={{ marginTop: 24, fontSize: 12 }}>
        &copy; {new Date().getFullYear()} NetLab
      </Text>
    </div>
  )
}
