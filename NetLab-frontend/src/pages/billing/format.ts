// 计费页面的展示格式化工具。

/** formatBytes 将字节数格式化为人类可读字符串（B/KB/MB/GB/TB）。 */
export function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value.toFixed(value >= 100 || unit === 0 ? 0 : 1)} ${units[unit]}`
}

/** formatDuration 将秒数格式化为人类可读时长（如 1d 2h、3h 25m、12m 5s）。 */
export function formatDuration(seconds: number): string {
  if (!seconds || seconds <= 0) return '0s'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (d > 0) return `${d}d ${h}h`
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

/** formatRate 将 Kbps 速率格式化为展示文本；0 表示不限。 */
export function formatRate(kbps: number, unlimited: string): string {
  if (!kbps || kbps <= 0) return unlimited
  if (kbps >= 1000) return `${(kbps / 1000).toFixed(kbps % 1000 === 0 ? 0 : 1)} Mbps`
  return `${kbps} Kbps`
}
