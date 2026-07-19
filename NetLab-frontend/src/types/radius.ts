// RADIUS 认证计费相关的 API 类型定义，与后端 camelCase JSON 对齐。

export interface RadiusUserItem {
  id: number
  username: string
  profileId?: number | null
  profileName: string
  profileLinkMode: number
  realname: string
  email: string
  mobile: string
  address: string
  macAddr: string
  vlanid1: number
  vlanid2: number
  bindMac: boolean
  bindVlan: boolean
  ipAddr: string
  ipV6Addr: string
  addrPool: string
  ipv6PrefixPool: string
  delegatedIpv6Prefix: string
  delegatedIpv6PrefixPool: string
  activeNum: number
  upRate: number
  downRate: number
  domain: string
  expireTime: string
  status: string
  remark: string
  onlineCount: number
  lastOnline?: string | null
  createdAt: string
  updatedAt: string
}

export interface RadiusUserListParams {
  page?: number
  size?: number
  keyword?: string
  status?: string
}

export interface RadiusUserListResult {
  items: RadiusUserItem[]
  total: number
  page: number
  size: number
}

/** 创建/更新认证用户的请求体；password 为空表示不修改（更新场景）。
 * expireTime 可选：创建留空后端默认一年，更新留空保持不变。 */
export interface RadiusUserPayload {
  username: string
  password?: string
  profileId?: number | null
  profileLinkMode?: number
  realname?: string
  email?: string
  mobile?: string
  address?: string
  macAddr?: string
  vlanid1?: number
  vlanid2?: number
  bindMac?: boolean
  bindVlan?: boolean
  ipAddr?: string
  ipV6Addr?: string
  addrPool?: string
  ipv6PrefixPool?: string
  delegatedIpv6Prefix?: string
  delegatedIpv6PrefixPool?: string
  activeNum?: number
  upRate?: number
  downRate?: number
  domain?: string
  expireTime?: string
  status?: string
  remark?: string
}

export interface RadiusProfileItem {
  id: number
  name: string
  addrPool: string
  ipv6PrefixPool: string
  delegatedIpv6PrefixPool: string
  activeNum: number
  upRate: number
  downRate: number
  domain: string
  bindMac: boolean
  bindVlan: boolean
  status: string
  remark: string
  userCount: number
  createdAt: string
  updatedAt: string
}

export interface RadiusProfileListParams {
  page?: number
  size?: number
  keyword?: string
}

export interface RadiusProfileListResult {
  items: RadiusProfileItem[]
  total: number
  page: number
  size: number
}

export interface RadiusProfileOption {
  id: number
  name: string
}

/** 创建/更新套餐的请求体。 */
export interface RadiusProfilePayload {
  name: string
  addrPool?: string
  ipv6PrefixPool?: string
  delegatedIpv6PrefixPool?: string
  activeNum?: number
  upRate?: number
  downRate?: number
  domain?: string
  bindMac?: boolean
  bindVlan?: boolean
  status?: string
  remark?: string
}

export interface RadiusNasItem {
  id: number
  name: string
  identifier: string
  hostname: string
  ipaddr: string
  coaPort: number
  model: string
  vendorCode: string
  status: string
  tags: string
  remark: string
  createdAt: string
  updatedAt: string
}

export interface RadiusNasListParams {
  page?: number
  size?: number
  keyword?: string
}

export interface RadiusNasListResult {
  items: RadiusNasItem[]
  total: number
  page: number
  size: number
}

/** 创建/更新 NAS 设备的请求体；secret 为空表示不修改（更新场景）。 */
export interface RadiusNasPayload {
  name: string
  identifier?: string
  hostname?: string
  ipaddr: string
  secret?: string
  coaPort?: number
  model?: string
  vendorCode?: string
  status?: string
  tags?: string
  remark?: string
}

/** 免认证类型：mac=按 Calling-Station-Id 匹配，ip=按 Framed-IP-Address 匹配（支持 CIDR）。 */
export type RadiusBypassType = 'mac' | 'ip'

export interface RadiusBypassItem {
  id: number
  type: RadiusBypassType
  value: string
  status: string
  remark: string
  createdAt: string
  updatedAt: string
}

export interface RadiusBypassListParams {
  page?: number
  size?: number
  keyword?: string
}

export interface RadiusBypassListResult {
  items: RadiusBypassItem[]
  total: number
  page: number
  size: number
}

/** 创建/更新免认证规则的请求体。 */
export interface RadiusBypassPayload {
  type: RadiusBypassType
  value: string
  status?: string
  remark?: string
}

export interface RadiusSessionItem {
  id: number
  username: string
  nasAddr: string
  nasPaddr: string
  nasId: string
  nasClass: string
  framedIpaddr: string
  framedNetmask: string
  framedIpv6Prefix: string
  framedIpv6Address: string
  delegatedIpv6Prefix: string
  macAddr: string
  nasPort: number
  nasPortId: string
  serviceType: number
  nasPortType: number
  acctSessionId: string
  acctSessionTime: number
  sessionTimeout?: number
  acctInputTotal: number
  acctOutputTotal: number
  acctInputPackets: number
  acctOutputPackets: number
  acctStartTime: string
  lastUpdate: string
}

export interface RadiusSessionListParams {
  page?: number
  size?: number
  username?: string
  nasAddr?: string
  macAddr?: string
}

