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
