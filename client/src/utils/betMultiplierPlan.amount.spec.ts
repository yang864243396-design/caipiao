import { describe, expect, it } from 'vitest'
import { buildPlanRowsFromTimes } from './betMultiplierPlan'

describe('bet multiplier plan actual stake', () => {
  it('calculates each period and total using Guaji-truncated amounts', () => {
    const rows = buildPlanRowsFromTimes([1, 1], 0.179, 1, 1)
    expect(rows[0]?.curBet).toBe('0.17')
    expect(rows[1]?.totalBet).toBe('0.34')
  })
})
