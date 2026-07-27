/**
 * 方案开始/结束时间解析与校验。
 * 墙钟一律按北京时间（Asia/Shanghai / UTC+8）解释，与后端 timeutil.PlatformLocation 一致。
 */

const BEIJING_TZ = 'Asia/Shanghai'
const BEIJING_OFFSET = '+08:00'

function normalizeRaw(raw: string): string {
  return raw.trim().replace(/：/g, ':')
}

function pad2(n: number): string {
  return String(n).padStart(2, '0')
}

/** 当前北京时间的年月日 */
function beijingYmd(nowMs = Date.now()): { y: number; mo: number; d: number } {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: BEIJING_TZ,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date(nowMs))
  const get = (type: string) => Number(parts.find((p) => p.type === type)?.value ?? NaN)
  return { y: get('year'), mo: get('month'), d: get('day') }
}

/** 将北京墙钟日期时间转为绝对毫秒时间戳 */
function beijingWallToMs(y: number, mo: number, d: number, h: number, mi: number, s: number): number | null {
  const iso = `${y}-${pad2(mo)}-${pad2(d)}T${pad2(h)}:${pad2(mi)}:${pad2(s)}${BEIJING_OFFSET}`
  const ms = Date.parse(iso)
  return Number.isNaN(ms) ? null : ms
}

/** 无效占位日期（如 0000-00-00 00:00:00） */
export function isInvalidSchemeDateTime(raw: string): boolean {
  const t = normalizeRaw(raw)
  if (!t) return true
  if (/0000[-/]0{2}[-/]0{2}/.test(t) || /^0{4}-0{2}-0{2}/.test(t)) return true
  return parseSchemeDateTimeMs(t) == null
}

/** 解析为毫秒时间戳（北京墙钟）；仅时刻（HH:mm）时取「今天（北京）」 */
export function parseSchemeDateTimeMs(raw: string, nowMs = Date.now()): number | null {
  const t = normalizeRaw(raw)
  if (!t) return null

  const full = t.match(/^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?$/)
  if (full) {
    const y = Number(full[1])
    const mo = Number(full[2])
    const d = Number(full[3])
    if (y <= 0 || mo <= 0 || d <= 0) return null
    return beijingWallToMs(
      y,
      mo,
      d,
      Number(full[4] ?? 0),
      Number(full[5] ?? 0),
      Number(full[6] ?? 0),
    )
  }

  const hm = t.match(/^(\d{1,2}):(\d{2})(?::(\d{2}))?$/)
  if (hm) {
    const h = Number(hm[1])
    const mi = Number(hm[2])
    const s = Number(hm[3] ?? 0)
    if (h > 23 || mi > 59 || s > 59) return null
    const { y, mo, d } = beijingYmd(nowMs)
    return beijingWallToMs(y, mo, d, h, mi, s)
  }

  return null
}

/** 开始时间是否严格早于结束时间 */
export function isSchemeStartBeforeEnd(start: string, end: string): boolean {
  const a = parseSchemeDateTimeMs(start)
  const b = parseSchemeDateTimeMs(end)
  if (a == null || b == null) return false
  return a < b
}

/**
 * 保存前校验时间范围；通过返回 null，否则返回提示文案。
 */
/** 开启方案：开始时间须严格晚于当前时刻（北京时间；提前开启后由 worker 在开始时间到达后下注） */
export function isSchemeStartAfterNow(start: string, nowMs = Date.now()): boolean {
  const ms = parseSchemeDateTimeMs(start, nowMs)
  if (ms == null) return false
  return ms > nowMs
}

export function schemeStartTimeOpenError(start: string): string | null {
  if (!start.trim()) return null
  if (isInvalidSchemeDateTime(start)) return '开始时间无效，请重新选择日期与时间'
  if (!isSchemeStartAfterNow(start)) return '预计开启时间小于现在时间 请修改后再执行开启'
  return null
}

/** 从 config 读取并规范化方案时间；空、无效占位、历史误存整日占位视为未配置 */
export function normalizeSchemeTimeFromConfig(raw: unknown): string {
  const t = normalizeRaw(String(raw ?? ''))
  if (!t) return ''
  if (isInvalidSchemeDateTime(t)) return ''
  return t
}

/** 成对规范化；历史误存 start=00:00 & end=23:59 视为无限期 */
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
  if (isInvalidSchemeDateTime(start)) return '开始时间无效，请重新选择日期与时间'
  if (isInvalidSchemeDateTime(end)) return '结束时间无效，请重新选择日期与时间'
  if (!isSchemeStartBeforeEnd(start, end)) return '开始时间必须早于结束时间'
  return null
}
