import { Typography, Card, Result, Row, Col } from 'antd'
import {
  DesktopOutlined,
  WifiOutlined,
  SafetyOutlined,
  CloudServerOutlined,
  ApartmentOutlined,
  ApiOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
const { Title, Text } = Typography

export default function DeviceLibraryPage() {
  const { t } = useTranslation(['operations', 'common'])
  const deviceCategories = [
    { icon: <WifiOutlined />, label: t('operations:managedRouters'), count: 12, color: '#1677FF' },
    { icon: <DesktopOutlined />, label: t('operations:managedSwitches'), count: 8, color: '#13C2C2' },
    { icon: <SafetyOutlined />, label: t('operations:managedFirewalls'), count: 6, color: '#F5222D' },
    { icon: <CloudServerOutlined />, label: t('operations:managedServers'), count: 10, color: '#722ED1' },
    { icon: <ApartmentOutlined />, label: t('operations:managedLoadBalancers'), count: 4, color: '#FA8C16' },
    { icon: <ApiOutlined />, label: t('operations:managedWireless'), count: 9, color: '#52C41A' },
  ]

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
              <Text type="secondary">{t('operations:devicesUnit', { count: cat.count })}</Text>
            </Card>
          </Col>
        ))}
      </Row>
      <Card style={{ marginTop: 24 }}>
        <Result
          title={t('operations:onboardingComingSoon')}
          subTitle={t('operations:snmpMonitoringDesc')}
        />
      </Card>
    </div>
  )
}
