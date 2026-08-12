import {
  LHC_ERQUANZHONG_NUM_MAX_PICKS,
  LHC_SX_DUIPENG_MAX_PICKS,
  LHC_WS_DUIPENG_MAX_PICKS,
  LHC_SW_DUIPENG_MAX_PICKS,
  LHC_TAIL_NUMBERS,
  LHC_TAIL_OPTIONS,
  LHC_ZODIAC_NUMBERS,
  LHC_ZODIACS,
  isLhcErquanzhongFushiConfig,
  isLhcErquanzhongNumInputConfig,
  isLhcErquanzhongTuotouConfig,
  isLhcLianmaNumInputConfig,
  lhcLianmaNumInputLabel,
  lhcLianmaNumInputMinPicks,
  isLhcRenyiDuipengConfig,
  isLhcSxDuipengConfig,
  isLhcWsDuipengConfig,
  isLhcSwDuipengConfig,
  isLhcTemaPlayConfig,
} from '@/constants/lhcPlay'
import type { PlayConfig } from '@/utils/betPayload'
import {
  dedupeDanshiTokens,
  hunheDigitLenFromConfig,
  isHunhePlayConfig,
  isRenxuanNeedsPositionConfig,
  isRenxuanZhixuanFushiPlayConfig,
  isSixingZu6PlayConfig,
  isSscDanshiLikeConfig,
  isYixingDingweiPlayConfig,
  isZhixuanZuhePlayConfig,
  isZu3DanshiConfig,
  isZu6DanshiConfig,
  isZuxuanDanshiConfig,
  normalizeLhcTemaContent,
  normalizeZu6DanshiContent,
  normalizeZu3DanshiContent,
  normalizeHunheGroupContent,
  normalizeZuxuanDanshiContent,
  parseZuDualZones,
  uniqueDigitsFromRun,
  zuDualMetaOf,
  isZuDualPlayConfig,
  zuDualFormatHint,
  isWuxingQuweiDigitPlayConfig,
  wuxingQuweiFormatHint,
  wuxingQuweiMaxPicks,
  YIXING_MAX_PICKS_PER_POS,
} from '@/utils/betPayload'
import {
  isLonghuPlayConfigLike,
  isPc28ModeConfigLike,
  isPerPosDxdsPlayConfig,
  isWuxingSumDxdsPlayConfig,
} from '@/utils/runTypeMatrix'
import { longhuPickOptionsForConfig } from '@/utils/longhuPickOptions'

export {
  isLhcErquanzhongFushiConfig,
  isLhcErquanzhongNumInputConfig,
  isLhcErquanzhongTuotouConfig,
  isLhcLianmaNumInputConfig,
  isLhcRenyiDuipengConfig,
  isLhcSxDuipengConfig,
  isLhcTemaPlayConfig,
}

/** 任意对碰：A/B 双区输入面板（勿走 01–49 单区点选） */
export function schemeGroupUsesLhcRenyiDuipengPanel(config: PlayConfig): boolean {
  return isLhcRenyiDuipengConfig(config) || config.betMode === 'renyi_dp'
}

/** 双区组选规范化：合法则去重保序；半截输入尽量保留「头区,尾区」形态 */
function normalizeZuDualDigitContent(raw: string, config: PlayConfig): string {
  const text = String(raw ?? '')
    .replace(/，/g, ',')
    .trim()
  if (!text) return ''
  const meta = zuDualMetaOf(config)
  if (meta) {
    const zones = parseZuDualZones(text, meta.minHead, meta.minTail, meta.equalCounts)
    if (zones) return zones.normalized
  }
  const parts = text.split(',')
  if (parts.length === 2) {
    const a = uniqueDigitsFromRun(parts[0] ?? '').join('')
    const b = uniqueDigitsFromRun(parts[1] ?? '').join('')
    if (a || b) return `${a},${b}`
  }
  return uniqueDigitsFromRun(text).join('')
}

/** 龙虎类玩法（龙/虎 或 龙/虎/和 文字选号，非 0-9） */
export function isLonghuTextPickConfig(config: PlayConfig): boolean {
  return isLonghuPlayConfigLike(config)
}

/** 方案内容是否用特码专用面板（19 属性 + 0–49 输入框） */
export function schemeGroupUsesLhcTemaPanel(config: PlayConfig): boolean {
  return isLhcTemaPlayConfig(config)
}

