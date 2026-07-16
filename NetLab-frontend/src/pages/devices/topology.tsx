import { Card, Result } from 'antd'
import { ClusterOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import '@/assets/css/topology.css'

export default function DeviceTopologyPage() {
  const { t } = useTranslation(['operations'])

  return (
    <div style={{ width: '100%' }}>
      <Card>
        <Result
          icon={<ClusterOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('operations:topologyComingSoon')}
          subTitle={t('operations:topologyUnderDevelopment')}
        />
      </Card>
    </div>
  )
}
