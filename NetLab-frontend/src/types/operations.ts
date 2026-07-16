export type DeviceStatus = 'online' | 'offline' | 'warning' | 'critical'

export type DeviceType = 'router' | 'switch' | 'firewall' | 'server' | 'load_balancer' | 'wireless' | 'other'

export interface DeviceGroup {
  id: string
  name: string
  status: DeviceStatus
  deviceCount: number
  onlineCount: number
  alertCount: number
  cpuUsage: number
  memUsage: number
  createdAt: string
  updatedAt: string
  description?: string
}

export interface ManagedDevice {
  id: string
  name: string
  type: DeviceType
  status: DeviceStatus
  vendor: string
  model: string
  managementIp: string
  site?: string
  groupId?: string
  snmpEnabled: boolean
  syslogEnabled: boolean
  radiusEnabled: boolean
  lastSeenAt?: string
}

export interface DevicePort {
  id: string
  name: string
  type: 'ethernet' | 'fiber' | 'serial' | 'loopback' | 'management'
  index: number
}

export interface DeviceTopologyNode {
  id: string
  deviceId: string
  x: number
  y: number
  label?: string
  status: DeviceStatus
}

export interface DeviceTopologyLink {
  id: string
  source: string
  target: string
  sourcePort: string
  targetPort: string
  status: DeviceStatus
  label?: string
}

export interface OperationsFilter {
  status: DeviceStatus | null
  search: string
}

export const DEVICE_STATUS_CONFIG: Record<DeviceStatus, { color: string; labelKey: string }> = {
  online: { color: 'success', labelKey: 'operations:statusOnline' },
  offline: { color: 'default', labelKey: 'operations:statusOffline' },
  warning: { color: 'warning', labelKey: 'operations:statusWarning' },
  critical: { color: 'error', labelKey: 'operations:statusCritical' },
}

export const DEVICE_TYPE_COLORS: Record<DeviceType, string> = {
  router: '#1677FF',
  switch: '#13C2C2',
  firewall: '#F5222D',
  server: '#722ED1',
  load_balancer: '#FA8C16',
  wireless: '#52C41A',
  other: '#8C8C8C',
}

export const DEVICE_TYPE_ICONS: Record<DeviceType, string> = {
  router: 'router',
  switch: 'switch',
  firewall: 'firewall',
  server: 'server',
  load_balancer: 'load_balancer',
  wireless: 'wireless',
  other: 'other',
}
