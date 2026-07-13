import { Card, Result } from 'antd'
import { ExperimentOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

export default function LabEditorPage() {
  const { t } = useTranslation(['topology', 'common'])

  return (
    <div style={{ width: '100%' }}>
      <Card>
        <Result
          icon={<ExperimentOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('comingSoon')}
          subTitle={t('underDevelopment')}
        />
      </Card>
    </div>
  )
}
