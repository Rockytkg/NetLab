import { Typography, Card, Result } from 'antd'
import { ExperimentOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'

const { Title, Text } = Typography

export default function LabEditorPage() {
  const { t } = useTranslation(['topology', 'common'])
  const { labId } = useParams<{ labId: string }>()

  return (
    <div style={{ width: '100%' }}>
      <div className="netlab-page-header">
        <div>
          <Title level={3}>{t('title')}</Title>
          <Text type="secondary">Lab ID: {labId}</Text>
        </div>
      </div>
      <Card>
        <Result
          icon={<ExperimentOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('comingSoon')}
          subTitle={t('underDevelopment')}
        />
      </Card>
    </div>
  )
}
