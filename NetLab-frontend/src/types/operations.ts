export type DeviceStatus = 'online' | 'offline' | 'warning' | 'critical'

export interface OperationsFilter {
  status: DeviceStatus | null
  search: string
}

export const DEVICE_STATUS_CONFIG: Record<DeviceStatus, { labelKey: string; color: string }> = {
  online: { labelKey: 'operations:statusOnline', color: 'success' },
  offline: { labelKey: 'operations:statusOffline', color: 'default' },
  warning: { labelKey: 'operations:statusWarning', color: 'warning' },
  critical: { labelKey: 'operations:statusCritical', color: 'error' },
}

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
