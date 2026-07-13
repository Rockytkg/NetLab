import { Card, Result } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

export default function TemplateUploadPage() {
  const { t } = useTranslation(['menu', 'common'])

  return (
    <div style={{ width: '100%' }}>
      <Card>
        <Result
          icon={<UploadOutlined style={{ fontSize: 64, color: '#1677FF' }} />}
          title={t('common:comingSoon')}
          subTitle={t('common:underDevelopment')}
        />
      </Card>
    </div>
  )
}
