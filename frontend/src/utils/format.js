/**
 * 把后端返回的 ISO 8601 时间戳格式化成 "YYYY-MM-DD HH:mm:ss"。
 *
 * 后端 Go 默认输出："2026-06-29T13:52:33.451856782+08:00"
 * 数据库无时区时也可能输出："2026-06-29T13:52:33+08:00"
 *
 * 入参为 null / undefined / 空串 → 返回 ''（保持表格对齐）
 * 解析失败 → 原样返回，避免吞掉调试信息
 */
export function formatDateTime(input) {
  if (!input) return ''
  const d = new Date(input)
  if (Number.isNaN(d.getTime())) return String(input)
  const pad = (n) => String(n).padStart(2, '0')
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ` +
    `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  )
}