/** 方案内容是否用 chip/选号面板（与 SchemeGroupPickPanel 同源） */
export function schemeGroupUsesPickPanel(config: PlayConfig): boolean {
  if (schemeGroupUsesLhcTemaPanel(config)) return false
  if (schemeGroupUsesLhcRenyiDuipengPanel(config)) return false
  // 二全中复式/拖头：改用逗号分隔输入框，勿点选 01–49
	if (isLhcLianmaNumInputConfig(config)) return false
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
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''}`
  // PC28 顶线和值：betMode 可能为空，仅按文案识别「和值」本身
  if (config.playTemplate === 'pc28_std' && label.trim() === '和值') return true
  // 任二直选和值等：目录 label 常为「直选和值」，须优先于组选复式/组六提示
  return /和值/.test(label) && !/尾数|跨度|单双|大小/.test(label)
}

/** 和值尾数号池（前三和值尾数等）：0–9，须逗号分隔（勿连写） */
function isWeishuPoolConfig(config: PlayConfig): boolean {
  if (config.betMode === 'weishu') return true
  const label = config.playMethodLabel ?? ''
  return /和值尾数/.test(label) || (label.includes('尾数') && !/单双|大小|对碰|不中|生肖/.test(label))
}

/** 跨度号池（前/中/后三直选跨度等）：0–9，须逗号分隔（勿连写成 039） */
function isKuaduPoolConfig(config: PlayConfig): boolean {
  if (config.betMode === 'kuadu') return true
  return /跨度/.test(config.playMethodLabel ?? '')
}

/** 不定位号池（前三一码/二码等）：0–9，须逗号分隔（如 1,2；勿连写成 12） */
function isBudingweiPoolConfig(config: PlayConfig): boolean {
  if (config.betMode === 'budingwei') return true
  const tid = String(config.playTypeId ?? '').toLowerCase()
  if (tid === 'g009' || tid === 'budingwei') return true
  const text = `${config.playTypeLabel ?? ''} ${config.playMethodLabel ?? ''} ${config.guajiGroup ?? ''}`
  return text.includes('不定位')
}

/** 组选复式号池（前二/后二/前三组选复式等）：0–9，须逗号分隔展示与录入 */
function isZuxuanFushiPoolPlay(config: PlayConfig): boolean {
  if (config.betMode === 'zuxuan_fs') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选复式|zuxuan_fs/i.test(text)
}

/** 组选24 号池：至少 4 码，逗号分隔 */
function isZu24PoolPlay(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu24') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选24|zu24/i.test(text)
}

/** 组选120 号池：至少 5 码，逗号分隔（五星组选120） */
function isZu120PoolPlay(config: PlayConfig): boolean {
  const bm = (config.betMode ?? '').trim()
  if (bm === 'zu120') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组选120|zu120/i.test(text)
}

function isZuDualPoolPlay(config: PlayConfig): boolean {
  return isZuDualPlayConfig(config)
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
  if (isWuxingQuweiDigitPlayConfig(config)) return []
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
  // 生肖对碰：十二生肖（开某投某正/反投与开出同源）
  if (isLhcSxDuipengConfig(config) || config.betMode === 'sx_dp') {
    return [...LHC_ZODIACS]
  }
  // 尾数对碰：0–9 尾
  if (isLhcWsDuipengConfig(config) || config.betMode === 'ws_dp') {
    return [...LHC_TAIL_OPTIONS]
  }
  // 生尾对碰：十二生肖 + 0–9 尾（正/反投与开出同源）
  if (isLhcSwDuipengConfig(config) || config.betMode === 'sw_dp') {
    return [...LHC_ZODIACS, ...LHC_TAIL_OPTIONS]
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
      // 五星趣味走数字输入框，勿给豹子/对子/顺子点选
      if (isWuxingQuweiDigitPlayConfig(config)) return []
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
  if (schemeGroupUsesLhcTemaPanel(config)) return false
  if (schemeGroupUsesLhcRenyiDuipengPanel(config)) return false
  // 二全中复式/拖头：输入框录入 01–49（即使未走 pick 分支也启用）
	if (isLhcLianmaNumInputConfig(config)) return true
  if (!schemeGroupUsesPickPanel(config)) return false
  if (config.inputMode === 'danshi') return false
  // 直选/组选/混合单式整注：走文本失焦面板，勿当复式按位号池
  if (isSscDanshiLikeConfig(config) || isHunhePlayConfig(config)) return false
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
 * 非任选的直选/组选/混合单式：用 SchemeGroupInputPanel 做整注文本 + 失焦校验。
 * （裸 el-input 的 blur 在定码分组场景偶发不触发；任选另走 SchemeRenxuanDanshiPanel）
 */
export function schemeGroupUsesDanshiTextInput(config: PlayConfig): boolean {
  if (isRenxuanNeedsPositionConfig(config)) return false
  if ((config.numberPoolMax ?? 9) > 9) return false
  if (config.inputMode === 'danshi') return true
  return isSscDanshiLikeConfig(config) || isHunhePlayConfig(config)
}

/** 方案内容走 SchemeGroupInputPanel（复式数字框或单式整注失焦框） */
export function schemeGroupUsesTextInputPanel(config: PlayConfig): boolean {
  return schemeGroupUsesDigitInput(config) || schemeGroupUsesDanshiTextInput(config)
}

/**
 * 变长号池（和值 0–27 / 组选和值 1–26 / 快三 3–18 等）必须逗号分隔录入。
 * 定宽补零号池（11选5 01–11、PK10）仍可连写按宽度切块。
 */
export function poolUsesCommaSeparatedInput(config: PlayConfig): boolean {
  if (isHezhiPoolConfig(config)) return true
  // 和值尾数虽为 0–9，连写会把多选拆错，与直选和值一致用逗号分隔
  if (isWeishuPoolConfig(config)) return true
  // 跨度 0–9：输入框内每个数字用逗号分隔（如 0,3,9），勿连写成 039
  if (isKuaduPoolConfig(config)) return true
  // 不定位（前三一码等）：每个数字用逗号分隔（如 1,2），勿连写成 12
  if (isBudingweiPoolConfig(config)) return true
  // 五星趣味（一帆风顺等）：每个数字用逗号分隔（如 0,3,9），勿连写成 039
  if (isWuxingQuweiDigitPlayConfig(config)) return true
  // 二全中复式/拖头：01–49 须逗号分隔（如 01,13,25），勿连写成 011325
	if (isLhcLianmaNumInputConfig(config)) return true
  // 组选复式 / 组三 / 组六 / 组选24 / 组选120：号池多选须逗号分隔（如 0,1,2），勿连写成 012
  // 组选12 为双区连写（12,34），勿按扁选逗号号池拆码
  if (
    isZuxuanFushiPoolPlay(config) ||
    isZu3PoolPlay(config) ||
    isZu6PoolPlay(config) ||
    isSixingZu6PoolPlay(config) ||
    isZu24PoolPlay(config) ||
    isZu120PoolPlay(config)
  ) {
    return true
  }
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
  const singleWidth = options.length > 0 && options.every((o) => o.length === 1)
  const push = (match: string) => {
    if (seen.has(match)) return
    seen.add(match)
    out.push(match)
  }
  for (const p of parts) {
    if (!/^\d+$/.test(p)) continue
    const n = Number(p)
    const match = options.find((o) => Number(o) === n)
    if (match) {
      push(match)
      continue
    }
    // 跨度/尾数等单位数号池：粘连录入 "039" → 0,3,9（失焦后输入框也按逗号展示）
    if (singleWidth && p.length > 1) {
      for (const ch of p) {
        const one = options.find((o) => o === ch || Number(o) === Number(ch))
        if (one) push(one)
      }
    }
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
  // 组选12/4：保留双区形态，勿拆成扁选号池
  if (isZuDualPoolPlay(config)) {
    return normalizeZuDualDigitContent(box, config)
  }
  // 直选组合：始终按位拆分（勿因 ruleId 误判组选6 而扁选）
  if (isZhixuanZuhePlayConfig(config) && segLen > 1) {
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
  // 号池型（组选复式/组三/组六/和值等）：单行逗号多选，勿按 segmentLen 拆成按位
  if (
    segLen <= 1 ||
    config.inputMode === 'pool' ||
    isZuxuanFushiPoolPlay(config) ||
    isZu3PoolPlay(config) ||
    isZu6PoolPlay(config) ||
    isSixingZu6PoolPlay(config)
  ) {
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
  if (c.replace(/[\s,，|#]/g, '') === '') return ''
  // 二全中拖头：旧 胆|拖 展成逗号扁选展示
	if (isLhcErquanzhongTuotouConfig(config) || /拖头$/.test(lhcLianmaNumInputLabel(config))) {
    const toks = parseDigitSegmentTokens(c.replace(/[|#]/g, ','), config)
    return toks.join(',')
  }
  const segLen = Math.max(1, config.segmentLen || 1)
  if (isZuDualPoolPlay(config)) {
    return normalizeZuDualDigitContent(c, config)
  }
  // 直选组合：按位压缩展示（勿误走组选6 扁选号池）
  if (isZhixuanZuhePlayConfig(config) && segLen > 1) {
    if (!c.includes('\n')) {
      const parts = c.split(/[,，]/)
      if (parts.some((p) => /[0-9]/.test(p)) && parts.length > 0 && parts.length <= segLen) {
        return Array.from({ length: segLen }, (_, i) =>
          (parts[i] ?? '').replace(/[^0-9]/g, ''),
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
  if (
    segLen <= 1 ||
    config.inputMode === 'pool' ||
    isZuxuanFushiPoolPlay(config) ||
    isZu3PoolPlay(config) ||
    isZu6PoolPlay(config) ||
    isSixingZu6PoolPlay(config)
  ) {
    // 和值/跨度/组选复式/组三等：显示时保留逗号；粘连串先按号池解析再拼回（039 → 0,3,9）
    if (
      poolUsesCommaSeparatedInput(config) ||
      isZuxuanFushiPoolPlay(config) ||
      isZu3PoolPlay(config) ||
      isZu6PoolPlay(config) ||
      isSixingZu6PoolPlay(config)
    ) {
      const toks = parseDigitSegmentTokens(c.replace(/\n/g, ','), config)
      return toks.join(',')
    }
    const toks = c
      .split(/[,，\s\n]+/)
      .map((t) => t.trim())
      .filter(Boolean)
    return toks.join('')
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

/**
 * 方案内容文本框失焦时统一规范化（不弹保存级错误）。
 * - 数字录入（复式/号池等）：box↔content 往返
 * - 单式/组选单式/混合组选：按位长过滤并去重；结果为空则保留原文（避免半截输入被清空）
 * - 仅逗号/空白 → 空串
 */
export function commitSchemeGroupContentOnBlur(raw: string, config: PlayConfig): string {
  const src = String(raw ?? '').replace(/\r/g, '')
  if (!schemeGroupContentHasDigits(src)) return ''

  if (schemeGroupUsesLhcTemaPanel(config)) {
    return normalizeLhcTemaContent(src)
  }

  // 数字录入玩法（前三直选复式等）：与 SchemeGroupInputPanel 既有失焦一致
  if (schemeGroupUsesDigitInput(config)) {
    return normalizeSchemeGroupDigitContent(src, config)
  }

  const bm = String(config.betMode ?? '').trim()
  if (bm === 'hunhe' || (typeof config.playMethodLabel === 'string' && config.playMethodLabel.includes('混合组选'))) {
    const digitLen = hunheDigitLenFromConfig(config)
    const seg = digitLen > 0 ? digitLen : 3
    const next = normalizeHunheGroupContent(src, seg)
    if (next) return next
    // 无一合法（豹子/超长等）→ 只留半截碎片
    return keepIncompleteDigitFragments(src, seg)
  }
  if (isZu3DanshiConfig(config)) {
    const seg = config.segmentLen > 0 ? config.segmentLen : 3
    const next = normalizeZu3DanshiContent(src, seg)
    if (next) return next
    // 无一合法形态：剔除 ≥N 位的完整/超长废票，仅保留半截碎片（如 01）
    return keepIncompleteDigitFragments(src, seg)
  }
  if (isZu6DanshiConfig(config)) {
    const seg = config.segmentLen > 0 ? config.segmentLen : 3
    const next = normalizeZu6DanshiContent(src, seg)
    if (next) return next
    // 无一合法形态：剔除 ≥N 位的完整/超长废票（如 1234,6548），仅保留半截碎片
    return keepIncompleteDigitFragments(src, seg)
  }
  if (isZuxuanDanshiConfig(config)) {
    const seg = config.segmentLen > 0 ? config.segmentLen : 2
    const next = normalizeZuxuanDanshiContent(src, seg)
    return next || src
  }
  if (isSscDanshiLikeConfig(config) || config.inputMode === 'danshi') {
    const seg = config.segmentLen > 0 ? config.segmentLen : 0
    const toks = dedupeDanshiTokens(src, seg)
    if (toks.length) return toks.join(',')
    // 无一合法整注：剔除 ≥N 位废票，仅保留半截（与组六/混合失焦同口径）
    return keepIncompleteDigitFragments(src, seg)
  }

  // 其它落到文本框的玩法：尽量按数字录入规则整形
  const next = normalizeSchemeGroupDigitContent(src, config)
  return schemeGroupContentHasDigits(next) ? next : src
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
  // 组三单式走整注文本框，勿当号池
  if (isZu3DanshiConfig(config)) return false
  if (config.betMode === 'zu3') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组三|zu3/i.test(text) && !/组选3|组选30|zu30|和值|单式|_ds/i.test(text)
}

function isZu6PoolPlay(config: PlayConfig): boolean {
  // 组六单式走整注文本框，勿当号池（否则失焦会把 012 拆成 0,1,2）
  if (isZu6DanshiConfig(config)) return false
  // 四星/任四组选6 另走 isSixingZu6PoolPlay（C(n,2)），勿当三星组六
  if (isSixingZu6PlayConfig(config)) return false
  if (config.betMode === 'zu6') return true
  const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  return /组六|zu6/i.test(text) && !/组选6|组选60|组选120|zu60|zu120|和值|单式|_ds/i.test(text)
}

/** 四星/任四组选6 号池（至少 2 码，C(n,2)） */
function isSixingZu6PoolPlay(config: PlayConfig): boolean {
  return isSixingZu6PlayConfig(config)
}

/**
 * 失焦时：合法票已滤空后，只保留长度 < segmentLen 的半截数字碎片。
 * ≥N 位的完整/超长废票（如 112、1234）一律丢掉，避免「1234,6548」失焦原样不动。
 */
function keepIncompleteDigitFragments(raw: string, segmentLen: number): string {
  if (segmentLen <= 0) return String(raw ?? '').trim()
  const parts = String(raw ?? '')
    .split(/[,，\s\n]+/)
    .map((s) => s.trim())
    .filter(Boolean)
  const incomplete = parts.filter((t) => /^\d+$/.test(t) && t.length < segmentLen)
  return incomplete.join(',')
}

/** 组选包胆（前三组选包胆等）：单选 0–9 */
function isBaodanPoolPlay(config: PlayConfig): boolean {
  if (config.betMode === 'baodan') return true
  return /包胆/.test(config.playMethodLabel ?? '')
}

/** 组选复式最少选号：二星/任二=2，其余=3（任选剥位后 playTypeId 被清空，须看 segmentLen/renPositionCount） */
function zuxuanFushiMinPickHint(config: PlayConfig): number {
  const renK = config.renPositionCount ?? 0
  if (renK === 2) return 2
  if (renK >= 3 && renK <= 5) return 3
  // bareConfigForRenxuanPicks 将 segmentLen 置为任 k
  if (config.segmentLen === 2) return 2
  const text = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.playTypeId ?? ''} ${config.guajiGroup ?? ''}`
  if (/前二|后二|任二|任选二|g004|g005|g008|qian2|hou2|ren2/i.test(text)) return 2
  return 3
}

