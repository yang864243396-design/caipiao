import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { groupContentPlaceholder } from './betPayload'

const qian2ZuxuanDs = {
  playTemplate: 'ssc_std',
  playTypeId: 'g004',
  subPlayId: '43',
  catalogSubId: '43',
  betMode: 'zuxuan_ds',
  playTypeLabel: '前二码',
  playMethodLabel: '前二组选单式',
  inputMode: 'danshi',
  segmentLen: 2,
  segmentLabels: ['万', '千'],
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('前二组选单式 placeholder', () => {
  it('hints 2-digit tickets separated by commas', () => {
    const tip = groupContentPlaceholder(qian2ZuxuanDs)
    expect(tip).toMatch(/组选单式/)
    expect(tip).toMatch(/2\s*位/)
    expect(tip).toMatch(/逗号/)
    expect(tip).not.toMatch(/选号池/)
    expect(tip).not.toMatch(/0–9（如 0,1,2,3）/)
  })

  it('mentions excluding pairs and form dedupe', () => {
    const tip = groupContentPlaceholder(qian2ZuxuanDs)
    expect(tip).toMatch(/对子/)
    expect(tip).toMatch(/12,21/)
  })
})
