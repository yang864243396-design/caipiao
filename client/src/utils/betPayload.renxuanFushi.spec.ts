import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  countBetUnits,
  renxuanZhixuanFushiMaxBetUnits,
  validateGroupContent,
  zhixuanFushiMaxBetUnits,
} from './betPayload'

const ren2Fs = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '74',
  catalogSubId: '74',
  betMode: 'fushi',
  segmentLen: 2,
  segmentLabels: ['万', '千', '百', '十', '个'],
  inputMode: 'multiline',
  playMethodLabel: '直选复式',
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

const content = [
  '1,2,3,4,5,6,7,8,9,0',
  '1,2,3,4,5,6,7,8,9,0',
  '1,2,3,4,5,6,7,8,9',
  '1,2,3,4,5,6,7,8,9',
  '1,2,3,4,5,6,7,8,9',
].join('\n')

describe('任二直选复式注数与上限', () => {
  it('上限为 900（非前二 90）', () => {
    expect(renxuanZhixuanFushiMaxBetUnits(ren2Fs)).toBe(900)
    expect(zhixuanFushiMaxBetUnits(ren2Fs)).toBe(900)
  })

  it('五位选号按 C(5,2) 计 883 注', () => {
    expect(countBetUnits(ren2Fs, content)).toBe(883)
  })

  it('883 注可保存', () => {
    const r = validateGroupContent(ren2Fs, content)
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(883)
  })
})
