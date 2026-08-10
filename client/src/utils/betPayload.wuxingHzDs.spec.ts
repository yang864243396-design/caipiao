import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { countBetUnits, groupContentPlaceholder, validateGroupContent } from './betPayload'
import { poolMaxPicksForConfig, togglePoolPick } from './pickPanelOptions'
import { resolvePlayConfigFromCatalogIds } from './playConfig'

const wuxingHzDs = {
  ...resolvePlayConfigFromCatalogIds('g016', '268', 'danshuang'),
  playTemplate: 'ssc_std',
  playTypeLabel: '大小单双',
  playMethodLabel: '五星和值单双',
  betMode: 'danshuang',
  poolMaxPicks: 1,
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'pool',
} as PlayConfig

describe('五星和值单双：仅选一个', () => {
  it('poolMaxPicks=1，点选替换', () => {
    expect(poolMaxPicksForConfig(wuxingHzDs)).toBe(1)
    expect(togglePoolPick(['单'], '双', 1)).toEqual(['双'])
  })

  it('合法单选计 1 注', () => {
    const r = validateGroupContent(wuxingHzDs, '单')
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.normalized).toBe('单')
      expect(r.betUnits).toBe(1)
    }
    expect(countBetUnits(wuxingHzDs, '单')).toBe(1)
  })

  it('多选拒绝', () => {
    const r = validateGroupContent(wuxingHzDs, '单,双')
    expect(r.ok).toBe(false)
    if (!r.ok) {
      expect(r.message).toBe('仅能选择一个选项（单/双）')
    }
  })

  it('placeholder 提示仅选一个', () => {
    expect(groupContentPlaceholder(wuxingHzDs)).toMatch(/仅选一个/)
  })
})
