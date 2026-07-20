import { Tag } from 'antd'
import dayjs from 'dayjs'
import type { TFunction } from 'i18next'

/** 时间列渲染：YYYY-MM-DD HH:mm:ss。 */
export function renderTime(val?: string | null) {
  return val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'
}

/** 启用/停用状态 Tag 渲染；统一走 radius:common.{enabled,disabled} i18n。 */
export function renderStatusTag(t: TFunction<'radius'>, val: string) {
  const key = val === 'enabled' ? 'common.enabled' : 'common.disabled'
  return <Tag color={val === 'enabled' ? 'success' : 'error'}>{t(key)}</Tag>
}
