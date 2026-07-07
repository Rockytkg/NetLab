import { Typography, Card, Result } from 'antd'
import { CloudOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

const { Title, Text } = Typography

export default function TemplateMarketPage() {
  const { t } = useTranslation(['menu', 'common'])

  return (
    <div style={{ width: '100%' }}>
      <div className="netlab-page-header">
        <div>
          <Title level={3}>{t('templateMarket')}</Title>
          <Text type="secondary">Browse and install network topology templates</Text>
        </div>
      </div>
      <Card>
        <Result
          icon={<CloudOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('common:comingSoon')}
          subTitle={t('common:underDevelopment')}
        />
      </Card>
    </div>
  )
}
