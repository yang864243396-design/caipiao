import type { PlayConfig } from '@/utils/betPayload'
import {
  isYixingDingweiPlayConfig,
  YIXING_MAX_PICKS_PER_POS,
} from '@/utils/betPayload'
import {
  isLonghuPlayConfigLike,
  isPc28ModeConfigLike,
} from '@/utils/runTypeMatrix'
import {
  longhuPickOptionsForConfig,
} from '@/utils/longhuPickOptions'

/** 龙虎类玩法（龙/虎 或 龙/虎/和 文字选号，非 0-9） */
export function isLonghuTextPickConfig(config: PlayConfig): boolean {
  return isLonghuPlayConfigLike(config)
}

/** 方案内容是否用 chip/选号面板（与 SchemeGroupPickPanel 同源） */
export function schemeGroupUsesPickPanel(config: PlayConfig): boolean {
  const mode = config.inputMode
  if (textPickOptionsForConfig(config).length > 0) return true
  if (isLonghuTextPickConfig(config)) return true
  if (mode === 'danshi') {
    return (config.numberPoolMax ?? 9) > 9
  }
  return ['lhc_num', 'lhc_zodiac', 'lhc_tail', 'lhc_attr', 'pool', 'dingwei', 'multiline'].includes(mode)
}

/** 和值号池（直选/组选）：变长数字，禁止补零连写（组选和值 1–26 勿变成 01–26） */
function isHezhiPoolConfig(config: PlayConfig): boolean {
  if (config.betMode === 'hezhi') return true
  // PC28 顶线和值：betMode 可能为空，仅按文案识别「和值」本身
  return config.playTemplate === 'pc28_std' && (config.playMethodLabel ?? '').trim() === '和值'
}

/** 和值尾数号池（前三和值尾数等）：0–9，须逗号分隔（勿连写） */
function isWeishuPoolConfig(config: PlayConfig): boolean {
  if (config.betMode === 'weishu') return true
  const label = config.playMethodLabel ?? ''
  return /和值尾数/.test(label) || (label.includes('尾数') && !/单双|大小|对碰|不中|生肖/.test(label))
}

/** 投注/方案面板：按玩法号池生成可选号码 */
export function digitOptionsForConfig(config: PlayConfig): string[] {
  const min = config.numberPoolMin ?? 0
  const max = config.numberPoolMax ?? 9
  // 11选5/PK10 等从 1 起且 max≥11 时补零；和值（含组选和值 1–26）保持自然数展示，须逗号分隔
  const pad = max >= 11 && min >= 1 && !isHezhiPoolConfig(config)
  const out: string[] = []
  for (let i = min; i <= max; i++) {
    out.push(pad ? String(i).padStart(2, '0') : String(i))
  }
  return out.length ? out : ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9']
}

function inferTextPickFromLabels(config: PlayConfig): string[] {
  const subLabel = config.playMethodLabel?.trim() ?? ''
  if (subLabel === '大小单双' || subLabel.includes('大小单双')) return ['大', '小', '单', '双']
  if (subLabel === '龙虎豹') return ['龙', '虎', '豹']
  if (subLabel === '特殊号' || subLabel.includes('特殊号')) {
    if (config.playTemplate === 'pc28_std') return ['豹子', '对子', '顺子', '极大', '极小']
    return ['豹子', '对子', '顺子']
  }
  if (subLabel.includes('幸运庄闲') || subLabel.includes('庄闲')) return ['庄', '闲']
  if (subLabel === '和值' && isPc28ModeConfigLike(config)) return []
  return []
}

/** 龙虎 / 大小单双 / PC28 特殊号等文字选项 */
export function textPickOptionsForConfig(config: PlayConfig): string[] {
  if (isLonghuTextPickConfig(config)) {
    return longhuPickOptionsForConfig(config)
  }
  const bm = config.betMode ?? ''
  switch (bm) {
    case 'longhu':
    case 'longhuhe':
      return longhuPickOptionsForConfig(config)
    case 'daxiao':
      return ['大', '小']
    case 'danshuang':
      return ['单', '双']
    case 'dxds':
      return ['大', '小', '单', '双']
    case 'zhuangxian':
      return ['庄', '闲']
    case 'teshu':
      return config.playTemplate === 'pc28_std'
        ? ['豹子', '对子', '顺子', '极大', '极小']
        : ['豹子', '对子', '顺子']
    case 'longhubao':
      return ['龙', '虎', '豹']
    default:
      return inferTextPickFromLabels(config)
  }
}

export function useCompactPickChips(config: PlayConfig): boolean {
  return (
    config.inputMode === 'lhc_num' ||
    config.inputMode === 'lhc_zodiac' ||
    config.inputMode === 'lhc_tail' ||
    config.inputMode === 'lhc_attr' ||
    (config.numberPoolMax ?? 0) >= 11
  )
}

