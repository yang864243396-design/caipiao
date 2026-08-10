import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { groupDigitInputHint, schemeGroupInputBoxToContent } from './pickPanelOptions'

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

describe('前后四直选组合录入提示与按位转换', () => {
  it('按序 4 位、逗号分位，示例含连写多选', () => {
    const tip = groupDigitInputHint(qianhou4Zuhe)
    expect(tip).toBe(
      '按顺序填 4 个位置的号码（0–9），位与位用逗号分隔；每位可多选连写，如 1,2,3,4 或 12,2,3,45',
    )
  })

  it('12,2,3,45 按位拆成四行，勿扁选成组选6号池', () => {
    expect(schemeGroupInputBoxToContent('12,2,3,45', qianhou4Zuhe)).toBe('1,2\n2\n3\n4,5')
  })
})
