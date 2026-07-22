import request from './request'
import type { PortalListResult, PortalNasItem, PortalNasPayload, PortalSessionItem } from '@/types/portal'

export const portalApi = {
	authenticate(data: { wlanacname: string; wlanuserip: string; username?: string; password?: string; authType?: 'chap' | 'pap' }): Promise<{ id: string }> { return request.post('/portal/authenticate', data) },
  listNas(params: { page?: number; size?: number; keyword?: string }): Promise<PortalListResult<PortalNasItem>> { return request.get('/portal/nas', { params }) },
  createNas(data: PortalNasPayload): Promise<PortalNasItem> { return request.post('/portal/nas', data) },
  updateNas(id: number, data: PortalNasPayload): Promise<PortalNasItem> { return request.put(`/portal/nas/${id}`, data) },
  deleteNas(id: number): Promise<{ deleted: number }> { return request.delete(`/portal/nas/${id}`) },
  listSessions(params: { page?: number; size?: number; username?: string; nasId?: string }): Promise<PortalListResult<PortalSessionItem>> { return request.get('/portal/sessions', { params }) },
  terminateSession(id: string): Promise<{ terminated: boolean }> { return request.delete(`/portal/sessions/${id}`) },
}
