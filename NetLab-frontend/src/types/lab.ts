/** 实验室状态 */
export type LabStatus = 'running' | 'stopped' | 'error' | 'paused'

/** 设备类型 */
export type DeviceType = 'router' | 'switch' | 'firewall' | 'server' | 'endpoint' | 'cloud'

/** 实验室项 */
export interface LabItem {
  id: string
  name: string
  status: LabStatus
  nodeCount: number
  cpuUsage: number
  memUsage: number
  createdAt: string
  updatedAt: string
  description?: string
}

/** 设备项 */
export interface DeviceItem {
  id: string
  name: string
  type: DeviceType
  vendor: string
  model: string
  icon: string
  ports: DevicePort[]
}

/** 设备端口 */
export interface DevicePort {
  id: string
  name: string
  type: 'ethernet' | 'serial' | 'loopback' | 'management'
  index: number
}

/** 拓扑节点 */
export interface TopologyNode {
  id: string
  deviceId: string
  x: number
  y: number
  label?: string
}

/** 拓扑边 */
export interface TopologyEdge {
  id: string
  source: string
  target: string
  sourcePort: string
  targetPort: string
  label?: string
}

/** 实验室筛选条件 */
export interface LabFilter {
  status: LabStatus | null
  search: string
}

/** 状态标签配置 */
export const LAB_STATUS_CONFIG: Record<LabStatus, { color: string; labelKey: string }> = {
  running: { color: 'success', labelKey: 'lab:statusRunning' },
  stopped: { color: 'default', labelKey: 'lab:statusStopped' },
  error: { color: 'error', labelKey: 'lab:statusError' },
  paused: { color: 'warning', labelKey: 'lab:statusPaused' },
}

/** 设备类型颜色映射（Ant Design 预设色板） */
export const DEVICE_TYPE_COLORS: Record<DeviceType, string> = {
  router: '#1677FF',
  switch: '#13C2C2',
  firewall: '#F5222D',
  server: '#722ED1',
  endpoint: '#52C41A',
  cloud: '#2F54EB',
}

/** 设备类型图标标识 */
export const DEVICE_TYPE_ICONS: Record<DeviceType, string> = {
  router: 'router',
  switch: 'switch',
  firewall: 'firewall',
  server: 'server',
  endpoint: 'endpoint',
  cloud: 'cloud',
}