export interface RadiusSessionListResult {
  items: RadiusSessionItem[]
  total: number
  page: number
  size: number
}

export interface RadiusKickResult {
  success: boolean
  responseCode: string
  target: string
  message: string
}

/** CoA 下发请求体；sessionTimeout 与 filterId 至少提供一项。 */
export interface RadiusCoAPayload {
  sessionTimeout?: number
  filterId?: string
}

/** CoA 下发结果；errorCause/errorCauseText/rttMs 由 NAS 响应决定，可能缺省。 */
export interface RadiusCoAResult {
  success: boolean
  responseCode: string
  target: string
  message: string
  errorCause?: string
  errorCauseText?: string
  rttMs?: number
}

export interface RadiusAccountingItem {
  id: number
  username: string
  nasAddr: string
  nasId: string
  nasClass: string
  framedIpaddr: string
  framedNetmask: string
  framedIpv6Address: string
  framedIpv6Prefix?: string
  delegatedIpv6Prefix: string
  macAddr: string
  nasPort?: number
  nasPortId: string
  serviceType: number
  nasPortType: number
  acctSessionId: string
  acctSessionTime: number
  sessionTimeout: number
  acctInputTotal: number
  acctOutputTotal: number
  acctInputPackets: number
  acctOutputPackets: number
  acctTerminateCause: string
  acctStartTime: string
  acctStopTime?: string | null
  lastUpdate?: string | null
}

export interface RadiusAccountingListParams {
  page?: number
  size?: number
  username?: string
  startTime?: string
  endTime?: string
}

export interface RadiusAccountingListResult {
  items: RadiusAccountingItem[]
  total: number
  page: number
  size: number
}

export interface RadiusAuthLogItem {
  id: number
  username: string
  nasAddr: string
  nasPaddr: string
  macAddr: string
  authType: string
  result: string
  reason: string
  createdAt: string
}

export interface RadiusAuthLogListParams {
  page?: number
  size?: number
  keyword?: string
  result?: string
}

export interface RadiusAuthLogListResult {
  items: RadiusAuthLogItem[]
  total: number
  page: number
  size: number
}

/** 证书类型：server=服务器证书（含私钥），ca=CA 证书。 */
export type RadiusCertType = 'server' | 'ca'

export interface RadiusCertItem {
  id: number
  name: string
  certType: RadiusCertType
  certPem: string
  subject: string
  issuer: string
  serial: string
  fingerprint: string
  notBefore: string
  notAfter: string
  hasKey: boolean
  remark: string
  createdAt: string
  updatedAt: string
}

export interface RadiusCertListParams {
  page?: number
  size?: number
  keyword?: string
  certType?: string
}

export interface RadiusCertListResult {
  items: RadiusCertItem[]
  total: number
  page: number
  size: number
}

/** 创建/更新证书的请求体；更新时 certPem/keyPem 留空表示不替换，certType 不可修改。 */
export interface RadiusCertPayload {
  name: string
  certType?: RadiusCertType
  certPem?: string
  keyPem?: string
  remark?: string
}

/** RADIUS 系统设置（/settings/system）。 */
export interface RadiusSystemSettings {
  enabled: boolean
  bindHost: string
  authPort: number
  acctPort: number
  radsecEnabled: boolean
  radsecPort: number
  radsecCertId: number
  radsecCaCertId: number
}

/** RADIUS 服务行为设置（/settings/server）。 */
export interface RadiusServerSettings {
  messageAuthMode: 'disabled' | 'warn' | 'enforce'
  ignorePassword: boolean
  sessionTimeout: number
  acctInterimInterval: number
  historyDays: number
  rejectDelayMaxRejects: number
  rejectDelayWindowSeconds: number
}

/** EAP 认证方式。 */
export type RadiusEapMethod = 'eap-md5' | 'eap-mschapv2' | 'eap-tls' | 'eap-peap' | 'eap-ttls'

/** EAP 设置（/settings/eap）；enabledHandlers 为逗号分隔的方法名或 "*"。 */
export interface RadiusEapSettings {
  enabled: boolean
  method: RadiusEapMethod
  enabledHandlers: string
  tlsServerCertId: number
  tlsClientCaId: number
  tlsMinVersion: '1.2' | '1.3'
}

/** GET /settings 返回的完整配置。 */
export interface RadiusSettings {
  system: RadiusSystemSettings
  server: RadiusServerSettings
  eap: RadiusEapSettings
}

/** NAS 厂商代码选项（与后端 vendors 包的厂商代码对应）。 */
export const RADIUS_VENDOR_CODES = [
  { value: '', labelKey: 'radius:nas.vendors.default' },
  { value: '2011', labelKey: 'radius:nas.vendors.huawei' },
  { value: '14988', labelKey: 'radius:nas.vendors.mikrotik' },
  { value: '25506', labelKey: 'radius:nas.vendors.h3c' },
  { value: '3902', labelKey: 'radius:nas.vendors.zte' },
  { value: '9', labelKey: 'radius:nas.vendors.cisco' },
  { value: '14823', labelKey: 'radius:nas.vendors.aruba' },
  { value: '6527', labelKey: 'radius:nas.vendors.alcatel' },
  { value: '2636', labelKey: 'radius:nas.vendors.juniper' },
  { value: '10055', labelKey: 'radius:nas.vendors.ikuai' },
  { value: '2352', labelKey: 'radius:nas.vendors.radback' },
] as const
