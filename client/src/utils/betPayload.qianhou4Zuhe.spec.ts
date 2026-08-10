import { describe, expect, it } from 'vitest'
import {
  countBetUnits,
  isSixingZu6PlayConfig,
  isZhixuanZuhePlayConfig,
  validateGroupContent,
  type PlayConfig,
} from './betPayload'
import {
  groupDigitInputHint,
  schemeGroupInputBoxToContent,
} from './pickPanelOptions'

const qianhou4Zuhe = {
  playTemplate: 'ssc_std',
  playTypeId: 'g014',
  playTypeLabel: '前后四',
  subPlayId: '136',
  catalogSubId: '136',
  betMode: 'zuhe',
  playMethodLabel: '直选组合',
  guajiGroup: '前后四',
  inputMode: 'multiline',
  segmentLen: 4,
  segmentLabels: ['千', '百', '十', '个'],
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('前后四直选组合（rule136）按位逗号录入', () => {
  it('不得误判为四星/前后四组选6', () => {
    expect(isZhixuanZuhePlayConfig(qianhou4Zuhe)).toBe(true)
    expect(isSixingZu6PlayConfig(qianhou4Zuhe)).toBe(false)
  })

  it('录入框 12,2,3,45 → 按位存储并计 32 注', () => {
    const content = schemeGroupInputBoxToContent('12,2,3,45', qianhou4Zuhe)
    expect(content).toBe('1,2\n2\n3\n4,5')
    // 位积 2×1×1×2=4 × 段长4 × 前后四区位2 = 32
    expect(countBetUnits(qianhou4Zuhe, content)).toBe(32)
    expect(countBetUnits(qianhou4Zuhe, '12,2,3,45')).toBe(32)
    const r = validateGroupContent(qianhou4Zuhe, '12,2,3,45')
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.betUnits).toBe(32)
      expect(r.normalized).toBe('1,2\n2\n3\n4,5')
    }
  })

  it('单码示例 1,2,3,4 → 8 注', () => {
    expect(countBetUnits(qianhou4Zuhe, '1,2,3,4')).toBe(8)
  })

  it('提示按序 4 位、逗号分位', () => {
    const tip = groupDigitInputHint(qianhou4Zuhe)
    expect(tip).toContain('4')
    expect(tip).toContain('逗号')
    expect(tip).toMatch(/1,2,3,4/)
    expect(tip).toMatch(/12,2,3,45/)
  })
})
