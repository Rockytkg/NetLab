import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { SupportedLocale } from '@/types/i18n'
import { DEFAULT_LOCALE } from '@/types/i18n'

export type ThemeMode = 'light' | 'dark' | 'system'

interface AppState {
  /** 侧边栏折叠状态 */
  sidebarCollapsed: boolean
  /** 当前语言 */
  locale: SupportedLocale
  /** 主题模式 */
  themeMode: ThemeMode

  /** 切换侧边栏 */
  toggleSidebar: () => void
  /** 设置语言 */
  setLocale: (locale: SupportedLocale) => void
  /** 设置主题模式 */
  setThemeMode: (mode: ThemeMode) => void
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      locale: DEFAULT_LOCALE,
      themeMode: 'system',

      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      setLocale: (locale: SupportedLocale) => set({ locale }),

      setThemeMode: (themeMode: ThemeMode) => set({ themeMode }),
    }),
    {
      name: 'netlab-app',
      partialize: (state) => ({
        locale: state.locale,
        sidebarCollapsed: state.sidebarCollapsed,
        themeMode: state.themeMode,
      }),
    }
  )
)
