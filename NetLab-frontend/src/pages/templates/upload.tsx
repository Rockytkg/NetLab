import { Typography, Card, Result } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

const { Title, Text } = Typography

export default function TemplateUploadPage() {
  const { t } = useTranslation(['menu', 'common'])

  return (
    <div style={{ width: '100%' }}>
      <div className="netlab-page-header">
        <div>
          <Title level={3}>{t('myUploads')}</Title>
          <Text type="secondary">Upload your custom topology templates</Text>
        </div>
      </div>
      <Card>
        <Result
          icon={<UploadOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('common:comingSoon')}
          subTitle={t('common:underDevelopment')}
        />
      </Card>
    </div>
  )
}
