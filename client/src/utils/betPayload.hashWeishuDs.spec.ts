import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { countBetUnits, groupContentPlaceholder, validateGroupContent } from './betPayload'
import { poolMaxPicksForConfig, togglePoolPick } from './pickPanelOptions'
import { resolvePlayConfigFromCatalogIds } from './playConfig'
import { isWuxingSumDxdsPlayConfig } from './runTypeMatrix'

/** 波场/哈希 尾数单双（g017 / 387） */
const hashWeishuDs = {
  ...resolvePlayConfigFromCatalogIds('g017', '387', 'danshuang'),
  playTemplate: 'fast_ssc_std',
  playTypeId: 'g017',
  catalogSubId: '387',
  playTypeLabel: '哈希玩法',
  playMethodLabel: '尾数单双',
  guajiGroup: '哈希玩法',
  betMode: 'danshuang',
  poolMaxPicks: 1,
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'pool',
} as PlayConfig

describe('哈希尾数单双：仅选一个', () => {
  it('识别为单 token 单双', () => {
    expect(isWuxingSumDxdsPlayConfig(hashWeishuDs)).toBe(true)
    expect(poolMaxPicksForConfig(hashWeishuDs)).toBe(1)
  })

  it('poolMaxPicks=1，点选替换', () => {
    expect(togglePoolPick(['单'], '双', 1)).toEqual(['双'])
  })

  it('合法单选计 1 注', () => {
    const r = validateGroupContent(hashWeishuDs, '单')
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.normalized).toBe('单')
      expect(r.betUnits).toBe(1)
    }
    expect(countBetUnits(hashWeishuDs, '单')).toBe(1)
  })

  it('多选拒绝', () => {
    const r = validateGroupContent(hashWeishuDs, '单,双')
    expect(r.ok).toBe(false)
    if (!r.ok) {
      expect(r.message).toBe('仅能选择一个选项（单/双）')
    }
  })

  it('placeholder 提示尾数单双', () => {
    expect(groupContentPlaceholder(hashWeishuDs)).toMatch(/尾数单双.*仅选一个/)
  })

  it('旧 ruleId 267 亦识别', () => {
    const cfg = {
      playTypeId: 'g017',
      catalogSubId: '267',
      betMode: 'danshuang',
      playMethodLabel: '',
      playTypeLabel: '哈希玩法',
    }
    expect(isWuxingSumDxdsPlayConfig(cfg)).toBe(true)
  })
})
