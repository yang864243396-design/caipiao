import { describe, expect, it } from 'vitest'
import {
  countBetUnits,
  expandZuxuanDigitPoolToDanshi,
  normalizeZuxuanDanshiContent,
  type PlayConfig,
  validateGroupContent,
} from './betPayload'

function ren2ZuxuanDsConfig(): PlayConfig {
  return {
    playTemplate: 'ssc_std',
    playTypeId: 'g011',
    subPlayId: '78',
    catalogSubId: '78',
    playTypeLabel: '任选',
    playMethodLabel: '任二组选单式',
    guajiGroup: '任选',
    betMode: 'zuxuan_ds',
    inputMode: 'danshi',
    segmentLen: 2,
    segmentLabels: ['选号'],
    renPositionCount: 2,
    numberPoolMin: 0,
    numberPoolMax: 9,
  } as PlayConfig
}

describe('任二组选单式号池两两组合', () => {
  it('expandZuxuanDigitPoolToDanshi: 1,2,3 → 12,13,23', () => {
    expect(expandZuxuanDigitPoolToDanshi('1,2,3', 2)).toBe('12,13,23')
  })

  it('normalize：号池与整注均可', () => {
    expect(normalizeZuxuanDanshiContent('1,2,3', 2)).toBe('12,13,23')
    expect(normalizeZuxuanDanshiContent('12,21,34', 2)).toBe('12,34')
  })

  it('校验号池+选位通过，注数=C(选位,2)×C(号池,2)', () => {
    const cfg = ren2ZuxuanDsConfig()
    const ok = validateGroupContent(cfg, '万,千\n1,2,3')
    expect(ok.ok).toBe(true)
    if (ok.ok) {
      expect(ok.betUnits).toBe(3)
      expect(ok.normalized).toContain('12')
    }
    expect(countBetUnits(cfg, '万,千,个\n1,2,3')).toBe(9)
  })
})
