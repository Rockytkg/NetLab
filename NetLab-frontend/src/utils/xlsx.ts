import * as XLSX from 'xlsx'

/**
 * 浏览器端 xlsx 导出工具。
 *
 * 与 services/admin.ts 中的实现同源；新代码统一走这里，
 * admin.ts 保持原样以免影响既有用户导入导出功能。
 */

/** 触发浏览器将 Blob 保存为本地文件。 */
export function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** 将二维数据（首行为表头）生成 xlsx 并触发浏览器下载。 */
export function saveWorkbook(rows: unknown[][], headers: string[], filename: string, widths: number[]) {
  const worksheet = XLSX.utils.aoa_to_sheet([headers, ...rows])
  worksheet['!cols'] = widths.map((wch) => ({ wch }))
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, worksheet, 'Sheet1')
  const bytes = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' })
  saveBlob(new Blob([bytes], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }), filename)
}
