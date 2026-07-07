import { useEffect, useRef } from 'react'
import { Spin, Typography } from 'antd'
import { LoadingOutlined } from '@ant-design/icons'

const { Text } = Typography

/**
 * OAuth 回调页面
 *
 * 该页面在 OAuth 流程中以弹窗形式打开。
 * OAuth 提供方会携带 `?code=...&state=...` 重定向到此处，
 * 我们解析这些参数并通过 postMessage 发送给父级（登录）窗口，
 * 然后关闭自身。
 */
export default function OAuthCallbackPage() {
  const doneRef = useRef(false)

  useEffect(() => {
    if (doneRef.current) return
    doneRef.current = true

    const params = new URLSearchParams(window.location.search)
    const code = params.get('code')
    const state = params.get('state')

    if (code && state && window.opener) {
      window.opener.postMessage(
        { type: 'oauth-callback', code, state },
        window.location.origin
      )
      // 关闭前给父级窗口留出片刻时间接收消息
      setTimeout(() => window.close(), 1000)
    }
  }, [])

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 16,
        background: '#f5f5f5',
      }}
    >
      <Spin indicator={<LoadingOutlined style={{ fontSize: 32 }} spin />} />
      <Text type="secondary">Completing sign in...</Text>
    </div>
  )
}
