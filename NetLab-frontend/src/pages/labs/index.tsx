import { Typography, Card, Button, Space } from 'antd'
import { PlusOutlined, ExperimentOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

const { Title, Text } = Typography

export default function LabListPage() {
  const { t } = useTranslation(['lab', 'common', 'menu'])

  return (
    <div style={{ width: '100%' }}>
      <div className="netlab-page-header">
        <div>
          <Title level={3}>{t('menu:myLabs')}</Title>
          <Text type="secondary">{t('lab:createFirstLab')}</Text>
        </div>
        <Button type="primary" icon={<PlusOutlined />}>
          {t('lab:createLab')}
        </Button>
      </div>
      <Card>
        <Space orientation="vertical" size="large" style={{ width: '100%', textAlign: 'center', padding: '80px 0' }}>
          <ExperimentOutlined style={{ fontSize: 48, color: '#BFBFBF' }} />
          <div>
            <Title level={4} type="secondary">{t('lab:noLabs')}</Title>
            <Text type="secondary">{t('common:comingSoon')}</Text>
          </div>
          <Button type="primary" icon={<PlusOutlined />}>
            {t('lab:createFirstLab')}
          </Button>
        </Space>
      </Card>
    </div>
  )
}
