import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { groupDigitInputHint } from './pickPanelOptions'

const ren2ZhixuanFs = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '74',
  catalogSubId: '74',
  betMode: 'fushi',
  playMethodLabel: '直选复式',
  inputMode: 'multiline',
  segmentLen: 5,
  numberPoolMin: 0,
  numberPoolMax: 9,
  segmentLabels: ['万', '千', '百', '十', '个'],
} as PlayConfig

describe('任二直选复式录入提示', () => {
  it('提示至少选两位，示例含空位', () => {
    const tip = groupDigitInputHint(ren2ZhixuanFs)
    expect(tip).toBe(
      '请对应万到个，以“，”分隔，输入对应位置的号码，至少选两位输入数字；如：01,,,45',
    )
  })
})
