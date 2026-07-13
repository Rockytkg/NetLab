import { Card, Result } from 'antd'
import { QuestionCircleOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

export default function HelpPage() {
  const { t } = useTranslation(['menu', 'common'])

  return (
    <div style={{ width: '100%' }}>
      <Card>
        <Result
          icon={<QuestionCircleOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('common:comingSoon')}
          subTitle={t('common:underDevelopment')}
        />
      </Card>
    </div>
  )
}
