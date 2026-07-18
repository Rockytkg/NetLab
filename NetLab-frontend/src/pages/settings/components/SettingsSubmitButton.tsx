import { Button } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import type { ReactNode } from 'react'
import Can from '@/components/auth/Can'

interface SettingsSubmitButtonProps {
  loading?: boolean
  /** 覆盖默认的「保存 / 保存中…」文案 */
  children?: ReactNode
}

/** 系统设置统一的保存按钮，按 setting.update 权限渲染。 */
export default function SettingsSubmitButton({ loading, children }: SettingsSubmitButtonProps) {
  const { t } = useTranslation('settings')
  return (
    <Can permission="setting.update">
      <Button type="primary" htmlType="submit" loading={loading} icon={<SaveOutlined />}>
        {children ?? (loading ? t('saving') : t('save'))}
      </Button>
    </Can>
  )
}
