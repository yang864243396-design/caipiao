import type { PlayConfig } from '@/utils/betPayload'
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

/** 投注/方案面板：按玩法号池生成可选号码 */
export function digitOptionsForConfig(config: PlayConfig): string[] {
  const min = config.numberPoolMin ?? 0
  const max = config.numberPoolMax ?? 9
  // 11选5/PK10 等从 1 起且 max≥11 时补零；和值 0–18 等保持单位数展示（对齐第三方）
  const pad = max >= 11 && min >= 1
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
 * 引擎存储内容 → 数字录入框压缩格式（与 SchemeGroupInputPanel 一致）。
 * 多位型：每位号码连写、逗号分隔各位，如 `1,2\n3,4` → `12,34`；
 * 单位型：号码连写，如 `1,2` → `12`。
 */
export function schemeGroupContentToInputBox(content: string, config: PlayConfig): string {
  const c = String(content ?? '').replace(/\r/g, '')
  const segLen = Math.max(1, config.segmentLen || 1)
  if (segLen <= 1) {
    return c
      .split(/[,，\s]+/)
      .map((t) => t.trim())
      .filter(Boolean)
      .join('')
  }
  // 已是录入框形态（段数对齐、无换行）则原样规范后返回
  if (!c.includes('\n')) {
    const parts = c.split(/[,，]/)
    if (parts.length === segLen) {
      return parts.map((p) => p.replace(/[^0-9A-Za-z]/g, '')).join(',')
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

/** 构造数字录入示例：逐位连写号码、逗号分隔各位（各位取数不同长度以示意） */
function buildDigitInputExample(options: string[], segLen: number): string {
  if (!options.length) return ''
  const segs: string[] = []
  for (let i = 0; i < segLen; i++) {
    const count = (i % 4) + 2
    const toks: string[] = []
    for (let k = 0; k < count; k++) toks.push(options[(i + k) % options.length]!)
    segs.push(toks.join(''))
  }
  return segs.join(',')
}

/**
 * 数字玩法方案内容录入提示（按玩法动态生成）：多位型逐位对应位名、逗号分隔、每位皆须录入；
 * 单位型直接连写。示例号码取自当前号池。
 */
export function groupDigitInputHint(config: PlayConfig): string {
  const options = digitOptionsForConfig(config)
  if (!options.length) return ''
  const range = `${options[0]}-${options[options.length - 1]}`
  const segLen = Math.max(1, config.segmentLen || 1)
  const example = buildDigitInputExample(options, segLen)
  if (segLen <= 1) {
    return `直接连写号码（可选 ${range}），如：${example}`
  }
  const labels = config.segmentLabels ?? []
  const first = labels[0] ?? '第1位'
  const last = labels[segLen - 1] ?? `第${segLen}位`
  return `请对应${first}到${last}，以“，”分隔，输入对应位置的号码，每一位置皆要输入号码；如：${example}`
}

/** 号池多选上限；包胆 / 龙虎（和）对齐第三方仅单选 */
export function poolMaxPicksForConfig(config: PlayConfig): number | null {
  if (config.poolMaxPicks != null && config.poolMaxPicks > 0) return config.poolMaxPicks
  if (config.betMode === 'baodan') return 1
  if (config.betMode === 'longhu' || config.betMode === 'longhuhe') return 1
  if (isLonghuPlayConfigLike(config)) return 1
  const method = config.playMethodLabel ?? ''
  if (method.includes('包胆')) return 1
  return null
}

/** 在上限内切换号池选中（max=1 时点选替换，行为同单选） */
export function togglePoolPick(selected: string[], digit: string, maxPicks: number | null): string[] {
  const set = new Set(selected)
  if (set.has(digit)) {
    set.delete(digit)
    return [...set].sort()
  }
  if (maxPicks === 1) return [digit]
  if (maxPicks != null && maxPicks > 0 && set.size >= maxPicks) {
    return [digit]
  }
  set.add(digit)
  return [...set].sort()
}
