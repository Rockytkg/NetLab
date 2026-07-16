import { Card, Col, Row, Statistic, Typography, theme } from 'antd'
import {
  AlertOutlined,
  AuditOutlined,
  ClusterOutlined,
  DatabaseOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useOperationsStore } from '@/stores/operationsStore'

const { Title, Text } = Typography

export default function DashboardPage() {
  const { t } = useTranslation(['operations'])
  const { token } = theme.useToken()
  const deviceGroups = useOperationsStore((s) => s.deviceGroups)
  const devices = useOperationsStore((s) => s.devices)

  const onlineDevices = devices.filter((device) => device.status === 'online').length
  const alertCount = deviceGroups.reduce((sum, group) => sum + group.alertCount, 0)
  const snmpEnabled = devices.filter((device) => device.snmpEnabled).length

  const cards = [
    {
      key: 'devices',
      title: t('operations:deviceInventory'),
      value: devices.length,
      icon: <DatabaseOutlined />,
      color: token.colorPrimary,
    },
    {
      key: 'online',
      title: t('operations:onlineCount'),
      value: onlineDevices,
      icon: <ClusterOutlined />,
      color: token.colorSuccess,
    },
    {
      key: 'alerts',
      title: t('operations:alertCount'),
      value: alertCount,
      icon: <AlertOutlined />,
      color: token.colorWarning,
    },
    {
      key: 'snmp',
      title: t('operations:snmpMonitoring'),
      value: snmpEnabled,
      icon: <AuditOutlined />,
      color: token.colorInfo,
    },
  ]

  return (
    <div style={{ width: '100%' }}>
      <div style={{ marginBottom: token.marginLG }}>
        <Title level={3} style={{ marginBottom: token.marginXXS }}>
          {t('operations:overviewTitle')}
        </Title>
        <Text type="secondary">{t('operations:overviewSubtitle')}</Text>
      </div>

      <Row gutter={[token.margin, token.margin]}>
        {cards.map((card) => (
          <Col xs={24} sm={12} xl={6} key={card.key}>
            <Card>
              <Statistic
                title={card.title}
                value={card.value}
                prefix={<span style={{ color: card.color }}>{card.icon}</span>}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[token.margin, token.margin]} style={{ marginTop: token.margin }}>
        {[
          { title: t('operations:snmpMonitoring'), desc: t('operations:snmpMonitoringDesc') },
          { title: t('operations:syslogCenter'), desc: t('operations:syslogCenterDesc') },
          { title: t('operations:radiusAudit'), desc: t('operations:radiusAuditDesc') },
          { title: t('operations:alerts'), desc: t('operations:alertsDesc') },
        ].map((item) => (
          <Col xs={24} lg={12} key={item.title}>
            <Card title={item.title}>
              <Text type="secondary">{item.desc}</Text>
            </Card>
          </Col>
        ))}
      </Row>
    </div>
  )
}
