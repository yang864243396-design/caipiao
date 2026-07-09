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
  const out: string[] = []
  for (let i = min; i <= max; i++) {
    out.push(max >= 11 ? String(i).padStart(2, '0') : String(i))
  }
  return out.length ? out : ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9']
}

function inferTextPickFromLabels(config: PlayConfig): string[] {
  const subLabel = config.playMethodLabel?.trim() ?? ''
  if (subLabel === '大小单双') return ['大', '小', '单', '双']
  if (subLabel === '龙虎豹') return ['龙', '虎', '豹']
  if (subLabel === '特殊号') return ['豹子', '对子', '顺子', '极大', '极小']
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
    case 'teshu':
      return ['豹子', '对子', '顺子', '极大', '极小']
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