/**
 * 是否用「数字输入框」录入方案内容（对齐第三方：直接键入号码、逗号分隔，不点选）。
 *
 * 适用：定位胆 / 号池 / 直选复式等按数字选号的玩法（0-9、1-10、01-11 等）。
 * 排除：大小单双/龙虎/庄闲/特殊号（有限文字选项，保留点选）、六合彩生肖/尾数/号码/属性、
 * 以及单式（整注按 N 位数字录入，另有面板）。
 */
export function schemeGroupUsesDigitInput(config: PlayConfig): boolean {
  if (!schemeGroupUsesPickPanel(config)) return false
  if (config.inputMode === 'danshi') return false
  if (
    config.inputMode === 'lhc_num' ||
    config.inputMode === 'lhc_zodiac' ||
    config.inputMode === 'lhc_tail' ||
    config.inputMode === 'lhc_attr'
  ) {
    return false
  }
  if (textPickOptionsForConfig(config).length > 0) return false
  return true
}

/**
 * 变长号池（和值 0–27 / 组选和值 1–26 / 快三 3–18 等）必须逗号分隔录入。
 * 定宽补零号池（11选5 01–11、PK10）仍可连写按宽度切块。
 */
export function poolUsesCommaSeparatedInput(config: PlayConfig): boolean {
  if (isHezhiPoolConfig(config)) return true
  // 和值尾数虽为 0–9，连写会把多选拆错，与直选和值一致用逗号分隔
  if (isWeishuPoolConfig(config)) return true
  // 组三/组六：第三方提示为逗号多选（如 0,1,2,3…），勿连写/按位
  if (isZu3PoolPlay(config) || isZu6PoolPlay(config)) return true
  const options = digitOptionsForConfig(config)
  if (options.length < 2) return false
  const widths = new Set(options.map((o) => o.length))
  if (widths.size > 1) return true
  const w = options[0]?.length ?? 1
  const max = config.numberPoolMax ?? 9
  // 单位数展示但上限 >9：连写会把 27 拆成 2,7
  return w === 1 && max > 9
}

/** 按逗号/空白解析号池 token（保留号池展示形态，如 07 / 27） */
function parseCommaSeparatedPoolTokens(raw: string, options: string[]): string[] {
  const parts = String(raw ?? '')
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  const seen = new Set<string>()
  const out: string[] = []
  for (const p of parts) {
    if (!/^\d+$/.test(p)) continue
    const n = Number(p)
    const match = options.find((o) => Number(o) === n)
    if (!match || seen.has(match)) continue
    seen.add(match)
    out.push(match)
  }
  return out
}

/** 解析单位内号码为号池合法 token（定宽连写切块 / 变长逗号分隔） */
function parseDigitSegmentTokens(seg: string, config: PlayConfig): string[] {
  const options = digitOptionsForConfig(config)
  if (!options.length) return []
  if (poolUsesCommaSeparatedInput(config)) {
    return parseCommaSeparatedPoolTokens(seg, options)
  }
  const w = options[0]?.length || 1
  const digits = String(seg ?? '').replace(/\D/g, '')
  const seen = new Set<string>()
  const out: string[] = []
  for (let i = 0; i + w <= digits.length; i += w) {
    const chunk = digits.slice(i, i + w)
    const n = Number(chunk)
    const match = options.find((o) => Number(o) === n)
    if (!match || seen.has(match)) continue
    seen.add(match)
    out.push(match)
  }
  return out
}

/**
 * 录入框（逗号分位压缩格式）→ 引擎内容（单位型单行、多位型按位换行）。
 * 与 SchemeGroupInputPanel / 定码轮换落库一致。
 */
export function schemeGroupInputBoxToContent(box: string, config: PlayConfig): string {
  const segLen = Math.max(1, config.segmentLen || 1)
  const cap = poolMaxPicksForConfig(config)
  // 号池型（组三/组六/和值等）：单行逗号多选，勿按 segmentLen 拆成按位
  if (segLen <= 1 || config.inputMode === 'pool' || isZu3PoolPlay(config) || isZu6PoolPlay(config)) {
    let toks = parseDigitSegmentTokens(box, config)
    if (cap != null && cap > 0) toks = toks.slice(0, cap)
    return toks.join(',')
  }
  const segs = String(box ?? '').split(/[,，]/)
  const lines: string[] = []
  let any = false
  for (let i = 0; i < segLen; i++) {
    let toks = parseDigitSegmentTokens(segs[i] ?? '', config)
    if (cap != null && cap > 0 && toks.length > cap) toks = toks.slice(0, cap)
    if (toks.length) any = true
    lines.push(toks.join(','))
  }
  return any ? lines.join('\n') : ''
}

