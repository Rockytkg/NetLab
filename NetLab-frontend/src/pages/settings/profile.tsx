import { Typography, Card, Descriptions, Avatar } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/authStore'

const { Title } = Typography

export default function SettingsProfilePage() {
  const { t } = useTranslation(['common', 'menu'])
  const userInfo = useAuthStore((s) => s.userInfo)

  return (
    <div style={{ width: '100%' }}>
      <div className="netlab-page-header">
        <Title level={3}>{t('profile')}</Title>
      </div>
      <Card style={{ maxWidth: 600 }}>
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <Avatar size={80} icon={<UserOutlined />} src={userInfo?.avatar} />
          <Title level={4} style={{ marginTop: 16, marginBottom: 4 }}>
            {userInfo?.username || 'Admin'}
          </Title>
        </div>
        <Descriptions column={1} bordered size="middle">
          <Descriptions.Item label="Username">
            {userInfo?.username || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Email">
            {userInfo?.email || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="Roles">
            {userInfo?.roles?.join(', ') || '-'}
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  )
}
