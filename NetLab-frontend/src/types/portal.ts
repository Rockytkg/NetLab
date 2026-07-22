export interface PortalNasItem {
  id: number
  name: string
	identifier: string
  vendor: 'mobile' | 'huawei'
  protocolProfile: 'mobile-v2' | 'cmcc-v1' | 'cmcc-v2' | 'huawei-v1' | 'huawei-v2'
	sourceIp: string
	acPort: number
  radiusNasId?: number
  coaEnabled: boolean
  status: 'enabled' | 'disabled'
  remark: string
  createdAt: string
  updatedAt: string
}

export interface PortalNasPayload {
  name: string
	identifier: string
  vendor: 'mobile' | 'huawei'
  protocolProfile: 'mobile-v2' | 'cmcc-v1' | 'cmcc-v2' | 'huawei-v1' | 'huawei-v2'
	sourceIp: string
	acPort?: number
  sharedSecret?: string
  radiusNasId?: number
  coaEnabled: boolean
  status: 'enabled' | 'disabled'
  remark?: string
}

export interface PortalSessionItem {
  id: string
  portalNasId: number
  username: string
  macAddr: string
  clientIp: string
  state: 'active' | 'terminated'
  authenticatedAt: string
  lastSeenAt: string
  terminatedAt?: string
  terminateReason: string
}

export interface PortalListResult<T> { items: T[]; total: number; page: number; size: number }

export interface PortalSystemSettings { enabled: boolean; bindHost: string; notifyPort: number }
