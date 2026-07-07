import { Typography, Card, Result } from 'antd'
import { MonitorOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

const { Title, Text } = Typography

export default function LabMonitorPage() {
  const { t } = useTranslation(['topology', 'menu'])

  return (
    <div style={{ width: '100%' }}>
      <div className="netlab-page-header">
        <div>
          <Title level={3}>{t('menu:runMonitor')}</Title>
          <Text type="secondary">Real-time lab monitoring dashboard</Text>
        </div>
      </div>
      <Card>
        <Result
          icon={<MonitorOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('comingSoon')}
          subTitle={t('underDevelopment')}
        />
      </Card>
    </div>
  )
}
