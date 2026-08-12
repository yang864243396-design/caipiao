import { describe, expect, it } from 'vitest'
import { toDisplayRow } from './betRecords'

describe('cloud bet record amount display', () => {
  it('shows the third-party truncated stake', () => {
    expect(toDisplayRow({
      id: 'r1',
      recordNo: 'r1',
      period: 'p1',
      playType: '定位胆',
      multiplier: '1',
      round: '1',
      amount: 0.179,
      pnl: 0,
      status: 'pending',
    }).amount).toBe('0.17')
  })
})
