import { Spin } from 'antd'
import { LoadingOutlined } from '@ant-design/icons'

interface LoadingProps {
  fullScreen?: boolean
  tip?: string
}

export default function Loading({ fullScreen = true, tip }: LoadingProps) {
  const indicator = <LoadingOutlined style={{ fontSize: 32 }} spin />

  if (fullScreen) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '100vh',
          width: '100vw',
        }}
      >
        <Spin indicator={indicator} description={tip} size="large">
          {/* Spin 需要 children 才能显示 tip */}
          <div style={{ padding: 50 }} />
        </Spin>
      </div>
    )
  }

  return <Spin indicator={indicator} description={tip} size="medium" />
}
