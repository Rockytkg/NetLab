import { useEffect, useState, useCallback, useRef, Fragment } from 'react'
import { Navigate } from 'react-router-dom'
import {
  Typography,
  Layout,
  theme,
  Spin,
  Space,
} from 'antd'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/authStore'
import {
  LockOutlined,
  CopyrightOutlined,
  SafetyCertificateOutlined,
  SafetyOutlined,
} from '@ant-design/icons'
import { authApi } from '@/services/auth'
import type { SystemConfig } from '@/types/auth'
import LoginForm from './LoginForm'
import RegisterForm from './RegisterForm'
import ForgotPasswordModal from './ForgotPasswordModal'
import OAuthSection from './OAuthSection'
import ThemeSwitcher from '@/components/common/ThemeSwitcher'
import LanguageSwitcher from '@/components/common/LanguageSwitcher'

const { Title, Text } = Typography

/* ── 动画网络拓扑画布（左侧面板装饰） ── */
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

    // 初始化节点
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

      // 更新并绘制节点
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

      // 在邻近节点之间绘制连线
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

      // 中心处缓慢脉动的光晕
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

  return (
    <canvas
      ref={canvasRef}
      style={{ position: 'absolute', inset: 0, width: '100%', height: '100%' }}
    />
  )
}

type AuthFlow = 'login' | 'register'

/* 固定的跳转链接模板 —— 现在后端只返回备案号。 */
const ICP_BEIAN_URL = 'https://beian.miit.gov.cn/#/Integrated/index'
const POLICE_BEIAN_URL = 'https://beian.mps.gov.cn/'

/* ── 单行版权 + 备案页脚 ──
   将版权、ICP、公安备案渲染在同一水平行中，每项前
   带一个 Ant Design 图标并以中点分隔。在较窄宽度下
   优雅换行（Growing）。 */
function BeianFooter({ config, inverted = false }: { config: SystemConfig | null; inverted?: boolean }) {
  const { t } = useTranslation(['login', 'common'])
  const { token: themeToken } = theme.useToken()

  const icp = config?.icpBeian?.trim()
  const police = config?.policeBeian?.trim()

  // 在深色画面上我们保持柔和的石板色调；其他情况下回退到
  // 主题的 on-surface-variant token（浅色主题解析为约 #595959）。
  const textColor = inverted ? 'rgba(148,163,184,0.7)' : themeToken.colorTextTertiary
  const linkColor = inverted ? 'rgba(148,163,184,0.9)' : themeToken.colorTextSecondary

  const itemStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: themeToken.marginXXS, // 4px —— 间距单位
    fontSize: themeToken.fontSizeSM, // body-sm：12px
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
        <span style={{ ...itemStyle, color: textColor }}>
          <CopyrightOutlined />
          {new Date().getFullYear()} {t('common:appName')}
        </span>
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
    <div
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        columnGap: themeToken.marginXS, // 各项之间 8px
        rowGap: themeToken.marginXXS,
        color: textColor,
      }}
    >
      {items.map((item, idx) => (
        <Fragment key={item.key}>
          {idx > 0 && (
            <span aria-hidden style={{ color: textColor, opacity: 0.5, fontSize: themeToken.fontSizeSM }}>·</span>
          )}
          {item.node}
        </Fragment>
      ))}
    </div>
  )
}

