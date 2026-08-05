import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { groupContentPlaceholder } from './betPayload'
import { groupDigitInputHint } from './pickPanelOptions'

const qian3Fs = {
  playTemplate: 'ssc_std',
  playTypeId: 'qian3',
  playTypeLabel: '前三',
  subPlayId: 'zhixuan_fs',
  catalogSubId: 'qian3_zhixuan_fs',
  betMode: 'fushi',
  playMethodLabel: '前三直选复式',
  inputMode: 'multiline',
  segmentLen: 3,
  segmentLabels: ['万', '千', '百'],
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('前三直选复式录入提示', () => {
  it('数字框提示为「请对应万位到百位…」而非按位分行旧文案', () => {
    const tip = groupDigitInputHint(qian3Fs)
    expect(tip).toMatch(/^请对应万位到百位，以“，”分隔/)
    expect(tip).toMatch(/每一位置皆要输入号码/)
    expect(tip).not.toMatch(/按位分行输入/)
    expect(tip).not.toMatch(/不含豹子/)
  })

  it('groupContentPlaceholder 与数字框口径一致（裸 textarea 兜底）', () => {
    const tip = groupContentPlaceholder(qian3Fs)
    expect(tip).toMatch(/^请对应万位到百位，以“，”分隔/)
    expect(tip).not.toMatch(/按位分行输入/)
  })
})
