import type { PlayConfig } from '@/utils/betPayload'
import { isLonghuPlayConfigLike } from '@/utils/runTypeMatrix'

export const LONGHU_PICK_DOU = ['龙', '虎'] as const
export const LONGHU_PICK_HE = ['龙', '虎', '和'] as const

export function isLonghuPlayConfig(
  config: Pick<PlayConfig, 'betMode' | 'playTypeId' | 'playTypeLabel'>,
): boolean {
  return isLonghuPlayConfigLike(config)
}

/** 龙虎「和」子玩法：betMode=longhuhe 或 subId 含 _he（如 lh_wanqian_he） */
export function isLonghuHeSubPlay(config: PlayConfig): boolean {
  if (config.betMode === 'longhuhe') return true
  const sub = (config.catalogSubId ?? config.subPlayId ?? '').toLowerCase()
  return sub.endsWith('_he')
}

export function longhuPickOptionsForConfig(config: PlayConfig): string[] {
  if (!isLonghuPlayConfig(config)) return []
  return isLonghuHeSubPlay(config) ? [...LONGHU_PICK_HE] : [...LONGHU_PICK_DOU]
}

export function longhuPickHint(config: PlayConfig): string {
  return isLonghuHeSubPlay(config) ? '龙、虎、和' : '龙、虎'
}
