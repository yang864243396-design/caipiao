function normalizeRaw(raw: string): string {
  return raw.trim().replace(/：/g, ':')
}

export function isInvalidSchemeDateTime(raw: string): boolean {
  const t = normalizeRaw(raw)
  if (!t) return true
  if (/0000[-/]0{2}[-/]0{2}/.test(t) || /^0{4}-0{2}-0{2}/.test(t)) return true
  return parseSchemeDateTimeMs(t) == null
}

export function parseSchemeDateTimeMs(raw: string): number | null {
  const t = normalizeRaw(raw)
  if (!t) return null

  const full = t.match(/^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?$/)
  if (full) {
    const y = Number(full[1])
    const mo = Number(full[2])
    const d = Number(full[3])
    if (y <= 0 || mo <= 0 || d <= 0) return null
    const dt = new Date(
      y,
      mo - 1,
      d,
      Number(full[4] ?? 0),
      Number(full[5] ?? 0),
      Number(full[6] ?? 0),
    )
    return Number.isNaN(dt.getTime()) ? null : dt.getTime()
  }

  const hm = t.match(/^(\d{1,2}):(\d{2})(?::(\d{2}))?$/)
  if (hm) {
    const h = Number(hm[1])
    const mi = Number(hm[2])
    const s = Number(hm[3] ?? 0)
    if (h > 23 || mi > 59 || s > 59) return null
    return new Date(2000, 0, 1, h, mi, s).getTime()
  }

  return null
}

export function isSchemeStartBeforeEnd(start: string, end: string): boolean {
  const a = parseSchemeDateTimeMs(start)
  const b = parseSchemeDateTimeMs(end)
  if (a == null || b == null) return false
  return a < b
}

export function normalizeSchemeTimeFromConfig(raw: unknown): string {
  const t = normalizeRaw(String(raw ?? ''))
  if (!t) return ''
  if (isInvalidSchemeDateTime(t)) return ''
  return t
}

export function normalizeSchemeTimePairFromConfig(
  startRaw: unknown,
  endRaw: unknown,
): { start: string; end: string } {
  let start = normalizeSchemeTimeFromConfig(startRaw)
  let end = normalizeSchemeTimeFromConfig(endRaw)
  if (start === '00:00' && end === '23:59') {
    return { start: '', end: '' }
  }
  if (start === '00:00' && !end) start = ''
  if (end === '23:59' && !start) end = ''
  return { start, end }
}

export function schemeTimeRangeError(start: string, end: string): string | null {
  const s = start.trim()
  const e = end.trim()
  if (!s && !e) return null
  if (!s || !e) {
    return '开始时间与结束时间须同时填写，或同时留空（无限期运行）'
  }
  if (isInvalidSchemeDateTime(start)) return '开始时间无效，请重新选择'
  if (isInvalidSchemeDateTime(end)) return '结束时间无效，请重新选择'
  if (!isSchemeStartBeforeEnd(start, end)) return '开始时间必须早于结束时间'
  return null
}

const DATETIME_RE = /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?$/
const HM_RE = /^(\d{1,2}):(\d{2})(?::(\d{2}))?$/

/** 规范为 el-date-picker value-format：YYYY-MM-DD HH:mm:ss */
export function toDatePickerValue(raw: string): string {
  const t = normalizeRaw(raw)
  if (!t) return ''
  const full = t.match(DATETIME_RE)
  if (full) {
    const h = (full[4] ?? '00').padStart(2, '0')
    const m = (full[5] ?? '00').padStart(2, '0')
    const s = (full[6] ?? '00').padStart(2, '0')
    return `${full[1]}-${full[2]}-${full[3]} ${h}:${m}:${s}`
  }
  const hm = t.match(HM_RE)
  if (hm) {
    const h = String(Number(hm[1])).padStart(2, '0')
    const s = (hm[3] ?? '00').padStart(2, '0')
    return `2000-01-01 ${h}:${hm[2]}:${s}`
  }
  return ''
}

/** 读取 date-picker 值，保证秒位补齐 */
export function fromDatePickerValue(value: string | null | undefined): string {
  if (value == null || value === '') return ''
  return toDatePickerValue(String(value))
}
