import { describe, expect, it } from 'vitest'
import { validateGroupContent, type PlayConfig } from './betPayload'

const kuaduConfig = {
  playTemplate: 'ssc_std',
  playTypeId: 'g002',
  subPlayId: '17',
  betMode: 'kuadu',
  playMethodLabel: '中三直选跨度',
  segmentLen: 1,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('validateGroupContent kuadu', () => {
  it('rejects values above 9', () => {
    const r = validateGroupContent(kuaduConfig, '3,11,15')
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.message).toMatch(/0–9|0-9|不能填写/)
  })

  it('rejects only-invalid content (no betUnits||1 bypass)', () => {
    const r = validateGroupContent(kuaduConfig, '11')
    expect(r.ok).toBe(false)
  })

  it('accepts 0–9 and normalizes', () => {
    const r = validateGroupContent(kuaduConfig, '0,3,09,9')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.normalized).toBe('0,3,9')
  })

  it('rejects combinatorial units over 900 (full 0–9 = 1000)', () => {
    const cfg = { ...kuaduConfig, segmentLen: 3, playTypeId: 'g002' }
    const r = validateGroupContent(cfg, '0,1,2,3,4,5,6,7,8,9')
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.message).toMatch(/900/)
  })
})
