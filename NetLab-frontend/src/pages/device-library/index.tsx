import { Typography, Card, Result, Row, Col } from 'antd'
import {
  DesktopOutlined,
  WifiOutlined,
  SafetyOutlined,
  CloudServerOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
const { Title, Text } = Typography

const deviceCategories = [
  { icon: <WifiOutlined />, label: 'Routers', count: 12, color: '#1677FF' },
  { icon: <DesktopOutlined />, label: 'Switches', count: 8, color: '#13C2C2' },
  { icon: <SafetyOutlined />, label: 'Firewalls', count: 6, color: '#F5222D' },
  { icon: <CloudServerOutlined />, label: 'Servers', count: 10, color: '#722ED1' },
]

export default function DeviceLibraryPage() {
  const { t } = useTranslation(['menu', 'common'])

  return (
    <div style={{ width: '100%' }}>
      <Row gutter={[16, 16]}>
        {deviceCategories.map((cat) => (
          <Col xs={12} sm={8} md={6} key={cat.label}>
            <Card
              hoverable
              style={{ textAlign: 'center' }}
              styles={{ body: { padding: 24 } }}
            >
              <div style={{ fontSize: 32, color: cat.color, marginBottom: 12 }}>
                {cat.icon}
              </div>
              <Title level={5} style={{ margin: 0 }}>{cat.label}</Title>
              <Text type="secondary">{cat.count} devices</Text>
            </Card>
          </Col>
        ))}
      </Row>
      <Card style={{ marginTop: 24 }}>
        <Result
          title={t('common:comingSoon')}
          subTitle={t('common:underDevelopment')}
        />
      </Card>
    </div>
  )
}
