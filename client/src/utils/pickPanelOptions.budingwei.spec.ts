import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  groupDigitInputHint,
  poolMaxPicksForConfig,
  poolUsesCommaSeparatedInput,
} from './pickPanelOptions'

const qian3Yima = {
  playTemplate: 'ssc_std',
  playTypeId: 'g009',
  playTypeLabel: '不定位',
  guajiGroup: '不定位',
  subPlayId: '113',
  catalogSubId: '113',
  betMode: 'budingwei',
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'pool',
  playMethodLabel: '前三一码不定位',
  numberPoolMin: 0,
  numberPoolMax: 9,
  poolMaxPicks: 2,
} as PlayConfig

describe('不定位号池逗号分隔', () => {
  it('前三一码须逗号分隔录入', () => {
    expect(poolUsesCommaSeparatedInput(qian3Yima)).toBe(true)
  })

  it('提示每个数字用逗号分隔', () => {
    const tip = groupDigitInputHint(qian3Yima)
    expect(tip).toContain('逗号分隔')
    expect(tip).toMatch(/1,\s*2/)
  })

  it('前三一码号池勾选上限为 2', () => {
    const withoutExplicit = { ...qian3Yima, poolMaxPicks: undefined }
    expect(poolMaxPicksForConfig(withoutExplicit)).toBe(2)
    expect(poolMaxPicksForConfig(qian3Yima)).toBe(2)
  })
})