/**
 * 引擎存储内容 → 数字录入框压缩格式（与 SchemeGroupInputPanel 一致）。
 * 多位型：每位号码连写、逗号分隔各位，如 `1,2\n3,4` → `12,34`；
 * 单位型：号码连写，如 `1,2` → `12`。
 */
export function schemeGroupContentToInputBox(content: string, config: PlayConfig): string {
  const c = String(content ?? '').replace(/\r/g, '')
  // 无有效号码时保持空串，避免 '' → ',,,,' 盖住 placeholder
  if (c.replace(/[\s,，]/g, '') === '') return ''
  const segLen = Math.max(1, config.segmentLen || 1)
  if (segLen <= 1 || config.inputMode === 'pool' || isZu3PoolPlay(config) || isZu6PoolPlay(config)) {
    const toks = c
      .split(/[,，\s\n]+/)
      .map((t) => t.trim())
      .filter(Boolean)
    // 和值/组三等号池：显示时保留逗号，避免 0,1,2 → 012 再被当三位按位
    return poolUsesCommaSeparatedInput(config) || isZu3PoolPlay(config) || isZu6PoolPlay(config)
      ? toks.join(',')
      : toks.join('')
  }
  // 已是录入框形态（无换行、逗号分位）：段数 ≤ 位宽时按位补齐（「1,2,3」→「1,2,3,,」）
  if (!c.includes('\n')) {
    const parts = c.split(/[,，]/)
    const hasAny = parts.some((p) => /[0-9A-Za-z]/.test(p))
    if (hasAny && parts.length > 0 && parts.length <= segLen) {
      return Array.from({ length: segLen }, (_, i) =>
        (parts[i] ?? '').replace(/[^0-9A-Za-z]/g, ''),
      ).join(',')
    }
  }
  const lines = c.split('\n')
  const segs: string[] = []
  let any = false
  for (let i = 0; i < segLen; i++) {
    const toks = (lines[i] ?? '')
      .split(/[,，\s]+/)
      .map((t) => t.trim())
      .filter(Boolean)
    if (toks.length) any = true
    segs.push(toks.join(''))
  }
  return any ? segs.join(',') : ''
}

/** 数字玩法内容规范为引擎存储（逗号分位存量 → 按位换行，与定码轮换一致） */
export function normalizeSchemeGroupDigitContent(content: string, config: PlayConfig): string {
  return schemeGroupInputBoxToContent(schemeGroupContentToInputBox(content, config), config)
}

/** 方案内容是否有有效号码（勿对绝对位用 trim，避免弄丢前导空行） */
export function schemeGroupContentHasDigits(content: string): boolean {
  return String(content ?? '').replace(/[\s,，]/g, '') !== ''
}

/** 构造数字录入示例：定宽连写 / 变长逗号分隔；多位型再按位用逗号分开 */
function buildDigitInputExample(options: string[], segLen: number, commaPool: boolean): string {
  if (!options.length) return ''
  if (commaPool) {
    const mid = Math.floor(options.length / 2)
    const sample = [options[mid]!, options[Math.min(mid + 1, options.length - 1)]!, options[Math.min(mid + 2, options.length - 1)]!]
    return [...new Set(sample)].join(',')
  }
  const segs: string[] = []
  for (let i = 0; i < segLen; i++) {
    const count = (i % 4) + 2
    const toks: string[] = []
    for (let k = 0; k < count; k++) toks.push(options[(i + k) % options.length]!)
    segs.push(toks.join(''))
  }
  return segs.join(',')
}

/** 组三 / 组六号池玩法（前三组三、前三组六等） */
function isZu3PoolPlay(config: PlayConfig): boolean {
  if (config.betMode === 'zu3') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组三|zu3/i.test(text) && !/组选3|组选30|zu30/i.test(text)
}

function isZu6PoolPlay(config: PlayConfig): boolean {
  if (config.betMode === 'zu6') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组六|zu6/i.test(text) && !/组选6|组选60|组选120|zu60|zu120/i.test(text)
}

/** 组选包胆（前三组选包胆等）：单选 0–9 */
function isBaodanPoolPlay(config: PlayConfig): boolean {
  if (config.betMode === 'baodan') return true
  return /包胆/.test(config.playMethodLabel ?? '')
}

/**
 * 数字玩法方案内容录入提示（按玩法动态生成）：多位型逐位对应位名、逗号分隔、每位皆须录入；
 * 单位定宽可连写；和值等变长号池须逗号分隔。
 */
