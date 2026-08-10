import {
  LHC_TEMA_ATTR_OPTIONS,
  LHC_TEMA_WAVE_OPTIONS,
  LHC_ZODIACS,
  LHC_ZODIAC_NUMBERS,
  LHC_TAIL_NUMBERS,
  LHC_TAIL_OPTIONS,
  LHC_ERQUANZHONG_NUM_MAX_PICKS,
  LHC_ERQUANZHONG_NUM_MIN_PICKS,
  LHC_SX_DUIPENG_MAX_PICKS,
  LHC_SX_DUIPENG_MIN_PICKS,
  LHC_WS_DUIPENG_MAX_PICKS,
  LHC_WS_DUIPENG_MIN_PICKS,
  LHC_SW_DUIPENG_MAX_PICKS,
  LHC_SW_DUIPENG_MIN_PICKS,
  isLhcErquanzhongFushiConfig,
  isLhcErquanzhongNumInputConfig,
  isLhcErquanzhongTuotouConfig,
  isLhcSxDuipengConfig,
  isLhcWsDuipengConfig,
  isLhcSwDuipengConfig,
  isLhcTemaAttrOption,
  isLhcTemaPlayConfig,
  isLhcTemaWaveOption,
  lhcMinPickCount,
} from '@/constants/lhcPlay'

export {
  isLhcErquanzhongFushiConfig,
  isLhcErquanzhongNumInputConfig,
  isLhcErquanzhongTuotouConfig,
  isLhcSxDuipengConfig,
  isLhcWsDuipengConfig,
  isLhcSwDuipengConfig,
  isLhcTemaPlayConfig,
}
import { isBetUnitValue } from '@/constants/betModeOptions'
import {
  isCatalogPlayTypeId,
  mapGuajiTypeIdToCatalog,
  resolvePlayConfigFromCatalogIds,
} from '@/utils/playConfig'
import {
  countHunheZuxuanUnits,
  countOrderedSpanCombinations,
  countOrderedSumCombinations,
  countZuxuanSumCombinations,
  hezhiDigitLenFromText,
  hunheDigitLenFromConfig,
} from '@/utils/playInputProfile'

export { hunheDigitLenFromConfig } from '@/utils/playInputProfile'
import {
  isPerPosDxdsPlayConfig,
  isWuxingSumDxdsPlayConfig,
  segmentBetMultiplier,
} from '@/utils/runTypeMatrix'
import {
  isLonghuPlayConfig,
  longhuPickHint,
  longhuPickOptionsForConfig,
} from '@/utils/longhuPickOptions'
import { resolvePlayTypeLabel } from '@/utils/playTypeLabels'

/** 保序去重（直选/组选单式对齐第三方：重复号码只计 1 注） */
export function uniquePreserveOrder(items: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const raw of items) {
    const t = raw.trim()
    if (!t || seen.has(t)) continue
    seen.add(t)
    out.push(t)
  }
  return out
}

/** 是否组选单式（须排除对子/豹子，并按组选形态去重） */
export function isZuxuanDanshiConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zuxuan_ds') return true
  const sub = (config.subPlayId ?? '').trim()
  if (sub === 'zuxuan_ds') return true
  const catalog = (config.catalogSubId ?? '').trim()
  if (catalog === 'zuxuan_ds' || catalog.endsWith('_zuxuan_ds')) return true
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  if (label.includes('组选单式')) return true
  return false
}

/** 组三单式（含任三组三单式；不含组选30 等） */
export function isZu3DanshiConfig(config: PlayConfig): boolean {
  const text = `${config.betMode ?? ''} ${config.subPlayId ?? ''} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  if (/组选3|组选30|zu30/i.test(text)) return false
  if (config.betMode === 'zu3' && (config.inputMode === 'danshi' || /单式|_ds/i.test(text))) return true
  if (/组三单式|zu3_ds|ren\d_zu3_ds/i.test(text)) return true
  // 数字 rule：任三组三单式 84
  const sid = Number.parseInt(String(config.catalogSubId ?? config.subPlayId ?? '').trim(), 10)
  if (sid === 84) return true
  return /组三|zu3/i.test(text) && /单式|danshi|_ds/i.test(text)
}

/**
 * 是否按「单式号码串」计注（须保序去重）。
 * 目录 subId 常为数字（如 g004/39），不能只认 zhixuan_ds；
 * 例：前二直选单式输入 12,13,14,15,12 → 4 注（重复 12 只计 1）。
 */
export function isSscDanshiLikeConfig(config: PlayConfig): boolean {
  if (isZuxuanDanshiConfig(config)) return true
  const bm = (config.betMode ?? '').trim()
  if (bm === 'danshi' || bm === 'zuxuan_ds') return true
  const sub = (config.subPlayId ?? '').trim()
  if (sub === 'zhixuan_ds' || sub === 'zuxuan_ds' || sub.endsWith('_ds')) return true
  const catalog = (config.catalogSubId ?? '').trim()
  if (catalog.endsWith('_ds')) return true
  if (config.inputMode === 'danshi' && config.playTemplate !== 'lhc_std') {
    // 混合组选走 hunhe 分支；此处排除已单独处理的玩法
    if (bm === 'hunhe' || bm === 'tuotou' || bm.endsWith('_dp')) return false
    return true
  }
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  if (label.includes('直选单式') || label.includes('组选单式')) return true
  if (label.includes('单式') && (label.includes('直选') || label.includes('组选') || label.includes('组三') || label.includes('组六'))) {
    return true
  }
  return false
}

/** 单式内容按位长过滤后保序去重（对齐第三方预览注数） */
export function dedupeDanshiTokens(raw: string, segmentLen: number): string[] {
  const parts = raw
    .split(/[,，\s\n]+/)
    .map((t) => t.trim())
    .filter(Boolean)
  const expect = segmentLen > 0 ? segmentLen : 0
  return uniquePreserveOrder(
    parts.filter((t) => /^\d+$/.test(t) && (!expect || t.length === expect)),
  )
}

/**
 * 冷热出号等「按位号池」内容：每位一行、行内为单码池（如 "4,5\\n3,5\\n2,5"）。
 * 与直选单式整注串（"452,455"）区分。
 */
export function isZhixuanPositionPoolContent(content: string, segmentLen: number): boolean {
  if (segmentLen <= 1) return false
  const raw = String(content ?? '').replace(/\r/g, '')
  if (!raw.includes('\n')) return false
  const lines = splitGroupLinesPad(raw, segmentLen).slice(0, segmentLen)
  for (let i = 0; i < segmentLen; i++) {
    const tokens = parsePickTokens(lines[i] ?? '')
    if (!tokens.length) return false
    if (tokens.some((t) => !/^[0-9]$/.test(t))) return false
  }
  return true
}

/** 按位号池笛卡尔积注数（直选复式口径） */
export function countZhixuanPositionPoolUnits(content: string, segmentLen: number): number {
  if (segmentLen <= 0) return 0
  const lines = splitGroupLinesPad(String(content ?? '').replace(/\r/g, ''), segmentLen).slice(0, segmentLen)
  let units = 1
  for (let i = 0; i < segmentLen; i++) {
    const n = [...new Set(parsePickTokens(lines[i] ?? ''))].length
    if (!n) return 0
    units *= n
  }
  return units
}

/** 按位号池 → 直选单式票（笛卡尔积）。例：`5\\n5\\n5` → `555` */
export function expandZhixuanPositionPoolToDanshi(content: string, segmentLen: number): string {
  if (!isZhixuanPositionPoolContent(content, segmentLen)) return ''
  const lines = splitGroupLinesPad(String(content ?? '').replace(/\r/g, ''), segmentLen).slice(0, segmentLen)
  const pools = lines.map((line) => [...new Set(parsePickTokens(line))])
  let cur = ['']
  for (const pool of pools) {
    const next: string[] = []
    for (const prefix of cur) {
      for (const d of pool) next.push(prefix + d)
    }
    cur = next
  }
  return uniquePreserveOrder(cur).join(',')
}

/** 直选单式展开结果剔除豹子票（随机出号 / 占位样例用） */
export function filterBaoziFromDanshiTickets(raw: string, segmentLen: number): string {
  const seg = segmentLen > 0 ? segmentLen : 0
  return dedupeDanshiTokens(raw, seg)
    .filter((t) => !isBaoziDigitTicket(t))
    .join(',')
}

/**
 * 按位号池展开为直选单式并剔除豹子；若配置禁止单独豹子且结果为空则返回空串。
 */
export function expandZhixuanPoolToDanshiWithoutBaozi(
  content: string,
  segmentLen: number,
): string {
  const expanded = expandZhixuanPositionPoolToDanshi(content, segmentLen)
  if (!expanded) return ''
  return filterBaoziFromDanshiTickets(expanded, segmentLen)
}

/** 直选单式 / 直选复式 / 混合组选：禁止「仅豹子号」时的统一提示 */
export const SOLO_BAOZI_FORBIDDEN_MSG =
  '当前方案不允许单独下注 111、222、333等类似的豹子号'

/** @deprecated 使用 SOLO_BAOZI_FORBIDDEN_MSG */
export const ZHIXUAN_DANSHI_SOLO_BAOZI_MSG = SOLO_BAOZI_FORBIDDEN_MSG

/** 单注是否为豹子（各位数字相同，如 111 / 22 / 55555） */
export function isBaoziDigitTicket(token: string): boolean {
  const t = String(token ?? '').trim()
  if (t.length < 2 || !/^\d+$/.test(t)) return false
  const head = t[0]!
  return [...t].every((c) => c === head)
}

/** 组三形态：恰好两同号 + 一异号（如 112 / 121；不含 111、012） */
export function isZu3DigitTicket(token: string): boolean {
  const t = String(token ?? '').trim()
  if (t.length !== 3 || !/^\d{3}$/.test(t)) return false
  const counts = new Map<string, number>()
  for (const c of t) counts.set(c, (counts.get(c) ?? 0) + 1)
  if (counts.size !== 2) return false
  return [...counts.values()].every((n) => n === 1 || n === 2)
}

/** 组三单式整注归一：仅保留组三形态，按形态去重保序 */
export function normalizeZu3DanshiContent(raw: string, segmentLen = 3): string {
  const expect = segmentLen > 0 ? segmentLen : 3
  const parts = String(raw ?? '')
    .split(/[,，\s\n]+/)
    .map((t) => t.trim())
    .filter(Boolean)
  const seen = new Set<string>()
  const out: string[] = []
  for (const t of parts) {
    if (!/^\d+$/.test(t) || t.length !== expect) continue
    if (expect === 3 ? !isZu3DigitTicket(t) : false) continue
    if (expect !== 3) {
      // 非三星暂按组选单式口径（不应走到任三组三）
      if (isBaoziDigitTicket(t)) continue
    }
    const key = [...t].sort().join('')
    if (seen.has(key)) continue
    seen.add(key)
    out.push(t)
  }
  return out.join(',')
}

export const ZU3_DANSHI_PATTERN_MSG =
  '组三单式：每注须为两个相同号码和一个不同号码（如 112），不含豹子与组六'

/** 组六单式（含任三组六单式；不含组选6/60 等） */
export function isZu6DanshiConfig(config: PlayConfig): boolean {
  const text = `${config.betMode ?? ''} ${config.subPlayId ?? ''} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  if (/组选6|组选60|组选120|zu60|zu120/i.test(text)) return false
  if (config.betMode === 'zu6' && (config.inputMode === 'danshi' || /单式|_ds/i.test(text))) return true
  if (/组六单式|zu6_ds|ren\d_zu6_ds/i.test(text)) return true
  // 数字 rule：任三组六单式 86
  const sid = Number.parseInt(String(config.catalogSubId ?? config.subPlayId ?? '').trim(), 10)
  if (sid === 86) return true
  return /组六|zu6/i.test(text) && /单式|danshi|_ds/i.test(text)
}

/** 组六形态：三位互不相同（如 012；不含 111、112） */
export function isZu6DigitTicket(token: string): boolean {
  const t = String(token ?? '').trim()
  if (t.length !== 3 || !/^\d{3}$/.test(t)) return false
  return new Set([...t]).size === 3
}

/** 组六单式整注归一：仅保留组六形态，按形态去重保序 */
export function normalizeZu6DanshiContent(raw: string, segmentLen = 3): string {
  const expect = segmentLen > 0 ? segmentLen : 3
  const parts = String(raw ?? '')
    .split(/[,，\s\n]+/)
    .map((t) => t.trim())
    .filter(Boolean)
  const seen = new Set<string>()
  const out: string[] = []
  for (const t of parts) {
    if (!/^\d+$/.test(t) || t.length !== expect) continue
    if (expect === 3 ? !isZu6DigitTicket(t) : false) continue
    if (expect !== 3 && isBaoziDigitTicket(t)) continue
    const key = [...t].sort().join('')
    if (seen.has(key)) continue
    seen.add(key)
    out.push(t)
  }
  return out.join(',')
}

export const ZU6_DANSHI_PATTERN_MSG =
  '组六单式：每注须为三个各不相同的号码（如 012），不含豹子与组三'

export const HUNHE_DANSHI_PATTERN_MSG =
  '混合组选：每注须为三个号码且不含豹子；组选形态相同只计 1 注（如 123 与 321），顺序不限'

/** 组六形态全集大小：C(10,3)=120 */
export const ZU6_DANSHI_FORM_COUNT = 120

/**
 * 随机生成 count 注组六单式整注（三位互异、形态去重，如 012）。
 */
export function randomZu6DanshiTickets(count: number, max = ZU6_DANSHI_FORM_COUNT): string {
  const want = Math.min(max, Math.max(1, Math.trunc(count) || 1))
  const out: string[] = []
  const seen = new Set<string>()
  let guard = 0
  while (out.length < want && guard++ < 3000) {
    const pool = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
    for (let i = pool.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      ;[pool[i], pool[j]] = [pool[j]!, pool[i]!]
    }
    const digits = pool.slice(0, 3).map(String)
    for (let i = digits.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      ;[digits[i], digits[j]] = [digits[j]!, digits[i]!]
    }
    const t = digits.join('')
    const key = [...t].sort().join('')
    if (seen.has(key)) continue
    seen.add(key)
    out.push(t)
  }
  return out.join(',')
}

/** 组三形态全集大小：C(10,2)×2（选对子号与异号） */
export const ZU3_DANSHI_FORM_COUNT = 90

/**
 * 随机生成 count 注组三单式整注（形态去重，如 112）。
 * 供开某投某「全部随机」/ 随机出号预览共用。
 */
export function randomZu3DanshiTickets(count: number, max = ZU3_DANSHI_FORM_COUNT): string {
  const want = Math.min(max, Math.max(1, Math.trunc(count) || 1))
  const out: string[] = []
  const seen = new Set<string>()
  let guard = 0
  while (out.length < want && guard++ < 2000) {
    const pair = Math.floor(Math.random() * 10)
    let single = Math.floor(Math.random() * 10)
    if (single === pair) continue
    const digits = [String(pair), String(pair), String(single)]
    for (let i = digits.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      ;[digits[i], digits[j]] = [digits[j]!, digits[i]!]
    }
    const t = digits.join('')
    const key = [...t].sort().join('')
    if (seen.has(key)) continue
    seen.add(key)
    out.push(t)
  }
  return out.join(',')
}

/** 直选复式（前二/前三/… 同类） */
export function isZhixuanFushiPlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'fushi' || bm === 'zhixuan_fs') return true
  const sub = (config.subPlayId ?? '').trim()
  if (sub === 'zhixuan_fs') return true
  const catalog = (config.catalogSubId ?? '').trim()
  if (catalog === 'zhixuan_fs' || catalog.endsWith('_zhixuan_fs')) return true
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  if (label.includes('直选复式')) return true
  if (label.includes('直选') && label.includes('复式') && !label.includes('组选')) return true
  return false
}

/** 混合组选（前三混合组选等） */
export function isHunhePlayConfig(config: PlayConfig): boolean {
  if ((config.betMode ?? '').trim() === 'hunhe') return true
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  return label.includes('混合组选')
}

/** 是否受「禁止单独豹子」约束的玩法 */
export function isSoloBaoziRestrictedPlay(config: PlayConfig): boolean {
  if (isSscDanshiLikeConfig(config) && !isZuxuanDanshiConfig(config)) return true
  if (isZhixuanFushiPlayConfig(config)) return true
  if (isHunhePlayConfig(config)) return true
  return false
}

function hunheTicketsFromContent(raw: string, digitLen: number): string[] {
  const parts = String(raw ?? '')
    .replace(/\r/g, '')
    .split(/[,，\s\n]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  const tickets = parts.filter((t) => /^\d+$/.test(t) && t.length === digitLen)
  if (tickets.length) return uniquePreserveOrder(tickets)
  const digits = String(raw ?? '').replace(/\D/g, '')
  if (digitLen > 0 && digits.length >= digitLen && digits.length % digitLen === 0) {
    const out: string[] = []
    for (let i = 0; i + digitLen <= digits.length; i += digitLen) {
      out.push(digits.slice(i, i + digitLen))
    }
    return uniquePreserveOrder(out)
  }
  return []
}

/** 混合组选落库内容：排除豹子，按组选形态去重（保序），ASCII 逗号分隔 */
export function normalizeHunheGroupContent(raw: string, digitLen: number): string {
  const len = digitLen > 0 ? digitLen : 3
  const tickets = hunheTicketsFromContent(raw, len)
  const seen = new Set<string>()
  const out: string[] = []
  for (const t of tickets) {
    if (isBaoziDigitTicket(t)) continue
    const key = [...t].sort().join('')
    if (seen.has(key)) continue
    seen.add(key)
    out.push(t)
  }
  return out.join(',')
}

/**
 * 方案内容是否「仅含豹子号」：
 * - 直选单式：有效整注全是 111/222…（含冷热按位号池展开后仅豹子）
 * - 直选复式：各位同一单码（如 1\\n1\\n1 → 111）
 * - 混合组选：有效注全是豹子
 */
export function isSchemeSoloBaoziContent(config: PlayConfig, raw: string): boolean {
  if (!isSoloBaoziRestrictedPlay(config)) return false
  let content = String(raw ?? '').replace(/\r/g, '')
  if (!content.trim()) return false

  if (isHunhePlayConfig(config)) {
    const digitLen = hunheDigitLenFromConfig(config)
    if (digitLen < 2) return false
    const tickets = hunheTicketsFromContent(content, digitLen)
    if (!tickets.length) return false
    return tickets.every(isBaoziDigitTicket)
  }

  if (isZhixuanFushiPlayConfig(config)) {
    const seg = config.segmentLen > 0 ? config.segmentLen : 0
    if (seg < 2) return false
    if (content.includes('\n') || isZhixuanPositionPoolContent(content, seg)) {
      const lines = splitGroupLinesPad(content, seg).slice(0, seg)
      return isZhixuanFushiBaoziLines(lines, seg)
    }
    const toks = parsePickTokens(content)
    if (!toks.length) return false
    // 单码号池扩成各位相同 → 豹子；或「1,1,1」按位同码
    if (toks.length === 1) return true
    if (toks.length === seg && toks.every((t) => t.length === 1) && toks.every((t) => t === toks[0])) {
      return true
    }
    return false
  }

  // 直选单式
  if (!isSscDanshiLikeConfig(config) || isZuxuanDanshiConfig(config)) return false
  const seg = config.segmentLen > 0 ? config.segmentLen : 0
  if (seg < 2) return false
  if (isZhixuanPositionPoolContent(content, seg)) {
    content = expandZhixuanPositionPoolToDanshi(content, seg)
  }
  const tokens = dedupeDanshiTokens(content, seg)
  if (!tokens.length) return false
  return tokens.every(isBaoziDigitTicket)
}

/** @deprecated 使用 isSchemeSoloBaoziContent */
export function isZhixuanDanshiSoloBaoziContent(config: PlayConfig, raw: string): boolean {
  return isSchemeSoloBaoziContent(config, raw)
}

/** 任一分组仅为豹子号时返回提示文案，否则 null */
export function schemeSoloBaoziError(config: PlayConfig, contents: string[]): string | null {
  for (const raw of contents) {
    if (isSchemeSoloBaoziContent(config, raw)) return SOLO_BAOZI_FORBIDDEN_MSG
  }
  return null
}

/** @deprecated 使用 schemeSoloBaoziError */
export function zhixuanDanshiSoloBaoziError(config: PlayConfig, contents: string[]): string | null {
  return schemeSoloBaoziError(config, contents)
}

/**
 * 子玩法切换后适配方案内容：
 * - 切到直选单式：把复式按位号池（`5\\n5\\n5`）展开为整注（`555`）；无法识别则清空
 * - 切到直选复式等按位录入：单式整注串无法可靠还原为按位号池，清空以免串味
 */
export function adaptSchemeGroupContentForPlay(content: string, config: PlayConfig): string {
  const raw = String(content ?? '').replace(/\r/g, '')
  if (!raw.trim()) return ''
  const seg = config.segmentLen > 0 ? config.segmentLen : 0

  if (isSscDanshiLikeConfig(config) && !isZuxuanDanshiConfig(config)) {
    if (seg > 1 && isZhixuanPositionPoolContent(raw, seg)) {
      return expandZhixuanPositionPoolToDanshi(raw, seg)
    }
    // 已是整注串则保留；含换行却非按位号池 → 清空
    if (raw.includes('\n')) return ''
    return dedupeDanshiTokens(raw, seg).join(',')
  }

  // 复式/按位：若仍是单式整注（无换行且 token 位长=segmentLen），无法安全还原，清空
  if (
    (config.inputMode === 'multiline' || config.subPlayId === 'zhixuan_fs' || config.betMode === 'fushi') &&
    seg > 1 &&
    !raw.includes('\n')
  ) {
    const tokens = dedupeDanshiTokens(raw, seg)
    if (tokens.length > 0 && tokens.every((t) => t.length === seg)) {
      return ''
    }
  }
  return raw
}

export const SSC_POSITION_LABELS = ['万', '千', '百', '十', '个'] as const

const REN_POS_DEFAULT: Record<number, string[]> = {
  2: ['千', '个'],
  3: ['万', '千', '个'],
  4: ['万', '千', '百', '十'],
}

/** SSC 任选（g011）；不含十一选五任选复式/单式 */
export function isSscRenxuanPlayConfig(config: PlayConfig): boolean {
  return isSscRenxuanConfig(config)
}

/** 任选·直选复式（五位逗号定位，无需选位壳） */
export function isRenxuanZhixuanFushiPlayConfig(config: PlayConfig): boolean {
  return isSscRenxuanConfig(config) && isRenxuanZhixuanFushi(config)
}

/**
 * 任选非直选复式：均需万千百十个选位（参考任二直选单式）。
 * 排除任二/任三/任选四直选复式。
 */
export function isRenxuanNeedsPositionConfig(config: PlayConfig): boolean {
  if (!isSscRenxuanConfig(config)) return false
  if (isRenxuanZhixuanFushi(config)) return false
  const k = config.renPositionCount ?? renPickCountFromConfig(config)
  return k >= 2 && k <= 5
}

/**
 * 任选选位 + 单式票面（直选/组选/混合/组三组六单式）。
 * 号池/和值类见 isRenxuanPositionPoolConfig。
 */
export function isRenxuanPositionDanshiConfig(config: PlayConfig): boolean {
  if (!isRenxuanNeedsPositionConfig(config)) return false
  const bm = config.betMode ?? ''
  if (bm === 'danshi' || bm === 'zuxuan_ds' || bm === 'hunhe') return true
  const label = `${config.playMethodLabel ?? ''}`
  if (/直选单式|组选单式|混合组选|组三单式|组六单式/.test(label)) return true
  return config.inputMode === 'danshi'
}

/** 任选选位 + 号池/和值（组选复式、组三/六复式、和值、组选24/12/6 等） */
export function isRenxuanPositionPoolConfig(config: PlayConfig): boolean {
  return isRenxuanNeedsPositionConfig(config) && !isRenxuanPositionDanshiConfig(config)
}

/** 任选选位内容：剥位后交给原玩法计注/选号面板（避免和值/号池分支误吃位名前缀） */
export function bareConfigForRenxuanPicks(config: PlayConfig): PlayConfig {
  const k = config.renPositionCount ?? renPickCountFromConfig(config)
  const bm = (config.betMode ?? '').trim()
  const label = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  // 和值/跨度/尾数：内层按任 k 位计组合；号池组选（组选24 等）须保持 segmentLen=1，勿当成按位录入
  const hezhiLike =
    bm === 'hezhi' ||
    bm === 'kuadu' ||
    bm === 'weishu' ||
    (/和值|跨度|尾数/.test(label) && !/单式|复式/.test(label))
  const poolZuxuan =
    config.inputMode === 'pool' ||
    ['zu24', 'zu12', 'zu4', 'zu3', 'zu6', 'zuxuan_fs', 'baodan'].includes(bm) ||
    /组选24|组选12|组选6|组选4|组选复式|组三|组六|包胆/i.test(label)
  let segmentLen = config.segmentLen
  if (hezhiLike && k >= 2 && k <= 5) {
    segmentLen = k
  } else if (poolZuxuan) {
    segmentLen = 1
  } else if (k >= 2 && k <= 5) {
    segmentLen = k
  }
  const bare: PlayConfig = {
    ...config,
    renPositionCount: undefined,
    playTypeId: '',
    guajiGroup: '',
    playTypeLabel: '',
    segmentLen,
  }
  // 剥位会清掉 renPositionCount/playTypeId；目录常把任四组选6 标成「组六」，
  // 若不保留识别信号，内层会按三星组六要求 ≥3 码并误报「组六至少选择 3 个号码」。
  if (isSixingZu6PlayConfig(config)) {
    const method = String(bare.playMethodLabel ?? '')
    if (!/组选6/i.test(method)) {
      bare.playMethodLabel = method ? `${method}组选6` : '组选6'
    }
    const sid = String(bare.catalogSubId ?? '').trim()
    if (!sid || !/^\d+$/.test(sid)) {
      const fromSub = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
      if (/^(132|139|145)$/.test(fromSub)) bare.catalogSubId = fromSub
      else if (!sid) bare.catalogSubId = '145'
    }
  }
  // 剥位会清掉 renPositionCount / 任选文案；组选复式最少选号等仍依赖「任二/任三」信号
  if (k >= 2 && k <= 5) {
    const tag = k === 2 ? '任二' : k === 3 ? '任三' : k === 4 ? '任四' : `任${k}`
    const method = String(bare.playMethodLabel ?? '')
    if (!new RegExp(tag).test(method)) {
      bare.playMethodLabel = method ? `${method}${tag}` : tag
    }
  }
  return bare
}

function countRenxuanNeedsPositionUnits(config: PlayConfig, content: string): number {
  const k = config.renPositionCount ?? renPickCountFromConfig(config)
  const { positions, picks } = parseRenxuanPositionContent(content, k)
  if (positions.length < k || positions.length > RENXUAN_POS_MAX || !picks.trim()) return 0
  const mul = comboCount(positions.length, k)
  if (mul <= 0) return 0

  if (isRenxuanPositionDanshiConfig(config)) {
    const digitLen = config.segmentLen > 0 ? config.segmentLen : k
    let picksBody = picks
    if (digitLen > 1 && isZhixuanPositionPoolContent(picksBody, digitLen)) {
      picksBody = expandZhixuanPositionPoolToDanshi(picksBody, digitLen)
      if (!picksBody) return 0
    }
    let tickets = 0
    if (isZu3DanshiConfig(config)) {
      tickets = normalizeZu3DanshiContent(picksBody, digitLen)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean).length
    } else if (isZu6DanshiConfig(config)) {
      tickets = normalizeZu6DanshiContent(picksBody, digitLen)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean).length
    } else if (isHunhePlayConfig(config)) {
      tickets = normalizeHunheGroupContent(picksBody, digitLen)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean).length
    } else if (isZuxuanDanshiConfig(config)) {
      tickets = normalizeZuxuanDanshiContent(picksBody, digitLen)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean).length
    } else {
      tickets = dedupeDanshiTokens(picksBody, digitLen).length
    }
    if (!tickets) return 0
    return mul * tickets
  }

  const base = countBetUnits(bareConfigForRenxuanPicks(config), picks)
  return base > 0 ? mul * base : 0
}

