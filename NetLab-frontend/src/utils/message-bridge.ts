/**
 * Message Bridge —— 将非 React 模块（如 request.ts 中的 Axios 拦截器）
 * 与 Ant Design 的 message API 解耦。
 *
 * Ant Design v6 在使用静态 `message` 导出时会告警，因为它无法读取
 * context（动态主题 / 语言）。官方推荐的做法是使用 `App` 组件的
 * `App.useApp()` hook —— 但该实例仅在 React 树内部可用。
 * 本 bridge 让一个顶层组件注册该实例一次，
 * 以便非 React 代码复用它。
 *
 * 用法:
 *   // 在渲染于 <App> 之下的组件中（见 App.tsx）:
 *   const { message } = App.useApp()
 *   useEffect(() => setMessageApi(message), [message])
 *
 *   // 在 request.ts 中:
 *   import { getMessageApi } from '@/utils/message-bridge'
 *   getMessageApi().error('...')
 */
import type { MessageInstance } from 'antd/es/message/interface'

/**
 * 在真实实例注册之前（或在非 UI 上下文中）使用的空操作兜底。
 * 防止拦截器在 <App> 挂载之前触发导致崩溃。
 */
const noopMessageApi = {
  open: () => noopClose,
  success: () => noopClose,
  error: () => noopClose,
  info: () => noopClose,
  warning: () => noopClose,
  loading: () => noopClose,
  destroy: () => {},
} as unknown as MessageInstance

function noopClose(): void {}

let _messageApi: MessageInstance = noopMessageApi

/** 注册 App 作用域内的 message 实例。从一个顶层组件调用一次。 */
export function setMessageApi(api: MessageInstance): void {
  _messageApi = api
}

/** 获取当前活跃的 message 实例。在注册前调用也安全（返回空操作实例）。 */
export function getMessageApi(): MessageInstance {
  return _messageApi
}
