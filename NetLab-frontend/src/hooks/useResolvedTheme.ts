import { useEffect, useState } from 'react'
import type { ThemeMode } from '@/stores/appStore'

export type ResolvedTheme = 'light' | 'dark'

const colorSchemeQuery = '(prefers-color-scheme: dark)'

function getSystemTheme(): ResolvedTheme {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia(colorSchemeQuery).matches ? 'dark' : 'light'
}

export function useResolvedTheme(themeMode: ThemeMode): ResolvedTheme {
  const [systemTheme, setSystemTheme] = useState<ResolvedTheme>(getSystemTheme)

  useEffect(() => {
    const media = window.matchMedia(colorSchemeQuery)
    setSystemTheme(media.matches ? 'dark' : 'light')

    if (themeMode !== 'system') return

    const handleChange = (event: MediaQueryListEvent) => {
      setSystemTheme(event.matches ? 'dark' : 'light')
    }

    media.addEventListener('change', handleChange)
    return () => media.removeEventListener('change', handleChange)
  }, [themeMode])

  return themeMode === 'system' ? systemTheme : themeMode
}
