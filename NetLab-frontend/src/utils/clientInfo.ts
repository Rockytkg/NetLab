import { UAParser } from 'ua-parser-js'

/**
 * 客户端操作系统/浏览器识别（登录日志用途）。
 *
 * 模块级缓存识别结果：initClientInfo() 幂等，多次调用只解析一次；
 * 通过 UA Client Hints 获取高熵值（可准确识别 Windows 11、CPU 架构），
 * 非 Chromium 浏览器自动回退为纯 UA 解析；失败全程静默，不影响登录流程。
 */

export interface ClientInfo {
  os: string
  browser: string
}

let clientInfo: ClientInfo | null = null
let loadPromise: Promise<void> | null = null

/** 初始化客户端信息识别，幂等；失败时静默降级为不携带客户端信息。 */
export function initClientInfo(): Promise<void> {
  if (loadPromise) return loadPromise
  loadPromise = (async () => {
    try {
      const parser = new UAParser()
      const [os, cpu, browser] = await Promise.all([
        parser.getOS().withClientHints(),
        parser.getCPU().withClientHints(),
        parser.getBrowser().withClientHints(),
      ])
      clientInfo = {
        os: formatOS(os.name, os.version, cpu.architecture),
        browser: formatBrowser(browser.name, browser.major),
      }
    } catch {
      // 识别失败时保持 clientInfo 为 null，请求头不携带客户端信息
    }
  })()
  return loadPromise
}

/** 同步返回已缓存的客户端信息；未初始化或识别失败时返回 null。 */
export function getClientInfo(): ClientInfo | null {
  return clientInfo
}

/** 操作系统："Windows 11 (amd64)"；无架构时省略括号，空值兜底为空串片段。 */
function formatOS(name?: string, version?: string, architecture?: string): string {
  const base = [name ?? '', version ?? ''].filter(Boolean).join(' ')
  return architecture ? `${base} (${architecture})` : base
}

/** 浏览器："Chrome 126"（名称 + 主版本号）。 */
function formatBrowser(name?: string, major?: string): string {
  return [name ?? '', major ?? ''].filter(Boolean).join(' ')
}
