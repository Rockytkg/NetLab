import { Tag } from 'antd'
import dayjs from 'dayjs'
import type { TFunction } from 'i18next'
import type { KeyboardEvent, MouseEvent } from 'react'

const INTERACTIVE_SELECTOR = 'button, a, input, textarea, select, [role="button"], [role="checkbox"], [contenteditable="true"]'

function isInteractiveTarget(target: EventTarget | null) {
  return target instanceof Element && Boolean(target.closest(INTERACTIVE_SELECTOR))
}

/** Open details from a table row while leaving embedded controls untouched. */
export function billingDetailRow<T>(onOpen: (record: T) => void) {
  return (record: T) => ({
    onClick: (event: MouseEvent<HTMLElement>) => {
      if (isInteractiveTarget(event.target)) return
      onOpen(record)
    },
    onKeyDown: (event: KeyboardEvent<HTMLElement>) => {
      if (isInteractiveTarget(event.target) || !['Enter', ' '].includes(event.key)) return
      event.preventDefault()
      onOpen(record)
    },
    tabIndex: 0,
    style: { cursor: 'pointer' },
  })
}

/** 时间列渲染：YYYY-MM-DD HH:mm:ss。 */
export function renderTime(val?: string | null) {
  return val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-'
}

/** 启用/停用状态 Tag 渲染；统一走 radius:common.{enabled,disabled} i18n。 */
export function renderStatusTag(t: TFunction<'radius'>, val: string) {
  const key = val === 'enabled' ? 'common.enabled' : 'common.disabled'
  return <Tag color={val === 'enabled' ? 'success' : 'error'}>{t(key)}</Tag>
}
