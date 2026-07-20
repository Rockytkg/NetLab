import { useState } from 'react'
import { Card, Result, Tabs, theme } from 'antd'
import { useTranslation } from 'react-i18next'
import { usePermission } from '@/hooks/usePermission'
import SystemForm from './SystemForm'
import AuthPolicyForm, { type PolicySection } from './AuthPolicyForm'

/** RADIUS 服务设置页：认证策略、拒绝防护、会话记账与 RadSec 独立保存。 */
export default function RadiusSettingsPage() {
  const { t } = useTranslation(['radius', 'settings'])
  const { can } = usePermission()
  const { token } = theme.useToken()
  const canReadRadius = can('radius.read')
  const [activeKey, setActiveKey] = useState<PolicySection | 'radsec'>('authentication')

  if (!canReadRadius) {
    return <Result status="403" title="403" subTitle={t('settings:permissionDenied')} />
  }

  return (
    <div>
      <Card variant="outlined">
        <Tabs
          activeKey={activeKey}
          onChange={(key) => setActiveKey(key as PolicySection | 'radsec')}
          tabBarStyle={{ marginBottom: token.marginLG }}
          items={[
            {
              key: 'authentication',
              label: t('radius:settings.tabs.authentication'),
            },
            {
              key: 'protection',
              label: t('radius:settings.tabs.protection'),
            },
            {
              key: 'session',
              label: t('radius:settings.tabs.session'),
            },
            {
              key: 'radsec',
              label: t('radius:settings.tabs.radsec'),
            },
          ]}
        />
        {activeKey === 'radsec' ? <SystemForm /> : <AuthPolicyForm section={activeKey} />}
      </Card>
    </div>
  )
}
