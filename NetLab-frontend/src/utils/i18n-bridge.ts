/**
 * i18n Bridge —— 将请求拦截器与 i18n 模块解耦，
 * 以避免循环依赖（request.ts → i18n/index.ts → … → request.ts）。
 *
 * 用法:
 *   // 在 i18n/index.ts 中（i18n.init 之后）:
 *   import { setI18nT } from '@/utils/i18n-bridge'
 *   setI18nT(i18n.t.bind(i18n))
 *
 *   // 在 request.ts 中:
 *   import { getI18nT } from '@/utils/i18n-bridge'
 *   const t = getI18nT()
 */

type TFunc = (key: string, options?: Record<string, unknown>) => string

let _t: TFunc = (key: string) => key

/** 设置 i18n 的 t 函数。在 i18n 初始化后调用一次。 */
export function setI18nT(t: TFunc): void {
  _t = t
}

/** 获取 i18n 的 t 函数。在初始化前调用也安全（返回透传函数）。 */
export function getI18nT(): TFunc {
  return _t
}
