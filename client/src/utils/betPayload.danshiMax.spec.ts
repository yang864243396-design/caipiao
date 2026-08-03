import { describe, expect, it } from 'vitest'
import { validateGroupContent, type PlayConfig } from './betPayload'

const qian2Danshi: PlayConfig = {
  playTemplate: 'ssc_std',
  playTypeId: 'g004',
  playTypeLabel: '前二',
  subPlayId: 'zhixuan_ds',
  betMode: 'danshi',
  segmentLen: 2,
  segmentLabels: ['十', '个'],
  inputMode: 'danshi',
  playMethodLabel: '前二直选单式',
}

function buildTickets(n: number): string {
  const parts: string[] = []
  for (let a = 0; a <= 9 && parts.length < n; a++) {
    for (let b = 0; b <= 9 && parts.length < n; b++) {
      parts.push(`${a}${b}`)
    }
  }
  return parts.join(',')
}

describe('validateGroupContent 前二直选单式最大注数', () => {
  it('90 注通过', () => {
    const r = validateGroupContent(qian2Danshi, buildTickets(90))
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(90)
  })

  it('100 注（00–99）拦截为超过 90', () => {
    const r = validateGroupContent(qian2Danshi, buildTickets(100))
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.message).toBe('投注注数超过最大投注注数:90')
  })
})
