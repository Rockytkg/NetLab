import FingerprintJS from '@fingerprintjs/fingerprintjs'

/**
 * 浏览器指纹采集（登录日志用途）。
 *
 * 模块级缓存 visitorId：initFingerprint() 幂等，多次调用只加载一次；
 * 采集失败（隐私模式、广告拦截等）全程静默，不影响登录流程。
 */

let visitorId: string | null = null
let loadPromise: Promise<void> | null = null

/** 初始化浏览器指纹，幂等；失败时静默降级为不携带指纹。 */
export function initFingerprint(): Promise<void> {
  if (loadPromise) return loadPromise
  loadPromise = (async () => {
    try {
      const fp = await FingerprintJS.load()
      const result = await fp.get()
      visitorId = result.visitorId
    } catch {
      // 采集失败时保持 visitorId 为 null，请求头不携带指纹
    }
  })()
  return loadPromise
}

/** 同步返回已缓存的浏览器指纹；未初始化或采集失败时返回 null。 */
export function getFingerprint(): string | null {
  return visitorId
}