/**
 * 数字玩法方案内容录入提示（按玩法动态生成）：多位型逐位对应位名、逗号分隔、每位皆须录入；
 * 单位定宽可连写；和值等变长号池须逗号分隔。
 */
export function groupDigitInputHint(config: PlayConfig): string {
  if (isWuxingQuweiDigitPlayConfig(config)) {
    return wuxingQuweiFormatHint(config)
  }
	if (isLhcLianmaNumInputConfig(config)) {
	  const label = lhcLianmaNumInputLabel(config)
	  return /拖头$/.test(label)
	    ? `${label}：输入 ${lhcLianmaNumInputMinPicks(config)}–${LHC_ERQUANZHONG_NUM_MAX_PICKS} 个 01–49 号码，逗号分隔（首个为胆，其余为拖；如 01,13,25）`
	    : `${label}：输入 ${lhcLianmaNumInputMinPicks(config)}–${LHC_ERQUANZHONG_NUM_MAX_PICKS} 个 01–49 号码，逗号分隔（如 01,13,25）`
  }
  // 和值须最先匹配：避免「直选和值」被组选复式/组六文案抢提示
  if (isHezhiPoolConfig(config)) {
    const min = config.numberPoolMin ?? 0
    const max = config.numberPoolMax ?? 27
    return `和值：输入 ${min}–${max}，多选用逗号分隔（如 14,15,16）`
  }
  // 前后四直选组合：按序 4 位、位与位逗号分隔；每位可多选连写（如 12,2,3,45）
  if (isZhixuanZuhePlayConfig(config) && isQianhou4PlayConfig(config)) {
    return '按顺序填 4 个位置的号码（0–9），位与位用逗号分隔；每位可多选连写，如 1,2,3,4 或 12,2,3,45'
  }
  if (isZu3PoolPlay(config)) {
    return '输入两个及以上 0-9 的号码，多选用逗号分隔，如 1,3,5,7'
  }
  // 四星/任四组选6：至少 2 码（区别于三星组六 ≥3）
  if (isSixingZu6PoolPlay(config)) {
    return '输入两个及以上的0-9的号码，多选用逗号分隔，如1,2'
  }
  if (isZu6PoolPlay(config)) {
    return '输入三个及以上 0-9 的号码，多选用逗号分隔，如 1,3,5,7'
  }
  // 双区组选（12/4/五星60·30·20·10·5）
  if (isZuDualPoolPlay(config)) {
    return zuDualFormatHint(config)
  }
  // 组选24：至少 4 码（任四剥位后亦走此提示）
  if (isZu24PoolPlay(config)) {
    return '输入4个及以上0-9的号码，多选用逗号分隔，如：1,3,5,7'
  }
  // 组选120：至少 5 码（五星组选120）
  if (isZu120PoolPlay(config)) {
    return '输入5个及以上0-9的号码，多选用逗号分隔，如：1,3,5,7,9'
  }
  // 前二/后二/任二组选复式：至少 2 个号；三星/任三组选复式至少 3 个
  if (isZuxuanFushiPoolPlay(config)) {
    const min = zuxuanFushiMinPickHint(config)
    return `输入 ${min} 个及以上 0-9 的号码，多选用逗号分隔，如 1,3,5,7`
  }
  // 和值尾数：对齐直选和值提示/分隔方式，号池 0–9
  if (isWeishuPoolConfig(config)) {
    const min = config.numberPoolMin ?? 0
    const max = config.numberPoolMax ?? 9
    return `和值尾数：输入 ${min}–${max}，多选用逗号分隔（如 1,3,5）`
  }
  // 跨度：0–9，输入框内每个数字用逗号分隔
  if (isKuaduPoolConfig(config)) {
    const min = config.numberPoolMin ?? 0
    const max = config.numberPoolMax ?? 9
    return `跨度：输入 ${min}–${max}，每个数字用逗号分隔（如 0,3,9）`
  }
  // 不定位：每个数字用逗号分隔（一码最多 2 个）
  if (isBudingweiPoolConfig(config)) {
    if (isYimaBudingweiConfig(config)) {
      return '一码不定位：输入 1–2 个 0–9 号码，每个数字用逗号分隔（如 1,2）'
    }
    const text = `${config.playMethodLabel ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
    const sid = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
    if (/五星|wuxing/i.test(text) || sid === '151' || sid === '152') {
      return '五星不定位：至少 4 个 0–9 号码，每个数字用逗号分隔（如 1,2,3,4）'
    }
    if (/三码|_3ma/i.test(text) || sid === '152') {
      return '三码不定位：至少 3 个 0–9 号码，每个数字用逗号分隔（如 1,2,3,4）'
    }
    if (/二码|_2ma/i.test(text)) {
      return '二码不定位：至少 2 个 0–9 号码，每个数字用逗号分隔（如 1,2）'
    }
    return '不定位：输入 0–9 号码，每个数字用逗号分隔（如 1,2）'
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
  // 任选直选复式：五位面板但只需填满 k 位（任二至少两位，如 01,,,45）
  if (isRenxuanZhixuanFushiPlayConfig(config) && segLen >= 5) {
    const pickN = renxuanZhixuanFushiMinFilledHint(config)
    const cn = pickN === 2 ? '两' : (['零', '一', '二', '三', '四', '五'][pickN] ?? String(pickN))
    const example = buildRenxuanZhixuanFushiExample(options, pickN)
    return `请对应万位到个位，以“，”分隔，输入对应位置的号码，至少选${cn}位输入数字；如：${example}`
  }
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
  const first = formatHintPosLabel(labels[0] ?? '第1位')
  const last = formatHintPosLabel(labels[segLen - 1] ?? `第${segLen}位`)
  return `请对应${first}到${last}，以“，”分隔，输入对应位置的号码，每一位置皆要输入号码；如：${example}`
}

/** 万/千/百/十/个 → 万位…，已有「位」后缀则原样 */
function formatHintPosLabel(lab: string): string {
  const t = String(lab ?? '').trim()
  if (!t) return t
  if (/^[万千百十个]$/.test(t)) return `${t}位`
  return t
}

/** 任选直选复式至少填满位数（任二=2 / 任三=3 / 任四=4） */
function renxuanZhixuanFushiMinFilledHint(config: PlayConfig): number {
  const s = `${config.catalogSubId ?? ''} ${config.subPlayId ?? ''} ${config.playMethodLabel ?? ''} ${config.guajiGroup ?? ''} ${config.playTypeLabel ?? ''}`
  if (/ren4|任选四|任四/i.test(s)) return 4
  if (/ren3|任选三|任三/i.test(s)) return 3
  if (/ren2|任选二|任二/i.test(s)) return 2
  const sid = Number.parseInt(String(config.catalogSubId ?? config.subPlayId ?? '').trim(), 10)
  if (Number.isFinite(sid)) {
    if (sid >= 141 && sid <= 145) return 4
    if (sid >= 80 && sid <= 88) return 3
    if (sid >= 74 && sid <= 79) return 2
  }
  return 2
}

/** 任选直选复示例：首尾填号、中间留空（任二对齐产品文案 01,,,45） */
function buildRenxuanZhixuanFushiExample(options: string[], pickN: number): string {
  const dig = (i: number) => options[i % Math.max(1, options.length)] ?? String(i % 10)
  const pair = (a: number, b: number) => `${dig(a)}${dig(b)}`
  if (pickN <= 2) return `${pair(0, 1)},,,${pair(4, 5)}`
  // 任三/任四：五位面板，前 (pickN-1) 位 + 个位有号
  const segs = ['', '', '', '', '']
  const front = Math.min(4, Math.max(1, pickN - 1))
  for (let i = 0; i < front; i++) segs[i] = pair(i, i + 1)
  segs[4] = pair(4, 5)
  return segs.join(',')
}

/** 跨段组合玩法（前中后三 / 前后二 / 前后三 / 前后四）：位置非连续，提示用「N 个顺序号码」。 */
function usesSequentialGroupHint(config: PlayConfig): boolean {
  if (isQianhou4PlayConfig(config)) return true
  const text = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''} ${config.subPlayId ?? ''} ${config.playTypeId ?? ''} ${config.catalogSubId ?? ''}`
  if (/前中后三|前后二|前后三|前后四/.test(text)) return true
  return /qianzhonghou3|qianhou3|combo24/i.test(text)
}

/** 前后四（含 typeId=g014 / qianhou4，勿仅靠子玩法文案「直选组合」）。 */
function isQianhou4PlayConfig(config: PlayConfig): boolean {
  const tid = String(config.playTypeId ?? '').trim().toLowerCase()
  if (tid === 'g014' || tid === 'qianhou4') return true
  const text = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''} ${config.subPlayId ?? ''} ${config.catalogSubId ?? ''}`
  return /前后四|qianhou4/i.test(text)
}

/** 号池多选上限；一星/定位胆每位最多 9；包胆 / 龙虎（和）对齐第三方仅单选 */
export function poolMaxPicksForConfig(config: PlayConfig): number | null {
  if (config.poolMaxPicks != null && config.poolMaxPicks > 0) return config.poolMaxPicks
  if (config.betMode === 'baodan') return 1
  if (config.betMode === 'longhu' || config.betMode === 'longhuhe') return 1
  if (isLonghuPlayConfigLike(config)) return 1
  // 前二/后二/前三/后三大小单双：每位仅 1 个（大/小/单/双）
  if (isPerPosDxdsPlayConfig(config)) return 1
  // 五星和值单双/大小、哈希尾数单双/大小：仅 1 个选项
  if (isWuxingSumDxdsPlayConfig(config)) return 1
  const method = config.playMethodLabel ?? ''
  if (method.includes('包胆')) return 1
  // 一星：0–9 共 10 个号，禁止单位置满号（对齐第三方/既定规则）
  if (isYixingDingweiPlayConfig(config)) return YIXING_MAX_PICKS_PER_POS
  // 一码不定位：最多 2 个号（第三方「投注数字不可超过两位数」）
  if (isYimaBudingweiConfig(config)) return 2
  // 一帆风顺：最多 2 个号（第三方「投注数字不可超过两位」）
  if (isWuxingQuweiDigitPlayConfig(config)) {
    const max = wuxingQuweiMaxPicks(config)
    if (max > 0 && max < 10) return max
  }
  // 二全中复式/拖头：最多 10 个号（失焦截断 / 与保存校验一致）
	if (isLhcLianmaNumInputConfig(config)) return LHC_ERQUANZHONG_NUM_MAX_PICKS
  // 生肖对碰：最多 2 个生肖
  if (isLhcSxDuipengConfig(config) || config.betMode === 'sx_dp') return LHC_SX_DUIPENG_MAX_PICKS
  // 尾数对碰：最多 2 个尾数
  if (isLhcWsDuipengConfig(config) || config.betMode === 'ws_dp') return LHC_WS_DUIPENG_MAX_PICKS
  // 生尾对碰：1 肖 + 1 尾
  if (isLhcSwDuipengConfig(config) || config.betMode === 'sw_dp') return LHC_SW_DUIPENG_MAX_PICKS
  return null
}

/** 生尾对碰点选：肖/尾各自最多 1 个，点同侧替换 */
export function toggleSwDuipengPick(selected: string[], digit: string): string[] {
  const isZodiac = (LHC_ZODIACS as readonly string[]).includes(digit)
  const isTail = (LHC_TAIL_OPTIONS as readonly string[]).includes(digit.replace(/尾$/, ''))
  if (!isZodiac && !isTail) return selected
  const tok = isTail ? digit.replace(/尾$/, '') : digit
  const curZ = selected.find((s) => (LHC_ZODIACS as readonly string[]).includes(s))
  const curT = selected.find((s) => (LHC_TAIL_OPTIONS as readonly string[]).includes(s.replace(/尾$/, '')))
  if (isZodiac) {
    if (curZ === tok) return curT ? [curT] : []
    return curT ? [tok, curT] : [tok]
  }
  if (curT === tok) return curZ ? [curZ] : []
  return curZ ? [curZ, tok] : [tok]
}

function isYimaBudingweiConfig(config: PlayConfig): boolean {
  const tid = String(config.playTypeId ?? '').toLowerCase()
  const bm = String(config.betMode ?? '')
  const text = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''} ${config.catalogSubId ?? ''} ${config.subPlayId ?? ''}`
  const isBdw =
    bm === 'budingwei' || tid === 'g009' || tid === 'budingwei' || text.includes('不定位')
  if (!isBdw) return false
  if (/三码|_3ma|(?:^|[^a-z])3ma/.test(text.toLowerCase()) && text.includes('不定位')) return false
  if (/二码|_2ma|(?:^|[^a-z])2ma/.test(text.toLowerCase()) && text.includes('不定位')) return false
  const sid = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
  if (['114', '116', '118', '147', '149', '151', '152'].includes(sid)) return false
  return true
}

