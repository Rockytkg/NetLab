import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './i18n' // 初始化 i18n（必须在 App 前导入）
import './index.css'
import App from './App'
import { initFingerprint } from './utils/fingerprint'
import { initClientInfo } from './utils/clientInfo'

// 提前采集浏览器指纹与客户端信息（fire-and-forget），供登录请求携带
initFingerprint()
initClientInfo()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
