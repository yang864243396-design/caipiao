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

const ren3Fs = {
  ...ren2Fs,
  subPlayId: '80',
  catalogSubId: '80',
  segmentLen: 3,
  playMethodLabel: '任三直选复式',
} as PlayConfig

const full5x10 = [
  '0,1,2,3,4,5,6,7,8,9',
  '0,1,2,3,4,5,6,7,8,9',
  '0,1,2,3,4,5,6,7,8,9',
  '0,1,2,3,4,5,6,7,8,9',
  '0,1,2,3,4,5,6,7,8,9',
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

describe('任三直选复式注数与上限', () => {
  it('上限为 9000（前三 900×C(5,3)，非 900）', () => {
    expect(renxuanZhixuanFushiMaxBetUnits(ren3Fs)).toBe(9000)
    expect(zhixuanFushiMaxBetUnits(ren3Fs)).toBe(9000)
  })

  it('五位满号 C(5,3)×1000=10000，超过 9000', () => {
    expect(countBetUnits(ren3Fs, full5x10)).toBe(10000)
    const r = validateGroupContent(ren3Fs, full5x10)
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.message).toContain('9000')
  })
})
