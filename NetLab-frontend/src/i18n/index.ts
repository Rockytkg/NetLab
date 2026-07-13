import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import dayjs from 'dayjs'
// dayjs 语言包数据 —— 驱动 DatePicker/RangePicker 日历内部逻辑
// （月份名称、星期表头、“今天”按钮）。Ant Design v6 通过 dayjs
// 渲染日期，因此当前激活的 dayjs 语言必须与应用语言保持同步，
// 且独立于 ConfigProvider 自身的语言包。
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/en'
import { languageDetector } from '@/types/i18n'
import type { SupportedLocale } from '@/types/i18n'
import { DEFAULT_LOCALE } from '@/types/i18n'
import { setI18nT } from '@/utils/i18n-bridge'

// 中文翻译资源
import zhCommon from './locales/zh-CN/common.json'
import zhLogin from './locales/zh-CN/login.json'
import zhMenu from './locales/zh-CN/menu.json'
import zhLab from './locales/zh-CN/lab.json'
import zhTopology from './locales/zh-CN/topology.json'
import zhSettings from './locales/zh-CN/settings.json'

// 英文翻译资源
import enCommon from './locales/en-US/common.json'
import enLogin from './locales/en-US/login.json'
import enMenu from './locales/en-US/menu.json'
import enLab from './locales/en-US/lab.json'
import enTopology from './locales/en-US/topology.json'
import enSettings from './locales/en-US/settings.json'

/** 翻译资源 */
const resources = {
  'zh-CN': {
    common: zhCommon,
    login: zhLogin,
    menu: zhMenu,
    lab: zhLab,
    topology: zhTopology,
    settings: zhSettings,
  },
  'en-US': {
    common: enCommon,
    login: enLogin,
    menu: enMenu,
    lab: enLab,
    topology: enTopology,
    settings: enSettings,
  },
} as const

/**
 * 初始化 i18n
 */
i18n
  .use(languageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: DEFAULT_LOCALE,
    defaultNS: 'common',
    interpolation: {
      escapeValue: false, // React 已处理 XSS
    },
  })

// 将应用语言 → dayjs 语言标识符映射
const DAYJS_LOCALE_MAP: Record<SupportedLocale, string> = {
  'zh-CN': 'zh-cn',
  'en-US': 'en',
}

/** 将全局 dayjs 语言与给定的应用语言同步。 */
function syncDayjsLocale(locale: SupportedLocale): void {
  dayjs.locale(DAYJS_LOCALE_MAP[locale] ?? 'en')
}

// 将 t 函数接入请求拦截器的 i18n 桥接层
setI18nT(i18n.t.bind(i18n))

// 应用初始的 dayjs 语言（i18next 已解析出检测到的语言）
syncDayjsLocale(getCurrentLanguage())

/** 切换语言 */
export function changeLanguage(locale: SupportedLocale): void {
  i18n.changeLanguage(locale)
  syncDayjsLocale(locale)
}

/** 获取当前语言 */
export function getCurrentLanguage(): SupportedLocale {
  return (i18n.language as SupportedLocale) || DEFAULT_LOCALE
}

export default i18n
