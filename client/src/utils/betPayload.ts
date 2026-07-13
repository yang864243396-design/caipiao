import { LHC_ZODIAC_NUMBERS, lhcMinPickCount } from '@/constants/lhcPlay'
import { isBetUnitValue } from '@/constants/betModeOptions'
import {
  isCatalogPlayTypeId,
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
  // 旧订单兼容：hou4 映射为 catalog sixing
  const typeId = playTypeId === 'hou4' ? 'sixing' : playTypeId
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
    const typeId = playTypeId === 'hou4' ? 'sixing' : playTypeId
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
    return {
      digits: [],
      lines: splitGroupLines(trimmed).map((line) => parsePickTokens(line, pool)),
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
    return parts.filter((s) => new RegExp(`^\\d{${config.segmentLen}}$`).test(s)).join(',')
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

  if (config.subPlayId === 'zhixuan_ds') {
    return content.split(',').filter((t) => t.length === config.segmentLen).length || 0
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

  if (config.betMode === 'dingwei' && config.inputMode === 'multiline' && config.segmentLen > 1) {
    const lines = splitGroupLines(content)
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
  const label = `${config.guajiGroup ?? ''} ${config.playTypeLabel ?? ''} ${config.playMethodLabel ?? ''}`
  if (label.includes('五星')) return 5
  if (label.includes('四星') || label.includes('前后四')) return 4
  if (label.includes('前二') || label.includes('后二') || label.includes('前后二')) return 2
  if (config.segmentLen > 1 && config.segmentLen <= 5) return config.segmentLen
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

  if (sub === 'zhixuan_ds') {
    const parts = content.split(/[,，\s\n]+/).map((s) => s.trim()).filter(Boolean)
    if (!parts.length) {
      return { ok: false, message: `直选单式须为 ${config.segmentLen} 位数字，每注用逗号分隔` }
    }
    const valid: string[] = []
    for (const p of parts) {
      if (!/^\d+$/.test(p)) return { ok: false, message: '存在非数字内容' }
      if (p.length !== config.segmentLen) {
        return { ok: false, message: `每注须为 ${config.segmentLen} 位数字，请用逗号分隔` }
      }
      valid.push(p)
    }
    const normalized = valid.join(',')
    return { ok: true, normalized, betUnits: valid.length }
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
      return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
    }
    if (!isValidDigitPoolLine(content)) {
      return { ok: false, message: '选号格式不合法，请使用 0-9 并以逗号分隔' }
    }
    const pool = parsePickTokens(content)
    if (!pool.length) return { ok: false, message: '选号池不能为空' }
    const normalized = [...new Set(pool)].join(',')
    return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
  }

  if (config.betMode === 'dingwei' && config.inputMode === 'multiline' && config.segmentLen > 1) {
    const lines = splitGroupLines(content)
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
      const digits = parsePickTokens(line, poolCfg)
      if (digits.length) hasAny = true
      normalizedLines.push([...new Set(digits)].join(','))
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
    const pool = parsePickTokens(content, poolCfg)
    if (!pool.length) return { ok: false, message: `选号须在 ${poolCfg.min}–${poolCfg.max} 范围内` }
    const normalized = [...new Set(pool)].join(',')
    return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
  }
  if (!isValidDigitPoolLine(content)) {
    return { ok: false, message: '选号格式不合法，请使用 0-9 并以逗号分隔每注' }
  }
  const pool = parsePickTokens(content)
  if (!pool.length) return { ok: false, message: '选号无效' }
  const normalized = [...new Set(pool)].join(',')
  return { ok: true, normalized, betUnits: countBetUnits(config, normalized) }
}

export interface SchemeGroupsValidation {
  ok: boolean
  normalized: string[]
  invalidIndexes: number[]
  message: string
}

/** 校验全部方案分组；返回不合法组下标 */
export function validateSchemeGroups(config: PlayConfig, groups: string[]): SchemeGroupsValidation {
  const normalized: string[] = []
  const invalidIndexes: number[] = []
  for (let i = 0; i < groups.length; i++) {
    const raw = groups[i]?.trim() ?? ''
    if (!raw) {
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
  if ((config.betMode ?? '').endsWith('_dp')) {
    return '对碰：A组|B组，如 马|龙 或 01,02|03,04'
  }
  if (config.subPlayId === 'zhixuan_ds') {
    return `每注 ${config.segmentLen} 位数字，多注用逗号分隔（如 1234,5678）`
  }
  if (config.subPlayId === 'zhixuan_fs' && config.inputMode === 'multiline') {
    const labels = config.segmentLabels.join('、')
    const poolHint = poolRangeHint(config)
    return `直选复式：按位分行输入，共 ${config.segmentLen} 行（${labels}），每位用逗号分隔${poolHint}`
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
