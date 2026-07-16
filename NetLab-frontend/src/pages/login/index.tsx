import { useEffect, useState, useCallback, useRef, Fragment } from 'react'
import { Navigate } from 'react-router-dom'
import {
  Typography,
  Layout,
  theme,
  Spin,
  Space,
  Flex,
  Grid,
  Card,
  Row,
  Col,
  Tag,
} from 'antd'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/authStore'
import {
  LockOutlined,
  CopyrightOutlined,
  SafetyCertificateOutlined,
  SafetyOutlined,
  ClusterOutlined,
  RadarChartOutlined,
  DeploymentUnitOutlined,
  ControlOutlined,
} from '@ant-design/icons'
import { authApi } from '@/services/auth'
import type { SystemConfig } from '@/types/auth'
import LoginForm from './LoginForm'
import RegisterForm from './RegisterForm'
import ForgotPasswordModal from './ForgotPasswordModal'
import OAuthSection from './OAuthSection'
import ThemeSwitcher from '@/components/common/ThemeSwitcher'
import LanguageSwitcher from '@/components/common/LanguageSwitcher'
import '@/assets/css/login.css'

const { Title, Text, Paragraph } = Typography
const { Content } = Layout
const { useBreakpoint } = Grid

function TopologyDecoration() {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    let raf = 0
    const nodes: { x: number; y: number; r: number; vx: number; vy: number; pulse: number }[] = []
    const NODE_COUNT = 25

    function resize() {
      const parent = canvas!.parentElement!
      canvas!.width = parent.offsetWidth
      canvas!.height = parent.offsetHeight
    }
    resize()
    window.addEventListener('resize', resize)

    for (let i = 0; i < NODE_COUNT; i++) {
      nodes.push({
        x: Math.random() * canvas.width,
        y: Math.random() * canvas.height,
        r: 1.5 + Math.random() * 2.5,
        vx: (Math.random() - 0.5) * 0.3,
        vy: (Math.random() - 0.5) * 0.3,
        pulse: Math.random() * Math.PI * 2,
      })
    }

    function draw() {
      const w = canvas!.width
      const h = canvas!.height
      ctx!.clearRect(0, 0, w, h)

      const t = Date.now() / 1000
      for (const n of nodes) {
        n.x += n.vx
        n.y += n.vy
        if (n.x < 0) n.x = w
        if (n.x > w) n.x = 0
        if (n.y < 0) n.y = h
        if (n.y > h) n.y = 0

        n.pulse += 0.02
        const alpha = 0.25 + Math.sin(n.pulse) * 0.2

        ctx!.beginPath()
        ctx!.arc(n.x, n.y, n.r, 0, Math.PI * 2)
        ctx!.fillStyle = `rgba(255,255,255,${alpha})`
        ctx!.fill()
      }

      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const dx = nodes[i].x - nodes[j].x
          const dy = nodes[i].y - nodes[j].y
          const dist = Math.sqrt(dx * dx + dy * dy)
          if (dist < 160) {
            const alpha = (1 - dist / 160) * 0.12
            ctx!.beginPath()
            ctx!.moveTo(nodes[i].x, nodes[i].y)
            ctx!.lineTo(nodes[j].x, nodes[j].y)
            ctx!.strokeStyle = `rgba(255,255,255,${alpha})`
            ctx!.lineWidth = 0.6
            ctx!.stroke()
          }
        }
      }

      const glowAlpha = 0.03 + Math.sin(t * 0.5) * 0.015
      const glow = ctx!.createRadialGradient(w * 0.4, h * 0.4, 0, w * 0.4, h * 0.4, Math.max(w, h) * 0.6)
      glow.addColorStop(0, `rgba(96,165,250,${glowAlpha * 3})`)
      glow.addColorStop(1, 'transparent')
      ctx!.fillStyle = glow
      ctx!.fillRect(0, 0, w, h)

      raf = requestAnimationFrame(draw)
    }
    draw()

    return () => {
      cancelAnimationFrame(raf)
      window.removeEventListener('resize', resize)
    }
  }, [])

  return <canvas ref={canvasRef} className="netlab-auth-canvas" />
}

type AuthFlow = 'login' | 'register'