export function defaultRenxuanPositions(k: number): string[] {
  return [...(REN_POS_DEFAULT[k] ?? REN_POS_DEFAULT[2]!)]
}

function extractSscPositionNames(raw: string): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  for (const tok of raw.split(/[,，\s]+/).map((t) => t.trim()).filter(Boolean)) {
    for (const lab of SSC_POSITION_LABELS) {
      if ((tok === lab || tok === `${lab}位`) && !seen.has(lab)) {
        seen.add(lab)
        out.push(lab)
      }
    }
  }
  if (out.length) return out
  for (const r of raw) {
    const lab = String(r)
    if ((SSC_POSITION_LABELS as readonly string[]).includes(lab) && !seen.has(lab)) {
      seen.add(lab)
      out.push(lab)
    }
  }
  return out
}

/** 任选单式选位上限（万千百十个） */
const RENXUAN_POS_MAX = 5

export function parseRenxuanPositionContent(
  raw: string,
  k: number,
): { positions: string[]; picks: string } {
  const text = (raw || '').trim()
  const want = k > 0 ? k : 2
  if (!text) {
    return { positions: defaultRenxuanPositions(want), picks: '' }
  }
  const pipe = text.indexOf('|')
  if (pipe > 0) {
    const positions = extractSscPositionNames(text.slice(0, pipe).trim()).slice(0, RENXUAN_POS_MAX)
    const picks = text.slice(pipe + 1).trim()
    // 允许多选位（want…5），不再截成恰好 k 个
    if (positions.length >= want) {
      return { positions, picks }
    }
  }
  const lines = text.split(/\n/).map((l) => l.trim())
  if (lines.length >= 1) {
    const positions = extractSscPositionNames(lines[0] ?? '').slice(0, RENXUAN_POS_MAX)
    if (positions.length >= want) {
      return {
        positions,
        picks: lines.slice(1).join('\n').trim(),
      }
    }
  }
  return { positions: defaultRenxuanPositions(want), picks: text }
}

/** 组内容：首行位名 + 换行 + 号码（与 guajibet.parseRenxuanPositionPick 对齐） */
export function buildRenxuanPositionContent(positions: string[], picks: string): string {
  const posLine = positions.join(',')
  const body = (picks || '').trim()
  return body ? `${posLine}\n${body}` : posLine
}

/**
 * 组选单式：排除对子/豹子，按组选形态去重（12 与 21 同一注），保序保留首次形态。
 * 例：11,12,13,14,15,16,17,22,24,25 → 8 注（去掉 11/22）
 */
export function dedupeZuxuanDanshiTokens(raw: string, segmentLen: number): string[] {
  const expect = segmentLen > 0 ? segmentLen : 2
  const parts = raw
    .split(/[,，\s\n]+/)
    .map((t) => t.trim())
    .filter(Boolean)
  const seen = new Set<string>()
  const out: string[] = []
  for (const t of parts) {
    if (!/^\d+$/.test(t) || t.length !== expect) continue
    if ([...t].every((c) => c === t[0])) continue
    const key = [...t].sort().join('')
    if (seen.has(key)) continue
    seen.add(key)
    out.push(t)
  }
  return out
}

/**
 * 单码号池展成组选单式整注（升序组合、不含对子/豹子）。
 * 对齐后端 guajibet.expandZuxuanDigitPoolToDanshi：如 1,2,3 → 12,13,23。
 */
export function expandZuxuanDigitPoolToDanshi(raw: string, segmentLen: number): string {
  const expect = segmentLen > 0 ? segmentLen : 2
  const parts = String(raw ?? '')
    .replace(/，/g, ',')
    .split(/[,，\s\n]+/)
    .map((t) => t.trim())
    .filter(Boolean)
  const digits: string[] = []
  const seen = new Set<string>()
  for (const p of parts) {
    const d = p.replace(/\D/g, '')
    if (d.length !== 1) return ''
    if (seen.has(d)) continue
    seen.add(d)
    digits.push(d)
  }
  if (digits.length < expect) return ''
  digits.sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
  const combos: string[] = []
  const walk = (start: number, cur: string[]) => {
    if (cur.length === expect) {
      const ticket = cur.join('')
      if (![...ticket].every((c) => c === ticket[0])) combos.push(ticket)
      return
    }
    for (let i = start; i < digits.length; i++) {
      walk(i + 1, [...cur, digits[i]!])
    }
  }
  walk(0, [])
  return combos.join(',')
}

/**
 * 组选单式内容归一：已有整注则形态去重；否则单码号池按 C(n,k) 展成整注。
 */
export function normalizeZuxuanDanshiContent(raw: string, segmentLen: number): string {
  const expect = segmentLen > 0 ? segmentLen : 2
  const tickets = dedupeZuxuanDanshiTokens(raw, expect)
  if (tickets.length) return tickets.join(',')
  return expandZuxuanDigitPoolToDanshi(raw, expect)
}

/** 与后端 schemes/play_api.go BetPayload 对齐 */
export interface GameBetPayload {  playTemplate?: string
  typeId?: string
  subId?: string
  playMethod?: string
  playTypeId?: string
  subPlayId?: string
  groupContent: string
}

export interface PlayConfig {
  playTypeId: string
  subPlayId: string
  segmentLen: number
  segmentLabels: string[]
  inputMode: 'dingwei' | 'pool' | 'multiline' | 'danshi' | 'lhc_num' | 'lhc_zodiac' | 'lhc_tail' | 'lhc_attr'
  betMode?: string
  catalogSubId?: string
  numberPoolMin?: number
  numberPoolMax?: number
  /** 号池最多可选个数；包胆等对齐第三方为 1（单选） */
  poolMaxPicks?: number
  /**
   * 任选直选/组选单式：从万千百十个中勾选的位数（任二=2）。
   * 内容格式：首行位名（千,个）+ 换行 + 号码（12,34）
   */
  renPositionCount?: number
  /** rules/v2 同步后来自 play_types.label */
  playTypeLabel?: string
  /** rules/v2 同步后来自 sub_plays.label */
  playMethodLabel?: string
  playTemplate?: string
  /** rules/v2 segment_rule.guajiGroup，用于前中后三等注数倍增 */
  guajiGroup?: string
}
const POSITION_LABELS = ['万', '千', '百', '十', '个'] as const

function configFromPlayIds(playTypeId: string, subPlayId: string): PlayConfig {
  // 旧订单兼容：hou4 映射为 catalog sixing；g004 等映射为 qian2
  const typeId = mapGuajiTypeIdToCatalog(playTypeId === 'hou4' ? 'sixing' : playTypeId)
  if (isCatalogPlayTypeId(typeId)) {
    return resolvePlayConfigFromCatalogIds(typeId, subPlayId)
  }
  const segmentLabels = POSITION_LABELS.slice(0, 1)
  return {
    playTypeId: typeId,
    subPlayId,
    segmentLen: 1,
    segmentLabels,
    inputMode: 'dingwei',
  }
}
/** 优先使用 playTypeId/subPlayId；缺失时 fallback 中文 playMethod 解析 */
export function resolvePlayConfig(options: {
  playMethod?: string
  playTypeId?: string
  subPlayId?: string
  betMode?: string
}): PlayConfig {
  const playTypeId = options.playTypeId?.trim()
  const subPlayId = options.subPlayId?.trim() ?? ''
  const betMode = options.betMode?.trim() ?? ''
  if (playTypeId) {
    const typeId = mapGuajiTypeIdToCatalog(playTypeId === 'hou4' ? 'sixing' : playTypeId)
    if (isCatalogPlayTypeId(typeId)) {
      return resolvePlayConfigFromCatalogIds(typeId, subPlayId, betMode)
    }
    return configFromPlayIds(playTypeId, subPlayId)
  }
  return inferPlayConfig(options.playMethod?.trim() || '定位胆万位')
}

function dingweiSubFromMethod(pm: string): string {
  if (pm.includes('万位')) return 'dingwei_wan'
  if (pm.includes('千位')) return 'dingwei_qian'
  if (pm.includes('百位')) return 'dingwei_bai'
  if (pm.includes('十位')) return 'dingwei_shi'
  if (pm.includes('个位')) return 'dingwei_ge'
  return ''
}

