import { LHC_ZODIAC_NUMBERS, lhcMinPickCount } from '@/constants/lhcPlay'
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
  hunheDigitLenFromConfig,
} from '@/utils/playInputProfile'
import { segmentBetMultiplier } from '@/utils/runTypeMatrix'
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

/** 任选单式是否需要万千百十个位选（对齐第三方） */
export function isRenxuanPositionDanshiConfig(config: PlayConfig): boolean {
  const k = config.renPositionCount ?? 0
  if (k < 2 || k > 5) return false
  const isRen =
    config.playTypeId === 'renxuan' ||
    config.playTypeId === 'g011' ||
    config.guajiGroup === '任选' ||
    (config.playTypeLabel ?? '') === '任选'
  if (!isRen) return false
  const bm = config.betMode ?? ''
  if (bm === 'danshi' || bm === 'zuxuan_ds') return true
  const label = `${config.playMethodLabel ?? ''}`
  return label.includes('直选单式') || label.includes('组选单式')
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
    const positions = extractSscPositionNames(text.slice(0, pipe).trim())
    const picks = text.slice(pipe + 1).trim()
    if (positions.length >= want) {
      return { positions: positions.slice(0, want), picks }
    }
  }
  const lines = text.split(/\n/).map((l) => l.trim())
  if (lines.length >= 1) {
    const positions = extractSscPositionNames(lines[0] ?? '')
    if (positions.length >= want) {
      return {
        positions: positions.slice(0, want),
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
  return raw
    .split(/[\s,，\n]+/)
    .map((s) => s.trim())
    .filter((s) => /^[0-9]$/.test(s))
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
  for (const p of parts) {
    if (!/^\d{1,2}$/.test(p)) continue
    const n = Number(p)
    if (n < min || n > max) continue
    const tok = max >= 11 ? String(n).padStart(2, '0') : String(n)
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
  return 1
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
  if (betMode === 'sx_dp') return tokens.length
  if (betMode === 'ws_dp') {
    return tokens.filter((s) => /^[0-9]$/.test(s)).length
  }
  const nums = parseLhcNumberTokens(tokens.join(','))
  if (nums.length) return nums.length
  if (betMode === 'sw_dp') {
    const zodiacNums = new Set<string>()
    for (const z of tokens) {
      for (const n of LHC_ZODIAC_NUMBERS[z] ?? []) zodiacNums.add(n)
      if (/^[0-9]$/.test(z)) {
        for (let n = 1; n <= 49; n++) {
          if (String(n % 10) === z) zodiacNums.add(String(n).padStart(2, '0'))
        }
      }
    }
    return zodiacNums.size || tokens.length
  }
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
    return parseLhcNumberTokens(content).length
  }
  if (betMode.endsWith('_dp')) {
    const sep = content.includes('|') ? '|' : content.includes('#') ? '#' : ''
    if (sep) {
      const [a, b] = content.split(sep)
      const units = lhcDuipengGroupSize(betMode, a ?? '') * lhcDuipengGroupSize(betMode, b ?? '')
      return units || (content ? 1 : 0)
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
    return {
      digits: trimmed
        .split(/[,，\s]+/)
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
    return tokens.length
  }

  // 不定位：一码=选几个号几注（最多2）；二码/三码=C(n,k)（对齐第三方 / guajibet）
  // 五星二码/三码：第三方要求至少 4 个号
  if (isBudingweiPlayConfig(config)) {
    const pool = poolFromConfig(config) ?? { min: 0, max: 9 }
    const tokens = [...new Set(parsePickTokens(content, pool))]
    const need = inferBudingweiNeed(config)
    const label = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''}`
    const wuxingMulti = label.includes('五星') && need >= 2
    if (wuxingMulti && tokens.length < 4) return 0
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
    const pool = parseLhcNumberTokens(content)
    if (!pool.length) return 0
    const betMode = config.betMode ?? ''
    const subId = config.catalogSubId ?? config.subPlayId
    const min = lhcMinPickCount(betMode, subId)
    if (betMode === 'fushi' || betMode === 'buzhong' || betMode === 'xuanyi') {
      return comboCount(pool.length, min)
    }
    if (betMode === 'tuotou' && content.includes('|')) {
      const [dan, tuo] = content.split('|')
      const d = parseLhcNumberTokens(dan ?? '').length
      const t = parseLhcNumberTokens(tuo ?? '').length
      return d * comboCount(t, Math.max(min - 1, 1))
    }
    return pool.length
  }
  if (config.inputMode === 'lhc_zodiac' || config.inputMode === 'lhc_tail' || config.inputMode === 'lhc_attr') {
    const parts = content.split(/[,，]/).map((s) => s.trim()).filter(Boolean)
    return parts.length || 0
  }

  if (isRenxuanPositionDanshiConfig(config)) {
    const k = config.renPositionCount ?? renPickCountFromConfig(config)
    const digitLen = config.segmentLen > 0 ? config.segmentLen : k
    const { positions, picks } = parseRenxuanPositionContent(content, k)
    if (positions.length < k || !picks.trim()) return 0
    if (isZuxuanDanshiConfig(config)) {
      return dedupeZuxuanDanshiTokens(picks, digitLen).length
    }
    return dedupeDanshiTokens(picks, digitLen).length || 0
  }

  if (isSscDanshiLikeConfig(config)) {
    // 组选单式：排除对子/豹子 + 形态去重（11,12,22,13 → 2；12,21 → 1）
    if (isZuxuanDanshiConfig(config)) {
      return applySegmentBetMultiplier(
        config,
        dedupeZuxuanDanshiTokens(content, config.segmentLen).length,
      )
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
  if (
    config.inputMode === 'multiline' &&
    config.segmentLen > 1 &&
    (config.betMode === 'zuhe' ||
      config.subPlayId === 'zuhe' ||
      /(^|[^组选])组合/.test(config.playMethodLabel ?? '') ||
      (config.playMethodLabel ?? '').endsWith('组合') ||
      (config.playMethodLabel ?? '').includes('直选组合'))
  ) {
    const lines = splitGroupLines(content)
    let units = 1
    for (let i = 0; i < config.segmentLen; i++) {
      const n = parsePickTokens(lines[i] ?? '').length
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

  // 组选号池：二星 C(n,2)；三星组三/组六/通用组选复式（对齐 guajibet countZuxuanFushiBetNums）
  const zuxuanText = `${config.betMode ?? ''} ${config.subPlayId} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''}`
  const isZuFsLike =
    config.subPlayId === 'zuxuan_fs' ||
    config.betMode === 'zu3' ||
    config.betMode === 'zu6' ||
    config.betMode === 'zuxuan_fs' ||
    /组三|组六|组选复式/.test(zuxuanText)
  if (isZuFsLike && config.segmentLen === 2) {
    const n = new Set(pool).size
    if (n < 2) return 0
    return applySegmentBetMultiplier(config, (n * (n - 1)) / 2)
  }
  const isZuPool = config.segmentLen === 3 && isZuFsLike
  if (isZuPool) {
    const n = new Set(pool).size
    const isZu6Only =
      config.betMode === 'zu6' ||
      ((/组六|zu6/i.test(zuxuanText) && !/组选6|组选60|组选120|zu60|zu120/i.test(zuxuanText)))
    const isZu3Only =
      !isZu6Only &&
      (config.betMode === 'zu3' || /组三|zu3/i.test(zuxuanText))
    if (isZu6Only) {
      if (n < 3) return 0
      return applySegmentBetMultiplier(config, (n * (n - 1) * (n - 2)) / 6)
    }
    if (isZu3Only) {
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

function applySegmentBetMultiplier(config: PlayConfig, units: number): number {
  if (units <= 0) return units
  const m = segmentBetMultiplier(config.guajiGroup ?? config.playTypeLabel ?? '')
  return m > 1 ? units * m : units
}

function isSscRenxuanConfig(config: PlayConfig): boolean {
  return (
    config.playTypeId === 'renxuan' ||
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
  const label = `${config.guajiGroup ?? ''} ${config.playTypeLabel ?? ''} ${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  const fromRen = renPickCountFromConfig(config)
  if (fromRen >= 2 && fromRen <= 5 && /任|ren/i.test(label)) return fromRen
  if (label.includes('五星')) return 5
  if (label.includes('四星') || label.includes('前后四') || label.includes('任四') || label.includes('任选四')) return 4
  if (label.includes('任三') || label.includes('任选三')) return 3
  if (
    label.includes('前二') ||
    label.includes('后二') ||
    label.includes('前后二') ||
    label.includes('任二') ||
    label.includes('任选二')
  ) {
    return 2
  }
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

/** 和值单组最大注数（前三直选和值等，对齐第三方） */
export const HEZHI_MAX_BET_UNITS = 900
export const HEZHI_MAX_BET_UNITS_MSG = '投注注数超过最大投注注数:900'

/** 和值尾数单组最大注数（前三和值尾数等，对齐第三方） */
export const WEISHU_MAX_BET_UNITS = 9
export const WEISHU_MAX_BET_UNITS_MSG = '投注注数超过最大投注注数:9'

/** 直选组合单组最大注数（前三组合等，对齐第三方） */
export const ZUHE_MAX_BET_UNITS = 2700
export const ZUHE_MAX_BET_UNITS_MSG = '投注注数超过最大投注注数:2700'

/**
 * 直选复式单组最大注数（对齐第三方）：满号位积 − 对子/豹子（各位同一号码，共 P 组）。
 * P=每位号池大小、n=位数：max = P^n − P。
 * 例：前二/后二 10^2−10=90；前三/后三 10^3−10=990；四星 9990；五星 99990；十一选五(P=11)前二 121−11=110。
 */
export function zhixuanFushiMaxBetUnits(config: PlayConfig): number {
  const pool = poolFromConfig(config)
  const size = pool ? pool.max - pool.min + 1 : 10
  const n = Math.max(1, config.segmentLen || 1)
  if (size <= 1 || n <= 1) return 0
  return Math.pow(size, n) - size
}

/** 是否「超过最大投注注数」类提示（保存时原样弹窗、不清空内容） */
export function isMaxBetUnitsExceededMessage(message: string): boolean {
  return String(message ?? '').startsWith('投注注数超过最大投注注数:')
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

/** 单行选号池是否仅含 0-9 单 digit，逗号/空格分隔 */
function isValidDigitPoolLine(raw: string): boolean {
  const t = raw.trim()
  if (!t) return false
  const parts = t.split(/[,，\s]+/).map((s) => s.trim()).filter(Boolean)
  if (!parts.length) return false
  return parts.every((p) => /^[0-9]$/.test(p))
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
    return { ok: true, normalized, betUnits }
  }

  if (isRenxuanPositionDanshiConfig(config)) {
    const k = config.renPositionCount ?? renPickCountFromConfig(config)
    const digitLen = config.segmentLen > 0 ? config.segmentLen : k
    const { positions, picks } = parseRenxuanPositionContent(content, k)
    if (positions.length < k) {
      return { ok: false, message: `请从万千百十个中勾选 ${k} 个位置` }
    }
    if (!picks.trim()) {
      return { ok: false, message: `请输入 ${digitLen} 位号码，每注用逗号分隔` }
    }
    const parts = picks.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
    for (const p of parts) {
      if (!/^\d+$/.test(p)) return { ok: false, message: '号码存在非数字内容' }
      if (p.length !== digitLen) {
        return { ok: false, message: `每注须为 ${digitLen} 位数字，请用逗号分隔` }
      }
    }
    const uniq = isZuxuanDanshiConfig(config)
      ? dedupeZuxuanDanshiTokens(picks, digitLen)
      : dedupeDanshiTokens(picks, digitLen)
    if (!uniq.length) return { ok: false, message: '选号无效' }
    const normalized = buildRenxuanPositionContent(positions, uniq.join(','))
    return { ok: true, normalized, betUnits: uniq.length }
  }

  if (isSscDanshiLikeConfig(config)) {
    const seg = config.segmentLen > 0 ? config.segmentLen : 0
    // 冷热/复式残留的按位号池先展开再校验
    let danshiRaw = content
    if (seg > 1 && isZhixuanPositionPoolContent(danshiRaw, seg)) {
      danshiRaw = expandZhixuanPositionPoolToDanshi(danshiRaw, seg)
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
    if (isZuxuanDanshiConfig(config)) {
      const uniq = dedupeZuxuanDanshiTokens(danshiRaw, config.segmentLen)
      if (!uniq.length) {
        return {
          ok: false,
          message: `组选单式须为 ${config.segmentLen} 位且各位不全相同（不含对子/豹子），组选形态相同只计 1 注`,
        }
      }
      // 前中后三/前后二三四等跨段玩法按段倍乘（前中后三×3），与后端 evaluateMultiZone、第三方一致
      return { ok: true, normalized: uniq.join(','), betUnits: applySegmentBetMultiplier(config, uniq.length) }
    }
    const uniq = dedupeDanshiTokens(danshiRaw, config.segmentLen)
    if (uniq.length && uniq.every(isBaoziDigitTicket)) {
      return { ok: false, message: ZHIXUAN_DANSHI_SOLO_BAOZI_MSG }
    }
    // 前中后三/前后二三四等跨段玩法按段倍乘（前中后三×3），与后端 evaluateMultiZone、第三方一致
    return { ok: true, normalized: uniq.join(','), betUnits: applySegmentBetMultiplier(config, uniq.length) }
  }

  // 直选复式 / 直选组合：按位号池，每一位都必须有号。
  // 禁止把「123，，」→「1,2,3\\n\\n」再误归一成单行「1,2,3」（会被录入框当成万=1/千=2/百=3）。
  if (
    (isZhixuanFushiPlayConfig(config) || isZhixuanZuhePlayConfig(config)) &&
    config.segmentLen > 1
  ) {
    const rawContent = String(raw ?? '').replace(/\r/g, '')
    const lines = splitGroupLinesPad(rawContent, config.segmentLen).slice(0, config.segmentLen)
    const normalizedLines: string[] = []
    for (let i = 0; i < config.segmentLen; i++) {
      const line = lines[i] ?? ''
      const pos = config.segmentLabels?.[i] ?? `第 ${i + 1} 位`
      if (!line.trim()) {
        return { ok: false, message: `${pos}选号不能为空，每一位都需要输入号码` }
      }
      if (!isValidDigitPoolLine(line)) {
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
    if (isZhixuanZuhePlayConfig(config) && betUnits > ZUHE_MAX_BET_UNITS) {
      return { ok: false, message: ZUHE_MAX_BET_UNITS_MSG }
    }
    // 直选复式：满号位积 − 对子/豹子（P^n − P）为单组上限（前二=90、前三=990…）
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
    if (betUnits > HEZHI_MAX_BET_UNITS) {
      return { ok: false, message: HEZHI_MAX_BET_UNITS_MSG }
    }
    return { ok: true, normalized, betUnits }
  }

  // 和值尾数：0–9 逗号分隔；单组最多 9 注
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
    if (betUnits > WEISHU_MAX_BET_UNITS) {
      return { ok: false, message: WEISHU_MAX_BET_UNITS_MSG }
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

  const specialBetModes = new Set([
    'kuadu',
    'longhu',
    'longhuhe',
    'dxds',
    'daxiao',
    'danshuang',
    'budingwei',
    // zuhe 已在上方按位校验 + 2700 注上限
    // baodan / weishu 已在上方单独校验
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
      if (isSchemeSoloBaoziContent(config, content)) {
        return { ok: false, message: SOLO_BAOZI_FORBIDDEN_MSG }
      }
      const digitLen = hunheDigitLenFromConfig(config)
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
      if (betUnits <= 0) {
        return { ok: false, message: '特殊号：请选择豹子、对子、顺子等，多选以逗号分隔' }
      }
      return { ok: true, normalized: content, betUnits }
    }
    return { ok: true, normalized: content, betUnits: betUnits || 1 }
  }

  // 和值/跨度/龙虎等特殊玩法：允许非空自由文本（与后端 validateGroupContent 对齐）
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
      normalized.push(isMaxBetUnitsExceededMessage(detail) ? raw : '')
    } else {
      normalized.push(r.normalized)
    }
  }
  const ok = invalidIndexes.length === 0
  if (ok) return { ok, normalized, invalidIndexes, message: '' }
  // 优先返回具体原因（如和值超 900 / 组合超 2700），便于弹窗原样展示
  if (isMaxBetUnitsExceededMessage(firstDetail)) {
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

export function groupContentPlaceholder(config: PlayConfig): string {
  {
    const zuxuanText = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''} ${config.betMode ?? ''}`
    if (
      config.betMode === 'zu3' ||
      (/组三|zu3/i.test(zuxuanText) && !/组选3|组选30|zu30/i.test(zuxuanText))
    ) {
      return '输入两个及以上0-9的号码，多选用逗号分隔，如1.3.5.7'
    }
    if (
      config.betMode === 'zu6' ||
      (/组六|zu6/i.test(zuxuanText) && !/组选6|组选60|组选120|zu60|zu120/i.test(zuxuanText))
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
  }
  if (config.inputMode === 'lhc_num') {
    const mode = config.betMode ?? ''
    if (mode === 'buzhong' || mode === 'xuanyi') {
      return '六合彩：选 1–49 号码，逗号分隔（注数按玩法最少选号数计算）'
    }
    return '六合彩：选 1–49 号码，逗号分隔（如 01,13,25）'
  }
  if (config.inputMode === 'lhc_zodiac') {
    return '生肖：马,龙,蛇 等，逗号分隔'
  }
  if (config.inputMode === 'lhc_tail') {
    return '尾数：0–9，逗号分隔'
  }
  if (config.inputMode === 'lhc_attr') {
    if (config.betMode === 'zongxiao') return '总肖：二肖–七肖，逗号分隔（如 二肖,五肖）'
    if (config.betMode === 'tematouwei') return '特码头尾：头0–头4、尾0–尾9，逗号分隔'
    if (config.betMode === 'qima') return '七码：单/双/大/小 + 0–7，如 双1'
    return '选属性项，逗号分隔（如 红,金,家）'
  }
  if (config.betMode === 'tuotou') {
    return '拖头：胆码|拖码，如 01,02|03,04,05'
  }
  if (isRenxuanPositionDanshiConfig(config)) {
    const k = config.renPositionCount ?? 2
    const n = config.segmentLen > 0 ? config.segmentLen : k
    return `先勾选 ${k} 个位置，再输入 ${n} 位号码（逗号分隔），如：千,个↵12,34`
  }
  if ((config.betMode ?? '').endsWith('_dp')) {
    return '对碰：A组|B组，如 马|龙 或 01,02|03,04'
  }
  if (config.subPlayId === 'zhixuan_ds') {
    return `每注 ${config.segmentLen} 位数字，多注用逗号分隔；重复号码只计 1 注（如 12,13,14,12 计 3 注）`
  }
  if (config.subPlayId === 'zhixuan_fs' && config.inputMode === 'multiline') {
    const labels = config.segmentLabels.join('、')
    const poolHint = poolRangeHint(config)
    return `直选复式：按位分行输入，共 ${config.segmentLen} 行（${labels}），每位用逗号分隔；不含豹子/对子（各位同一单码）${poolHint}`
  }
  if (isLonghuPlayConfig(config)) {
    return `龙虎：${longhuPickHint(config)}，逗号分隔`
  }
  if (config.betMode === 'daxiao' || config.betMode === 'danshuang' || config.betMode === 'dxds') {
    return '大小单双：大、小、单、双，逗号分隔'
  }
  if (config.betMode === 'hezhi') {
    const pool = poolFromConfig(config)
    if (pool) return `和值：输入 ${pool.min}–${pool.max}，多选用逗号分隔（如 14,15,16）`
    return '和值：输入和值数字，多选用逗号分隔（前三直选 0–27，前二 0–18，快三 3–18）'
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
