import { useOutletContext } from 'react-router-dom'
import type { AdminSettings, OAuthProviderSettings } from '@/types/settings'

/** 系统设置布局通过 Outlet context 向各子路由面板分发设置数据。 */
export interface SettingsOutletContext {
  settings: AdminSettings
  patchSettings: (partial: Partial<AdminSettings>) => void
  patchOAuthProvider: (next: OAuthProviderSettings) => void
}

export function useSettingsContext() {
  return useOutletContext<SettingsOutletContext>()
}
