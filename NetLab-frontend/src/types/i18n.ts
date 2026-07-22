import type { LanguageDetectorModule } from 'i18next'

/** 支持的语言 */
export type SupportedLocale = 'zh-CN' | 'en-US'

/** 语言选项 */
export interface LocaleOption {
  value: SupportedLocale
  label: string
}

/** i18n 命名空间 */
export type I18nNamespace = 'common' | 'login' | 'menu' | 'operations' | 'portal' | 'radius' | 'settings'

const APP_STORE_KEY = 'netlab-app'

function isSupportedLocale(value: unknown): value is SupportedLocale {
  return value === 'zh-CN' || value === 'en-US'
}

function getPersistedAppLocale(): SupportedLocale | null {
  try {
    const stored = localStorage.getItem(APP_STORE_KEY)
    if (!stored) return null

    const parsed = JSON.parse(stored) as {
      state?: {
        locale?: unknown
      }
    }

    return isSupportedLocale(parsed.state?.locale) ? parsed.state.locale : null
  } catch {
    return null
  }
}

/**
 * i18next 自定义检测器。
 * 优先级: Zustand app store > navigator > fallback
 */
export const languageDetector: LanguageDetectorModule = {
  type: 'languageDetector',
  detect(): string {
    const stored = getPersistedAppLocale()
    if (stored) return stored

    const nav = navigator.language
    if (nav.startsWith('zh')) return 'zh-CN'
    if (nav.startsWith('en')) return 'en-US'

    return 'zh-CN'
  },
  cacheUserLanguage(): void {
    // 语言持久化由 useAppStore 负责（localStorage 键: netlab-app）。
  },
}

/** 语言选项列表 */
export const LOCALE_OPTIONS: LocaleOption[] = [
  { value: 'zh-CN', label: '简体中文' }, // i18n-allow: locale name
  { value: 'en-US', label: 'English' },
]

/** 默认语言 */
export const DEFAULT_LOCALE: SupportedLocale = 'zh-CN'
