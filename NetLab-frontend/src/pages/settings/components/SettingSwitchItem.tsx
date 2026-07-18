import { Flex, Form, Switch, Typography } from 'antd'
import type { ReactNode } from 'react'

const { Text } = Typography

interface SettingSwitchItemProps {
  /** 表单字段名 */
  name: string
  label: ReactNode
  help?: ReactNode
}

/** 设置项开关行：左侧为名称与说明，右侧为接入表单的 Switch。 */
export default function SettingSwitchItem({ name, label, help }: SettingSwitchItemProps) {
  return (
    <Flex align="flex-start" justify="space-between" gap="middle">
      <Flex vertical flex={1}>
        <Text>{label}</Text>
        {help ? <Text type="secondary">{help}</Text> : null}
      </Flex>
      <Form.Item name={name} valuePropName="checked" noStyle>
        <Switch />
      </Form.Item>
    </Flex>
  )
}