/** 在上限内切换号池选中（max=1 时点选替换；达上限时拒绝再加，保留原选） */
export function togglePoolPick(selected: string[], digit: string, maxPicks: number | null): string[] {
  const numericPool =
    /^\d+$/.test(digit) && selected.every((s) => /^\d+$/.test(s))
  const sortOrKeep = (arr: string[]) =>
    numericPool
      ? [...arr].sort((a, b) => Number(a) - Number(b) || a.localeCompare(b))
      : arr // 生肖等：保选择序（对碰首个|第二个）
  if (selected.includes(digit)) {
    return sortOrKeep(selected.filter((s) => s !== digit))
  }
  if (maxPicks === 1) return [digit]
  if (maxPicks != null && maxPicks > 0 && selected.length >= maxPicks) {
    return sortOrKeep(selected)
  }
  return sortOrKeep([...selected, digit])
}

/** 生肖对碰 chip 副文案：马 → 01,13,25,37,49 */
export function lhcZodiacChipSub(zodiac: string): string {
  const nums = LHC_ZODIAC_NUMBERS[zodiac]
  return nums?.length ? nums.join(',') : ''
}

/** 尾数对碰 chip 主文案：0 → 0尾（落库仍为 0） */
export function lhcTailChipLabel(tail: string): string {
  const t = String(tail ?? '')
    .trim()
    .replace(/尾$/, '')
  return t ? `${t}尾` : ''
}

/** 尾数对碰 chip 副文案：0 → 10,20,30,40 */
export function lhcTailChipSub(tail: string): string {
  const nums = LHC_TAIL_NUMBERS[String(tail).replace(/尾$/, '')]
  return nums?.length ? nums.join(',') : ''
}
