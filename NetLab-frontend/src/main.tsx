import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './i18n' // 初始化 i18n（必须在 App 前导入）
import './index.css'
import App from './App'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
