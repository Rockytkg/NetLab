import { Card, Result, Tabs } from 'antd'
import { useTranslation } from 'react-i18next'
import { usePermission } from '@/hooks/usePermission'
import SystemForm from './SystemForm'
import AuthPolicyForm from './AuthPolicyForm'

/** RADIUS 服务设置页：基础设置（监听/RadSec）与认证会话策略的合并入口。
 * 两个 Tab 各自独立表单、独立保存（对应后端 system / server 两个 PUT）。 */
export default function RadiusSettingsPage() {
  const { t } = useTranslation(['radius', 'settings'])
  const { can } = usePermission()
  const canReadRadius = can('radius.read')

  if (!canReadRadius) {
    return <Result status="403" title="403" subTitle={t('settings:permissionDenied')} />
  }

  return (
    <div style={{ width: '100%' }}>
      <Card variant="outlined">
        <Tabs
          items={[
            {
              key: 'basic',
              label: t('radius:settings.tabs.basic'),
              children: <SystemForm />,
            },
            {
              key: 'authSession',
              label: t('radius:settings.tabs.authSession'),
              children: <AuthPolicyForm />,
            },
          ]}
        />
      </Card>
    </div>
  )
}