const ICP_BEIAN_URL = 'https://beian.miit.gov.cn/#/Integrated/index'
const POLICE_BEIAN_URL = 'https://beian.mps.gov.cn/'

function BeianFooter({ config, inverted = false }: { config: SystemConfig | null; inverted?: boolean }) {
  const { t } = useTranslation(['login', 'common'])
  const { token } = theme.useToken()

  const icp = config?.icpBeian?.trim()
  const police = config?.policeBeian?.trim()
  const textColor = inverted ? 'rgba(148,163,184,0.72)' : token.colorTextTertiary
  const linkColor = inverted ? 'rgba(203,213,225,0.9)' : token.colorTextSecondary

  const itemStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: token.marginXXS,
    color: textColor,
    fontSize: token.fontSizeSM,
    lineHeight: '20px',
    whiteSpace: 'nowrap',
  }
  const linkStyle: React.CSSProperties = {
    ...itemStyle,
    color: linkColor,
    textDecoration: 'none',
  }

  type FooterItem = { key: string; node: React.ReactNode }
  const items: FooterItem[] = [
    {
      key: 'copyright',
      node: (
        <Text style={itemStyle}>
          <CopyrightOutlined />
          {new Date().getFullYear()} {t('common:appName')}
        </Text>
      ),
    },
  ]

  if (icp) {
    items.push({
      key: 'icp',
      node: (
        <a href={ICP_BEIAN_URL} target="_blank" rel="noreferrer" style={linkStyle} title={t('icpBeianTip')}>
          <SafetyCertificateOutlined />
          {icp}
        </a>
      ),
    })
  }

  if (police) {
    items.push({
      key: 'police',
      node: (
        <a href={POLICE_BEIAN_URL} target="_blank" rel="noreferrer" style={linkStyle} title={t('policeBeianTip')}>
          <SafetyOutlined />
          {police}
        </a>
      ),
    })
  }

  return (
    <Flex wrap align="center" gap={token.marginXS}>
      {items.map((item, index) => (
        <Fragment key={item.key}>
          {index > 0 && <Text style={{ color: textColor, opacity: 0.5 }}>·</Text>}
          {item.node}
        </Fragment>
      ))}
    </Flex>
  )
}

