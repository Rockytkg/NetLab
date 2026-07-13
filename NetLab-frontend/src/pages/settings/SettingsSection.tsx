import type { ReactNode } from 'react'
import { Flex, theme } from 'antd'

interface SettingsSectionProps {
  extra?: ReactNode
  children: ReactNode
}

export default function SettingsSection({
  extra,
  children,
}: SettingsSectionProps) {
  const { token } = theme.useToken()

  return (
    <Flex
      vertical
      className="netlab-settings-section"
      style={{
        height: '100%',
        minWidth: 0,
        paddingBlock: token.padding,
        width: '100%',
        overflow: 'hidden',
      }}
    >
      {extra && (
        <Flex justify="flex-end" style={{ marginBottom: token.marginLG }}>
          {extra}
        </Flex>
      )}
      {children}
    </Flex>
  )
}
