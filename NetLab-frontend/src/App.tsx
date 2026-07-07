import { Suspense, useEffect, useMemo } from 'react'
import { App as AntdApp, ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import enUS from 'antd/locale/en_US'
import { useAppStore } from '@/stores/appStore'
import { useTranslation } from 'react-i18next'
import AppRouter from '@/router'
import Loading from '@/components/common/Loading'
import { createAppTheme } from '@/theme'
import { useResolvedTheme } from '@/hooks/useResolvedTheme'
import { setMessageApi } from '@/utils/message-bridge'

/** antd locale 映射 */
const antdLocales: Record<string, typeof zhCN> = {
  'zh-CN': zhCN,
  'en-US': enUS,
}

/**
 * Registers the App-scoped message instance into the message bridge so
 * non-React modules (e.g. the Axios interceptor) can emit context-aware
 * messages instead of the static `message` API. Must render under <AntdApp>.
 */
function MessageBridge() {
  const { message } = AntdApp.useApp()
  useEffect(() => {
    setMessageApi(message)
  }, [message])
  return null
}

export default function App() {
  const locale = useAppStore((s) => s.locale)
  const themeMode = useAppStore((s) => s.themeMode)
  const resolvedTheme = useResolvedTheme(themeMode)
  const { t } = useTranslation('common')
  const isDark = resolvedTheme === 'dark'
  const theme = useMemo(() => createAppTheme(isDark), [isDark])

  useEffect(() => {
    document.documentElement.dataset.theme = resolvedTheme
    document.documentElement.style.colorScheme = resolvedTheme
  }, [resolvedTheme])

  useEffect(() => {
    document.title = `${t('appName')} - ${t('appSubtitle')}`
  }, [t, locale])

  return (
    <Suspense fallback={<Loading />}>
      <ConfigProvider
        locale={antdLocales[locale] || zhCN}
        theme={theme}
      >
        <AntdApp>
          <MessageBridge />
          <AppRouter />
        </AntdApp>
      </ConfigProvider>
    </Suspense>
  )
}
