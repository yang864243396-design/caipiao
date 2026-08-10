import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { countBetUnits, groupContentPlaceholder, validateGroupContent } from './betPayload'
import { poolMaxPicksForConfig, togglePoolPick } from './pickPanelOptions'
import { resolvePlayConfigFromCatalogIds } from './playConfig'

const hou2Dxds = {
  ...resolvePlayConfigFromCatalogIds('g016', '266', 'dxds'),
  playTemplate: 'ssc_std',
  playTypeLabel: '大小单双',
  playMethodLabel: '后二大小单双',
} as PlayConfig

describe('后二大小单双：十位/个位各选一个', () => {
  it('poolMaxPicks=1，点选替换而非累加', () => {
    expect(poolMaxPicksForConfig(hou2Dxds)).toBe(1)
    expect(togglePoolPick(['大'], '小', 1)).toEqual(['小'])
    expect(togglePoolPick(['大'], '大', 1)).toEqual([])
  })

  it('合法内容计 1 注并规范化', () => {
    const r = validateGroupContent(hou2Dxds, '大\n小')
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.normalized).toBe('大\n小')
      expect(r.betUnits).toBe(1)
    }
    expect(countBetUnits(hou2Dxds, '大\n小')).toBe(1)
  })

  it('每位多选拒绝保存', () => {
    const r = validateGroupContent(hou2Dxds, '大,小\n单')
    expect(r.ok).toBe(false)
    if (!r.ok) {
      expect(r.message).toBe('仅能选择一个选项（大/小/单/双）')
    }
  })

  it('缺位拒绝保存', () => {
    const r = validateGroupContent(hou2Dxds, '大\n')
    expect(r.ok).toBe(false)
  })

  it('placeholder 提示每位各选一个', () => {
    expect(groupContentPlaceholder(hou2Dxds)).toMatch(/各选一个/)
  })
})