export function inferPlayConfig(playMethod: string): PlayConfig {
  const pm = playMethod.trim() || '定位胆万位'
  let playTypeId = 'dingwei'
  if (pm.includes('五星')) playTypeId = 'wuxing'
  else if (pm.includes('四星') || pm.includes('后四')) playTypeId = 'sixing'
  else if (pm.includes('前三')) playTypeId = 'qian3'
  else if (pm.includes('中三')) playTypeId = 'zhong3'
  else if (pm.includes('后三')) playTypeId = 'hou3'
  else if (pm.includes('前二')) playTypeId = 'qian2'
  else if (pm.includes('后二')) playTypeId = 'hou2'

  let subPlayId = dingweiSubFromMethod(pm)
  if (pm.includes('直选复式')) subPlayId = 'zhixuan_fs'
  else if (pm.includes('直选单式')) subPlayId = 'zhixuan_ds'
  else if (pm.includes('组选') || pm.includes('组三') || pm.includes('组六')) subPlayId = 'zuxuan_fs'

  return configFromPlayIds(playTypeId, subPlayId)
}
export function parsePickTokens(raw: string, pool?: { min?: number; max?: number }): string[] {
  const min = pool?.min ?? 0
  const max = pool?.max ?? 9
  if (max > 9) {
    return parsePoolTokens(raw, min, max)
  }
  // 0–9 号池：粘连「12」「1234567890」按位拆开（对齐录入失焦与后端 formatSSCZuxuanPoolDigits）
  const parts = String(raw ?? '')
    .split(/[\s,，\n]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  const seen = new Set<string>()
  const out: string[] = []
  const push = (ch: string) => {
    if (!/^[0-9]$/.test(ch)) return
    const n = Number(ch)
    if (n < min || n > max) return
    if (seen.has(ch)) return
    seen.add(ch)
    out.push(ch)
  }
  for (const p of parts) {
    if (!/^\d+$/.test(p)) continue
    if (p.length === 1) {
      push(p)
      continue
    }
    for (const ch of p) push(ch)
  }
  return out
}

/** 直选复式各位均为同一单码（豹子/对子）——第三方网页计 0 注 */
export function isZhixuanFushiBaoziLines(lines: string[], segmentLen: number): boolean {
  if (segmentLen < 2 || lines.length < segmentLen) return false
  let first = ''
  for (let i = 0; i < segmentLen; i++) {
    const toks = [...new Set(parsePickTokens(lines[i] ?? ''))]
    if (toks.length !== 1) return false
    const d = toks[0] ?? ''
    if (!d) return false
    if (i === 0) first = d
    else if (d !== first) return false
  }
  return true
}

export function parsePoolTokens(raw: string, min: number, max: number): string[] {
  const parts = raw.split(/[\s,，\n]+/).map((s) => s.trim()).filter(Boolean)
  const seen = new Set<string>()
  const out: string[] = []
  // 仅 11选5(01–11)/PK10(01–10) 定宽号球补零；和值号池（1–26/0–27/3–18）须自然数，补零会被第三方拒「投注注数不正确」
  const padFixedBall = min >= 1 && max >= 10 && max <= 11
  for (const p of parts) {
    if (!/^\d{1,2}$/.test(p)) continue
    const n = Number(p)
    if (n < min || n > max) continue
    const tok = padFixedBall ? String(n).padStart(2, '0') : String(n)
    if (seen.has(tok)) continue
    seen.add(tok)
    out.push(tok)
  }
  return out
}

function poolFromConfig(config: PlayConfig): { min: number; max: number } | undefined {
  if (config.numberPoolMax != null && config.numberPoolMax > 9) {
    return { min: config.numberPoolMin ?? 1, max: config.numberPoolMax }
  }
  if (config.numberPoolMax != null && config.numberPoolMin != null) {
    return { min: config.numberPoolMin, max: config.numberPoolMax }
  }
  return undefined
}

function syxwRenxuanNM(subId: string): { pickN: number; matchM: number } | null {
  const s = subId.toLowerCase().replace(/_ds$/, '')
  const m = /^rx_(\d+)z(\d+)/.exec(s)
  if (!m) return null
  return { pickN: Number(m[1]), matchM: Number(m[2]) }
}

export function parseLhcNumberTokens(raw: string): string[] {
  return raw
    .split(/[\s,，\n|#]+/)
    .map((s) => s.trim())
    .filter((s) => {
      if (!/^\d{1,2}$/.test(s)) return false
      const n = Number(s)
      return n >= 1 && n <= 49
    })
    .map((s) => String(Number(s)).padStart(2, '0'))
}

const LHC_ZODIAC_SET = new Set<string>(LHC_ZODIACS)

/** 解析生肖 token（保序去重；兼容 马|龙 / 马,龙） */
export function parseLhcZodiacTokens(raw: string): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const t of String(raw ?? '')
    .split(/[\s,，\n|#]+/)
    .map((s) => s.trim())
    .filter(Boolean)) {
    if (!LHC_ZODIAC_SET.has(t) || seen.has(t)) continue
    seen.add(t)
    out.push(t)
  }
  return out
}

const LHC_TAIL_SET = new Set<string>(LHC_TAIL_OPTIONS)

/** 解析尾数 token（保序去重；兼容 0|1 / 0,1 / 0尾） */
export function parseLhcTailTokens(raw: string): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const t of String(raw ?? '')
    .split(/[\s,，\n|#]+/)
    .map((s) => s.trim().replace(/尾$/, ''))
    .filter(Boolean)) {
    if (!LHC_TAIL_SET.has(t) || seen.has(t)) continue
    seen.add(t)
    out.push(t)
  }
  return out
}

/** 解析生尾对碰：恰好 1 肖 + 1 尾 → [肖, 尾]；顺序归一为肖在前 */
export function parseLhcSwDuipengTokens(raw: string): string[] {
  const zs = parseLhcZodiacTokens(raw)
  const ts = parseLhcTailTokens(raw)
  if (zs.length !== 1 || ts.length !== 1) return []
  return [zs[0]!, ts[0]!]
}

/** 生尾对碰注数：肖展开 × 尾展开 − 共有号码（对齐第三方） */
export function countLhcSwDuipengUnits(zodiac: string, tail: string): number {
  const left = LHC_ZODIAC_NUMBERS[String(zodiac ?? '').trim()] ?? []
  const t = String(tail ?? '')
    .trim()
    .replace(/尾$/, '')
  const right = LHC_TAIL_NUMBERS[t] ?? []
  if (!left.length || !right.length) return 0
  const rightSet = new Set(right)
  let overlap = 0
  for (const n of left) {
    if (rightSet.has(n)) overlap++
  }
  return left.length * right.length - overlap
}

export type LhcTemaParts = {
  nums: string[]
  attrs: string[]
  waves: string[]
}

function aliasLhcTemaToken(raw: string): string {
  let t = String(raw ?? '').trim()
  if (t.endsWith('||')) t = t.slice(0, -2).trim()
  if (t === '洪波') return '红波'
  if (t === '绿播') return '绿波'
  return t
}

function sortLhcTemaByCanon(tokens: string[], canon: readonly string[]): string[] {
  const rank = new Map(canon.map((v, i) => [v, i]))
  return [...tokens].sort((a, b) => (rank.get(a) ?? 999) - (rank.get(b) ?? 999))
}

/** 特码方案内容：`号码|属性|波色`；兼容旧逗号混选与 `07||,13||` */
export function parseLhcTemaParts(raw: string): LhcTemaParts {
  const text = String(raw ?? '').trim()
  const numSet = new Set<string>()
  const attrSet = new Set<string>()
  const waveSet = new Set<string>()

  const take = (token: string) => {
    const t = aliasLhcTemaToken(token)
    if (!t) return
    if (isLhcTemaWaveOption(t)) {
      waveSet.add(t)
      return
    }
    if (isLhcTemaAttrOption(t)) {
      attrSet.add(t)
      return
    }
    if (!/^\d{1,2}$/.test(t)) return
    const n = Number(t)
    // 第三方特码仅 01–49；00 会回「投注数字不合规」
    if (n < 1 || n > 49) return
    numSet.add(String(n).padStart(2, '0'))
  }

  if (text.includes('|')) {
    // 旧多注：07||,13|| / 大|| → 按逗号拆再分类
    if (/,\s*\S+\|\|/.test(text) || /\|\|,\s*/.test(text)) {
      for (const part of text.split(/[,，]+/)) take(part)
    } else {
      const sections = text.split('|')
      for (const sec of sections.slice(0, 3)) {
        for (const part of sec.split(/[,，\s\n]+/)) take(part)
      }
      // 多余段也扫一遍，避免脏数据丢 token
      for (const sec of sections.slice(3)) {
        for (const part of sec.split(/[,，\s\n]+/)) take(part)
      }
    }
  } else {
    for (const part of text.split(/[,，\s\n]+/)) take(part)
  }

  const nums = [...numSet].sort((a, b) => Number(a) - Number(b))
  const attrs = sortLhcTemaByCanon([...attrSet], LHC_TEMA_ATTR_OPTIONS)
  const waves = sortLhcTemaByCanon([...waveSet], LHC_TEMA_WAVE_OPTIONS)
  return { nums, attrs, waves }
}

export function formatLhcTemaParts(parts: LhcTemaParts): string {
  const nums = parts.nums.join(',')
  const attrs = parts.attrs.join(',')
  const waves = parts.waves.join(',')
  if (!nums && !attrs && !waves) return ''
  return `${nums}|${attrs}|${waves}`
}

export function parseLhcTemaContentTokens(raw: string): string[] {
  const { nums, attrs, waves } = parseLhcTemaParts(raw)
  return [...nums, ...attrs, ...waves]
}

export function normalizeLhcTemaContent(raw: string): string {
  return formatLhcTemaParts(parseLhcTemaParts(raw))
}

/** 拆特码录入 token（兼容 flat 逗号混选与 号码|属性|波色） */
export function splitLhcTemaRawTokens(raw: string): string[] {
  const text = String(raw ?? '').trim()
  if (!text) return []
  if (text.includes('|')) {
    if (/,\s*\S+\|\|/.test(text) || /\|\|,\s*/.test(text)) {
      return text
        .split(/[,，]+/)
        .map((s) => s.trim())
        .filter(Boolean)
    }
    return text
      .split('|')
      .flatMap((sec) => sec.split(/[,，\s\n]+/))
      .map((s) => s.trim())
      .filter(Boolean)
  }
  return text
    .split(/[,，\s\n]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 单 token：合法则返回规范化值（01 / 大 / 红波），否则 invalid */
export function classifyLhcTemaToken(
  raw: string,
): { ok: true; value: string } | { ok: false; raw: string } {
  const original = String(raw ?? '').trim()
  if (!original) return { ok: false, raw: '' }
  const t = aliasLhcTemaToken(original)
  if (!t) return { ok: false, raw: original }
  if (isLhcTemaWaveOption(t) || isLhcTemaAttrOption(t)) return { ok: true, value: t }
  if (!/^\d{1,2}$/.test(t)) return { ok: false, raw: original }
  const n = Number(t)
  if (n < 1 || n > 49) return { ok: false, raw: original }
  return { ok: true, value: String(n).padStart(2, '0') }
}

export function lhcTemaInvalidTokens(raw: string): string[] {
  const bad: string[] = []
  const seen = new Set<string>()
  for (const part of splitLhcTemaRawTokens(raw)) {
    const c = classifyLhcTemaToken(part)
    if (c.ok) continue
    if (!c.raw || seen.has(c.raw)) continue
    seen.add(c.raw)
    bad.push(c.raw)
  }
  return bad
}

/**
 * 开某投某正/反投录入：逗号混选保序去重，如 01,02,大,03,蓝波。
 * 下单时再由 normalizeLhcTemaContent / 后端 wire 合成 号码|属性|波色。
 */
export function normalizeLhcTemaFlatContent(raw: string): string {
  const out: string[] = []
  const seen = new Set<string>()
  for (const part of splitLhcTemaRawTokens(raw)) {
    const c = classifyLhcTemaToken(part)
    if (!c.ok) continue
    if (seen.has(c.value)) continue
    seen.add(c.value)
    out.push(c.value)
  }
  return out.join(',')
}

function comboCount(n: number, k: number): number {
  if (n < k || k <= 0) return n > 0 ? n : 0
  let out = 1
  for (let i = 0; i < k; i++) out = (out * (n - i)) / (i + 1)
  return Math.round(out)
}

/** 不定位码数：一码/二码/三码（勿把「前二码/后二码」里的「二码」当成不定位二码） */
function inferBudingweiNeed(config: PlayConfig): number {
  const text =
    `${config.catalogSubId ?? ''} ${config.subPlayId} ${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''}`.toLowerCase()
  if (text.includes('_3ma') || text.includes('3ma') || (text.includes('不定位') && text.includes('三码'))) return 3
  if (text.includes('_2ma') || text.includes('2ma') || (text.includes('不定位') && text.includes('二码'))) return 2
  // rules/v2 数字 id（文案缺失时，对齐 guajibet.budingweiPickCount）
  const sid = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
  if (sid === '152') return 3
  if (['114', '116', '118', '147', '149', '151'].includes(sid)) return 2
  if (['113', '115', '117', '146', '148', '150'].includes(sid)) return 1
  return 1
}

/** 五星二码/三码不定位：第三方要求号池至少 4 个号（含目录 id 151/152） */
function isWuxingBudingweiMulti(config: PlayConfig): boolean {
  if (inferBudingweiNeed(config) < 2) return false
  const text =
    `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`.toLowerCase()
  if (text.includes('五星') || text.includes('wuxing')) return true
  const sid = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
  return sid === '151' || sid === '152'
}

/**
 * 不定位号池最少选号（第三方：二码不能低于两个；五星二/三码至少 4；三码至少 3）。
 * 非不定位返回 null。
 */
export function budingweiMinPicks(config: PlayConfig): number | null {
  if (!isBudingweiPlayConfig(config)) return null
  const need = inferBudingweiNeed(config)
  if (isWuxingBudingweiMulti(config)) return 4
  if (need <= 1) return 1
  return need
}

/** 不定位号池不足最少选号时的提示（二码对齐第三方文案） */
export function budingweiMinPicksMessage(config: PlayConfig): string {
  const min = budingweiMinPicks(config)
  if (min == null) return '选号无效'
  const need = inferBudingweiNeed(config)
  if (need === 2 && min === 2) return '投注数字不能低于两个'
  if (min === 4) {
    return need === 3 ? '五星三码不定位：至少选择 4 个号码' : '五星二码不定位：至少选择 4 个号码'
  }
  if (need === 3) return '三码不定位：至少选择 3 个号码'
  return `不定位：至少选择 ${min} 个号码`
}

function isBudingweiPlayConfig(config: PlayConfig): boolean {
  if (config.betMode === 'budingwei') return true
  const tid = (config.playTypeId || '').toLowerCase()
  // SSC 不定位=g009；syxw 不定位=g004。SSC 的 g004 是前二码，绝不能当不定位。
  if (tid === 'budingwei' || tid === 'g009') return true
  if (tid === 'g004' && config.playTemplate === 'syxw_std') return true
  const label = `${config.playTypeLabel ?? ''} ${config.playMethodLabel ?? ''} ${config.guajiGroup ?? ''}`
  return label.includes('不定位')
}

function isLhcDanshiBetMode(betMode: string): boolean {
  return betMode === 'guoguan' || betMode === 'tuotou' || betMode.endsWith('_dp')
}

function lhcDuipengGroupSize(betMode: string, raw: string): number {
  const tokens = raw
    .split(/[,，]/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (!tokens.length) return 0
  // 生肖对碰：一侧展开为该肖全部号码个数（马=5，其余肖=4）
  if (betMode === 'sx_dp') {
    let n = 0
    for (const t of tokens) {
      n += (LHC_ZODIAC_NUMBERS[t] ?? []).length
    }
    return n
  }
  // 尾数对碰：一侧展开为该尾全部号码个数（0=4，1–9=5）
  if (betMode === 'ws_dp') {
    let n = 0
    for (const t of tokens) {
      const tok = t.replace(/尾$/, '')
      n += (LHC_TAIL_NUMBERS[tok] ?? []).length
    }
    return n
  }
  // 生尾对碰单侧：生肖或尾数
  if (betMode === 'sw_dp') {
    let n = 0
    for (const t of tokens) {
      const zNums = LHC_ZODIAC_NUMBERS[t]
      if (zNums?.length) {
        n += zNums.length
        continue
      }
      const tok = t.replace(/尾$/, '')
      n += (LHC_TAIL_NUMBERS[tok] ?? []).length
    }
    return n
  }
  const nums = parseLhcNumberTokens(tokens.join(','))
  if (nums.length) return nums.length
  return tokens.length
}

function countLhcDanshiUnits(config: PlayConfig, content: string): number {
  const betMode = config.betMode ?? ''
  if (betMode === 'guoguan') {
    const parts = content.split(/[,，]/).map((s) => s.trim()).filter(Boolean)
    return parts.length || (content ? 1 : 0)
  }
  if (betMode === 'tuotou') {
    const sep = content.includes('|') ? '|' : content.includes('#') ? '#' : ''
    if (sep) {
      const [dan, tuo] = content.split(sep)
      const d = parseLhcNumberTokens(dan ?? '').length
      const t = parseLhcNumberTokens(tuo ?? '').length
      const subId = config.catalogSubId ?? config.subPlayId
      const min = lhcMinPickCount('fushi', subId)
      return Math.max(d, 1) * comboCount(t, Math.max(min - 1, 1))
    }
    // 二全中拖头扁选：首号为胆，其余为拖
    const pool = [...new Set(parseLhcNumberTokens(content))]
    if (pool.length >= 2) {
      const subId = config.catalogSubId ?? config.subPlayId
      const min = lhcMinPickCount('fushi', subId)
      return comboCount(pool.length - 1, Math.max(min - 1, 1))
    }
    return 0
  }
  if (betMode.endsWith('_dp')) {
    const sep = content.includes('|') ? '|' : content.includes('#') ? '#' : ''
    if (sep) {
      const [a, b] = content.split(sep)
      if (betMode === 'sw_dp') {
        const parts = parseLhcSwDuipengTokens(content)
        if (parts.length !== LHC_SW_DUIPENG_MAX_PICKS) return 0
        return countLhcSwDuipengUnits(parts[0]!, parts[1]!)
      }
      const units = lhcDuipengGroupSize(betMode, a ?? '') * lhcDuipengGroupSize(betMode, b ?? '')
      return units || 0
    }
    // 生肖对碰扁选：恰好两个生肖 → |侧号码数 × |侧号码数（马×肖=20，肖×肖=16）
    if (betMode === 'sx_dp') {
      const zs = parseLhcZodiacTokens(content)
      if (zs.length !== LHC_SX_DUIPENG_MAX_PICKS) return 0
      return lhcDuipengGroupSize('sx_dp', zs[0]!) * lhcDuipengGroupSize('sx_dp', zs[1]!)
    }
    // 尾数对碰扁选：恰好两个尾数 → 展开积（0×1=20；1×2=25）
    if (betMode === 'ws_dp') {
      const ts = parseLhcTailTokens(content)
      if (ts.length !== LHC_WS_DUIPENG_MAX_PICKS) return 0
      return lhcDuipengGroupSize('ws_dp', ts[0]!) * lhcDuipengGroupSize('ws_dp', ts[1]!)
    }
    // 生尾对碰扁选：1 肖 + 1 尾 → 展开积 − 共有号码（狗|5=19，与第三方一致）
    if (betMode === 'sw_dp') {
      const parts = parseLhcSwDuipengTokens(content)
      if (parts.length !== LHC_SW_DUIPENG_MAX_PICKS) return 0
      return countLhcSwDuipengUnits(parts[0]!, parts[1]!)
    }
    return content ? 1 : 0
  }
  return 0
}

function parseTextPickTokens(raw: string, allowed: string[]): string[] {
  const set = new Set(allowed)
  return raw
    .split(/[\s,，\n]+/)
    .map((s) => s.trim())
    .filter((s) => set.has(s))
}

export function parseGroupPicks(
  config: PlayConfig,
  content: string,
): { digits: string[]; lines: string[][] } {
  const trimmed = content.trim()
  if (isLonghuPlayConfig(config)) {
    return {
      digits: parseTextPickTokens(trimmed, longhuPickOptionsForConfig(config)),
      lines: [],
    }
  }
  // 五星趣味：0–9 数字池（勿当豹子/对子/顺子）
  if (isWuxingQuweiDigitPlayConfig(config)) {
    return { digits: parseWuxingQuweiDigits(trimmed), lines: [] }
  }
  const textModes = ['daxiao', 'danshuang', 'dxds', 'teshu', 'longhubao', 'zhuangxian'] as const
  if (config.betMode && (textModes as readonly string[]).includes(config.betMode)) {
    const opts: Record<string, string[]> = {
      daxiao: ['大', '小'],
      danshuang: ['单', '双'],
      dxds: ['大', '小', '单', '双'],
      teshu:
        config.playTemplate === 'pc28_std'
          ? ['豹子', '对子', '顺子', '极大', '极小']
          : ['豹子', '对子', '顺子'],
      longhubao: ['龙', '虎', '豹'],
      zhuangxian: ['庄', '闲'],
    }
    const allowed = opts[config.betMode] ?? []
    if (config.inputMode === 'multiline' && config.segmentLen > 1) {
      return {
        digits: [],
        lines: splitGroupLines(trimmed).map((line) => parseTextPickTokens(line, allowed)),
      }
    }
    return { digits: parseTextPickTokens(trimmed, allowed), lines: [] }
  }
  const pool = poolFromConfig(config)
  if (config.inputMode === 'multiline') {
    const padded = isDingweiMultilineConfig(config)
      ? dingweiPositionLines(String(content ?? '').replace(/\r/g, ''), config.segmentLen)
      : splitGroupLines(trimmed)
    return {
      digits: [],
      lines: padded.map((line) => parsePickTokens(line, pool)),
    }
  }
  if (config.inputMode === 'lhc_num') {
    return { digits: parseLhcNumberTokens(trimmed), lines: [] }
  }
  if (
    config.inputMode === 'lhc_zodiac' ||
    config.inputMode === 'lhc_tail' ||
    config.inputMode === 'lhc_attr'
  ) {
    if (isLhcSxDuipengConfig(config) || config.betMode === 'sx_dp') {
      return { digits: parseLhcZodiacTokens(trimmed), lines: [] }
    }
    if (isLhcWsDuipengConfig(config) || config.betMode === 'ws_dp') {
      return { digits: parseLhcTailTokens(trimmed), lines: [] }
    }
    if (isLhcSwDuipengConfig(config) || config.betMode === 'sw_dp') {
      return { digits: parseLhcSwDuipengTokens(trimmed), lines: [] }
    }
    return {
      digits: trimmed
        .split(/[,，\s|#]+/)
        .map((s) => s.trim())
        .filter(Boolean),
      lines: [],
    }
  }
  return { digits: parsePickTokens(trimmed, pool), lines: [] }
}

export function buildGroupContent(
  config: PlayConfig,
  picks: {
    digits?: string[]
    lines?: string[][]
    danshi?: string
  },
): string {
  if (isLonghuPlayConfig(config)) {
    return (picks.digits ?? []).join(',')
  }
  if (isWuxingQuweiDigitPlayConfig(config)) {
    return (picks.digits ?? []).join(',')
  }
  const textModes = ['daxiao', 'danshuang', 'dxds', 'teshu', 'longhubao', 'zhuangxian'] as const
  if (config.betMode && (textModes as readonly string[]).includes(config.betMode)) {
    if (config.inputMode === 'multiline' && config.segmentLen > 1) {
      const lines = picks.lines ?? []
      return Array.from({ length: config.segmentLen }, (_, i) => (lines[i] ?? []).join(',')).join('\n')
    }
    return (picks.digits ?? []).join(',')
  }
  if (config.inputMode === 'danshi') {
    const rawInput = (picks.danshi ?? '').trim() || (picks.digits ?? []).join(',')
    const parts = rawInput
      .split(/[\n,，\s]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    if (
      config.betMode === 'guoguan' ||
      config.betMode === 'tuotou' ||
      (config.betMode ?? '').endsWith('_dp') ||
      parts.some((s) => !/^\d+$/.test(s))
    ) {
      return (picks.danshi ?? '').trim()
    }
    if (isZuxuanDanshiConfig(config)) {
      return dedupeZuxuanDanshiTokens(rawInput, config.segmentLen).join(',')
    }
    return dedupeDanshiTokens(rawInput, config.segmentLen).join(',')
  }
  if (config.inputMode === 'lhc_num') {
    return [...new Set(parseLhcNumberTokens((picks.digits ?? []).join(',')))].join(',')
  }
  if (config.inputMode === 'lhc_zodiac' || config.inputMode === 'lhc_tail' || config.inputMode === 'lhc_attr') {
    // 生肖对碰：两个生肖合成 肖A|肖B
    if (isLhcSxDuipengConfig(config) || config.betMode === 'sx_dp') {
      const zs = parseLhcZodiacTokens((picks.digits ?? []).join(','))
      if (zs.length >= 2) return `${zs[0]}|${zs[1]}`
      return zs.join('|')
    }
    // 尾数对碰：两个尾数合成 尾A|尾B
    if (isLhcWsDuipengConfig(config) || config.betMode === 'ws_dp') {
      const ts = parseLhcTailTokens((picks.digits ?? []).join(','))
      if (ts.length >= 2) return `${ts[0]}|${ts[1]}`
      return ts.join('|')
    }
    // 生尾对碰：1 肖 + 1 尾 → 肖|尾
    if (isLhcSwDuipengConfig(config) || config.betMode === 'sw_dp') {
      const parts = parseLhcSwDuipengTokens((picks.digits ?? []).join(','))
      if (parts.length === 2) return `${parts[0]}|${parts[1]}`
      return (picks.digits ?? []).join('|')
    }
    return (picks.digits ?? []).join(',')
  }
  const pool = poolFromConfig(config)
  if (config.inputMode === 'multiline') {
    const lines = picks.lines ?? []
    return lines
      .map((line) => {
        const valid = pool
          ? line.filter((d) => parsePoolTokens(d, pool.min, pool.max).length > 0 || /^\d{1,2}$/.test(d))
          : line.filter((d) => /^[0-9]$/.test(d))
        return [...new Set(valid)].join(',')
      })
      .join('\n')
  }
  const digits = picks.digits ?? []
  if (pool) {
    return [...new Set(digits.filter((d) => parsePoolTokens(d, pool.min, pool.max).length > 0 || /^\d{1,2}$/.test(d)))].join(',')
  }
  return [...new Set(digits.filter((d) => /^[0-9]$/.test(d)))].join(',')
}

export function countBetUnits(config: PlayConfig, groupContent: string): number {
  const content = groupContent.trim()
  if (!content) return 0

  // 任选非直选复式：须先剥位再计注（和值/号池勿误吃「万,千\n…」）
  if (isRenxuanNeedsPositionConfig(config)) {
    return countRenxuanNeedsPositionUnits(config, content)
  }

  // 双区组选（12/4/五星60·20·10·5）；任选剥位后内层 bare 也会走到此
  if (isZuDualPlayConfig(config)) {
    const n = countZuDualBetUnits(config, content)
    return n > 0 ? applySegmentBetMultiplier(config, n) : 0
  }

  if (config.betMode === 'hezhi' || (config.playTemplate === 'pc28_std' && config.playMethodLabel?.trim() === '和值')) {
    const pool = poolFromConfig(config) ?? { min: 0, max: 27 }
    const tokens = parsePickTokens(content, pool)
    if (!tokens.length) return content ? 1 : 0
    // PC28 / K3 / PK10：选几个和值即几注
    if (
      config.playTemplate === 'pc28_std' ||
      config.playTemplate === 'k3_std' ||
      config.playTemplate === 'pk10_std'
    ) {
      return tokens.length
    }
    // SSC：按位组合数求和
    const segLen = inferHezhiSegmentLen(config)
    const zuxuan = (config.playMethodLabel ?? '').includes('组选')
    let total = 0
    for (const t of tokens) {
      const sum = Number(t)
      if (!Number.isFinite(sum)) continue
      total += zuxuan
        ? countZuxuanSumCombinations(sum, segLen)
        : countOrderedSumCombinations(sum, segLen)
    }
    return applySegmentBetMultiplier(config, total || tokens.length)
  }

  if (config.betMode === 'kuadu') {
    const pool = poolFromConfig(config) ?? { min: 0, max: 9 }
    const tokens = parsePickTokens(content, pool)
    if (!tokens.length) return 0
    const segLen = inferHezhiSegmentLen(config)
    let total = 0
    for (const t of tokens) {
      const span = Number(t)
      if (!Number.isFinite(span)) continue
      total += countOrderedSpanCombinations(span, segLen)
    }
    return applySegmentBetMultiplier(config, total || tokens.length)
  }

  if (config.betMode === 'weishu' || config.betMode === 'baodan') {
    const pool = poolFromConfig(config) ?? { min: 0, max: 9 }
    const tokens = parsePickTokens(content, pool)
    if (config.betMode === 'baodan') {
      // 三星包胆约 54 注/胆；二星 9 注
      const n = tokens.length
      if (!n) return 0
      const segLen = inferHezhiSegmentLen(config)
      const per = segLen === 2 ? 9 : 54
      return applySegmentBetMultiplier(config, n * per)
    }
    // 和值尾数：选几个尾数即几注，再×区位（前中后三 9×3=27）
    return applySegmentBetMultiplier(config, tokens.length)
  }

  // 不定位：一码=选几个号几注（最多2）；二码/三码=C(n,k)（对齐第三方 / guajibet）
  // 五星二码/三码：第三方要求至少 4 个号
  if (isBudingweiPlayConfig(config)) {
    const pool = poolFromConfig(config) ?? { min: 0, max: 9 }
    const tokens = [...new Set(parsePickTokens(content, pool))]
    const need = inferBudingweiNeed(config)
    if (isWuxingBudingweiMulti(config) && tokens.length < 4) return 0
    if (need <= 1) {
      if (!tokens.length) return 0
      return Math.min(tokens.length, 2)
    }
    if (tokens.length < need) return 0
    return comboCount(tokens.length, need)
  }

  if (isLonghuPlayConfig(config)) {
    return parseGroupPicks(config, content).digits.length
  }

  // 五星趣味：选几个 0–9 计几注
  if (isWuxingQuweiDigitPlayConfig(config)) {
    return applySegmentBetMultiplier(config, parseWuxingQuweiDigits(content).length)
  }
  // 前二/后二/前三/后三大小单双：按位各 1 选项 → 1 注（勿按换行扁平计 2）
  if (isPerPosDxdsPlayConfig(config)) {
    const lines = splitGroupLines(content)
    const allowed = ['大', '小', '单', '双']
    for (let i = 0; i < config.segmentLen; i++) {
      if (parseTextPickTokens(lines[i] ?? '', allowed).length !== 1) return 0
    }
    return 1
  }

  // 五星和值单双/大小、哈希尾数单双/大小：仅 1 选项 → 1 注
  if (isWuxingSumDxdsPlayConfig(config)) {
    const allowed =
      config.betMode === 'daxiao' || /和值大小|尾数大小/.test(config.playMethodLabel ?? '')
        ? ['大', '小']
        : ['单', '双']
    const n = parseTextPickTokens(content, allowed).length
    return n === 1 ? 1 : 0
  }

  // 特殊号 / 大小单双等文字选项：选几个计几注（对齐第三方）
  const textBetModes = ['daxiao', 'danshuang', 'dxds', 'teshu', 'longhubao', 'zhuangxian'] as const
  if (config.betMode && (textBetModes as readonly string[]).includes(config.betMode)) {
    const picks = parseGroupPicks(config, content).digits
    if (picks.length > 0) {
      return applySegmentBetMultiplier(config, picks.length)
    }
    const raw = content
      .split(/[\s,，\n]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    return applySegmentBetMultiplier(config, raw.length)
  }

  // 混合组选：排除豹子，按组选形态去重（对齐第三方）
  if (config.betMode === 'hunhe') {
    const digitLen = hunheDigitLenFromConfig(config)
    return applySegmentBetMultiplier(config, countHunheZuxuanUnits(content, digitLen))
  }

  // SSC 任选直选复式：按 C(5,n) 位组合计注（对齐后端 evaluateRenxuanZhixuan / 第三方）
  if (isSscRenxuanConfig(config) && isRenxuanZhixuanFushi(config)) {
    const pickN = renPickCountFromConfig(config)
    const lines = splitGroupLinesPad(content, 5)
    const units = countRenxuanZhixuanUnits(lines, pickN, poolFromConfig(config))
    return applySegmentBetMultiplier(config, units)
  }

  if (config.inputMode === 'danshi' && isLhcDanshiBetMode(config.betMode ?? '')) {
    return countLhcDanshiUnits(config, content)
  }

  if (config.inputMode === 'lhc_num') {
    if (isLhcTemaPlayConfig(config)) {
      return parseLhcTemaContentTokens(content).length
    }
    const pool = parseLhcNumberTokens(content)
    if (!pool.length) return 0
    const betMode = config.betMode ?? ''
    const subId = config.catalogSubId ?? config.subPlayId
    const min = lhcMinPickCount(betMode, subId)
    if (betMode === 'fushi' || betMode === 'buzhong' || betMode === 'xuanyi') {
      return comboCount(pool.length, min)
    }
    if (betMode === 'tuotou') {
      if (content.includes('|')) {
        const [dan, tuo] = content.split('|')
        const d = parseLhcNumberTokens(dan ?? '').length
        const t = parseLhcNumberTokens(tuo ?? '').length
        return d * comboCount(t, Math.max(min - 1, 1))
      }
      // 二全中拖头扁选：首号为胆，其余为拖
      const uniq = [...new Set(pool)]
      if (uniq.length >= 2) {
        return comboCount(uniq.length - 1, Math.max(min - 1, 1))
      }
      return 0
    }
    return pool.length
  }
  if (config.inputMode === 'lhc_zodiac' || config.inputMode === 'lhc_tail' || config.inputMode === 'lhc_attr') {
    if (isLhcSxDuipengConfig(config) || config.betMode === 'sx_dp') {
      return countLhcDanshiUnits(config, content)
    }
    if (isLhcWsDuipengConfig(config) || config.betMode === 'ws_dp') {
      return countLhcDanshiUnits(config, content)
    }
    if (isLhcSwDuipengConfig(config) || config.betMode === 'sw_dp') {
      return countLhcDanshiUnits(config, content)
    }
    const parts = content.split(/[,，]/).map((s) => s.trim()).filter(Boolean)
    return parts.length || 0
  }

  if (isSscDanshiLikeConfig(config)) {
    // 组选单式：整注形态去重，或单码号池 C(n,k)（1,2,3 → 3）
    if (isZuxuanDanshiConfig(config)) {
      const n = normalizeZuxuanDanshiContent(content, config.segmentLen)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean).length
      return applySegmentBetMultiplier(config, n)
    }
    // 冷热出号按位号池：按直选复式位积计注（万4,5×千3,5×百2,5 = 8）
    const seg = config.segmentLen > 0 ? config.segmentLen : 0
    if (seg > 1 && isZhixuanPositionPoolContent(content, seg)) {
      return applySegmentBetMultiplier(config, countZhixuanPositionPoolUnits(content, seg))
    }
    // 直选单式：相同号码重复录入只计 1 注（如 12,13,14,15,12 → 4；12,12,12 → 1）；
    // 前中后三/前后二三四等跨段玩法需按段倍乘（前中后三×3：111,234 → 2×3=6，对齐 v6 第三方）
    return applySegmentBetMultiplier(config, dedupeDanshiTokens(content, config.segmentLen).length) || 0
  }

  // 直选组合：按位乘积 × 段长（三星×3，对齐第三方「组合」）
  // 支持多行按位，或单行逗号按位（前后四如 `12,2,3,45`）
  if (
    config.inputMode === 'multiline' &&
    config.segmentLen > 1 &&
    (config.betMode === 'zuhe' ||
      config.subPlayId === 'zuhe' ||
      /(^|[^组选])组合/.test(config.playMethodLabel ?? '') ||
      (config.playMethodLabel ?? '').endsWith('组合') ||
      (config.playMethodLabel ?? '').includes('直选组合'))
  ) {
    const parts = splitZhixuanPositionParts(content, config.segmentLen)
    if (!parts) return 0
    let units = 1
    for (let i = 0; i < config.segmentLen; i++) {
      const n = [...new Set(parsePickTokens(parts[i] ?? ''))].length
      if (!n) return 0
      units *= n
    }
    return applySegmentBetMultiplier(config, units * config.segmentLen)
  }

  // 直选复式按位乘积（subPlayId 可能是数字目录 id，不能只认 zhixuan_fs）
  if (
    config.inputMode === 'multiline' &&
    config.segmentLen > 1 &&
    (isZhixuanFushiPlayConfig(config) ||
      (config.betMode === 'fushi' && !/组选/.test(`${config.playMethodLabel ?? ''} ${config.subPlayId}`)))
  ) {
    // 保留空位：`1,2,3\n\n` 不得压成单行号池；任一位无号即 0 注
    const lines = splitGroupLinesPad(content, config.segmentLen).slice(0, config.segmentLen)
    if (isZhixuanFushiBaoziLines(lines, config.segmentLen)) return 0
    let units = 1
    for (let i = 0; i < config.segmentLen; i++) {
      const n = [...new Set(parsePickTokens(lines[i] ?? ''))].length
      if (!n) return 0
      units *= n
    }
    return applySegmentBetMultiplier(config, units)
  }

  if (config.betMode === 'dxds' && config.inputMode === 'multiline' && config.segmentLen > 1) {
    const lines = splitGroupLines(content)
    const allowed = ['大', '小', '单', '双']
    let units = 1
    for (let i = 0; i < config.segmentLen; i++) {
      const n = parseTextPickTokens(lines[i] ?? '', allowed).length
      if (!n) return 0
      units *= n
    }
    return units
  }

  if (isDingweiMultilineConfig(config)) {
    // 保留前导空位：",,12,," / "\n\n1,2\n\n" 不得压成首位
    const lines = dingweiPositionLines(content, config.segmentLen)
    const poolCfg = poolFromConfig(config)
    let total = 0
    for (let i = 0; i < config.segmentLen; i++) {
      total += parsePickTokens(lines[i] ?? '', poolCfg).length
    }
    return total
  }

  const poolCfg = poolFromConfig(config)
  if (config.playTypeId === 'renxuan_fs' || config.playTypeId === 'renxuan_ds') {
    if (config.betMode === 'danshi' || (config.catalogSubId ?? '').endsWith('_ds')) {
      const lines = splitGroupLines(content)
      return lines.filter((l) => parsePickTokens(l, poolCfg).length > 0).length || (content ? 1 : 0)
    }
    const nm = syxwRenxuanNM(config.catalogSubId ?? config.subPlayId)
    if (nm) {
      const picks = parsePickTokens(content, poolCfg)
      if (picks.length < nm.pickN) return 0
      let units = 1
      for (let i = 0; i < nm.pickN; i++) units = (units * (picks.length - i)) / (i + 1)
      return Math.round(units)
    }
  }
  const pool = parsePickTokens(content, poolCfg)
  if (!pool.length) {
    // 和值/跨度等特殊玩法：有内容即计 1 注
    if (!config.subPlayId) return applySegmentBetMultiplier(config, 1)
    return 0
  }

  if (config.subPlayId === 'zhixuan_fs' && config.segmentLen > 1) {
    // 单码号池扩成各位相同 → 豹子，第三方计 0
    if (new Set(pool).size === 1) return 0
    return applySegmentBetMultiplier(config, pool.length ** config.segmentLen)
  }

  // 组选24/12/4：对齐 guajibet countZuGroupBetNums（号池 UI segmentLen=1）
  {
    const bm = (config.betMode ?? '').trim()
    const zuxuanN = `${config.betMode ?? ''} ${config.subPlayId ?? ''} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''}`
    const n = new Set(pool).size
    if (bm === 'zu24' || /组选24|zu24/i.test(zuxuanN)) {
      return n < 4 ? 0 : applySegmentBetMultiplier(config, comboCount(n, 4))
    }
    if (bm === 'zu120' || /组选120|zu120/i.test(zuxuanN)) {
      return n < 5 ? 0 : applySegmentBetMultiplier(config, comboCount(n, 5))
    }
    if (bm === 'zu12' || (/组选12|zu12/i.test(zuxuanN) && !/组选120|zu120/i.test(zuxuanN))) {
      // 双区「二重,单号」：C(m,1)×C(n,2)；扁选号池不再按 C(n,2)*2 估算
      const dual = countZu12BetUnits(content)
      return dual > 0 ? applySegmentBetMultiplier(config, dual) : 0
    }
    if (bm === 'zu4' || (/组选4|zu4/i.test(zuxuanN) && !/组选24|zu24|组选12|zu12/i.test(zuxuanN))) {
      const dual = countZu4BetUnits(content)
      if (dual > 0) return applySegmentBetMultiplier(config, dual)
      return 0
    }
    // 四星/任四组选6：C(n,2)（须在三星组六 C(n,3) 之前）
    if (isSixingZu6PlayConfig(config)) {
      if (n < 2) return 0
      return applySegmentBetMultiplier(config, (n * (n - 1)) / 2)
    }
  }

  // 组选号池：二星 C(n,2)；三星组三/组六/通用组选复式（对齐 guajibet countZuxuanFushiBetNums）
  // 注意：号池 UI 会把 segmentLen 置 1，计注不能再依赖 segmentLen===3。
  const zuxuanText = `${config.betMode ?? ''} ${config.subPlayId} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''}`
  const isZuFsLike =
    config.subPlayId === 'zuxuan_fs' ||
    config.betMode === 'zu3' ||
    config.betMode === 'zu6' ||
    config.betMode === 'zuxuan_fs' ||
    /组三|组六|组选复式|组选6/.test(zuxuanText)
  if (isZuFsLike) {
    const n = new Set(pool).size
    const starLen = zuxuanStarLen(config)
    // 四星/任四组选6：C(n,2)
    if (isSixingZu6PlayConfig(config)) {
      if (n < 2) return 0
      return applySegmentBetMultiplier(config, (n * (n - 1)) / 2)
    }
    const isZu6Only =
      config.betMode === 'zu6' ||
      (/组六|zu6/i.test(zuxuanText) && !/组选6|组选60|组选120|zu60|zu120/i.test(zuxuanText))
    const isZu3Only =
      !isZu6Only &&
      (config.betMode === 'zu3' || /组三|zu3/i.test(zuxuanText))
    if (starLen === 2) {
      if (n < 2) return 0
      return applySegmentBetMultiplier(config, (n * (n - 1)) / 2)
    }
    if (isZu6Only) {
      if (n < 3) return 0
      return applySegmentBetMultiplier(config, (n * (n - 1) * (n - 2)) / 6)
    }
    if (isZu3Only) {
      // 组三：n*(n-1)；前中后三再 ×3 → 10 码 = 90×3 = 270（对齐第三方）
      if (n < 2) return 0
      return applySegmentBetMultiplier(config, n * (n - 1))
    }
    // 通用组选复式：组三注 + 组六注
    if (n < 2) return 0
    if (n < 3) return applySegmentBetMultiplier(config, n * (n - 1))
    return applySegmentBetMultiplier(config, n * (n - 1) + (n * (n - 1) * (n - 2)) / 6)
  }

  return applySegmentBetMultiplier(config, pool.length || 1)
}

/**
 * 组选号池最少选号数（对齐后端 zuxuanPoolMinPick）。
 * 组三 ≥2；三星组六 ≥3；四星/任四组选6 ≥2；组选24 ≥4；组选12 双区另验。
 */
export function zuxuanPoolMinPick(config: PlayConfig): number | null {
  const text = `${config.betMode ?? ''} ${config.subPlayId ?? ''} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''}`
  // 包胆每组仅 1 胆，勿套组三/组六下限
  if (config.betMode === 'baodan' || /包胆|baodan|_bd\b/i.test(text)) {
    return null
  }
  // 组选120 须先于组选12（避免「组选12」前缀 / zu12⊂zu120 误匹配）
  if (config.betMode === 'zu120' || /组选120|zu120/i.test(text)) {
    return 5
  }
  // 双区组选（12/4/五星60·20·10·5）：见 validateZuDualContent，勿套扁选下限
  if (isZuDualPlayConfig(config)) {
    return null
  }
  if (config.betMode === 'zu24' || /组选24|zu24/i.test(text)) {
    return 4
  }
  // 四星/任四组选6：C(n,2) 至少 2 码（区别于三星组六 ≥3）
  if (isSixingZu6PlayConfig(config)) {
    return 2
  }
  if (
    config.betMode === 'zu6' ||
    (/组六|zu6/i.test(text) && !/组选6|组选60|组选120|zu60|zu120/i.test(text))
  ) {
    return 3
  }
  if (
    config.betMode === 'zu3' ||
    (/组三|zu3/i.test(text) && !/组选3|组选30|zu30/i.test(text))
  ) {
    return 2
  }
  return null
}

/** 组选号池不足时的保存提示 */
export function zuxuanPoolMinPickMessage(config: PlayConfig): string {
  const min = zuxuanPoolMinPick(config)
  if (min == null) return '选号无效'
  const text = `${config.betMode ?? ''} ${config.playMethodLabel ?? ''} ${config.subPlayId ?? ''} ${config.catalogSubId ?? ''}`
  if (config.betMode === 'zu24' || /组选24|zu24/i.test(text)) {
    return `组选24至少选择 ${min} 个号码`
  }
  if (config.betMode === 'zu120' || /组选120|zu120/i.test(text)) {
    return `组选120至少选择 ${min} 个号码`
  }
  if (isSixingZu6PlayConfig(config)) {
    return `组选6至少选择 ${min} 个号码`
  }
  if (
    config.betMode === 'zu6' ||
    (/组六|zu6/i.test(text) && !/组选6|组选60|组选120|zu60|zu120/i.test(text))
  ) {
    return `组六至少选择 ${min} 个号码`
  }
  return `组三至少选择 ${min} 个号码`
}

/**
 * 四星/任四「组选6」：号池 C(n,2)、至少 2 码。
 * 区别于三星「组六」复式 C(n,3)、至少 3 码。
 * 注意：目录短名常为「组六」，须结合任四/四星区位或规则 id 识别，勿仅靠「组选6」文案。
 */
export function isSixingZu6PlayConfig(config: PlayConfig): boolean {
  if (isZu6DanshiConfig(config)) return false
  // 直选组合（前后四 rule136 等）绝非组选6
  if (isZhixuanZuhePlayConfig(config)) return false
  const method = String(config.playMethodLabel ?? '')
  const sidRaw = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
  // catalog 可能是「组选6 145」合并串，取出数字 id
  const sid =
    /^\d+$/.test(sidRaw)
      ? sidRaw
      : (sidRaw.match(/(?:^|[\s|,/])(132|139|145)(?:$|[\s|,/])/)?.[1] ?? sidRaw)
  const text = `${method} ${sidRaw} ${config.betMode ?? ''}`
  if (/组选60|组选120|zu60|zu120/i.test(text)) return false
  // 文案明确「组选6」
  if (/组选6/i.test(method)) return true
  // 规则 id：四星组选6=132；前后四组选6=139；任四组选6=145
  // （136 是前后四直选组合，勿列入）
  if (['132', '139', '145'].includes(sid)) return true
  const bm = (config.betMode ?? '').trim()
  const isZu6 =
    bm === 'zu6' ||
    (/zu6/i.test(text) && !/组六/i.test(method)) ||
    // 目录短名「组六」在任四/四星语境下即组选6
    (/组六/i.test(method) && !/组六单式/i.test(method))
  if (!isZu6 && !/组选6|zu6/i.test(text)) return false
  // 任四选位 / 四星区位
  const renK = config.renPositionCount ?? renPickCountFromConfig(config)
  if (renK >= 4) return true
  const loc = `${config.playTypeLabel ?? ''} ${config.playTypeId ?? ''} ${config.guajiGroup ?? ''}`
  if (/四星|前后四|qian4|hou4|g013|g014/i.test(loc)) return true
  // 任选 + 组六/zu6：任四规则段（勿把任三组六 80–88 算进来）
  if (/任选|renxuan|g011/i.test(loc) && (/组六|组选6|zu6/i.test(text) || bm === 'zu6')) {
    const n = Number.parseInt(sid, 10)
    if (Number.isFinite(n) && n >= 141 && n <= 145) return true
  }
  return false
}

/** 组选12：双区「二重号池,单号池」（如 12,3234） */
export function isZu12PlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu12') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选12|zu12/i.test(text) && !/组选120|zu120/i.test(text)
}

/** 五星趣味（一帆风顺/好事成双/三星报喜/四季发财）：0–9 数字池，非豹子/对子/顺子 */
export function isWuxingQuweiDigitPlayConfig(config: PlayConfig): boolean {
  const text = [
    config.playMethodLabel,
    config.playTypeLabel,
    config.subPlayId,
    config.catalogSubId,
    config.betMode,
  ]
    .map((s) => String(s ?? '').trim())
    .join(' ')
  if (/一帆风顺|好事成双|三星报喜|四季发财/i.test(text)) return true
  if (/yifan|haoshi|sanxing|siji|wuxing_yifan|wuxing_haoshi|wuxing_sanxing|wuxing_siji/i.test(text)) {
    return true
  }
  const sid = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
  return ['162', '163', '164', '165'].includes(sid)
}

/** 一帆风顺第三方最多 2 码；好事成双/三星报喜/四季发财可至 10 */
export function wuxingQuweiMaxPicks(config: PlayConfig): number {
  const text = [
    config.playMethodLabel,
    config.subPlayId,
    config.catalogSubId,
  ]
    .map((s) => String(s ?? ''))
    .join(' ')
  if (/一帆风顺|yifan|wuxing_yifan|\b162\b/i.test(text)) return 2
  return 10
}

export function wuxingQuweiFormatHint(config: PlayConfig): string {
  const label = String(config.playMethodLabel ?? '')
  const name = /四季发财/.test(label)
    ? '四季发财'
    : /三星报喜/.test(label)
      ? '三星报喜'
      : /好事成双/.test(label)
        ? '好事成双'
        : '一帆风顺'
  const max = wuxingQuweiMaxPicks(config)
  const example = max <= 2 ? '0,3' : '0,3,9'
  if (max <= 2) {
    return `${name}：输入 1–2 个 0–9 号码，每个数字用逗号分隔（如 ${example}）`
  }
  return `${name}：输入 0–9，每个数字用逗号分隔（如 ${example}）`
}

/** 趣味数字池：保序去重，连写拆位 */
export function parseWuxingQuweiDigits(raw: string): string[] {
  const text = String(raw ?? '')
    .replace(/，/g, ',')
    .trim()
  if (!text) return []
  const parts = text.split(/[,，\s|]+/).map((s) => s.trim()).filter(Boolean)
  const out: string[] = []
  const seen = new Set<string>()
  for (const p of parts) {
    if (/^\d+$/.test(p) && p.length > 1) {
      for (const ch of p) {
        if (ch >= '0' && ch <= '9' && !seen.has(ch)) {
          seen.add(ch)
          out.push(ch)
        }
      }
      continue
    }
    if (p.length === 1 && p >= '0' && p <= '9' && !seen.has(p)) {
      seen.add(p)
      out.push(p)
    }
  }
  return out
}

/** 组选4：双区「三重号池,单号池」（如 1,2 / 12,34） */
export function isZu4PlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu4') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选4|zu4/i.test(text) && !/组选24|zu24|组选12|zu12/i.test(text)
}

/** 五星组选60：双区「二重号,单号」 */
export function isZu60PlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu60') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选60|zu60/i.test(text)
}

/** 五星组选30：双区「二重号,单号」（二重≥3、单号≥1） */
export function isZu30PlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu30') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选30|zu30/i.test(text)
}

/** 五星组选20：双区「三重号,单号」，两区个数须相同 */
export function isZu20PlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu20') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  // 勿误伤组选120（文案含 zu120）
  return /组选20|zu20/i.test(text) && !/组选120|zu120/i.test(text)
}

/** 五星组选10：双区「三重号,二重号」 */
export function isZu10PlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu10') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选10|zu10/i.test(text)
}

/** 五星组选5：双区「四重号,单号」 */
export function isZu5PlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu5') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选5|zu5/i.test(text) && !/组选50|组选5\d/i.test(text)
}

export type ZuDualKind = 'zu12' | 'zu4' | 'zu60' | 'zu30' | 'zu20' | 'zu10' | 'zu5'

/** 双区组选元信息（头区/尾区标签与下限） */
export type ZuDualMeta = {
  kind: ZuDualKind
  headLabel: string
  tailLabel: string
  minHead: number
  minTail: number
  /** 两区选号个数必须相同（组选20） */
  equalCounts: boolean
  example: string
}

export function zuDualKindOf(config: PlayConfig): ZuDualKind | null {
  if (isZu60PlayConfig(config)) return 'zu60'
  if (isZu30PlayConfig(config)) return 'zu30'
  if (isZu20PlayConfig(config)) return 'zu20'
  if (isZu10PlayConfig(config)) return 'zu10'
  if (isZu5PlayConfig(config)) return 'zu5'
  if (isZu12PlayConfig(config)) return 'zu12'
  if (isZu4PlayConfig(config)) return 'zu4'
  return null
}

export function zuDualMetaOf(config: PlayConfig): ZuDualMeta | null {
  const kind = zuDualKindOf(config)
  if (!kind) return null
  switch (kind) {
    case 'zu12':
      return {
        kind,
        headLabel: '二重号',
        tailLabel: '单号',
        minHead: 1,
        minTail: 2,
        equalCounts: false,
        example: '12,3234',
      }
    case 'zu4':
      return {
        kind,
        headLabel: '三重号',
        tailLabel: '单号',
        minHead: 1,
        minTail: 1,
        equalCounts: false,
        example: '1,2',
      }
    case 'zu60':
      return {
        kind,
        headLabel: '二重号',
        tailLabel: '单号',
        minHead: 1,
        minTail: 3,
        equalCounts: false,
        example: '1,234',
      }
    case 'zu30':
      return {
        kind,
        headLabel: '二重号',
        tailLabel: '单号',
        minHead: 3,
        minTail: 1,
        equalCounts: false,
        example: '123,1',
      }
    case 'zu20':
      // 三重号与单号个数须相同，各≥2（对每个三重 t：C(|单号\{t}|, 2)）
      return {
        kind,
        headLabel: '三重号',
        tailLabel: '单号',
        minHead: 2,
        minTail: 2,
        equalCounts: true,
        example: '12,34',
      }
    case 'zu10':
      return {
        kind,
        headLabel: '三重号',
        tailLabel: '二重号',
        minHead: 1,
        minTail: 1,
        equalCounts: false,
        example: '1,2',
      }
    case 'zu5':
      return {
        kind,
        headLabel: '四重号',
        tailLabel: '单号',
        minHead: 1,
        minTail: 1,
        equalCounts: false,
        example: '1,2',
      }
  }
}

/** 组选12/4/五星组选60·30·20·10·5 双区玩法 */
export function isZuDualPlayConfig(config: PlayConfig): boolean {
  return zuDualKindOf(config) != null
}

/** 从连写/逗号串提取 0–9 数字（去重保序） */
export function uniqueDigitsFromRun(raw: string): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  for (const ch of String(raw ?? '')) {
    if (ch < '0' || ch > '9' || seen.has(ch)) continue
    seen.add(ch)
    out.push(ch)
  }
  return out
}

export type Zu12Zones = {
  doubles: string[]
  /** 单号区区内去重（允许与二重区有交集；出站/落库原样保留） */
  singles: string[]
  normalized: string
}

/**
 * 解析组选12 双区内容。须恰好一段逗号分隔：二重号,单号。
 * 二重 ≥1、单号区 ≥2（各位 0–9，区内去重保序；跨区重叠码保留）。
 */
export function parseZu12Zones(raw: string): Zu12Zones | null {
  const text = String(raw ?? '')
    .replace(/，/g, ',')
    .trim()
  if (!text) return null
  const parts = text.split(',')
  if (parts.length !== 2) return null
  const doubles = uniqueDigitsFromRun(parts[0] ?? '')
  const singles = uniqueDigitsFromRun(parts[1] ?? '')
  if (doubles.length < 1 || singles.length < 2) return null
  return {
    doubles,
    singles,
    // 区内去重即可；跨区重叠原样保留（如 23,123），交给第三方按双区计注
    normalized: `${doubles.join('')},${singles.join('')}`,
  }
}

export const ZU12_FORMAT_MSG =
  '组选12：从0-9中输入1个及以上二重号码、2个及以上单号，两区用逗号分隔，如：12,3234'

export const ZU12_OVERLAP_MSG =
  '组选12：每个二重号须能与单号区凑成至少 1 注（选该二重时单号区去掉该码后仍≥2；如 23,123 计 2 注，1,12 为 0 注）'

/** C(n,2)；n<2 → 0（勿用 comboCount：其对 n<k 会回落成 n） */
function combo2(n: number): number {
  if (n < 2) return 0
  return (n * (n - 1)) / 2
}

/**
 * 组选12 注数：对每个二重 d，C(|单号\{d}|, 2) 求和。
 * 跨区重叠时不整区剔除；仅在组合该二重时排除同码（23,123→2；2,123→1；1,12→0）。
 */
export function countZu12BetUnits(raw: string): number {
  const zones = parseZu12Zones(raw)
  if (!zones) return 0
  let total = 0
  for (const d of zones.doubles) {
    total += combo2(zones.singles.filter((s) => s !== d).length)
  }
  return total
}

/**
 * 组选12 随机双区内容。
 * @param doublesCount 二重号个数（1–10）
 * @param singlesCount 单号个数（2–10）；缺省 2
 * 保证总注数 ≥1（必要时重抽；跨区可重叠）。
 */
export function randomZu12DualContent(doublesCount = 1, singlesCount = 2): string {
  const shuffle = <T,>(arr: T[]): T[] => {
    const a = [...arr]
    for (let i = a.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      ;[a[i], a[j]] = [a[j]!, a[i]!]
    }
    return a
  }
  const fallback = '12,34'
  const wantD = Math.min(10, Math.max(1, Math.trunc(doublesCount) || 1))
  const wantS = Math.min(10, Math.max(2, Math.trunc(singlesCount) || 2))
  for (let guard = 0; guard < 80; guard++) {
    const pool = shuffle(Array.from({ length: 10 }, (_, i) => String(i)))
    // 先取二重；单号优先用剩余码，不够再与二重重叠
    const doubles = pool.slice(0, wantD)
    const outside = pool.slice(wantD)
    const singles: string[] = []
    for (const d of outside) {
      if (singles.length >= wantS) break
      singles.push(d)
    }
    for (const d of shuffle([...doubles])) {
      if (singles.length >= wantS) break
      if (!singles.includes(d)) singles.push(d)
    }
    if (doubles.length < wantD || singles.length < wantS) continue
    const raw = `${doubles.join('')},${singles.join('')}`
    const zones = parseZu12Zones(raw)
    if (zones && countZu12BetUnits(zones.normalized) > 0) return zones.normalized
  }
  return fallback
}

export function validateZu12Content(raw: string): GroupContentValidation {
  const zones = parseZu12Zones(raw)
  if (!zones) return { ok: false, message: ZU12_FORMAT_MSG }
  const betUnits = countZu12BetUnits(raw)
  if (betUnits <= 0) {
    const overlapped = zones.singles.some((d) => zones.doubles.includes(d))
    return { ok: false, message: overlapped ? ZU12_OVERLAP_MSG : ZU12_FORMAT_MSG }
  }
  return { ok: true, normalized: zones.normalized, betUnits }
}

export type Zu4Zones = {
  triples: string[]
  singles: string[]
  normalized: string
}

/**
 * 解析组选4 双区内容。须恰好一段逗号分隔：三重号,单号。
 * 三重 ≥1、单号区 ≥1（各位 0–9，区内去重保序；跨区重叠码保留）。
 */
export function parseZu4Zones(raw: string): Zu4Zones | null {
  const text = String(raw ?? '')
    .replace(/，/g, ',')
    .trim()
  if (!text) return null
  const parts = text.split(',')
  if (parts.length !== 2) return null
  const triples = uniqueDigitsFromRun(parts[0] ?? '')
  const singles = uniqueDigitsFromRun(parts[1] ?? '')
  if (triples.length < 1 || singles.length < 1) return null
  return {
    triples,
    singles,
    normalized: `${triples.join('')},${singles.join('')}`,
  }
}

export const ZU4_FORMAT_MSG =
  '组选4：从0-9中输入1个及以上三重号码、1个及以上单号，两区用逗号分隔，如：1,2'

export const ZU4_OVERLAP_MSG =
  '组选4：每个三重号须能与单号区凑成至少 1 注（选该三重时单号区去掉该码后仍≥1；如 12,34 计 4 注，1,1 为 0 注）'

/**
 * 组选4 注数：对每个三重 t，统计单号 s 中 s≠t 的个数并求和。
 */
export function countZu4BetUnits(raw: string): number {
  const zones = parseZu4Zones(raw)
  if (!zones) return 0
  let total = 0
  for (const t of zones.triples) {
    total += zones.singles.filter((s) => s !== t).length
  }
  return total
}

/**
 * 组选4 随机双区内容。
 * @param triplesCount 三重号个数（1–10）
 * @param singlesCount 单号个数（1–10）；缺省 1
 */
export function randomZu4DualContent(triplesCount = 1, singlesCount = 1): string {
  const shuffle = <T,>(arr: T[]): T[] => {
    const a = [...arr]
    for (let i = a.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      ;[a[i], a[j]] = [a[j]!, a[i]!]
    }
    return a
  }
  const fallback = '1,2'
  const wantT = Math.min(10, Math.max(1, Math.trunc(triplesCount) || 1))
  const wantS = Math.min(10, Math.max(1, Math.trunc(singlesCount) || 1))
  for (let guard = 0; guard < 80; guard++) {
    const pool = shuffle(Array.from({ length: 10 }, (_, i) => String(i)))
    const triples = pool.slice(0, wantT)
    const outside = pool.slice(wantT)
    const singles: string[] = []
    for (const d of outside) {
      if (singles.length >= wantS) break
      singles.push(d)
    }
    for (const d of shuffle([...triples])) {
      if (singles.length >= wantS) break
      if (!singles.includes(d)) singles.push(d)
    }
    if (triples.length < wantT || singles.length < wantS) continue
    const raw = `${triples.join('')},${singles.join('')}`
    const zones = parseZu4Zones(raw)
    if (zones && countZu4BetUnits(zones.normalized) > 0) return zones.normalized
  }
  return fallback
}

export function validateZu4Content(raw: string): GroupContentValidation {
  const zones = parseZu4Zones(raw)
  if (!zones) return { ok: false, message: ZU4_FORMAT_MSG }
  const betUnits = countZu4BetUnits(raw)
  if (betUnits <= 0) {
    const overlapped = zones.singles.some((d) => zones.triples.includes(d))
    return { ok: false, message: overlapped ? ZU4_OVERLAP_MSG : ZU4_FORMAT_MSG }
  }
  return { ok: true, normalized: zones.normalized, betUnits }
}

export type ZuDualZones = {
  head: string[]
  tail: string[]
  normalized: string
}

/** 通用双区解析：恰好一段逗号分隔，区内去重保序 */
export function parseZuDualZones(
  raw: string,
  minHead: number,
  minTail: number,
  equalCounts = false,
): ZuDualZones | null {
  const text = String(raw ?? '')
    .replace(/，/g, ',')
    .trim()
  if (!text) return null
  const parts = text.split(',')
  if (parts.length !== 2) return null
  const head = uniqueDigitsFromRun(parts[0] ?? '')
  const tail = uniqueDigitsFromRun(parts[1] ?? '')
  if (head.length < minHead || tail.length < minTail) return null
  if (equalCounts && head.length !== tail.length) return null
  return { head, tail, normalized: `${head.join('')},${tail.join('')}` }
}

/** C(n,3)；n<3 → 0 */
function combo3(n: number): number {
  if (n < 3) return 0
  return (n * (n - 1) * (n - 2)) / 6
}

export const ZU60_FORMAT_MSG =
  '组选60：从0-9中输入1个及以上二重号码、3个及以上单号，两区用逗号分隔，如：1,234'

export const ZU60_OVERLAP_MSG =
  '组选60：每个二重号须能与单号区凑成至少 1 注（选该二重时单号区去掉该码后仍≥3）'

/** 组选60 注数：对每个二重 d，C(|单号\{d}|, 3) 求和 */
export function countZu60BetUnits(raw: string): number {
  const zones = parseZuDualZones(raw, 1, 3)
  if (!zones) return 0
  let total = 0
  for (const d of zones.head) {
    total += combo3(zones.tail.filter((s) => s !== d).length)
  }
  return total
}

export function validateZu60Content(raw: string): GroupContentValidation {
  const zones = parseZuDualZones(raw, 1, 3)
  if (!zones) return { ok: false, message: ZU60_FORMAT_MSG }
  const betUnits = countZu60BetUnits(raw)
  if (betUnits <= 0) {
    const overlapped = zones.tail.some((d) => zones.head.includes(d))
    return { ok: false, message: overlapped ? ZU60_OVERLAP_MSG : ZU60_FORMAT_MSG }
  }
  return { ok: true, normalized: zones.normalized, betUnits }
}

export const ZU30_FORMAT_MSG =
  '组选30：从0-9中输入3个及以上二重号码、1个及以上单号，两区用逗号分隔，如：123,1'

export const ZU30_OVERLAP_MSG =
  '组选30：每组二重号须能与单号区凑成至少 1 注（选该对二重时单号区去掉这两码后仍≥1）'

/**
 * 组选30 注数：对每个二重对 (d1,d2)，计 |单号\{d1,d2}| 并求和。
 * 无跨区重叠时即 C(|二重|,2)×|单号|（如 123,45→6；1234,5→6）。
 */
export function countZu30BetUnits(raw: string): number {
  const zones = parseZuDualZones(raw, 3, 1)
  if (!zones) return 0
  let total = 0
  const head = zones.head
  for (let i = 0; i < head.length; i++) {
    for (let j = i + 1; j < head.length; j++) {
      const d1 = head[i]!
      const d2 = head[j]!
      total += zones.tail.filter((s) => s !== d1 && s !== d2).length
    }
  }
  return total
}

export function validateZu30Content(raw: string): GroupContentValidation {
  const zones = parseZuDualZones(raw, 3, 1)
  if (!zones) return { ok: false, message: ZU30_FORMAT_MSG }
  const betUnits = countZu30BetUnits(raw)
  if (betUnits <= 0) {
    const overlapped = zones.tail.some((d) => zones.head.includes(d))
    return { ok: false, message: overlapped ? ZU30_OVERLAP_MSG : ZU30_FORMAT_MSG }
  }
  return { ok: true, normalized: zones.normalized, betUnits }
}

export const ZU20_FORMAT_MSG =
  '组选20：三重号与单号个数须相同，至少各 2 个，两区用逗号分隔，如：12,34'

export const ZU20_OVERLAP_MSG =
  '组选20：每个三重号须能与单号区凑成至少 1 注（选该三重时单号区去掉该码后仍≥2）'

/** 组选20 注数：两区个数相同且各≥2；对每个三重 t，C(|单号\{t}|, 2) 求和 */
export function countZu20BetUnits(raw: string): number {
  const zones = parseZuDualZones(raw, 2, 2, true)
  if (!zones) return 0
  let total = 0
  for (const t of zones.head) {
    total += combo2(zones.tail.filter((s) => s !== t).length)
  }
  return total
}

export function validateZu20Content(raw: string): GroupContentValidation {
  const text = String(raw ?? '')
    .replace(/，/g, ',')
    .trim()
  const parts = text.split(',')
  if (parts.length === 2) {
    const head = uniqueDigitsFromRun(parts[0] ?? '')
    const tail = uniqueDigitsFromRun(parts[1] ?? '')
    if (head.length > 0 && tail.length > 0 && head.length !== tail.length) {
      return { ok: false, message: ZU20_FORMAT_MSG }
    }
  }
  const zones = parseZuDualZones(raw, 2, 2, true)
  if (!zones) return { ok: false, message: ZU20_FORMAT_MSG }
  const betUnits = countZu20BetUnits(zones.normalized)
  if (betUnits <= 0) {
    const overlapped = zones.tail.some((d) => zones.head.includes(d))
    return { ok: false, message: overlapped ? ZU20_OVERLAP_MSG : ZU20_FORMAT_MSG }
  }
  return { ok: true, normalized: zones.normalized, betUnits }
}

export const ZU10_FORMAT_MSG =
  '组选10：从0-9中输入1个及以上三重号码、1个及以上二重号码，两区用逗号分隔，如：1,2'

export const ZU10_OVERLAP_MSG =
  '组选10：每个三重号须能与二重号区凑成至少 1 注（选该三重时二重区去掉该码后仍≥1）'

/** 组选10 注数：对每个三重 t，统计二重 d 中 d≠t 的个数并求和 */
export function countZu10BetUnits(raw: string): number {
  const zones = parseZuDualZones(raw, 1, 1)
  if (!zones) return 0
  let total = 0
  for (const t of zones.head) {
    total += zones.tail.filter((d) => d !== t).length
  }
  return total
}

export function validateZu10Content(raw: string): GroupContentValidation {
  const zones = parseZuDualZones(raw, 1, 1)
  if (!zones) return { ok: false, message: ZU10_FORMAT_MSG }
  const betUnits = countZu10BetUnits(raw)
  if (betUnits <= 0) {
    const overlapped = zones.tail.some((d) => zones.head.includes(d))
    return { ok: false, message: overlapped ? ZU10_OVERLAP_MSG : ZU10_FORMAT_MSG }
  }
  return { ok: true, normalized: zones.normalized, betUnits }
}

export const ZU5_FORMAT_MSG =
  '组选5：从0-9中输入1个及以上四重号码、1个及以上单号，两区用逗号分隔，如：1,2'

export const ZU5_OVERLAP_MSG =
  '组选5：每个四重号须能与单号区凑成至少 1 注（选该四重时单号区去掉该码后仍≥1）'

/** 组选5 注数：同组选4（四重 × 单号） */
export function countZu5BetUnits(raw: string): number {
  return countZu10BetUnits(raw)
}

export function validateZu5Content(raw: string): GroupContentValidation {
  const zones = parseZuDualZones(raw, 1, 1)
  if (!zones) return { ok: false, message: ZU5_FORMAT_MSG }
  const betUnits = countZu5BetUnits(raw)
  if (betUnits <= 0) {
    const overlapped = zones.tail.some((d) => zones.head.includes(d))
    return { ok: false, message: overlapped ? ZU5_OVERLAP_MSG : ZU5_FORMAT_MSG }
  }
  return { ok: true, normalized: zones.normalized, betUnits }
}

/** 按双区元信息随机一注合法内容 */
export function randomZuDualContentForConfig(
  config: PlayConfig,
  headCount?: number,
  tailCount?: number,
): string {
  const meta = zuDualMetaOf(config)
  if (!meta) return '1,2'
  if (meta.kind === 'zu12') return randomZu12DualContent(headCount ?? 1, tailCount ?? 2)
  if (meta.kind === 'zu4') return randomZu4DualContent(headCount ?? 1, tailCount ?? 1)

  const shuffle = <T,>(arr: T[]): T[] => {
    const a = [...arr]
    for (let i = a.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      ;[a[i], a[j]] = [a[j]!, a[i]!]
    }
    return a
  }
  let wantH = Math.min(10, Math.max(meta.minHead, Math.trunc(headCount ?? meta.minHead) || meta.minHead))
  let wantT = Math.min(10, Math.max(meta.minTail, Math.trunc(tailCount ?? meta.minTail) || meta.minTail))
  if (meta.equalCounts) {
    const n = Math.max(wantH, wantT, meta.minHead, meta.minTail)
    wantH = n
    wantT = n
  }
  const countFn =
    meta.kind === 'zu60'
      ? countZu60BetUnits
      : meta.kind === 'zu30'
        ? countZu30BetUnits
        : meta.kind === 'zu20'
          ? countZu20BetUnits
          : meta.kind === 'zu10'
            ? countZu10BetUnits
            : countZu5BetUnits
  for (let guard = 0; guard < 80; guard++) {
    const pool = shuffle(Array.from({ length: 10 }, (_, i) => String(i)))
    const head = pool.slice(0, wantH)
    const outside = pool.slice(wantH)
    const tail: string[] = []
    for (const d of outside) {
      if (tail.length >= wantT) break
      tail.push(d)
    }
    for (const d of shuffle([...head])) {
      if (tail.length >= wantT) break
      if (!tail.includes(d)) tail.push(d)
    }
    if (head.length < wantH || tail.length < wantT) continue
    const raw = `${head.join('')},${tail.join('')}`
    if (countFn(raw) > 0) return raw
  }
  return meta.example
}

/** 校验任意双区组选（含五星 60/30/20/10/5） */
export function validateZuDualContent(config: PlayConfig, raw: string): GroupContentValidation {
  const kind = zuDualKindOf(config)
  if (kind === 'zu12') return validateZu12Content(raw)
  if (kind === 'zu4') return validateZu4Content(raw)
  if (kind === 'zu60') return validateZu60Content(raw)
  if (kind === 'zu30') return validateZu30Content(raw)
  if (kind === 'zu20') return validateZu20Content(raw)
  if (kind === 'zu10') return validateZu10Content(raw)
  if (kind === 'zu5') return validateZu5Content(raw)
  return { ok: false, message: '不支持的双区组选玩法' }
}

export function countZuDualBetUnits(config: PlayConfig, raw: string): number {
  const kind = zuDualKindOf(config)
  if (kind === 'zu12') return countZu12BetUnits(raw)
  if (kind === 'zu4') return countZu4BetUnits(raw)
  if (kind === 'zu60') return countZu60BetUnits(raw)
  if (kind === 'zu30') return countZu30BetUnits(raw)
  if (kind === 'zu20') return countZu20BetUnits(raw)
  if (kind === 'zu10') return countZu10BetUnits(raw)
  if (kind === 'zu5') return countZu5BetUnits(raw)
  return 0
}

export function zuDualFormatHint(config: PlayConfig): string {
  const meta = zuDualMetaOf(config)
  if (!meta) return ''
  if (meta.equalCounts) {
    return `从0-9中，${meta.headLabel}与${meta.tailLabel}个数须相同，至少各 ${meta.minHead} 个，两区用逗号分隔，如：${meta.example}`
  }
  return `从0-9中，输入${meta.minHead}个及以上的${meta.headLabel}，${meta.minTail}个及以上的${meta.tailLabel}，两个位置由逗号分隔，如：${meta.example}`
}

/** 组选星数：二星组选=2，组三/组六/三星组选复式=3（不受号池 UI 的 segmentLen=1 影响） */
function zuxuanStarLen(config: PlayConfig): number {
  if (config.segmentLen === 2) return 2
  if (config.segmentLen === 4) return 4
  // 任选组选：优先 renPositionCount / 文案任二三四
  const renK = config.renPositionCount ?? 0
  if (renK >= 2 && renK <= 5) return renK
  const renFromLabel = renPickCountFromConfig(config)
  if (
    isSscRenxuanConfig(config) ||
    /任[二三四]|ren[234]/i.test(`${config.playMethodLabel ?? ''}`)
  ) {
    return renFromLabel
  }
  const text = `${config.guajiGroup ?? ''} ${config.playTypeLabel ?? ''} ${config.playMethodLabel ?? ''} ${config.playTypeId ?? ''}`
  if (/前二|后二|g004|g005|g008/.test(text) && !/前中后|前后三|前后四|前三|中三|后三|g001|g002|g003|g007|g012/.test(text)) {
    return 2
  }
  if (config.betMode === 'zu3' || config.betMode === 'zu6') return 3
  if (config.segmentLen === 3) return 3
  if (/前三|中三|后三|前中后三|前后三|组三|组六/.test(text)) return 3
  return config.segmentLen > 1 ? config.segmentLen : 3
}

function applySegmentBetMultiplier(config: PlayConfig, units: number): number {
  if (units <= 0) return units
  let m = segmentBetMultiplier(config.guajiGroup ?? config.playTypeLabel ?? '')
  if (m <= 1) {
    const tid = String(config.playTypeId ?? '').trim().toLowerCase()
    if (tid === 'g007' || tid === 'qianzhonghou3') m = 3
    else if (
      tid === 'g008' ||
      tid === 'qianhou2' ||
      tid === 'g012' ||
      tid === 'qianhou3' ||
      tid === 'g014' ||
      tid === 'qianhou4'
    ) {
      m = 2
    }
  }
  return m > 1 ? units * m : units
}

/**
 * 直选复式/组合按位拆分：多行，或单行恰好 segmentLen 段逗号分位（如 `12,2,3,45`）。
 * 段数不合规 → null。
 */
export function splitZhixuanPositionParts(content: string, segmentLen: number): string[] | null {
  if (segmentLen <= 0) return null
  const raw = String(content ?? '')
    .replace(/\r/g, '')
    .replace(/，/g, ',')
  if (!raw.trim()) return null
  if (raw.includes('\n')) {
    return splitGroupLinesPad(raw, segmentLen).slice(0, segmentLen)
  }
  const parts = raw.split(',').map((p) => p.trim())
  if (parts.length !== segmentLen) return null
  return parts
}

function isSscRenxuanConfig(config: PlayConfig): boolean {
  const tid = String(config.playTypeId ?? '').toLowerCase()
  return (
    tid === 'renxuan' ||
    tid === 'g011' ||
    config.guajiGroup === '任选' ||
    (config.playTypeLabel ?? '') === '任选'
  )
}

function isRenxuanZhixuanFushi(config: PlayConfig): boolean {
  const text = `${config.betMode ?? ''} ${config.subPlayId} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''}`
  if (/单式|组选|和值|组三|组六|zu\d|hunhe|混合/i.test(text)) return false
  return (
    config.inputMode === 'multiline' ||
    /直选复式|zhixuan_fs|fushi/i.test(text) ||
    (config.betMode === 'fushi' && !/组选/.test(text))
  )
}

function renPickCountFromConfig(config: PlayConfig): number {
  const s = `${config.catalogSubId ?? ''} ${config.subPlayId} ${config.playMethodLabel ?? ''}`
  if (/ren4|任选四|任四/i.test(s)) return 4
  if (/ren3|任选三|任三/i.test(s)) return 3
  if (/ren2|任选二|任二/i.test(s)) return 2
  // rules/v2 数字 id（对齐 guajibet.renxuanSegmentLen）
  const sid = Number.parseInt(String(config.catalogSubId ?? config.subPlayId ?? '').trim(), 10)
  if (Number.isFinite(sid)) {
    if (sid >= 141 && sid <= 145) return 4
    if (sid >= 80 && sid <= 88) return 3
    if (sid >= 74 && sid <= 79) return 2
  }
  return 2
}

function combinationsIndices(n: number, k: number): number[][] {
  const out: number[][] = []
  const buf: number[] = []
  const dfs = (start: number) => {
    if (buf.length === k) {
      out.push([...buf])
      return
    }
    for (let i = start; i < n; i++) {
      buf.push(i)
      dfs(i + 1)
      buf.pop()
    }
  }
  dfs(0)
  return out
}

/** 与后端 evaluateRenxuanZhixuan 一致：五位号池，对 C(5,pickCount) 各位积求和 */
function countRenxuanZhixuanUnits(
  lines: string[],
  pickCount: number,
  pool?: { min: number; max: number } | null,
): number {
  const n = pickCount > 0 && pickCount <= 5 ? pickCount : 2
  const pools = Array.from({ length: 5 }, (_, i) => parsePickTokens(lines[i] ?? '', pool ?? undefined))
  let units = 0
  for (const combo of combinationsIndices(5, n)) {
    let u = 1
    for (const pos of combo) {
      const len = pools[pos]?.length ?? 0
      if (!len) {
        u = 0
        break
      }
      u *= len
    }
    units += u
  }
  return units
}

function inferHezhiSegmentLen(config: PlayConfig): number {
  const label = `${config.guajiGroup ?? ''} ${config.playTypeLabel ?? ''} ${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''} ${config.playTypeId ?? ''}`
  const fromRen = renPickCountFromConfig(config)
  if (fromRen >= 2 && fromRen <= 5 && /任|ren/i.test(label)) return fromRen
  // 文案优先（前中后三/前后三/前三…）；勿在 tid 回退里把 g007 当成四星、g001 当成五星
  const fromText = hezhiDigitLenFromText(label, 0)
  if (fromText >= 2) return fromText
  // 玩法树未回填文案时：UI segmentLen 恒为 1（选项池），须用 playTypeId 推断星数
  // 对齐 playConfig.sscSegmentRange / guajibet.legacyTypeSegmentRange
  const tid = String(config.playTypeId ?? '').toLowerCase()
  if (tid === 'g004' || tid === 'g005' || tid === 'g008' || tid === 'qian2' || tid === 'hou2' || tid === 'qianhou2') {
    return 2
  }
  if (
    tid === 'g001' ||
    tid === 'g002' ||
    tid === 'g003' ||
    tid === 'g007' ||
    tid === 'g012' ||
    tid === 'qian3' ||
    tid === 'zhong3' ||
    tid === 'hou3' ||
    tid === 'qianzhonghou3' ||
    tid === 'qianhou3'
  ) {
    return 3
  }
  if (
    tid === 'g013' ||
    tid === 'g014' ||
    tid === 'sixing' ||
    tid === 'qian4' ||
    tid === 'hou4' ||
    tid === 'qianhou4'
  ) {
    return 4
  }
  if (tid === 'g015' || tid === 'g000' || tid === 'wuxing') return 5
  if (config.segmentLen > 1 && config.segmentLen <= 5) return config.segmentLen
  if (config.renPositionCount && config.renPositionCount >= 2) return config.renPositionCount
  return 3
}

export function buildGameBetPayload(
  playMethod: string,
  groupContent: string,
  overrides?: Partial<
    Pick<GameBetPayload, 'playTemplate' | 'typeId' | 'subId' | 'playTypeId' | 'subPlayId'>
  >,
): GameBetPayload {
  const cfg = resolvePlayConfig({
    playMethod,
    playTypeId: overrides?.typeId ?? overrides?.playTypeId,
    subPlayId: overrides?.subId ?? overrides?.subPlayId,
  })
  const typeId = overrides?.typeId ?? overrides?.playTypeId ?? cfg.playTypeId
  const subId = overrides?.subId ?? overrides?.subPlayId ?? (cfg.subPlayId || undefined)
  return {
    playTemplate: overrides?.playTemplate,
    typeId,
    subId,
    playMethod: playMethod.trim() || undefined,
    playTypeId: typeId,
    subPlayId: subId,
    groupContent: groupContent.trim(),
  }
}

export function seedDigitsFromNumbers(numbers: string): string[] {
  return parsePickTokens(numbers.replace(/\s+/g, ','))
}

export function splitGroupLines(content: string): string[] {
  return content
    .split('\n')
    .map((l) => l.trim())
    .filter(Boolean)
}

/** 保留空行并补齐到 len（任选五位号池按位对齐） */
export function splitGroupLinesPad(content: string, len: number): string[] {
  const lines = content.split('\n').map((l) => l.trim())
  while (lines.length < len) lines.push('')
  return lines.slice(0, Math.max(len, lines.length))
}

/**
 * 定位胆多位内容 → 按位行（保留前导/中间空位）。
 * 支持换行格式「\\n\\n1,2\\n\\n」与逗号 wire「,,12,,」；禁止 trim/filter 空行导致位次前移。
 */
export function dingweiPositionLines(raw: string, segLen: number): string[] {
  const n = Math.max(1, segLen)
  const s = String(raw ?? '').replace(/\r/g, '')
  if (s.includes('\n')) {
    return splitGroupLinesPad(s, n).slice(0, n)
  }
  const parts = s.split(',')
  if (parts.length === n) {
    return parts.map((p) => {
      const digits = String(p ?? '')
        .replace(/\D/g, '')
        .split('')
        .filter((d) => d >= '0' && d <= '9')
      return [...new Set(digits)].join(',')
    })
  }
  return splitGroupLinesPad(s, n).slice(0, n)
}

/** 一星/定位胆多位（允许空位）：校验与计注须按位保留空槽 */
function isDingweiMultilineConfig(config: PlayConfig): boolean {
  if (config.inputMode !== 'multiline' || config.segmentLen <= 1) return false
  return isYixingDingweiPlayConfig(config)
}

/** 一星/定位胆：每位最多投注号码个数（0–9 共 10 个号，上限 9） */
export const YIXING_MAX_PICKS_PER_POS = 9
export const YIXING_MAX_PICKS_MSG = '每个位置最多只能投注9个号码'

/** 和值/跨度默认上限（三星）；二星等按区位动态取，见 hezhiKuaduMaxBetUnits */
export const HEZHI_MAX_BET_UNITS = 900
export const HEZHI_MAX_BET_UNITS_MSG = '投注注数超过最大投注注数:900'

/** 任二直选和值单组上限（第三方 900，对齐任二直选复式） */
export const REN2_ZHIXUAN_HEZHI_MAX_BET_UNITS = 900

/**
 * 和值/跨度单组最大注数（对齐第三方）：与直选复式同口径（每位 0–9）。
 * 前二/后二=90，前三/中三/后三=900；和值 UI 的 segmentLen 恒为 1，须用 inferHezhiSegmentLen。
 * 勿用和值号池宽度（前二 0–18→19）代入公式，否则会算成 342。
 */
export function hezhiKuaduMaxBetUnits(config: PlayConfig): number {
  // 任二直选和值：第三方上限 900（勿套前二 90）
  if (isRen2ZhixuanHezhiConfig(config)) return REN2_ZHIXUAN_HEZHI_MAX_BET_UNITS
  // 任三/任四直选和值：与任选直选复式同口径（9000 / 45000）；含剥位后的 bareConfig
  if (isRenxuanZhixuanHezhiConfig(config)) {
    return renxuanZhixuanFushiMaxBetUnits(config)
  }
  const segLen = inferHezhiSegmentLen(config)
  if (segLen <= 1) return HEZHI_MAX_BET_UNITS
  const size = 10
  const fullMinusSame = Math.pow(size, segLen) - size
  const oneShort = (size - 1) * Math.pow(size, segLen - 1)
  const base = Math.min(fullMinusSame, oneShort)
  const m = segmentBetMultiplier(config.guajiGroup ?? config.playTypeLabel ?? '')
  return base * Math.max(1, m)
}

export function hezhiKuaduMaxBetUnitsMsg(config: PlayConfig): string {
  return `投注注数超过最大投注注数:${hezhiKuaduMaxBetUnits(config)}`
}

/** 单个和值/跨度选项的组合注数 */
function hezhiKuaduTokenUnits(config: PlayConfig, token: string): number {
  const segLen = inferHezhiSegmentLen(config)
  const n = Number(token)
  if (!Number.isFinite(n)) return 0
  if (config.betMode === 'kuadu' || /跨度/.test(config.playMethodLabel ?? '')) {
    return applySegmentBetMultiplier(config, countOrderedSpanCombinations(n, segLen))
  }
  const zuxuan = (config.playMethodLabel ?? '').includes('组选')
  const base = zuxuan
    ? countZuxuanSumCombinations(n, segLen)
    : countOrderedSumCombinations(n, segLen)
  return applySegmentBetMultiplier(config, base)
}

/**
 * 选 k 个和值/跨度时，最小可能组合注数是否 ≤ max。
 * 对齐后端 attributeCountFeasibleUnderMax（用组合数最小的 k 个选项求和）。
 */
export function hezhiKuaduCountFeasibleUnderMax(
  config: PlayConfig,
  k: number,
  universe: string[],
  maxUnits = hezhiKuaduMaxBetUnits(config),
): boolean {
  if (k <= 0 || maxUnits <= 0) return true
  if (!universe.length || k > universe.length) return false
  const units = universe
    .map((t) => hezhiKuaduTokenUnits(config, t))
    .filter((n) => n > 0)
    .sort((a, b) => a - b)
  if (units.length < k) return false
  let sum = 0
  for (let i = 0; i < k; i++) sum += units[i]!
  return sum <= maxUnits
}

/** 随机出号「选项个数」上限：宇宙长度再受组合注数约束（前二满选 19→18） */
export function maxHezhiKuaduRandomCount(config: PlayConfig, universe: string[]): number {
  let n = Math.max(1, universe.length)
  const maxUnits = hezhiKuaduMaxBetUnits(config)
  if (maxUnits <= 0) return n
  while (n > 1 && !hezhiKuaduCountFeasibleUnderMax(config, n, universe, maxUnits)) n -= 1
  return n
}

/**
 * 贪心取组合数较小的 k 个选项且总注 ≤ 上限（保存占位/预览用；真实下注由引擎重抽）。
 * 若 k 过大则自动降到可行个数。
 */
export function greedyHezhiKuaduPicksUnderMax(
  config: PlayConfig,
  k: number,
  universe: string[],
): string[] {
  const maxUnits = hezhiKuaduMaxBetUnits(config)
  const scored = universe
    .map((t) => ({ t, u: hezhiKuaduTokenUnits(config, t) }))
    .filter((x) => x.u > 0)
    .sort((a, b) => a.u - b.u || Number(a.t) - Number(b.t) || a.t.localeCompare(b.t))
  if (!scored.length) return []
  let want = Math.min(Math.max(1, k), scored.length)
  while (want > 1 && !hezhiKuaduCountFeasibleUnderMax(config, want, universe, maxUnits)) want -= 1
  const picked: string[] = []
  let sum = 0
  for (const row of scored) {
    if (picked.length >= want) break
    if (maxUnits > 0 && sum + row.u > maxUnits) continue
    picked.push(row.t)
    sum += row.u
  }
  if (!picked.length) picked.push(scored[0]!.t)
  return picked.sort((a, b) => Number(a) - Number(b) || a.localeCompare(b))
}

/** 和值尾数单区最大注数（选项最多 9 个）；多区位再×段倍乘（前中后三→27） */
export const WEISHU_MAX_BET_UNITS = 9
export const WEISHU_MAX_BET_UNITS_MSG = '投注注数超过最大投注注数:9'

/** 和值尾数最大注数：单区 9 × 区位倍乘 */
export function weishuMaxBetUnits(config: PlayConfig): number {
  const m = segmentBetMultiplier(config.guajiGroup ?? config.playTypeLabel ?? '')
  return WEISHU_MAX_BET_UNITS * Math.max(1, m)
}

export function weishuMaxBetUnitsMsg(config: PlayConfig): string {
  return `投注注数超过最大投注注数:${weishuMaxBetUnits(config)}`
}

/** 三星直选组合单区上限（900×3）；四星等按 zhixuanFushiMax×段长 动态计算 */
export const ZUHE_MAX_BET_UNITS = 2700
/** @deprecated 请用 zuheMaxBetUnitsMsg(config)；固定 2700 仅覆盖三星单区 */
export const ZUHE_MAX_BET_UNITS_MSG = '投注注数超过最大投注注数:2700'

/**
 * 直选组合最大注数 = 直选复式上限 × 段长（复式上限已含区位倍乘）。
 * 例：前三 900×3=2700；前中后三 2700×3=8100；四星 9000×4=36000。
 */
export function zuheMaxBetUnits(config: PlayConfig): number {
  const seg = Math.max(1, config.segmentLen || 1)
  const fushiMax = zhixuanFushiMaxBetUnits(config)
  if (fushiMax > 0) return fushiMax * seg
  const m = segmentBetMultiplier(config.guajiGroup ?? config.playTypeLabel ?? '')
  return ZUHE_MAX_BET_UNITS * Math.max(1, m)
}

export function zuheMaxBetUnitsMsg(config: PlayConfig): string {
  return `投注注数超过最大投注注数:${zuheMaxBetUnits(config)}`
}

/** 第三方单次投注最高金额（对齐 guaji 40053） */
export const MAX_SINGLE_BET_AMOUNT = 100000

/** 最高下注限额文案（如 最高下注限额100000.00USDT） */
export function maxBetAmountExceededMessage(currency = 'USDT'): string {
  const cur = String(currency ?? 'USDT').trim().toUpperCase() || 'USDT'
  return `最高下注限额${MAX_SINGLE_BET_AMOUNT.toFixed(2)}${cur}`
}

export function isMaxBetAmountExceededMessage(message: string): boolean {
  return String(message ?? '').includes('最高下注限额')
}

/** 注数 × 投注单位 × 倍数 → 单次金额 */
export function calcBetAmount(betUnits: number, mult: number, unitYuan: number): number {
  const units = betUnits > 0 ? betUnits : 1
  const m = mult > 0 ? mult : 1
  const unit = unitYuan > 0 ? unitYuan : 2
  return Math.round(unit * units * m * 100) / 100
}

export function betAmountExceedsMax(amount: number): boolean {
  return amount > MAX_SINGLE_BET_AMOUNT + 1e-9
}

/**
 * 从倍投载荷估算模式最高倍率（扫描 rounds[].mult / multiples 等）。
 * 估高无妨：用于保存前预检，真正限额以后端下单前为准。
 */
export function maxModeMultiplierFromPayload(payload: unknown): number {
  let max = 1
  const visit = (v: unknown): void => {
    if (Array.isArray(v)) {
      for (const x of v) visit(x)
      return
    }
    if (!v || typeof v !== 'object') return
    const o = v as Record<string, unknown>
    for (const key of ['mult', 'multiple', 'multiplier'] as const) {
      const n = Number(o[key])
      if (Number.isFinite(n) && n > max) max = n
    }
    if (typeof o.multiples === 'string') {
      for (const part of o.multiples.split(/[,，\s]+/)) {
        const n = Number(part)
        if (Number.isFinite(n) && n > max) max = n
      }
    }
    for (const val of Object.values(o)) visit(val)
  }
  visit(payload)
  return max > 0 ? max : 1
}

/**
 * 直选（复式/单式）单组最大注数（对齐第三方）：
 * min(P^n − P, (P−1)·P^(n−1))，再乘区位倍乘。
 * - P^n−P：满号位积去掉「各位同一号码」的对子/豹子
 * - (P−1)·P^(n−1)：第三方前三等实限（SSC：前二 90、前三 900，而非公式外推的 990）
 * 例：前二/后二 90；前三/中三/后三 900；前中后三×3→2700。
 */
export function zhixuanFushiMaxBetUnits(config: PlayConfig): number {
  // 任选直选复式：按 C(5,n) 计注，上限单独取值（任二=900，勿套前二 90）
  if (isSscRenxuanConfig(config) && isRenxuanZhixuanFushi(config)) {
    return renxuanZhixuanFushiMaxBetUnits(config)
  }
  const pool = poolFromConfig(config)
  const size = pool ? pool.max - pool.min + 1 : 10
  const n = Math.max(1, config.segmentLen || 1)
  if (size <= 1 || n <= 1) return 0
  const fullMinusSame = Math.pow(size, n) - size
  const oneShort = (size - 1) * Math.pow(size, n - 1)
  const base = Math.min(fullMinusSame, oneShort)
  const m = segmentBetMultiplier(config.guajiGroup ?? config.playTypeLabel ?? '')
  return base * Math.max(1, m)
}

/** 任选·直选和值（不含组选和值；含剥位后的 bareConfig，靠文案/ruleId 识别） */
function isRenxuanZhixuanHezhiConfig(config: PlayConfig): boolean {
  const label = `${config.playMethodLabel ?? ''}`
  if (/组选/.test(label)) return false
  const bm = String(config.betMode ?? '').toLowerCase()
  const isHezhi =
    bm === 'hezhi' || /直选和值/.test(label) || (/和值/.test(label) && !/尾数|跨度/.test(label))
  if (!isHezhi) return false
  const k = config.renPositionCount ?? renPickCountFromConfig(config)
  if (k < 2 || k > 5) return false
  const sid = Number.parseInt(String(config.catalogSubId ?? config.subPlayId ?? '').trim(), 10)
  // 组选和值：79 任二 / 85 任三 / 145 任四，勿误判
  if (Number.isFinite(sid) && (sid === 79 || sid === 85 || sid === 145)) return false
  return (
    isSscRenxuanConfig(config) ||
    /任[二三四]|任选|g011|renxuan/i.test(
      `${config.playTypeId ?? ''} ${config.guajiGroup ?? ''} ${label}`,
    ) ||
    (Number.isFinite(sid) && ((sid >= 74 && sid <= 88) || (sid >= 141 && sid <= 144)))
  )
}

/** 任选·任二直选和值（不含组选和值；catalog 76） */
export function isRen2ZhixuanHezhiConfig(config: PlayConfig): boolean {
  if (!isRenxuanZhixuanHezhiConfig(config)) return false
  const k = config.renPositionCount ?? renPickCountFromConfig(config)
  return k === 2
}

/**
 * 任选直选复式单组上限：同星直选上限 × C(5,k)
 * 任二 90×10=900；任三 900×10=9000；任四 9000×5=45000（对齐第三方）。
 */
export function renxuanZhixuanFushiMaxBetUnits(config: PlayConfig): number {
  const n = renPickCountFromConfig(config)
  const k = n >= 2 && n <= 5 ? n : 2
  // 同星直选上限按位号 0–9；勿用和值号池（0–27）否则任三会被抬飞或误算
  const size = 10
  if (k <= 1) return 900
  const fullMinusSame = Math.pow(size, k) - size
  const oneShort = (size - 1) * Math.pow(size, k - 1)
  const base = Math.min(fullMinusSame, oneShort)
  const mul = comboCount(5, k)
  return mul > 0 ? base * mul : base
}

/** 任选选位类单组上限（任二直选和值=900，其余对齐直选复式） */
export function renxuanNeedsPositionMaxBetUnits(config: PlayConfig): number {
  if (isRen2ZhixuanHezhiConfig(config)) return REN2_ZHIXUAN_HEZHI_MAX_BET_UNITS
  return renxuanZhixuanFushiMaxBetUnits(config)
}

/** 直选单式最大注数：与复式同口径（前二=90、前三=900…）；任选直选单式用任选上限（任二=900） */
export function zhixuanDanshiMaxBetUnits(config: PlayConfig): number {
  if (
    isSscRenxuanConfig(config) &&
    isRenxuanPositionDanshiConfig(config) &&
    !isZuxuanDanshiConfig(config)
  ) {
    return renxuanZhixuanFushiMaxBetUnits(config)
  }
  return zhixuanFushiMaxBetUnits(config)
}

/** 是否「超过最大投注注数」类提示（保存时原样弹窗、不清空内容） */
export function isMaxBetUnitsExceededMessage(message: string): boolean {
  return String(message ?? '').startsWith('投注注数超过最大投注注数:')
}

/** 注数/金额超限类提示：保存时原样弹窗、不清空内容 */
export function isBetLimitExceededMessage(message: string): boolean {
  return isMaxBetUnitsExceededMessage(message) || isMaxBetAmountExceededMessage(message)
}

/** 直选组合（前三/中三/后三「组合」等） */
export function isZhixuanZuhePlayConfig(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zuhe') return true
  const sub = (config.subPlayId ?? '').trim()
  if (sub === 'zuhe') return true
  const catalog = (config.catalogSubId ?? '').trim()
  if (catalog === 'zuhe' || catalog.endsWith('_zuhe')) return true
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  if (label.includes('组选')) return false
  if (label.includes('直选组合')) return true
  if (/(^|[^组选])组合/.test(label) || label.trim().endsWith('组合')) return true
  return false
}

/** 是否一星/定位胆玩法（含 rules/v2 g006、guajiGroup=一星） */
export function isYixingDingweiPlayConfig(config: PlayConfig): boolean {
  if (config.betMode === 'dingwei') return true
  const tid = String(config.playTypeId ?? '')
  if (tid === 'dingwei' || tid === 'g006') return true
  if (config.guajiGroup === '一星') return true
  const label = `${config.playTypeLabel ?? ''} ${config.playMethodLabel ?? ''}`
  return label.includes('一星') || label.includes('定位胆')
}

/** 一星内容按位校验：任一位号码数 > 9 则返回固定提示 */
export function yixingContentMaxPicksError(config: PlayConfig, raw: string): string | null {
  if (!isYixingDingweiPlayConfig(config)) return null
  const poolCfg = poolFromConfig(config) ?? undefined
  if (isDingweiMultilineConfig(config)) {
    const lines = dingweiPositionLines(String(raw ?? '').replace(/\r/g, ''), config.segmentLen)
    for (let i = 0; i < config.segmentLen; i++) {
      const line = lines[i] ?? ''
      if (!line.trim()) continue
      const n = [...new Set(parsePickTokens(line, poolCfg))].length
      if (n > YIXING_MAX_PICKS_PER_POS) return YIXING_MAX_PICKS_MSG
    }
    return null
  }
  const n = [...new Set(parsePickTokens(String(raw ?? ''), poolCfg))].length
  if (n > YIXING_MAX_PICKS_PER_POS) return YIXING_MAX_PICKS_MSG
  return null
}

/** 直选单式：提取指定位数的数字串 */
export function parseNumberTokens(raw: string, expectLen: number): string[] {
  const parts = raw.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
  const out: string[] = []
  for (const p of parts) {
    if (!/^\d+$/.test(p)) continue
    if (expectLen > 0 && p.length !== expectLen) continue
    out.push(p)
  }
  return out
}

/** 单行选号池是否为 0-9 数字（允许粘连「12」/「1234567890」，校验时按位拆开） */
function isValidDigitPoolLine(raw: string): boolean {
  const t = raw.trim()
  if (!t) return false
  const parts = t.split(/[,，\s]+/).map((s) => s.trim()).filter(Boolean)
  if (!parts.length) return false
  return parts.every((p) => /^\d+$/.test(p))
}

export type GroupContentValidation =
  | { ok: true; normalized: string; betUnits: number }
  | { ok: false; message: string }

/**
 * 校验并规范化方案分组内容，规则与后端 schemes/play_api.go validateGroupContent 对齐。
 */
export function validateGroupContent(config: PlayConfig, raw: string): GroupContentValidation {
  const content = raw.trim()
  if (!content) return { ok: false, message: '方案内容不能为空' }

  // 双区组选：须在任选剥位 / 通用号池校验之前拦截（「12,34」不是扁选 0-9 号池）
  if (isZuDualPlayConfig(config) && !isRenxuanNeedsPositionConfig(config)) {
    const dual = validateZuDualContent(config, content)
    if (!dual.ok) return dual
    return {
      ok: true,
      normalized: dual.normalized,
      betUnits: applySegmentBetMultiplier(config, dual.betUnits),
    }
  }

  // 组三/组六/组选6/组选24 号池：保存时强制最低选号；粘连「12」「1234567890」按位展开后落库
  // 任选带位名前缀时先剥位再数码（万,千,百,十\n1,2,3）
  const zuxuanMin = zuxuanPoolMinPick(config)
  if (zuxuanMin != null) {
    let poolLine = content
    let renPrefix = ''
    if (isRenxuanNeedsPositionConfig(config)) {
      const k = config.renPositionCount ?? renPickCountFromConfig(config)
      const parsed = parseRenxuanPositionContent(content, k)
      poolLine = parsed.picks
      if (parsed.positions.length) {
        renPrefix = `${parsed.positions.join(',')}\n`
      }
    }
    // 双区组选已在上方拦截；此处仅扁选号池（含四星组选6）
    if (poolLine && !poolLine.includes('\n') && !isZuDualPlayConfig(config)) {
      const digits = [...new Set(parsePickTokens(poolLine))]
      if (isValidDigitPoolLine(poolLine) || digits.length > 0) {
        if (digits.length < zuxuanMin) {
          return { ok: false, message: zuxuanPoolMinPickMessage(config) }
        }
        if (digits.length > 0) {
          const normalized = `${renPrefix}${digits.join(',')}`
          return {
            ok: true,
            normalized,
            betUnits: countBetUnits(config, normalized),
          }
        }
      }
    }
  }

  const sub = config.subPlayId

  // 任选直选复式：允许部分位为空，至少填满 n 个位（须在通用 zhixuan_fs 校验之前）
  if (isSscRenxuanConfig(config) && isRenxuanZhixuanFushi(config)) {
    const pickN = renPickCountFromConfig(config)
    const rawLines = splitGroupLinesPad(content, 5)
    const normalizedLines: string[] = []
    let filled = 0
    for (let i = 0; i < 5; i++) {
      const line = rawLines[i] ?? ''
      if (!line) {
        normalizedLines.push('')
        continue
      }
      if (!isValidDigitPoolLine(line)) {
        const pos = config.segmentLabels[i] ?? `第 ${i + 1} 位`
        return { ok: false, message: `${pos}选号格式不合法，请使用 0-9 并以逗号分隔` }
      }
      const digits = parsePickTokens(line)
      if (digits.length) filled++
      normalizedLines.push([...new Set(digits)].join(','))
    }
    if (filled < pickN) {
      return { ok: false, message: `任选至少在 ${pickN} 个位置选号` }
    }
    const normalized = normalizedLines.join('\n')
    const betUnits = countBetUnits(config, normalized)
    if (betUnits <= 0) return { ok: false, message: '选号无效' }
    const maxRen = renxuanZhixuanFushiMaxBetUnits(config)
    if (maxRen > 0 && betUnits > maxRen) {
      return { ok: false, message: `投注注数超过最大投注注数:${maxRen}` }
    }
    return { ok: true, normalized, betUnits }
  }

  if (isRenxuanNeedsPositionConfig(config)) {
    const k = config.renPositionCount ?? renPickCountFromConfig(config)
    // 须显式带位名（parse 对无位名内容会填默认万千，不能当已选位）
    const head =
      content.includes('|')
        ? content.slice(0, content.indexOf('|'))
        : (content.split(/\n/)[0] ?? '')
    if (!extractSscPositionNames(head).length) {
      return { ok: false, message: `请从万千百十个中至少勾选 ${k} 个位置（最多 ${RENXUAN_POS_MAX} 个）` }
    }
    const { positions, picks } = parseRenxuanPositionContent(content, k)
    if (positions.length < k) {
      return { ok: false, message: `请从万千百十个中至少勾选 ${k} 个位置（最多 ${RENXUAN_POS_MAX} 个）` }
    }
    if (positions.length > RENXUAN_POS_MAX) {
      return { ok: false, message: `选位最多 ${RENXUAN_POS_MAX} 个` }
    }
    if (!picks.trim()) {
      return { ok: false, message: '请先选择或输入号码' }
    }
    const mul = comboCount(positions.length, k)
    if (mul <= 0) return { ok: false, message: '选位无效' }

    if (isRenxuanPositionDanshiConfig(config)) {
      const digitLen = config.segmentLen > 0 ? config.segmentLen : k
      // 冷热按位号池（1,2\n3,4\n5,6）先展成整注，再按单式形态校验（对齐后端）
      let picksBody = picks
      if (digitLen > 1 && isZhixuanPositionPoolContent(picksBody, digitLen)) {
        picksBody = expandZhixuanPositionPoolToDanshi(picksBody, digitLen)
        if (!picksBody) return { ok: false, message: '选号无效' }
      }
      let body = picksBody
      if (isZu3DanshiConfig(config)) {
        const parts = picksBody.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
        for (const p of parts) {
          if (!/^\d+$/.test(p)) return { ok: false, message: '号码存在非数字内容' }
          if (p.length !== digitLen) {
            return { ok: false, message: `每注须为 ${digitLen} 位数字，请用逗号分隔` }
          }
        }
        body = normalizeZu3DanshiContent(picksBody, digitLen)
        if (!body) return { ok: false, message: ZU3_DANSHI_PATTERN_MSG }
      } else if (isZu6DanshiConfig(config)) {
        const parts = picksBody.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
        for (const p of parts) {
          if (!/^\d+$/.test(p)) return { ok: false, message: '号码存在非数字内容' }
          if (p.length !== digitLen) {
            return { ok: false, message: `每注须为 ${digitLen} 位数字，请用逗号分隔` }
          }
        }
        body = normalizeZu6DanshiContent(picksBody, digitLen)
        if (!body) return { ok: false, message: ZU6_DANSHI_PATTERN_MSG }
      } else if (isHunhePlayConfig(config)) {
        const parts = picksBody.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
        for (const p of parts) {
          if (!/^\d+$/.test(p)) return { ok: false, message: '号码存在非数字内容' }
          if (p.length !== digitLen) {
            return { ok: false, message: `每注须为 ${digitLen} 位数字，请用逗号分隔` }
          }
        }
        body = normalizeHunheGroupContent(picksBody, digitLen)
        if (!body) return { ok: false, message: HUNHE_DANSHI_PATTERN_MSG }
      } else if (isZuxuanDanshiConfig(config)) {
        // 单码号池（1,2,3）→ 两两组合整注（12,13,23）；已有整注则形态去重
        body = normalizeZuxuanDanshiContent(picksBody, digitLen)
        if (!body) {
          return {
            ok: false,
            message: `请从号码池至少选 ${digitLen} 个不同号码（将自动组合为组选单式）`,
          }
        }
      } else {
        const parts = picksBody.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
        for (const p of parts) {
          if (!/^\d+$/.test(p)) return { ok: false, message: '号码存在非数字内容' }
          if (p.length !== digitLen) {
            return { ok: false, message: `每注须为 ${digitLen} 位数字，请用逗号分隔` }
          }
        }
        body = dedupeDanshiTokens(picksBody, digitLen).join(',')
        if (!body) return { ok: false, message: '选号无效' }
      }
      const uniq = body.split(',').map((s) => s.trim()).filter(Boolean)
      const normalized = buildRenxuanPositionContent(positions, uniq.join(','))
      const betUnits = mul * uniq.length
      if (
        !isZuxuanDanshiConfig(config) &&
        !isZu3DanshiConfig(config) &&
        !isZu6DanshiConfig(config) &&
        !isHunhePlayConfig(config)
      ) {
        const maxDanshi = zhixuanDanshiMaxBetUnits(config)
        if (maxDanshi > 0 && betUnits > maxDanshi) {
          return { ok: false, message: `投注注数超过最大投注注数:${maxDanshi}` }
        }
      }
      return { ok: true, normalized, betUnits }
    }

    // 号池/和值：剥位后走原玩法校验，再拼回选位；注数 × C(n,k)
    const inner = validateGroupContent(bareConfigForRenxuanPicks(config), picks)
    if (!inner.ok) return inner
    const normalized = buildRenxuanPositionContent(positions, inner.normalized)
    const betUnits = mul * inner.betUnits
    if (betUnits <= 0) return { ok: false, message: '选号无效' }
    const maxRen = renxuanNeedsPositionMaxBetUnits(config)
    if (maxRen > 0 && betUnits > maxRen) {
      return { ok: false, message: `投注注数超过最大投注注数:${maxRen}` }
    }
    return { ok: true, normalized, betUnits }
  }

  if (isSscDanshiLikeConfig(config)) {
    const seg = config.segmentLen > 0 ? config.segmentLen : 0
    // 冷热/复式残留的按位号池先展开再校验
    let danshiRaw = content
    if (seg > 1 && isZhixuanPositionPoolContent(danshiRaw, seg)) {
      danshiRaw = expandZhixuanPositionPoolToDanshi(danshiRaw, seg)
    }
    if (isZu3DanshiConfig(config)) {
      const parts = danshiRaw.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
      for (const p of parts) {
        if (!/^\d+$/.test(p)) return { ok: false, message: '存在非数字内容' }
        if (config.segmentLen > 0 && p.length !== config.segmentLen) {
          return { ok: false, message: `每注须为 ${config.segmentLen} 位数字，请用逗号分隔` }
        }
      }
      const uniq = normalizeZu3DanshiContent(danshiRaw, config.segmentLen || 3)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      if (!uniq.length) return { ok: false, message: ZU3_DANSHI_PATTERN_MSG }
      return {
        ok: true,
        normalized: uniq.join(','),
        betUnits: applySegmentBetMultiplier(config, uniq.length),
      }
    }
    if (isZu6DanshiConfig(config)) {
      const parts = danshiRaw.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
      for (const p of parts) {
        if (!/^\d+$/.test(p)) return { ok: false, message: '存在非数字内容' }
        if (config.segmentLen > 0 && p.length !== config.segmentLen) {
          return { ok: false, message: `每注须为 ${config.segmentLen} 位数字，请用逗号分隔` }
        }
      }
      const uniq = normalizeZu6DanshiContent(danshiRaw, config.segmentLen || 3)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      if (!uniq.length) return { ok: false, message: ZU6_DANSHI_PATTERN_MSG }
      return {
        ok: true,
        normalized: uniq.join(','),
        betUnits: applySegmentBetMultiplier(config, uniq.length),
      }
    }
    if (isZuxuanDanshiConfig(config)) {
      const uniq = normalizeZuxuanDanshiContent(danshiRaw, config.segmentLen)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      if (!uniq.length) {
        return {
          ok: false,
          message: `请从号码池至少选 ${config.segmentLen} 个不同号码（将自动组合为组选单式），或直接填整注如 12,13`,
        }
      }
      // 前中后三/前后二三四等跨段玩法按段倍乘（前中后三×3），与后端 evaluateMultiZone、第三方一致
      return { ok: true, normalized: uniq.join(','), betUnits: applySegmentBetMultiplier(config, uniq.length) }
    }
    const parts = danshiRaw.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
    if (!parts.length) {
      return { ok: false, message: `直选单式须为 ${config.segmentLen} 位数字，每注用逗号分隔` }
    }
    for (const p of parts) {
      if (!/^\d+$/.test(p)) return { ok: false, message: '存在非数字内容' }
      if (config.segmentLen > 0 && p.length !== config.segmentLen) {
        return { ok: false, message: `每注须为 ${config.segmentLen} 位数字，请用逗号分隔` }
      }
    }
    const uniq = dedupeDanshiTokens(danshiRaw, config.segmentLen)
    if (uniq.length && uniq.every(isBaoziDigitTicket)) {
      return { ok: false, message: ZHIXUAN_DANSHI_SOLO_BAOZI_MSG }
    }
    // 前中后三/前后二三四等跨段玩法按段倍乘（前中后三×3），与后端 evaluateMultiZone、第三方一致
    const betUnits = applySegmentBetMultiplier(config, uniq.length)
    // 直选单式与复式同第三方上限（前二=90、前三=900…）
    const maxDanshi = zhixuanDanshiMaxBetUnits(config)
    if (maxDanshi > 0 && betUnits > maxDanshi) {
      return { ok: false, message: `投注注数超过最大投注注数:${maxDanshi}` }
    }
    return { ok: true, normalized: uniq.join(','), betUnits }
  }

  // 直选复式 / 直选组合：按位号池，每一位都必须有号。
  // 支持多行，或单行逗号按位（前后四直选组合如 `12,2,3,45`）。
  // 禁止把「123，，」→「1,2,3\\n\\n」再误归一成单行「1,2,3」（会被录入框当成万=1/千=2/百=3）。
  if (
    (isZhixuanFushiPlayConfig(config) || isZhixuanZuhePlayConfig(config)) &&
    config.segmentLen > 1
  ) {
    const rawContent = String(raw ?? '').replace(/\r/g, '').replace(/，/g, ',')
    const lines =
      splitZhixuanPositionParts(rawContent, config.segmentLen) ??
      splitGroupLinesPad(rawContent, config.segmentLen).slice(0, config.segmentLen)
    const normalizedLines: string[] = []
    for (let i = 0; i < config.segmentLen; i++) {
      const line = lines[i] ?? ''
      const pos = config.segmentLabels?.[i] ?? `第 ${i + 1} 位`
      if (!line.trim()) {
        return { ok: false, message: `${pos}选号不能为空，每一位都需要输入号码` }
      }
      if (!isValidDigitPoolLine(line) && !/^\d+$/.test(line.trim())) {
        return { ok: false, message: `${pos}选号格式不合法，请使用 0-9 并以逗号分隔` }
      }
      const digits = [...new Set(parsePickTokens(line))]
      if (!digits.length) return { ok: false, message: `${pos}选号无效` }
      normalizedLines.push(digits.join(','))
    }
    const normalized = normalizedLines.join('\n')
    if (
      isZhixuanFushiPlayConfig(config) &&
      isZhixuanFushiBaoziLines(normalizedLines, config.segmentLen)
    ) {
      return { ok: false, message: SOLO_BAOZI_FORBIDDEN_MSG }
    }
    const betUnits = countBetUnits(config, normalized)
    if (betUnits <= 0) return { ok: false, message: '选号无效' }
    if (isZhixuanZuhePlayConfig(config)) {
      const maxZuhe = zuheMaxBetUnits(config)
      if (betUnits > maxZuhe) {
        return { ok: false, message: zuheMaxBetUnitsMsg(config) }
      }
    }
    // 直选复式：单组上限（前二=90、前三=900…）
    if (isZhixuanFushiPlayConfig(config) && !isZhixuanZuhePlayConfig(config)) {
      const maxFushi = zhixuanFushiMaxBetUnits(config)
      if (maxFushi > 0 && betUnits > maxFushi) {
        return { ok: false, message: `投注注数超过最大投注注数:${maxFushi}` }
      }
    }
    return { ok: true, normalized, betUnits }
  }

  if (isDingweiMultilineConfig(config)) {
    // 勿用 content=raw.trim()：会吃掉前导空行，",,12,," / "\n\n1,2\n\n" 被压成万位
    const lines = dingweiPositionLines(String(raw ?? '').replace(/\r/g, ''), config.segmentLen)
    const poolCfg = poolFromConfig(config)
    const normalizedLines: string[] = []
    let hasAny = false
    for (let i = 0; i < config.segmentLen; i++) {
      const line = lines[i] ?? ''
      if (!line.trim()) {
        normalizedLines.push('')
        continue
      }
      if (!isValidDigitPoolLine(line)) {
        const pos = config.segmentLabels[i] ?? `第 ${i + 1} 位`
        return { ok: false, message: `${pos}选号格式不合法，请使用 0-9 并以逗号分隔` }
      }
      const digits = [...new Set(parsePickTokens(line, poolCfg))]
      if (digits.length > YIXING_MAX_PICKS_PER_POS) {
        return { ok: false, message: YIXING_MAX_PICKS_MSG }
      }
      if (digits.length) hasAny = true
      normalizedLines.push(digits.join(','))
    }
    if (!hasAny) return { ok: false, message: '请至少在一位选择号码' }
    const normalized = normalizedLines.join('\n')
    return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
  }

  if (config.inputMode === 'danshi' && isLhcDanshiBetMode(config.betMode ?? '')) {
    if (!content) return { ok: false, message: '请输入选号内容' }
    const betMode = config.betMode ?? ''
    // 二全中拖头：允许扁选 01,13,25（与复式同口径）；勿强制要求 |
    if (isLhcErquanzhongTuotouConfig(config)) {
      const flat = content.includes('|') || content.includes('#')
        ? content.replace(/[|#]/g, ',')
        : content
      const nums = [...new Set(parseLhcNumberTokens(flat))]
      if (nums.length < LHC_ERQUANZHONG_NUM_MIN_PICKS) {
        return {
          ok: false,
          message: `二全中拖头：请输入 ${LHC_ERQUANZHONG_NUM_MIN_PICKS}–${LHC_ERQUANZHONG_NUM_MAX_PICKS} 个 01–49 号码，逗号分隔（首个为胆，其余为拖；如 01,13）`,
        }
      }
      if (nums.length > LHC_ERQUANZHONG_NUM_MAX_PICKS) {
        return {
          ok: false,
          message: `二全中拖头：最多 ${LHC_ERQUANZHONG_NUM_MAX_PICKS} 个 01–49 号码，逗号分隔`,
        }
      }
      const normalized = nums.join(',')
      const betUnits = countBetUnits(config, normalized)
      if (betUnits <= 0) return { ok: false, message: '二全中拖头：选号无效' }
      return { ok: true, normalized, betUnits }
    }
    if (
      (betMode === 'tuotou' || betMode.endsWith('_dp')) &&
      !content.includes('|') &&
      !content.includes('#')
    ) {
      return { ok: false, message: '拖头/对碰须用 | 分隔胆拖或对碰组' }
    }
    const betUnits = countBetUnits(config, content)
    if (betUnits <= 0) return { ok: false, message: '选号无效' }
    return { ok: true, normalized: content, betUnits }
  }

  if (
    config.inputMode === 'lhc_num' ||
    config.inputMode === 'lhc_zodiac' ||
    config.inputMode === 'lhc_tail' ||
    config.inputMode === 'lhc_attr'
  ) {
    if (!content) return { ok: false, message: '请先选择号码' }
    if (config.inputMode === 'lhc_num' && isLhcTemaPlayConfig(config)) {
      const invalid = lhcTemaInvalidTokens(content)
      if (invalid.length) {
        return {
          ok: false,
          message: `特码选项无效：${invalid.join('、')}（仅支持 1–49 号码与大/小/单/双/红波等）`,
        }
      }
      const normalized = normalizeLhcTemaContent(content)
      if (!normalized) return { ok: false, message: '请选择属性或输入 1–49 号码（逗号分隔）' }
      return { ok: true, normalized, betUnits: parseLhcTemaContentTokens(normalized).length }
    }
    if (config.inputMode === 'lhc_zodiac' && (isLhcSxDuipengConfig(config) || config.betMode === 'sx_dp')) {
      const zs = parseLhcZodiacTokens(content)
      if (zs.length < LHC_SX_DUIPENG_MIN_PICKS) {
        return {
          ok: false,
          message: `生肖对碰：请选择 ${LHC_SX_DUIPENG_MIN_PICKS} 个生肖（如 马、蛇）`,
        }
      }
      if (zs.length > LHC_SX_DUIPENG_MAX_PICKS) {
        return {
          ok: false,
          message: `生肖对碰：最多选择 ${LHC_SX_DUIPENG_MAX_PICKS} 个生肖`,
        }
      }
      const normalized = `${zs[0]}|${zs[1]}`
      const betUnits = countBetUnits(config, normalized)
      if (betUnits <= 0) return { ok: false, message: '生肖对碰：选号无效' }
      return { ok: true, normalized, betUnits }
    }
    if (config.inputMode === 'lhc_tail' && (isLhcWsDuipengConfig(config) || config.betMode === 'ws_dp')) {
      const ts = parseLhcTailTokens(content)
      if (ts.length < LHC_WS_DUIPENG_MIN_PICKS) {
        return {
          ok: false,
          message: `尾数对碰：请选择 ${LHC_WS_DUIPENG_MIN_PICKS} 个尾数（如 0、1）`,
        }
      }
      if (ts.length > LHC_WS_DUIPENG_MAX_PICKS) {
        return {
          ok: false,
          message: `尾数对碰：最多选择 ${LHC_WS_DUIPENG_MAX_PICKS} 个尾数`,
        }
      }
      const normalized = `${ts[0]}|${ts[1]}`
      const betUnits = countBetUnits(config, normalized)
      if (betUnits <= 0) return { ok: false, message: '尾数对碰：选号无效' }
      return { ok: true, normalized, betUnits }
    }
    if (
      (config.inputMode === 'lhc_attr' || config.inputMode === 'lhc_zodiac' || config.inputMode === 'lhc_tail') &&
      (isLhcSwDuipengConfig(config) || config.betMode === 'sw_dp')
    ) {
      const parts = parseLhcSwDuipengTokens(content)
      if (parts.length < LHC_SW_DUIPENG_MIN_PICKS) {
        return {
          ok: false,
          message: '生尾对碰：请各选择 1 个生肖和 1 个尾数（如 马|0）',
        }
      }
      const normalized = `${parts[0]}|${parts[1]}`
      const betUnits = countBetUnits(config, normalized)
      if (betUnits <= 0) return { ok: false, message: '生尾对碰：选号无效' }
      return { ok: true, normalized, betUnits }
    }
    if (config.inputMode === 'lhc_num' && isLhcErquanzhongNumInputConfig(config)) {
      // 兼容旧拖头 胆|拖：展成扁选再校验
      const flat = content.includes('|') || content.includes('#')
        ? content.replace(/[|#]/g, ',')
        : content
      const nums = [...new Set(parseLhcNumberTokens(flat))]
      const label = isLhcErquanzhongTuotouConfig(config) ? '二全中拖头' : '二全中复式'
      if (nums.length < LHC_ERQUANZHONG_NUM_MIN_PICKS) {
        return {
          ok: false,
          message: `${label}：请输入 ${LHC_ERQUANZHONG_NUM_MIN_PICKS}–${LHC_ERQUANZHONG_NUM_MAX_PICKS} 个 01–49 号码，逗号分隔（如 01,13）`,
        }
      }
      if (nums.length > LHC_ERQUANZHONG_NUM_MAX_PICKS) {
        return {
          ok: false,
          message: `${label}：最多 ${LHC_ERQUANZHONG_NUM_MAX_PICKS} 个 01–49 号码，逗号分隔`,
        }
      }
      const normalized = nums.join(',')
      const betUnits = countBetUnits(config, normalized)
      if (betUnits <= 0) return { ok: false, message: `${label}：选号无效` }
      return { ok: true, normalized, betUnits }
    }
    return { ok: true, normalized: content, betUnits: countBetUnits(config, content) || 1 }
  }

  if (isLonghuPlayConfig(config)) {
    const digits = parseGroupPicks(config, content).digits
    if (!digits.length) {
      return { ok: false, message: `请选择${longhuPickHint(config)}` }
    }
    const normalized = digits.join(',')
    return { ok: true, normalized, betUnits: digits.length }
  }

  // 和值：须落在号池范围内（前三直选和值 0–27 等），逗号分隔；禁止把「27」拆成 2,7 后放行
  if (
    config.betMode === 'hezhi' ||
    (config.playTemplate === 'pc28_std' && config.playMethodLabel?.trim() === '和值')
  ) {
    const pool = poolFromConfig(config) ?? { min: 0, max: 27 }
    const tokens = [...new Set(parsePickTokens(content, pool))]
    if (!tokens.length) {
      return {
        ok: false,
        message: `和值须在 ${pool.min}–${pool.max} 范围内，多选用逗号分隔（如 14,15,16）`,
      }
    }
    const normalized = tokens.join(',')
    const betUnits = countBetUnits(config, normalized)
    if (betUnits <= 0) return { ok: false, message: '选号无效' }
    const maxHezhi = hezhiKuaduMaxBetUnits(config)
    if (maxHezhi > 0 && betUnits > maxHezhi) {
      return { ok: false, message: hezhiKuaduMaxBetUnitsMsg(config) }
    }
    return { ok: true, normalized, betUnits }
  }

  // 和值尾数：0–9 逗号分隔；单区最多 9 注，前中后三等再×区位
  if (
    config.betMode === 'weishu' ||
    /和值尾数/.test(config.playMethodLabel ?? '') ||
    (/尾数/.test(config.playMethodLabel ?? '') &&
      !/单双|大小|对碰|不中|生肖/.test(config.playMethodLabel ?? ''))
  ) {
    const pool = poolFromConfig(config) ?? { min: 0, max: 9 }
    const tokens = [...new Set(parsePickTokens(content, pool))]
    if (!tokens.length) {
      return {
        ok: false,
        message: `和值尾数须在 ${pool.min}–${pool.max} 范围内，多选用逗号分隔（如 1,3,5）`,
      }
    }
    const normalized = tokens.join(',')
    const betUnits = countBetUnits(config, normalized)
    if (betUnits <= 0) return { ok: false, message: '选号无效' }
    const maxWeishu = weishuMaxBetUnits(config)
    if (betUnits > maxWeishu) {
      return { ok: false, message: weishuMaxBetUnitsMsg(config) }
    }
    return { ok: true, normalized, betUnits }
  }

  // 跨度：须落在号池 0–9（前/中/后三直选跨度等），禁止 10+；勿走下方 special 放行
  if (config.betMode === 'kuadu' || /跨度/.test(config.playMethodLabel ?? '')) {
    const pool = poolFromConfig(config) ?? { min: 0, max: 9 }
    const parts = String(content ?? '')
      .replace(/，/g, ',')
      .split(/[\s,\n]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    if (!parts.length) {
      return {
        ok: false,
        message: `跨度须在 ${pool.min}–${pool.max} 范围内，多选用逗号分隔（如 0,3,9）`,
      }
    }
    const tokens: string[] = []
    const seen = new Set<string>()
    for (const p of parts) {
      if (!/^\d{1,2}$/.test(p)) {
        return {
          ok: false,
          message: `跨度须在 ${pool.min}–${pool.max} 范围内，多选用逗号分隔（如 0,3,9）`,
        }
      }
      const n = Number(p)
      if (!Number.isFinite(n) || n < pool.min || n > pool.max) {
        return {
          ok: false,
          message: `跨度须在 ${pool.min}–${pool.max} 范围内，不能填写 ${p}`,
        }
      }
      const tok = String(n)
      if (seen.has(tok)) continue
      seen.add(tok)
      tokens.push(tok)
    }
    const normalized = tokens.join(',')
    const betUnits = countBetUnits(config, normalized)
    if (betUnits <= 0) return { ok: false, message: '选号无效' }
    // 直选跨度组合注数上限与和值一致（前二=90、三星满选 1000>900）
    const maxKuadu = hezhiKuaduMaxBetUnits(config)
    if (maxKuadu > 0 && betUnits > maxKuadu) {
      return { ok: false, message: hezhiKuaduMaxBetUnitsMsg(config) }
    }
    return { ok: true, normalized, betUnits }
  }

  // 不定位：一码最多 2；二码/三码/五星最少选号（勿落入下方 betUnits||1 误放行）
  if (isBudingweiPlayConfig(config)) {
    const pool = poolFromConfig(config) ?? { min: 0, max: 9 }
    const tokens = [...new Set(parsePickTokens(content, pool))]
    const need = inferBudingweiNeed(config)
    if (need <= 1) {
      if (!tokens.length) {
        return { ok: false, message: '一码不定位：须选择 1–2 个 0–9 号码，多选用逗号分隔' }
      }
      if (tokens.length > 2) {
        return { ok: false, message: '投注数字不可超过两位数' }
      }
      const normalized = tokens.join(',')
      return { ok: true, normalized, betUnits: tokens.length }
    }
    const min = budingweiMinPicks(config) ?? need
    if (tokens.length < min) {
      return { ok: false, message: budingweiMinPicksMessage(config) }
    }
    const normalized = tokens.join(',')
    const betUnits = countBetUnits(config, normalized)
    if (betUnits <= 0) {
      return { ok: false, message: budingweiMinPicksMessage(config) }
    }
    return { ok: true, normalized, betUnits }
  }

  // 组选包胆：仅允许一个 0–9 胆码
  if (config.betMode === 'baodan' || /包胆/.test(config.playMethodLabel ?? '')) {
    const pool = poolFromConfig(config) ?? { min: 0, max: 9 }
    const tokens = [...new Set(parsePickTokens(content, pool))]
    if (!tokens.length) {
      return { ok: false, message: '包胆：须输入一个 0–9 的号码（如 5）' }
    }
    if (tokens.length > 1) {
      return { ok: false, message: '包胆：只能选择一个 0–9 的号码' }
    }
    const normalized = tokens[0]!
    const betUnits = countBetUnits(config, normalized)
    if (betUnits <= 0) return { ok: false, message: '选号无效' }
    return { ok: true, normalized, betUnits }
  }

  // 前二/后二/前三/后三大小单双：每位恰好 1 个大/小/单/双（第三方 wire 大,小）
  if (isPerPosDxdsPlayConfig(config)) {
    const allowed = ['大', '小', '单', '双']
    const rawLines = splitGroupLines(content)
    const normalizedLines: string[] = []
    for (let i = 0; i < config.segmentLen; i++) {
      const pos = config.segmentLabels[i] ?? `第 ${i + 1} 位`
      const toks = parseTextPickTokens(rawLines[i] ?? '', allowed)
      if (!toks.length) {
        return { ok: false, message: `${pos}须选择一个选项（大/小/单/双）` }
      }
      if (toks.length > 1) {
        return { ok: false, message: '仅能选择一个选项（大/小/单/双）' }
      }
      normalizedLines.push(toks[0]!)
    }
    const normalized = normalizedLines.join('\n')
    return { ok: true, normalized, betUnits: 1 }
  }

  // 五星和值单双/大小、哈希尾数单双/大小：仅 1 个选项
  if (isWuxingSumDxdsPlayConfig(config)) {
    const isDx = config.betMode === 'daxiao' || /和值大小|尾数大小/.test(config.playMethodLabel ?? '')
    const allowed = isDx ? ['大', '小'] : ['单', '双']
    const hint = isDx ? '大/小' : '单/双'
    const toks = parseTextPickTokens(content, allowed)
    if (!toks.length) {
      return { ok: false, message: `须选择一个选项（${hint}）` }
    }
    if (toks.length > 1) {
      return { ok: false, message: `仅能选择一个选项（${hint}）` }
    }
    return { ok: true, normalized: toks[0]!, betUnits: 1 }
  }

  const specialBetModes = new Set([
    'longhu',
    'longhuhe',
    'dxds',
    'daxiao',
    'danshuang',
    // budingwei 已在上方按码数校验（二码≥2，勿 betUnits||1 误放行）
    // zuhe 已在上方按位校验 + 组合上限（复式上限×段长，含区位）
    // baodan / weishu / kuadu 已在上方单独校验
    'hunhe',
    'teshu',
    'longhubao',
    'tonghao',
    'butong',
    'lianhao',
    'sanlian',
    'shoudong',
    'dantiao',
    'zu24',
    'zu12',
    'zu60',
    'zu30',
    'zu120',
  ])

  if (config.betMode && specialBetModes.has(config.betMode)) {
    const betUnits = countBetUnits(config, content)
    if (config.betMode === 'hunhe') {
      const digitLen = hunheDigitLenFromConfig(config)
      // 高级开某投某等按位号池：保存时保留原格式；下注时再展开并排除豹子。
      // 排除后可能为 0 注（本期跳过），仍允许保存映射。
      if (digitLen > 1 && isZhixuanPositionPoolContent(content, digitLen)) {
        const expanded = expandZhixuanPositionPoolToDanshi(content, digitLen) || ''
        const units = countHunheZuxuanUnits(expanded, digitLen)
        return { ok: true, normalized: content, betUnits: units }
      }
      if (isSchemeSoloBaoziContent(config, content)) {
        return { ok: false, message: SOLO_BAOZI_FORBIDDEN_MSG }
      }
      if (betUnits <= 0) {
        return {
          ok: false,
          message: `混合组选：每注 ${digitLen} 位，不含豹子；组选形态相同只计 1 注（如 123 与 321）`,
        }
      }
      // 落库前过滤豹子/非法注，与计注及第三方 wire 一致（避免「注数 1 却带上 111」）
      return { ok: true, normalized: normalizeHunheGroupContent(content, digitLen), betUnits }
    }
    if (config.betMode === 'teshu') {
      if (isWuxingQuweiDigitPlayConfig(config)) {
        const digits = parseWuxingQuweiDigits(content)
        if (digits.length <= 0) {
          return { ok: false, message: wuxingQuweiFormatHint(config) }
        }
        const max = wuxingQuweiMaxPicks(config)
        if (digits.length > max) {
          return { ok: false, message: wuxingQuweiFormatHint(config) }
        }
        return { ok: true, normalized: digits.join(','), betUnits: digits.length }
      }
      if (betUnits <= 0) {
        return { ok: false, message: '特殊号：请选择豹子、对子、顺子等，多选以逗号分隔' }
      }
      return { ok: true, normalized: content, betUnits }
    }
    // 组选24/12 等：注数为 0 时勿用 ||1 放行（如 3 码 C(3,4)=0）
    if (betUnits <= 0) {
      const minMsg = zuxuanPoolMinPickMessage(config)
      return {
        ok: false,
        message: minMsg !== '选号无效' ? minMsg : '选号无效',
      }
    }
    return { ok: true, normalized: content, betUnits }
  }

  // 无子玩法时勿放行任意文本（跨度等已在上方校验）
  if (!sub) {
    return { ok: true, normalized: content, betUnits: 1 }
  }
  const poolCfg = poolFromConfig(config)
  if (poolCfg) {
    const pool = [...new Set(parsePickTokens(content, poolCfg))]
    if (!pool.length) return { ok: false, message: `选号须在 ${poolCfg.min}–${poolCfg.max} 范围内` }
    if (isYixingDingweiPlayConfig(config) && pool.length > YIXING_MAX_PICKS_PER_POS) {
      return { ok: false, message: YIXING_MAX_PICKS_MSG }
    }
    const normalized = pool.join(',')
    return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
  }
  if (!isValidDigitPoolLine(content)) {
    return { ok: false, message: '选号格式不合法，请使用 0-9 并以逗号分隔每注' }
  }
  const pool = [...new Set(parsePickTokens(content))]
  if (!pool.length) return { ok: false, message: '选号无效' }
  if (isYixingDingweiPlayConfig(config) && pool.length > YIXING_MAX_PICKS_PER_POS) {
    return { ok: false, message: YIXING_MAX_PICKS_MSG }
  }
  const normalized = pool.join(',')
  return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
}

export interface SchemeGroupsValidation {
  ok: boolean
  normalized: string[]
  invalidIndexes: number[]
  message: string
}

/** 分组是否无有效内容（勿用 trim 吃掉定位胆前导空行后再判空） */
function isBlankGroupContent(raw: string): boolean {
  return !String(raw ?? '').replace(/\r/g, '').trim()
}

/** 校验全部方案分组；返回不合法组下标 */
export function validateSchemeGroups(config: PlayConfig, groups: string[]): SchemeGroupsValidation {
  const normalized: string[] = []
  const invalidIndexes: number[] = []
  let firstDetail = ''
  for (let i = 0; i < groups.length; i++) {
    // 保留前导/尾随换行空位：",,12,," → "\n\n1,2\n\n"；trim 会压成万位 "1,2\n\n\n\n"
    const raw = String(groups[i] ?? '').replace(/\r/g, '')
    if (isBlankGroupContent(raw)) {
      invalidIndexes.push(i)
      normalized.push('')
      if (!firstDetail) firstDetail = '方案内容不能为空'
      continue
    }
    const r = validateGroupContent(config, raw)
    // 注数为 0 不得保存（避免缺位内容被误归一后仍 ok）
    if (!r.ok || r.betUnits <= 0) {
      invalidIndexes.push(i)
      // 超注数等业务拒绝：保留原文便于用户删减；格式错误仍清空
      const detail = !r.ok ? r.message : '选号无效'
      if (!firstDetail) firstDetail = detail
      normalized.push(isBetLimitExceededMessage(detail) ? raw : '')
    } else {
      normalized.push(r.normalized)
    }
  }
  const ok = invalidIndexes.length === 0
  if (ok) return { ok, normalized, invalidIndexes, message: '' }
  // 优先返回具体原因（如和值超限 / 组合超限 / 金额超限），便于弹窗原样展示
  if (isBetLimitExceededMessage(firstDetail)) {
    return { ok, normalized, invalidIndexes, message: firstDetail }
  }
  const message =
    invalidIndexes.length === 1
      ? firstDetail || `第 ${invalidIndexes[0]! + 1} 组输入内容与当前玩法不符，已清空该组`
      : `第 ${invalidIndexes.map((i) => i + 1).join('、')} 组输入内容与当前玩法不符，已清空这些组`
  return { ok, normalized, invalidIndexes, message }
}

const SUB_PLAY_LABELS: Record<string, string> = {
  zhixuan_fs: '直选复式',
  zhixuan_ds: '直选单式',
  zuxuan_fs: '组选复式',
}

type PlayConfigSummaryInput = PlayConfig & {
  playMethodLabel?: string
  playTypeLabel?: string
  typeId?: string
  subId?: string
  catalogSubId?: string
}

export function playConfigSummary(config: PlayConfigSummaryInput): string {
  const pt = resolvePlayTypeLabel(config)
  if (config.playMethodLabel) {
    return `${pt} · ${config.playMethodLabel}`
  }
  const subKey = config.catalogSubId ?? config.subId ?? config.subPlayId
  const sp = subKey ? (SUB_PLAY_LABELS[subKey] ?? subKey) : ''
  return sp ? `${pt} · ${sp}` : pt
}

export function catalogFieldsFromPlayConfig(
  config: PlayConfig & { playTemplate?: string; typeId?: string; subId?: string; betMode?: string },
): {
  playTemplate?: string
  typeId?: string
  subId?: string
  betMode?: string
} {
  if (!config.playTemplate) return {}
  const playBetMode = (config.betMode ?? '').trim()
  const out: {
    playTemplate?: string
    typeId?: string
    subId?: string
    betMode?: string
  } = {
    playTemplate: config.playTemplate,
    typeId: config.typeId ?? config.playTypeId,
    subId: config.subId ?? config.catalogSubId ?? config.subPlayId,
  }
  if (playBetMode && !isBetUnitValue(playBetMode)) {
    out.betMode = playBetMode
  }
  return out
}

/**
 * 任选选位面板 placeholder：前缀选位说明 + 按子玩法具体录入规则（勿用通用「再选择/输入号码；C(选位数,k)」）。
 */
export function renxuanPositionPanelPlaceholder(config: PlayConfig): string {
  const kRaw = config.renPositionCount ?? renPickCountFromConfig(config)
  const k = kRaw >= 2 && kRaw <= 5 ? kRaw : 2
  const maxPositions = 5
  const prefix = `从万、千、百、十、个中勾选至少 ${k} 个、最多 ${maxPositions} 个位置`
  const bm = (config.betMode ?? '').trim()
  const method = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''} ${bm}`

  // —— 单式票面 ——
  if (isRenxuanPositionDanshiConfig(config)) {
    if (isZu3DanshiConfig(config)) {
      return `${prefix}，再输入两个号相同号码和一个不同号码组成一注。所选位置与号码须与开奖一致，顺序不限。示例：112,223`
    }
    if (isZu6DanshiConfig(config)) {
      return `${prefix}，再输入三个各不相同的3个号码组成一注。所选位置与号码须与开奖一致，顺序不限。示例：012,345`
    }
    if (isHunhePlayConfig(config)) {
      return `${prefix}，再输入三个号码组成一注，所选${k}个位置的开奖号码与输入号码一致，顺序不限。示例：012,345`
    }
    const n = config.segmentLen > 0 ? config.segmentLen : k
    const example = Array.from({ length: 2 }, (_, ti) =>
      Array.from({ length: n }, (_, i) => String((i + ti * 3) % 10)).join(''),
    ).join(',')
    if (isZuxuanDanshiConfig(config)) {
      return `${prefix}，再输入 ${n} 位号码组成一注。所选位置与号码须与开奖一致，顺序不限。示例：${example}`
    }
    return `${prefix}，再输入 ${n} 位号码组成一注。所选位置与号码顺序均须与开奖一致。示例：${example}`
  }

  // —— 号池 / 和值 ——
  if (bm === 'zu24' || /组选24|zu24/i.test(method)) {
    return `${prefix}，再输入4个及以上0-9的号码，多选用逗号分隔，如：1,3,5,7`
  }
  if (bm === 'zu12' || (/组选12|zu12/i.test(method) && !/组选120|zu120/i.test(method))) {
    return `${prefix}，再从0-9中，输入1个及以上的二重号码，2个及以上的单号，两个位置由逗号分隔，如：12,3234`
  }
  if (isSixingZu6PlayConfig(config)) {
    return `${prefix}，再输入两个及以上的0-9的号码，多选用逗号分隔，如1,2`
  }
  if (
    !isZu3DanshiConfig(config) &&
    (bm === 'zu3' || (/组三|zu3/i.test(method) && !/组选3|组选30|zu30|和值|单式|_ds/i.test(method)))
  ) {
    return `${prefix}，再输入两个及以上0-9的号码，多选用逗号分隔，如1,3,5,7`
  }
  if (
    !isZu6DanshiConfig(config) &&
    !isSixingZu6PlayConfig(config) &&
    (bm === 'zu6' || (/组六|zu6/i.test(method) && !/组选6|组选60|组选120|zu60|zu120|和值|单式|_ds/i.test(method)))
  ) {
    return `${prefix}，再输入三个及以上0-9的号码，多选用逗号分隔，如1,3,5,7`
  }
  if (
    bm === 'hezhi' ||
    (/和值/.test(method) && !/尾数|跨度|单双|大小/.test(method))
  ) {
    const pool = poolFromConfig(config)
    const min = config.numberPoolMin ?? pool?.min ?? (k <= 2 ? 0 : 0)
    const max = config.numberPoolMax ?? pool?.max ?? (k <= 2 ? 18 : 27)
    return `${prefix}，再输入和值 ${min}–${max}，多选用逗号分隔（如 14,15,16）`
  }
  if (bm === 'baodan' || /包胆/.test(method)) {
    return `${prefix}，再输入一个 0–9 的号码（如 5）`
  }
  if (bm === 'kuadu' || /跨度/.test(method)) {
    const pool = poolFromConfig(config)
    const min = config.numberPoolMin ?? pool?.min ?? 0
    const max = config.numberPoolMax ?? pool?.max ?? 9
    return `${prefix}，再输入跨度 ${min}–${max}，每个数字用逗号分隔（如 0,3,9）`
  }
  if (
    bm === 'weishu' ||
    /和值尾数/.test(method) ||
    (/尾数/.test(method) && !/单双|大小/.test(method))
  ) {
    const pool = poolFromConfig(config)
    const min = config.numberPoolMin ?? pool?.min ?? 0
    const max = config.numberPoolMax ?? pool?.max ?? 9
    return `${prefix}，再输入和值尾数 ${min}–${max}，多选用逗号分隔（如 1,3,5）`
  }
  if (bm === 'zuxuan_fs' || /组选复式|zuxuan_fs/i.test(method)) {
    const min = k <= 2 ? 2 : 3
    const cn = min === 2 ? '两个' : '三个'
    const eg = min === 2 ? '1,2' : '1,3,5,7'
    return `${prefix}，再输入${cn}及以上的0-9的号码，多选用逗号分隔，如${eg}`
  }
  // 其它号池：按最少选号推断
  const minPick = zuxuanPoolMinPick(config) ?? (k <= 2 ? 2 : 3)
  const cn =
    minPick === 2 ? '两个' : minPick === 3 ? '三个' : minPick === 4 ? '四个' : `${minPick}个`
  const eg = minPick <= 2 ? '1,2' : '1,3,5,7'
  return `${prefix}，再输入${cn}及以上的0-9的号码，多选用逗号分隔，如${eg}`
}

export function groupContentPlaceholder(config: PlayConfig): string {
  // 任选选位：按子玩法拼具体录入规则（优先于下方和值/组三等扁选文案）
  if (isRenxuanNeedsPositionConfig(config)) {
    return renxuanPositionPanelPlaceholder(config)
  }
  // 和值优先：任二直选和值等勿落成组三/组六「N 个及以上」提示
  if (
    config.betMode === 'hezhi' ||
    (/和值/.test(`${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`) &&
      !/尾数|跨度|单双|大小/.test(config.playMethodLabel ?? ''))
  ) {
    const pool = poolFromConfig(config)
    if (pool) return `和值：输入 ${pool.min}–${pool.max}，多选用逗号分隔（如 14,15,16）`
    return '和值：输入和值数字，多选用逗号分隔（前三直选 0–27，前二/任二 0–18，快三 3–18）'
  }
  {
    const zuxuanText = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''} ${config.betMode ?? ''}`
    // 组三/组六复式号池提示；单式走下方任选选位/组选单式文案
    if (
      !isZu3DanshiConfig(config) &&
      !/单式|_ds/i.test(zuxuanText) &&
      (config.betMode === 'zu3' ||
        (/组三|zu3/i.test(zuxuanText) && !/组选3|组选30|zu30|和值/i.test(zuxuanText)))
    ) {
      return '输入两个及以上0-9的号码，多选用逗号分隔，如1.3.5.7'
    }
    // 四星/任四组选6 走下方选位文案或「两个及以上」；此处仅三星组六
    if (
      !isZu6DanshiConfig(config) &&
      !isSixingZu6PlayConfig(config) &&
      !/单式|_ds/i.test(zuxuanText) &&
      (config.betMode === 'zu6' ||
        (/组六|zu6/i.test(zuxuanText) && !/组选6|组选60|组选120|zu60|zu120|和值/i.test(zuxuanText)))
    ) {
      return '输入三个及以上0-9的号码，多选用逗号分隔，如1.3.5.7'
    }
    if (config.betMode === 'baodan' || /包胆/.test(zuxuanText)) {
      return '包胆：输入一个 0–9 的号码（如 5）'
    }
    if (
      config.betMode === 'weishu' ||
      /和值尾数/.test(zuxuanText) ||
      (/尾数/.test(zuxuanText) && !/单双|大小|对碰|不中|生肖/.test(zuxuanText))
    ) {
      const pool = poolFromConfig(config)
      const min = pool?.min ?? 0
      const max = pool?.max ?? 9
      return `和值尾数：输入 ${min}–${max}，多选用逗号分隔（如 1,3,5）`
    }
    if (isBudingweiPlayConfig(config)) {
      if (inferBudingweiNeed(config) <= 1) {
        return '一码不定位：输入 1–2 个 0–9 号码，每个数字用逗号分隔（如 1,2）'
      }
      if (isWuxingBudingweiMulti(config)) {
        return '五星不定位：至少 4 个 0–9 号码，每个数字用逗号分隔（如 1,2,3,4）'
      }
      if (inferBudingweiNeed(config) === 3) {
        return '三码不定位：至少 3 个 0–9 号码，每个数字用逗号分隔（如 1,2,3,4）'
      }
      return '二码不定位：至少 2 个 0–9 号码，每个数字用逗号分隔（如 1,2）'
    }
  }
  if (config.inputMode === 'lhc_num') {
    const mode = config.betMode ?? ''
    if (mode === 'buzhong' || mode === 'xuanyi') {
      return '六合彩：选 1–49 号码，逗号分隔（注数按玩法最少选号数计算）'
    }
    if (isLhcTemaPlayConfig(config)) {
      return '特码：点选上方属性；号码在下方输入 1–49，逗号分隔（提交格式：号码|属性|波色）'
    }
    if (isLhcErquanzhongTuotouConfig(config)) {
      return `二全中拖头：输入 ${LHC_ERQUANZHONG_NUM_MIN_PICKS}–${LHC_ERQUANZHONG_NUM_MAX_PICKS} 个 01–49 号码，逗号分隔（首个为胆，其余为拖；如 01,13,25）`
    }
    if (isLhcErquanzhongFushiConfig(config)) {
      return `二全中复式：输入 ${LHC_ERQUANZHONG_NUM_MIN_PICKS}–${LHC_ERQUANZHONG_NUM_MAX_PICKS} 个 01–49 号码，逗号分隔（如 01,13,25）`
    }
    return '六合彩：选 1–49 号码，逗号分隔（如 01,13,25）'
  }
  if (config.inputMode === 'lhc_zodiac') {
    if (isLhcSxDuipengConfig(config) || config.betMode === 'sx_dp') {
      return '生肖对碰：点选 2 个生肖（如 马|蛇）；每个生肖对应固定号码'
    }
    return '生肖：马,龙,蛇 等，逗号分隔'
  }
  if (config.inputMode === 'lhc_tail') {
    if (isLhcWsDuipengConfig(config) || config.betMode === 'ws_dp') {
      return '尾数对碰：点选 2 个尾数（如 0|1）；每个尾数对应固定号码'
    }
    return '尾数：0–9，逗号分隔'
  }
  if (config.inputMode === 'lhc_attr') {
    if (isLhcSwDuipengConfig(config) || config.betMode === 'sw_dp') {
      return '生尾对碰：各选 1 个生肖和 1 个尾数（如 马|0）；两侧展开号码对碰'
    }
    if (config.betMode === 'zongxiao') return '总肖：二肖–七肖，逗号分隔（如 二肖,五肖）'
    if (config.betMode === 'tematouwei') return '特码头尾：头0–头4、尾0–尾9，逗号分隔'
    if (config.betMode === 'qima') return '七码：单/双/大/小 + 0–7，如 双1'
    return '选属性项，逗号分隔（如 红,金,家）'
  }
  if (config.betMode === 'tuotou') {
    return '拖头：胆码|拖码，如 01,02|03,04,05'
  }
  if ((config.betMode ?? '').endsWith('_dp')) {
    return '对碰：A组|B组，如 马|龙 或 01,02|03,04'
  }
  // 前二/后二组选单式：每注 2 位；前三等每注 3 位；多注逗号分隔（勿落成「选号池 0-9」）
  if (isZuxuanDanshiConfig(config)) {
    const n = config.segmentLen > 1 ? config.segmentLen : zuxuanStarLen(config)
    if (n === 2) {
      return '组选单式：每注 2 位数字（不含对子），多注用逗号分隔；组选形态相同只计 1 注（如 12,21 计 1 注）'
    }
    return `组选单式：每注 ${n} 位数字（不含豹子），多注用逗号分隔；组选形态相同只计 1 注（如 123,321 计 1 注）`
  }
  if (
    config.subPlayId === 'zhixuan_ds' ||
    config.betMode === 'danshi' ||
    (isSscDanshiLikeConfig(config) && config.betMode !== 'hunhe')
  ) {
    const n = config.segmentLen > 0 ? config.segmentLen : 2
    const eg = n === 2 ? '12,13,14,12' : n === 3 ? '123,124,123' : `${'1'.repeat(n)},${'2'.repeat(n)}`
    return `每注 ${n} 位数字，多注用逗号分隔；重复号码只计 1 注（如 ${eg} 计 ${n === 2 ? 3 : 2} 注）`
  }
  if (config.subPlayId === 'zhixuan_fs' && config.inputMode === 'multiline') {
    // 与 groupDigitInputHint 对齐（数字录入框主路径）；此处供裸 textarea 等兜底
    const labels = config.segmentLabels ?? []
    const fmt = (lab: string) => (/^[万千百十个]$/.test(lab.trim()) ? `${lab.trim()}位` : lab.trim())
    const first = fmt(labels[0] ?? '第1位')
    const last = fmt(labels[config.segmentLen - 1] ?? `第${config.segmentLen}位`)
    const eg = Array.from({ length: config.segmentLen }, (_, i) => {
      const n = (i % 4) + 2
      return Array.from({ length: n }, (_, k) => String((i + k) % 10)).join('')
    }).join(',')
    return `请对应${first}到${last}，以“，”分隔，输入对应位置的号码，每一位置皆要输入号码；如：${eg}`
  }
  if (isLonghuPlayConfig(config)) {
    return `龙虎：${longhuPickHint(config)}，逗号分隔`
  }
  if (isPerPosDxdsPlayConfig(config)) {
    const labels = config.segmentLabels ?? []
    const posHint = labels.length
      ? labels.map((l) => (/^[万千百十个]$/.test(l.trim()) ? `${l.trim()}位` : l.trim())).join('、')
      : '每位'
    return `大小单双：${posHint}各选一个（大/小/单/双）`
  }
  if (isWuxingSumDxdsPlayConfig(config)) {
    const label = config.playMethodLabel ?? ''
    if (/尾数大小/.test(label) || (config.betMode === 'daxiao' && /尾数/.test(label))) {
      return '尾数大小：仅选一个（大 或 小）'
    }
    if (/尾数单双/.test(label) || (config.betMode === 'danshuang' && /尾数/.test(label))) {
      return '尾数单双：仅选一个（单 或 双）'
    }
    const isDx = config.betMode === 'daxiao' || /和值大小/.test(label)
    return isDx ? '五星和值大小：仅选一个（大 或 小）' : '五星和值单双：仅选一个（单 或 双）'
  }
  if (config.betMode === 'daxiao' || config.betMode === 'danshuang' || config.betMode === 'dxds') {
    return '大小单双：大、小、单、双，逗号分隔'
  }
  if (config.betMode === 'kuadu' || /跨度/.test(config.playMethodLabel ?? '')) {
    const pool = poolFromConfig(config)
    const min = pool?.min ?? 0
    const max = pool?.max ?? 9
    return `跨度：输入 ${min}–${max}，每个数字用逗号分隔（如 0,3,9）`
  }
  if (config.betMode === 'weishu') {
    const pool = poolFromConfig(config)
    const min = pool?.min ?? 0
    const max = pool?.max ?? 9
    return `和值尾数：输入 ${min}–${max}，多选用逗号分隔（如 1,3,5）`
  }
  if (config.betMode === 'hunhe') {
    const digitLen = hunheDigitLenFromConfig(config)
    return `混合组选：每注 ${digitLen} 位，不含豹子；组选形态相同只计 1 注（如 123,321 计 1 注）`
  }
  if (config.betMode === 'zu24' || /组选24|zu24/i.test(config.playMethodLabel ?? '')) {
    return '输入4个及以上0-9的号码，多选用逗号分隔，如：1,3,5,7'
  }
  if (config.betMode === 'zu120' || /组选120|zu120/i.test(config.playMethodLabel ?? '')) {
    return '输入5个及以上0-9的号码，多选用逗号分隔，如：1,3,5,7,9'
  }
  if (isZuDualPlayConfig(config)) {
    return zuDualFormatHint(config)
  }
  if (isSixingZu6PlayConfig(config)) {
    return '输入两个及以上的0-9的号码，多选用逗号分隔，如1,2'
  }
  if (isWuxingQuweiDigitPlayConfig(config)) {
    return wuxingQuweiFormatHint(config)
  }
  if (config.betMode === 'teshu') {
    return '特殊号：豹子、对子、顺子（PC28 另含极大/极小），多选各计 1 注'
  }
  if (config.betMode === 'longhubao') {
    return '龙虎豹：龙、虎、豹，逗号分隔'
  }
  if (config.playTypeId === 'renxuan_fs' || config.playTypeId === 'renxuan_ds') {
    return `任选：${poolRangeHint(config)}，逗号分隔`
  }
  if (config.betMode === 'dingwei' && config.inputMode === 'multiline' && config.segmentLen > 1) {
    const labels = config.segmentLabels.join('、')
    return `定位胆：${labels} 各位分别选号，每位 0-9，多选用逗号分隔`
  }
  if (config.playTypeId === 'dingwei' || config.betMode === 'dingwei') {
    const poolHint = poolRangeHint(config)
    return `定位胆：每注一个号码${poolHint}，多注用逗号分隔`
  }
  const poolHint = poolRangeHint(config)
  return `选号池：${poolHint}，用逗号分隔`
}

function poolRangeHint(config: PlayConfig): string {
  const min = config.numberPoolMin
  const max = config.numberPoolMax
  if (min != null && max != null && (max > 9 || min > 0)) {
    const pad = max >= 11 ? '（如 01,03,05）' : '（如 1,3,5）'
    return `${min}–${max} ${pad}`
  }
  return '0–9（如 0,1,2,3）'
}
