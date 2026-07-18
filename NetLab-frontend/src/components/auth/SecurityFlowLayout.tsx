import type { ReactNode } from 'react'
import { Card, Col, Flex, Grid, Row, Steps, Typography, theme } from 'antd'
import { useTranslation } from 'react-i18next'

const { Title, Text } = Typography
const { useBreakpoint } = Grid

export interface SecurityFlowStep {
  title: ReactNode
  content?: ReactNode
  icon?: ReactNode
}

interface SecurityFlowLayoutProps {
  /** 流程标题 */
  title: string
  /** 流程副标题 */
  subtitle: string
  /** 流程步骤（桌面端竖向展示；移动端折叠为仅标题的横向步骤条） */
  steps: SecurityFlowStep[]
  /** 右侧操作区内容 */
  children: ReactNode
}

/**
 * 强制安全流程（强制改密 / 强制绑定两步验证）的整页布局：
 * 单张悬浮卡片，视觉语言对齐登录页；桌面端左栏品牌与流程说明、
 * 右侧操作区，移动端折叠为上下堆叠的紧凑头部。
 */
export default function SecurityFlowLayout({ title, subtitle, steps, children }: SecurityFlowLayoutProps) {
  const { t } = useTranslation('common')
  const { token } = theme.useToken()
  const screens = useBreakpoint()
  const isCompact = !screens.md

  return (
    <Flex
      justify='center'
      align='center'
      style={{
        minHeight: '100dvh',
        padding: isCompact ? token.padding : token.paddingLG,
        background: token.colorBgLayout,
      }}
    >
      <Card
        variant='borderless'
        styles={{ body: { padding: 0 } }}
        style={{
          width: '100%',
          maxWidth: 900,
          overflow: 'hidden',
          background: token.colorBgElevated,
          boxShadow: token.boxShadowTertiary,
        }}
      >
        <Row align='stretch'>
          <Col xs={24} md={10} style={{ background: token.colorFillAlter }}>
            <Flex
              vertical
              gap={isCompact ? token.margin : token.marginXL}
              style={{ height: '100%', padding: isCompact ? token.paddingLG : token.paddingXL }}
            >
              <Flex align='center' gap={token.marginSM}>
                <img
                  src='/logo-mark.svg'
                  alt=''
                  aria-hidden
                  width={isCompact ? 24 : 28}
                  height={isCompact ? 24 : 28}
                  style={{ display: 'block', borderRadius: token.borderRadiusSM }}
                />
                <Text strong style={{ fontSize: isCompact ? token.fontSize : token.fontSizeLG }}>
                  {t('appName')}
                </Text>
              </Flex>
              <Flex vertical gap={token.marginXS}>
                <Title level={isCompact ? 3 : 2} style={{ margin: 0 }}>
                  {title}
                </Title>
                <Text type='secondary'>{subtitle}</Text>
              </Flex>
              {isCompact ? (
                <Steps size='small' current={0} items={steps.map(({ title: stepTitle }) => ({ title: stepTitle }))} />
              ) : (
                <Steps orientation='vertical' current={0} items={steps} />
              )}
            </Flex>
          </Col>
          <Col xs={24} md={14}>
            <Flex
              vertical
              justify='center'
              style={{ height: '100%', padding: isCompact ? token.paddingLG : token.paddingXL }}
            >
              {children}
            </Flex>
          </Col>
        </Row>
      </Card>
    </Flex>
  )
}
