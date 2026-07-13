import { Card, Result } from 'antd'
import { MonitorOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

export default function LabMonitorPage() {
  const { t } = useTranslation(['topology', 'menu'])

  return (
    <div style={{ width: '100%' }}>
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
