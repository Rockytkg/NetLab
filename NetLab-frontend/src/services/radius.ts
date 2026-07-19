import request from './request'
import type {
  RadiusAccountingListParams,
  RadiusAccountingListResult,
  RadiusAuthLogListParams,
  RadiusAuthLogListResult,
  RadiusBypassItem,
  RadiusBypassListParams,
  RadiusBypassListResult,
  RadiusBypassPayload,
  RadiusCertItem,
  RadiusCertListParams,
  RadiusCertListResult,
  RadiusCertPayload,
  RadiusCoAPayload,
  RadiusCoAResult,
  RadiusEapSettings,
  RadiusKickResult,
  RadiusNasItem,
  RadiusNasListParams,
  RadiusNasListResult,
  RadiusNasPayload,
  RadiusProfileItem,
  RadiusProfileListParams,
  RadiusProfileListResult,
  RadiusProfileOption,
  RadiusProfilePayload,
  RadiusServerSettings,
  RadiusSessionListParams,
  RadiusSessionListResult,
  RadiusSettings,
  RadiusSystemSettings,
  RadiusUserItem,
  RadiusUserListParams,
  RadiusUserListResult,
  RadiusUserPayload,
} from '@/types/radius'

/** 触发浏览器将 Blob 保存为本地文件（与 admin.ts 同款的轻量实现）。 */
function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** RADIUS 认证计费管理 API。 */
export const radiusApi = {
  // —— 认证用户 ——
  listUsers(params: RadiusUserListParams): Promise<RadiusUserListResult> {
    return request.get('/radius/users', { params })
  },
  createUser(data: RadiusUserPayload): Promise<RadiusUserItem> {
    return request.post('/radius/users', data)
  },
  updateUser(id: number, data: RadiusUserPayload): Promise<RadiusUserItem> {
    return request.put(`/radius/users/${id}`, data)
  },
  deleteUser(id: number): Promise<{ deleted: number }> {
    return request.delete(`/radius/users/${id}`)
  },

  // —— 策略套餐 ——
  listProfiles(params: RadiusProfileListParams): Promise<RadiusProfileListResult> {
    return request.get('/radius/profiles', { params })
  },
  listProfileOptions(): Promise<RadiusProfileOption[]> {
    return request.get('/radius/profiles/options')
  },
  createProfile(data: RadiusProfilePayload): Promise<RadiusProfileItem> {
    return request.post('/radius/profiles', data)
  },
  updateProfile(id: number, data: RadiusProfilePayload): Promise<RadiusProfileItem> {
    return request.put(`/radius/profiles/${id}`, data)
  },
  deleteProfile(id: number): Promise<{ deleted: number }> {
    return request.delete(`/radius/profiles/${id}`)
  },

  // —— NAS 设备 ——
  listNas(params: RadiusNasListParams): Promise<RadiusNasListResult> {
    return request.get('/radius/nas', { params })
  },
  createNas(data: RadiusNasPayload): Promise<RadiusNasItem> {
    return request.post('/radius/nas', data)
  },
  updateNas(id: number, data: RadiusNasPayload): Promise<RadiusNasItem> {
    return request.put(`/radius/nas/${id}`, data)
  },
  deleteNas(id: number): Promise<{ deleted: number }> {
    return request.delete(`/radius/nas/${id}`)
  },

  // —— 免认证 ——
  listBypass(params: RadiusBypassListParams): Promise<RadiusBypassListResult> {
    return request.get('/radius/bypass', { params })
  },
  createBypass(data: RadiusBypassPayload): Promise<RadiusBypassItem> {
    return request.post('/radius/bypass', data)
  },
  updateBypass(id: number, data: RadiusBypassPayload): Promise<RadiusBypassItem> {
    return request.put(`/radius/bypass/${id}`, data)
  },
  deleteBypass(id: number): Promise<{ deleted: number }> {
    return request.delete(`/radius/bypass/${id}`)
  },

  // —— 在线会话 ——
  listSessions(params: RadiusSessionListParams): Promise<RadiusSessionListResult> {
    return request.get('/radius/sessions', { params })
  },
  kickSession(id: number): Promise<RadiusKickResult> {
    return request.delete(`/radius/sessions/${id}`)
  },
  /** 向会话所在 NAS 下发 CoA（修改 Session-Timeout / Filter-Id）。 */
  coaSession(id: number, data: RadiusCoAPayload): Promise<RadiusCoAResult> {
    return request.post(`/radius/sessions/${id}/coa`, data)
  },

  // —— 记账记录 ——
  listAccounting(params: RadiusAccountingListParams): Promise<RadiusAccountingListResult> {
    return request.get('/radius/accounting', { params })
  },

  // —— 认证日志 ——
  listAuthLogs(params: RadiusAuthLogListParams): Promise<RadiusAuthLogListResult> {
    return request.get('/radius/auth-logs', { params })
  },
  deleteAuthLogs(ids: number[]): Promise<{ deleted: number }> {
    return request.delete('/radius/auth-logs', { data: { ids } })
  },

  // —— 证书管理 ——
  listCerts(params: RadiusCertListParams): Promise<RadiusCertListResult> {
    return request.get('/radius/certs', { params })
  },
  createCert(data: RadiusCertPayload): Promise<RadiusCertItem> {
    return request.post('/radius/certs', data)
  },
  updateCert(id: number, data: RadiusCertPayload): Promise<RadiusCertItem> {
    return request.put(`/radius/certs/${id}`, data)
  },
  deleteCert(id: number): Promise<{ deleted: number }> {
    return request.delete(`/radius/certs/${id}`)
  },
  /** 导出证书 PEM 文件并触发浏览器下载；includeKey 仅对持有私钥的服务器证书有效。 */
  async exportCert(id: number, name: string, includeKey: boolean): Promise<void> {
    // 响应拦截器对非信封响应（Blob）原样透传；出错时走统一的 HTTP 错误提示
    const blob = (await request.get(`/radius/certs/${id}/export`, {
      params: { includeKey },
      responseType: 'blob',
    })) as unknown as Blob
    saveBlob(blob, `${name}${includeKey ? '-with-key' : ''}.pem`)
  },

  // —— RADIUS 设置 ——
  getSettings(): Promise<RadiusSettings> {
    return request.get('/radius/settings')
  },
  updateSystemSettings(data: RadiusSystemSettings): Promise<RadiusSystemSettings> {
    return request.put('/radius/settings/system', data)
  },
  updateServerSettings(data: RadiusServerSettings): Promise<RadiusServerSettings> {
    return request.put('/radius/settings/server', data)
  },
  updateEapSettings(data: RadiusEapSettings): Promise<RadiusEapSettings> {
    return request.put('/radius/settings/eap', data)
  },
}
