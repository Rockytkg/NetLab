import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useAppStore } from '@/stores/appStore'
import { changeLanguage } from '@/i18n'
import type { SupportedLocale } from '@/types/i18n'

export function useI18n() {
  const { t, i18n } = useTranslation()
  const setLocale = useAppStore((s) => s.setLocale)

  const switchLanguage = useCallback(
    (locale: SupportedLocale) => {
      changeLanguage(locale)
      setLocale(locale)
    },
    [setLocale]
  )

  return {
    t,
    currentLanguage: i18n.language as SupportedLocale,
    switchLanguage,
  }
}
