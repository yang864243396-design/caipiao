import type { PlayConfig } from '@/utils/betPayload'
import { isLonghuPlayConfigLike } from '@/utils/runTypeMatrix'

export const LONGHU_PICK_DOU = ['龙', '虎'] as const
export const LONGHU_PICK_HE = ['龙', '虎', '和'] as const

export function isLonghuPlayConfig(
  config: Pick<PlayConfig, 'betMode' | 'playTypeId' | 'playTypeLabel'>,
): boolean {
  return isLonghuPlayConfigLike(config)
}

/** 龙虎「和」子玩法：betMode=longhuhe、subId 含 _he，或中文名含龙虎和 */
export function isLonghuHeSubPlay(config: PlayConfig): boolean {
  if (config.betMode === 'longhuhe') return true
  const sub = (config.catalogSubId ?? config.subPlayId ?? '').toLowerCase()
  if (sub.endsWith('_he') || sub.includes('_he_') || sub.includes('longhuhe')) return true
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''}`
  return label.includes('龙虎和')
}

export function longhuPickOptionsForConfig(config: PlayConfig): string[] {
  if (!isLonghuPlayConfig(config)) return []
  return isLonghuHeSubPlay(config) ? [...LONGHU_PICK_HE] : [...LONGHU_PICK_DOU]
}

export function longhuPickHint(config: PlayConfig): string {
  return isLonghuHeSubPlay(config) ? '龙、虎、和' : '龙、虎'
}
