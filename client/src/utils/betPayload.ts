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

/** 不定位码数：一码/二码/三码 */
function inferBudingweiNeed(config: PlayConfig): number {
  const text =
    `${config.catalogSubId ?? ''} ${config.subPlayId} ${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''}`.toLowerCase()
  if (text.includes('三码') || text.includes('_3ma') || text.includes('3ma')) return 3
  if (text.includes('二码') || text.includes('_2ma') || text.includes('2ma')) return 2
  return 1
}

function isBudingweiPlayConfig(config: PlayConfig): boolean {
  if (config.betMode === 'budingwei') return true
  const tid = (config.playTypeId || '').toLowerCase()
  if (tid === 'budingwei' || tid === 'g009' || tid === 'g004') return true
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
    // 直选单式：相同号码重复录入只计 1 注（如 12,13,14,15,12 → 4；12,12,12 → 1）
    return dedupeDanshiTokens(content, config.segmentLen).length || 0
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

  if (config.subPlayId === 'zhixuan_fs' && config.inputMode === 'multiline') {
    const lines = content.split('\n').filter(Boolean)
    if (lines.length < config.segmentLen) return 0
    // 各位同一单码（豹子/对子）：第三方网页计 0 注且无法下注
    if (isZhixuanFushiBaoziLines(lines, config.segmentLen)) return 0
    let units = 1
    for (let i = 0; i < config.segmentLen; i++) {
      const n = parsePickTokens(lines[i] ?? '').length || 1
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

  // 三星组选号池：组三 / 组六 / 通用组选复式分别计注（对齐第三方；勿把组三+组六混算）
  const zuxuanText = `${config.betMode ?? ''} ${config.subPlayId} ${config.catalogSubId ?? ''} ${config.playMethodLabel ?? ''}`
  const isZuPool =
    config.segmentLen === 3 &&
    (config.subPlayId === 'zuxuan_fs' ||
      config.betMode === 'zu3' ||
      config.betMode === 'zu6' ||
      config.betMode === 'zuxuan_fs' ||
      /组三|组六|组选复式/.test(zuxuanText))
  if (isZuPool) {
    const n = pool.length
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
    const parts = content.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
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
      const uniq = dedupeZuxuanDanshiTokens(content, config.segmentLen)
      if (!uniq.length) {
        return {
          ok: false,
          message: `组选单式须为 ${config.segmentLen} 位且各位不全相同（不含对子/豹子），组选形态相同只计 1 注`,
        }
      }
      return { ok: true, normalized: uniq.join(','), betUnits: uniq.length }
    }
    const uniq = dedupeDanshiTokens(content, config.segmentLen)
    return { ok: true, normalized: uniq.join(','), betUnits: uniq.length }
  }

  if (sub === 'zhixuan_fs' && config.segmentLen > 1) {
    const lines = splitGroupLines(content)
    if (lines.length >= config.segmentLen) {
      const normalizedLines: string[] = []
      for (let i = 0; i < config.segmentLen; i++) {
        const line = lines[i] ?? ''
        if (!isValidDigitPoolLine(line)) {
          return { ok: false, message: `第 ${i + 1} 位选号格式不合法，请使用 0-9 并以逗号分隔` }
        }
        const digits = parsePickTokens(line)
        if (!digits.length) return { ok: false, message: `第 ${i + 1} 位选号无效` }
        normalizedLines.push([...new Set(digits)].join(','))
      }
      const normalized = normalizedLines.join('\n')
      if (isZhixuanFushiBaoziLines(normalizedLines, config.segmentLen)) {
        return {
          ok: false,
          message: '直选复式不含豹子/对子（各位同一单码），请更换号码或改用直选单式',
        }
      }
      return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
    }
    if (!isValidDigitPoolLine(content)) {
      return { ok: false, message: '选号格式不合法，请使用 0-9 并以逗号分隔' }
    }
    const pool = parsePickTokens(content)
    if (!pool.length) return { ok: false, message: '选号池不能为空' }
    const normalized = [...new Set(pool)].join(',')
    if (new Set(pool).size === 1) {
      return {
        ok: false,
        message: '直选复式不含豹子/对子（各位同一单码），请更换号码或改用直选单式',
      }
    }
    return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
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

  const specialBetModes = new Set([
    'hezhi',
    'kuadu',
    'longhu',
    'longhuhe',
    'dxds',
    'daxiao',
    'danshuang',
    'budingwei',
    'zuhe',
    'baodan',
    'hunhe',
    'weishu',
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
      if (betUnits <= 0) {
        const digitLen = hunheDigitLenFromConfig(config)
        return {
          ok: false,
          message: `混合组选：每注 ${digitLen} 位，不含豹子；组选形态相同只计 1 注（如 123 与 321）`,
        }
      }
      return { ok: true, normalized: content, betUnits }
    }
    if (config.betMode === 'teshu') {
      if (betUnits <= 0) {
        return { ok: false, message: '特殊号：请选择豹子、对子、顺子等，多选以逗号分隔' }
      }
      return { ok: true, normalized: content, betUnits }
    }
    return { ok: true, normalized: content, betUnits: betUnits || 1 }
  }

  if (config.playTemplate === 'pc28_std' && config.playMethodLabel?.trim() === '和值') {
    const pool = { min: 0, max: 27 }
    const tokens = parsePickTokens(content, pool)
    if (!tokens.length) {
      return { ok: false, message: '和值须在 0–27 范围内，逗号分隔' }
    }
    const normalized = tokens.join(',')
    return { ok: true, normalized, betUnits: tokens.length }
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
  for (let i = 0; i < groups.length; i++) {
    // 保留前导/尾随换行空位：",,12,," → "\n\n1,2\n\n"；trim 会压成万位 "1,2\n\n\n\n"
    const raw = String(groups[i] ?? '').replace(/\r/g, '')
    if (isBlankGroupContent(raw)) {
      invalidIndexes.push(i)
      normalized.push('')
      continue
    }
    const r = validateGroupContent(config, raw)
    if (!r.ok) {
      invalidIndexes.push(i)
      normalized.push('')
    } else {
      normalized.push(r.normalized)
    }
  }
  const ok = invalidIndexes.length === 0
  const message = ok
    ? ''
    : invalidIndexes.length === 1
      ? `第 ${invalidIndexes[0]! + 1} 组输入内容与当前玩法不符，已清空该组`
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
    return '和值：输入和值数字，逗号分隔（快三 3–18，PC28 0–27）'
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
