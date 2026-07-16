import { Card, Result } from 'antd'
import { CloudOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

export default function OperationsTemplatesPage() {
  const { t } = useTranslation(['operations'])

  return (
    <div style={{ width: '100%' }}>
      <Card>
        <Result
          icon={<CloudOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('operations:templatesComingSoon')}
          subTitle={t('operations:alertsDesc')}
        />
      </Card>
    </div>
  )
}
