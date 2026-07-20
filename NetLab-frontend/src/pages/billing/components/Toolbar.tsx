import { Flex, Grid } from 'antd'
import type { ReactNode } from 'react'
import '@/assets/css/billing.css'

interface ToolbarProps {
  left?: ReactNode
  right?: ReactNode
  rightFullWidth?: boolean
}

/** 工具栏：操作区与筛选区在桌面端保持为完整分组，窄屏再整体重排。 */
export default function Toolbar({ left, right, rightFullWidth = false }: ToolbarProps) {
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.sm

  return (
    <Flex
      gap="small"
      wrap
      justify="space-between"
      align={isMobile ? 'stretch' : 'center'}
      style={{ width: '100%', marginBottom: 16 }}
    >
      <Flex
        gap="small"
        wrap
        align="center"
        flex={isMobile ? '1 1 100%' : '0 0 auto'}
        style={isMobile ? { width: '100%' } : undefined}
      >
        {left}
      </Flex>
      <Flex
        gap="small"
        wrap={isMobile}
        align="center"
        flex={isMobile || rightFullWidth ? '1 1 100%' : '0 1 auto'}
        style={{ minWidth: 0, maxWidth: '100%', ...(isMobile || rightFullWidth ? { width: '100%' } : {}) }}
      >
        {right}
      </Flex>
    </Flex>
  )
}
