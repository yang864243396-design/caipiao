import { describe, expect, it } from 'vitest'
import { countBetUnits, validateGroupContent, weishuMaxBetUnits, type PlayConfig } from './betPayload'

const qzh3Weishu = {
  playTemplate: 'ssc_std',
  playTypeId: 'g007',
  playTypeLabel: '前中后三',
  guajiGroup: '前中后三',
  subPlayId: '111',
  catalogSubId: '111',
  betMode: 'weishu',
  segmentLen: 1,
  segmentLabels: ['和值尾数'],
  inputMode: 'pool',
  playMethodLabel: '和值尾数',
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

const qian3Weishu = {
  playTemplate: 'ssc_std',
  playTypeId: 'g001',
  playTypeLabel: '前三',
  guajiGroup: '前三码',
  subPlayId: '11',
  betMode: 'weishu',
  segmentLen: 1,
  segmentLabels: ['和值尾数'],
  inputMode: 'pool',
  playMethodLabel: '和值尾数',
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('和值尾数注数', () => {
  it('前三 1–9 = 9', () => {
    expect(countBetUnits(qian3Weishu, '1,2,3,4,5,6,7,8,9')).toBe(9)
    expect(weishuMaxBetUnits(qian3Weishu)).toBe(9)
  })

  it('前中后三 1–9 = 9×3 = 27', () => {
    expect(countBetUnits(qzh3Weishu, '1,2,3,4,5,6,7,8,9')).toBe(27)
    expect(weishuMaxBetUnits(qzh3Weishu)).toBe(27)
  })

  it('前中后三满 1–9 校验通过', () => {
    const r = validateGroupContent(qzh3Weishu, '1,2,3,4,5,6,7,8,9')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(27)
  })
})
