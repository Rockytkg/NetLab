import { Card, Result } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

export default function OperationsTemplateUploadPage() {
  const { t } = useTranslation(['operations'])

  return (
    <div style={{ width: '100%' }}>
      <Card>
        <Result
          icon={<UploadOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('operations:templatesComingSoon')}
          subTitle={t('operations:alertsDesc')}
        />
      </Card>
    </div>
  )
}