export default function LoginPage() {
  const { t } = useTranslation(['login', 'common'])
  const token = useAuthStore((s) => s.accessToken)
  const { token: themeToken } = theme.useToken()

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

  if (token) {
    return <Navigate to="/dashboard" replace />
  }

  return (
    <>
      {/* ── 全屏布局 ── */}
      <Layout style={{ minHeight: '100vh' }}>
        <div style={{ display: 'flex', minHeight: '100vh' }}>

          {/* ═══════════════════════════════════════
              左侧面板 —— 品牌 + 拓扑动画
              ═══════════════════════════════════════ */}
          <div
            style={{
              flex: '1 1 48%',
              position: 'relative',
              overflow: 'hidden',
              background: 'linear-gradient(160deg, #0B1A33 0%, #132347 30%, #0F2348 60%, #0A1628 100%)',
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
              padding: '80px 64px',
            }}
          >
            {/* 动画画布图层 */}
            <TopologyDecoration />

            {/* 微妙的网格叠加层 */}
            <div
              aria-hidden="true"
              style={{
                position: 'absolute', inset: 0, opacity: 0.03,
                backgroundImage: 'linear-gradient(rgba(255,255,255,0.8) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.8) 1px, transparent 1px)',
                backgroundSize: '60px 60px',
              }}
            />

            {/* 内容 */}
            <div style={{ position: 'relative', zIndex: 2, maxWidth: 520 }}>
              {/* Logo 标识 */}
              <div style={{ marginBottom: 40 }}>
                <img
                  src="/logo-mark.svg"
                  alt="NetLab"
                  width={44}
                  height={44}
                  style={{
                    display: 'block',
                    borderRadius: 12,
                    boxShadow: '0 0 32px rgba(59,130,246,0.35)',
                    marginBottom: 20,
                  }}
                />
                <h1
                  style={{
                    fontSize: 32,
                    fontWeight: 700,
                    lineHeight: 1.25,
                    margin: 0,
                    color: '#F1F5F9',
                    letterSpacing: 0,
                  }}
                >
                  {t('brandTitle')}
                </h1>
              </div>

              <p
                style={{
                  fontSize: 15,
                  lineHeight: 1.7,
                  color: 'rgba(203,213,225,0.85)',
                  margin: '0 0 48px',
                  maxWidth: 420,
                }}
              >
                {t('brandSubtitle')}
              </p>

              {/* 功能标签 */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                {[
                  { label: t('vpTopologyTitle'), desc: t('vpTopologyDesc') },
                  { label: t('vpDevicesTitle'), desc: t('vpDevicesDesc') },
                  { label: t('vpRealtimeTitle'), desc: t('vpRealtimeDesc') },
                ].map((item) => (
                  <div key={item.label} style={{ display: 'flex', gap: 14, alignItems: 'flex-start' }}>
                    <div
                      style={{
                        width: 8, height: 8,
                        borderRadius: 9999,
                        background: '#3B82F6',
                        boxShadow: '0 0 8px rgba(59,130,246,0.5)',
                        flexShrink: 0,
                        marginTop: 6,
                      }}
                    />
                    <div>
                      <div style={{ fontSize: 14, fontWeight: 600, color: '#E2E8F0', marginBottom: 2 }}>
                        {item.label}
                      </div>
                      <div style={{ fontSize: 12, color: 'rgba(148,163,184,0.8)', lineHeight: 1.5 }}>
                        {item.desc}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* 设置 + 版权/备案 —— 底部 */}
            <div style={{ position: 'relative', zIndex: 2, marginTop: 'auto', paddingTop: 48 }}>
              <Space size={4} align="center" wrap>
                <ThemeSwitcher inverted showLabel />
                <LanguageSwitcher inverted showLabel />
              </Space>
              <div style={{ marginTop: 12 }}>
                <BeianFooter config={config} inverted />
              </div>
            </div>
          </div>

          {/* ═════════════════════════════
              右侧面板 —— 表单
              ═════════════════════════════ */}
          <div
            style={{
              flex: '0 0 500px',
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
              alignItems: 'center',
              padding: '64px 56px',
              background: themeToken.colorBgContainer,
            }}
          >
            <div style={{ width: '100%', maxWidth: 380 }}>
              {/* 依赖流程的头部 */}
              {flow === 'register' ? (
                <div style={{ marginBottom: 28 }}>
                  <Title level={3} style={{ marginBottom: 4, fontWeight: 700, fontSize: 24 }}>
                    {t('registerTitle')}
                  </Title>
                  <Text type="secondary" style={{ fontSize: 14 }}>{t('registerSubtitle')}</Text>
                </div>
              ) : (
                <div style={{ marginBottom: 28 }}>
                  <Space size={8} align="center">
                    <img
                      src="/logo-mark.svg"
                      alt=""
                      aria-hidden
                      width={20}
                      height={20}
                      style={{ display: 'block', borderRadius: themeToken.borderRadiusSM }}
                    />
                    <span style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                      <Text style={{ fontSize: 13, fontWeight: 600, color: themeToken.colorText, lineHeight: '18px', letterSpacing: 0 }}>
                        {t('common:appName')}
                      </Text>
                      <Text style={{ fontSize: 11, color: themeToken.colorTextTertiary, lineHeight: '16px', letterSpacing: 0 }}>
                        {t('brandShortSubtitle')}
                      </Text>
                    </span>
                  </Space>
                  <Title level={3} style={{ marginTop: 4, marginBottom: 4, fontWeight: 700, fontSize: 24 }}>
                    {t('welcome')}
                  </Title>
                  <Text type="secondary" style={{ fontSize: 14 }}>{t('subtitle')}</Text>
                </div>
              )}

              {configLoading ? (
                <div style={{ padding: 48, textAlign: 'center' }}>
                  <Spin size="small" />
                </div>
              ) : (
                <>
                  {/* 主表单 */}
                  {flow === 'register' ? (
                    <RegisterForm onBack={handleBackToLogin} />
                  ) : (
                    <LoginForm
                      config={config}
                      onForgotPassword={handleForgotPassword}
                      onRegister={handleRegister}
                    />
                  )}

                  {/* 备用认证方式 —— Passkey + OAuth 同排显示 */}
                  {flow === 'login' && (
                    <OAuthSection
                      providers={config?.oauthProviders ?? []}
                      passkeyEnabled={config?.passkeyEnabled ?? false}
                    />
                  )}

                  {/* 安全页脚 */}
                  {flow === 'login' && (
                    <div style={{ marginTop: 28, textAlign: 'center' }}>
                      <Space size={4}>
                        <LockOutlined style={{ color: themeToken.colorTextQuaternary, fontSize: 11 }} />
                        <Text type="secondary" style={{ fontSize: 11 }}>{t('secureTip')}</Text>
                      </Space>
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        </div>
      </Layout>

      {config?.passwordResetEnabled !== false && (
        <ForgotPasswordModal open={forgotOpen} onClose={() => setForgotOpen(false)} />
      )}
    </>
  )
}