export function groupDigitInputHint(config: PlayConfig): string {
  if (isZu3PoolPlay(config)) {
    return '输入两个及以上 0-9 的号码，多选用逗号分隔，如 1,3,5,7'
  }
  if (isZu6PoolPlay(config)) {
    return '输入三个及以上 0-9 的号码，多选用逗号分隔，如 1,3,5,7'
  }
  // 直选/组选和值：与 groupContentPlaceholder 一致，逗号分隔；组选池为 1–26
  if (isHezhiPoolConfig(config)) {
    const min = config.numberPoolMin ?? 0
    const max = config.numberPoolMax ?? 27
    return `和值：输入 ${min}–${max}，多选用逗号分隔（如 14,15,16）`
  }
  // 和值尾数：对齐直选和值提示/分隔方式，号池 0–9
  if (isWeishuPoolConfig(config)) {
    const min = config.numberPoolMin ?? 0
    const max = config.numberPoolMax ?? 9
    return `和值尾数：输入 ${min}–${max}，多选用逗号分隔（如 1,3,5）`
  }
  // 组选包胆：仅单选一个胆码
  if (isBaodanPoolPlay(config)) {
    return '包胆：输入一个 0–9 的号码（如 5）'
  }
  const options = digitOptionsForConfig(config)
  if (!options.length) return ''
  const range = `${options[0]}-${options[options.length - 1]}`
  const segLen = Math.max(1, config.segmentLen || 1)
  const commaPool = poolUsesCommaSeparatedInput(config)
  const example = buildDigitInputExample(options, segLen, commaPool)
  if (segLen <= 1) {
    if (commaPool) {
      return `输入 ${range} 的号码，多选用逗号分隔，如：${example}`
    }
    return `直接连写号码（可选 ${range}），如：${example}`
  }
  // 跨段组合（前中后三 / 前后二 / 前后三 / 前后四）位置非连续，用「N 个顺序号码」；
  // 前三/中三/后三/前二/后二/四星/五星等连续固定位仍按「首位到末位」显示位置。
  if (usesSequentialGroupHint(config)) {
    const cnCount = ['零', '一', '二', '三', '四', '五', '六', '七', '八', '九', '十'][segLen] ?? String(segLen)
    return `请对应${cnCount}个顺序号码，以“，”分隔，输入对应位置的号码，每一位置皆要输入号码；如：${example}`
  }
  const labels = config.segmentLabels ?? []
  const first = labels[0] ?? '第1位'
  const last = labels[segLen - 1] ?? `第${segLen}位`
  return `请对应${first}到${last}，以“，”分隔，输入对应位置的号码，每一位置皆要输入号码；如：${example}`
}

/** 跨段组合玩法（前中后三 / 前后二 / 前后三 / 前后四）：位置非连续，提示用「N 个顺序号码」。 */
function usesSequentialGroupHint(config: PlayConfig): boolean {
  const text = `${config.playMethodLabel ?? ''} ${config.subPlayId ?? ''} ${config.playTypeId ?? ''} ${config.catalogSubId ?? ''}`
  if (/前中后三|前后二|前后三|前后四/.test(text)) return true
  return /qianzhonghou3|qianhou3|combo24/i.test(text)
}

/** 号池多选上限；一星/定位胆每位最多 9；包胆 / 龙虎（和）对齐第三方仅单选 */
export function poolMaxPicksForConfig(config: PlayConfig): number | null {
  if (config.poolMaxPicks != null && config.poolMaxPicks > 0) return config.poolMaxPicks
  if (config.betMode === 'baodan') return 1
  if (config.betMode === 'longhu' || config.betMode === 'longhuhe') return 1
  if (isLonghuPlayConfigLike(config)) return 1
  const method = config.playMethodLabel ?? ''
  if (method.includes('包胆')) return 1
  // 一星：0–9 共 10 个号，禁止单位置满号（对齐第三方/既定规则）
  if (isYixingDingweiPlayConfig(config)) return YIXING_MAX_PICKS_PER_POS
  return null
}

/** 在上限内切换号池选中（max=1 时点选替换；达上限时拒绝再加，保留原选） */
export function togglePoolPick(selected: string[], digit: string, maxPicks: number | null): string[] {
  const set = new Set(selected)
  if (set.has(digit)) {
    set.delete(digit)
    return [...set].sort((a, b) => Number(a) - Number(b) || a.localeCompare(b))
  }
  if (maxPicks === 1) return [digit]
  if (maxPicks != null && maxPicks > 0 && set.size >= maxPicks) {
    return [...set].sort((a, b) => Number(a) - Number(b) || a.localeCompare(b))
  }
  set.add(digit)
  return [...set].sort((a, b) => Number(a) - Number(b) || a.localeCompare(b))
}