export default function LoginPage() {
  const { t } = useTranslation(['login', 'common', 'operations'])
  const accessToken = useAuthStore((s) => s.accessToken)
  const { token } = theme.useToken()
  const screens = useBreakpoint()
  const isDesktop = Boolean(screens.lg)
  const isCompact = !screens.md

  const [flow, setFlow] = useState<AuthFlow>('login')
  const [forgotOpen, setForgotOpen] = useState(false)
  const [config, setConfig] = useState<SystemConfig | null>(null)
  const [configLoading, setConfigLoading] = useState(true)

  useEffect(() => {
    authApi.getSystemConfig()
      .then(setConfig)
      .catch(() => {
        setConfig({ registrationEnabled: true, captchaEnabled: false, passkeyEnabled: false, passwordResetEnabled: true, oauthProviders: [] })
      })
      .finally(() => setConfigLoading(false))
  }, [])

  const handleForgotPassword = useCallback(() => {
    if (config?.passwordResetEnabled === false) return
    setForgotOpen(true)
  }, [config?.passwordResetEnabled])

  const handleRegister = useCallback(() => setFlow('register'), [])
  const handleBackToLogin = useCallback(() => setFlow('login'), [])

  if (accessToken) {
    return <Navigate to="/dashboard" replace />
  }

  const capabilityDomains: Array<{ label: string; desc: string; icon: React.ReactNode; color: string }> = [
    { label: t('vpTopologyTitle'), desc: t('vpTopologyDesc'), icon: <ClusterOutlined />, color: '#22d3ee' },
    { label: t('vpDevicesTitle'), desc: t('vpDevicesDesc'), icon: <RadarChartOutlined />, color: '#60a5fa' },
    { label: t('vpRealtimeTitle'), desc: t('vpRealtimeDesc'), icon: <DeploymentUnitOutlined />, color: '#34d399' },
  ]

  const capabilityTags = ['SNMP', 'Syslog', 'RADIUS', 'NETCONF', 'Telemetry', 'RoCE', 'Flow', 'gNMI']

  const header = flow === 'register'
    ? { title: t('registerTitle'), subtitle: t('registerSubtitle') }
    : { title: t('welcome'), subtitle: t('subtitle') }

  return (
    <>
      <Layout className="netlab-auth-layout">
        <Flex className="netlab-auth-shell" align="stretch">
          {isDesktop && (
            <Content className="netlab-auth-visual">
              <TopologyDecoration />
              <Flex
                vertical
                justify="space-between"
                className="netlab-auth-visual-inner"
                style={{ padding: 'clamp(40px, 6vh, 80px) clamp(40px, 5vw, 72px)' }}
              >
                <Flex vertical justify="center" flex="1 1 auto" gap={token.marginXL}>
                  <Row gutter={[token.marginXL, token.marginLG]} align="middle">
                    <Col span={screens.xxl ? 11 : 24}>
                      <Space orientation="vertical" size={token.margin}>
                        <Flex align="center" gap={token.margin} wrap="nowrap">
                          <img
                            src="/logo-mark.svg"
                            alt="NetLab"
                            width={48}
                            height={48}
                            style={{
                              display: 'block',
                              flex: '0 0 auto',
                              borderRadius: token.borderRadiusLG,
                              boxShadow: '0 0 32px rgba(20, 184, 166, 0.26)',
                            }}
                          />
                          <Title level={1} style={{ margin: 0, color: '#f1f5f9', fontSize: screens.xxl ? 40 : 34, lineHeight: 1.2 }}>
                            {t('brandTitle')}
                          </Title>
                        </Flex>
                        <Paragraph style={{ maxWidth: 560, margin: 0, color: 'rgba(203,213,225,0.88)', fontSize: 15, lineHeight: 1.75 }}>
                          {t('brandSubtitle')}
                        </Paragraph>
                      </Space>
                    </Col>

                    <Col span={screens.xxl ? 13 : 24}>
                      <Row gutter={[token.marginSM, token.marginSM]} className="netlab-auth-feature-stack">
                        {capabilityDomains.map((item) => (
                          <Col span={screens.xl ? 8 : 24} key={item.label}>
                            <Card
                              size="small"
                              variant="borderless"
                              styles={{ body: { height: '100%', padding: `${token.padding}px ${token.paddingLG}px` } }}
                              style={{
                                height: '100%',
                                background: 'rgba(15, 23, 42, 0.34)',
                                backdropFilter: 'blur(8px)',
                                border: '1px solid rgba(148, 163, 184, 0.16)',
                              }}
                            >
                              <Space orientation="vertical" size={token.marginXS}>
                                <span style={{ color: item.color, fontSize: 22, lineHeight: 1 }}>
                                  {item.icon}
                                </span>
                                <Text strong style={{ color: '#e2e8f0' }}>{item.label}</Text>
                                <Text style={{ color: 'rgba(148,163,184,0.86)', fontSize: token.fontSizeSM, lineHeight: 1.6 }}>
                                  {item.desc}
                                </Text>
                              </Space>
                            </Card>
                          </Col>
                        ))}
                      </Row>
                    </Col>
                  </Row>

                  <Card
                    size="small"
                    variant="borderless"
                    className="netlab-auth-protocol-card"
                    styles={{ body: { padding: `${token.paddingSM}px ${token.paddingLG}px` } }}
                    style={{
                      background: 'rgba(15, 23, 42, 0.28)',
                      border: '1px solid rgba(148, 163, 184, 0.14)',
                    }}
                  >
                    <Flex justify="space-between" align="center" gap={token.margin} wrap>
                      <Space align="center" size={token.marginSM}>
                        <ControlOutlined style={{ color: '#93c5fd', fontSize: token.fontSizeLG }} />
                        <Text style={{ color: 'rgba(226,232,240,0.9)', fontSize: token.fontSizeSM }}>
                          {t('protocolEcosystem')}
                        </Text>
                      </Space>
                      <Space size={[token.marginXS, token.marginXS]} wrap>
                        {capabilityTags.map((item) => (
                          <Tag key={item} variant="filled" style={{ marginInlineEnd: 0, color: '#cbd5e1', background: 'rgba(148, 163, 184, 0.14)' }}>
                            {item}
                          </Tag>
                        ))}
                      </Space>
                    </Flex>
                  </Card>
                </Flex>

                <Space orientation="vertical" size={token.marginXS}>
                  <Space size={4} align="center" wrap>
                    <ThemeSwitcher inverted showLabel />
                    <LanguageSwitcher inverted showLabel />
                  </Space>
                  <BeianFooter config={config} inverted />
                </Space>
              </Flex>
            </Content>
          )}

          <Content
            style={{
              flex: isDesktop ? '0 0 min(500px, 46vw)' : '1 1 auto',
              minWidth: 0,
              background: token.colorBgContainer,
            }}
          >
            <Flex
              vertical
              justify="center"
              align="center"
              className="netlab-auth-panel"
              style={{
                height: '100%',
                padding: isCompact ? token.padding : `${token.paddingXL * 2}px ${token.paddingXL + token.padding}px`,
              }}
            >
              {!isDesktop && (
                <Flex justify="space-between" align="center" gap={token.marginSM} style={{ width: '100%', maxWidth: 420, marginBottom: token.marginLG }}>
                  <Space size={token.marginSM} align="center">
                    <img src="/logo-mark.svg" alt="" aria-hidden width={24} height={24} style={{ display: 'block', borderRadius: token.borderRadius }} />
                    <Space orientation="vertical" size={0}>
                      <Text strong>{t('common:appName')}</Text>
                      <Text type="secondary" style={{ fontSize: token.fontSizeSM }}>{t('brandShortSubtitle')}</Text>
                    </Space>
                  </Space>
                  <Space size={0}>
                    <ThemeSwitcher />
                    <LanguageSwitcher />
                  </Space>
                </Flex>
              )}

              <Card
                variant="borderless"
                className="netlab-auth-form-card"
                styles={{
                  body: {
                    padding: isCompact ? token.paddingLG : token.paddingXL,
                  },
                }}
                style={{
                  width: '100%',
                  maxWidth: isCompact ? 420 : 404,
                  background: isDesktop ? 'transparent' : token.colorBgElevated,
                  boxShadow: isDesktop ? 'none' : token.boxShadowTertiary,
                }}
              >
                <Space orientation="vertical" size={isCompact ? token.margin : token.marginLG} style={{ width: '100%' }}>
                  <Space orientation="vertical" size={token.marginXXS} style={{ width: '100%' }}>
                    <Flex align="center" gap={token.marginXS}>
                      <img
                        src="/logo-mark.svg"
                        alt=""
                        aria-hidden
                        width={isCompact ? 24 : 28}
                        height={isCompact ? 24 : 28}
                        style={{ display: 'block', flex: '0 0 auto', borderRadius: token.borderRadiusSM }}
                      />
                      <Title level={3} style={{ margin: 0, fontSize: isCompact ? 22 : 24, lineHeight: 1.3 }}>
                        {header.title}
                      </Title>
                    </Flex>
                    <Text type="secondary">{header.subtitle}</Text>
                  </Space>

                  {configLoading ? (
                    <Flex justify="center" align="center" style={{ minHeight: 220 }}>
                      <Spin size="small" />
                    </Flex>
                  ) : (
                    <Space orientation="vertical" size={0} style={{ width: '100%' }}>
                      {flow === 'register' ? (
                        <RegisterForm onBack={handleBackToLogin} />
                      ) : (
                        <LoginForm
                          config={config}
                          onForgotPassword={handleForgotPassword}
                          onRegister={handleRegister}
                        />
                      )}

                      {flow === 'login' && (
                        <OAuthSection
                          providers={config?.oauthProviders ?? []}
                          passkeyEnabled={config?.passkeyEnabled ?? false}
                        />
                      )}

                      {flow === 'login' && (
                        <Flex justify="center" align="center" gap={token.marginXXS} className="netlab-auth-secure-tip">
                          <LockOutlined style={{ color: token.colorTextQuaternary, fontSize: 11 }} />
                          <Text type="secondary" style={{ fontSize: 11 }}>{t('secureTip')}</Text>
                        </Flex>
                      )}
                    </Space>
                  )}
                </Space>
              </Card>
            </Flex>
          </Content>
        </Flex>
      </Layout>

      {config?.passwordResetEnabled !== false && (
        <ForgotPasswordModal open={forgotOpen} onClose={() => setForgotOpen(false)} />
      )}
    </>
  )
}
