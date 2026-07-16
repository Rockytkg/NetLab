import { Card, Col, Result, Row, Typography, theme } from 'antd'
import { AlertOutlined, AuditOutlined, FileSearchOutlined, RadarChartOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

const { Text } = Typography

export default function ObservabilityPage() {
  const { t } = useTranslation(['operations', 'common'])
  const { token } = theme.useToken()

  const capabilities = [
    { title: t('operations:snmpMonitoring'), desc: t('operations:snmpMonitoringDesc'), icon: <RadarChartOutlined /> },
    { title: t('operations:syslogCenter'), desc: t('operations:syslogCenterDesc'), icon: <FileSearchOutlined /> },
    { title: t('operations:radiusAudit'), desc: t('operations:radiusAuditDesc'), icon: <AuditOutlined /> },
    { title: t('operations:alerts'), desc: t('operations:alertsDesc'), icon: <AlertOutlined /> },
  ]

  return (
    <div style={{ width: '100%' }}>
      <Row gutter={[token.margin, token.margin]}>
        {capabilities.map((item) => (
          <Col xs={24} lg={12} key={item.title}>
            <Card title={item.title} extra={<span style={{ color: token.colorPrimary }}>{item.icon}</span>}>
              <Text type="secondary">{item.desc}</Text>
            </Card>
          </Col>
        ))}
      </Row>
      <Card style={{ marginTop: token.margin }}>
        <Result title={t('operations:observability')} subTitle={t('common:underDevelopment')} />
      </Card>
    </div>
  )
}
