import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  expandZhixuanPoolToDanshiWithoutBaozi,
  filterBaoziFromDanshiTickets,
  isBaoziDigitTicket,
} from './betPayload'

describe('随机出号剔除豹子', () => {
  it('filterBaoziFromDanshiTickets', () => {
    expect(filterBaoziFromDanshiTickets('123,111,222,122', 3)).toBe('123,122')
    expect(filterBaoziFromDanshiTickets('111,222', 3)).toBe('')
  })

  it('前后三按位号池展开后无豹子', () => {
    const expanded = expandZhixuanPoolToDanshiWithoutBaozi('1,2\n1,2\n1,2', 3)
    expect(expanded).toBeTruthy()
    for (const t of expanded.split(',')) {
      expect(isBaoziDigitTicket(t)).toBe(false)
    }
    expect(expandZhixuanPoolToDanshiWithoutBaozi('5\n5\n5', 3)).toBe('')
  })

  it('类型占位：PlayConfig 可识别', () => {
    const cfg = {
      betMode: 'danshi',
      segmentLen: 3,
      playMethodLabel: '直选单式',
      playTypeLabel: '前后三',
    } as PlayConfig
    expect(cfg.segmentLen).toBe(3)
  })
})
