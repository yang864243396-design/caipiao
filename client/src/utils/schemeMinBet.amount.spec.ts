import { describe, expect, it } from 'vitest'
import { schemeMinSingleBetAmount } from './schemeMinBet'

describe('scheme minimum actual bet amount', () => {
  it('truncates the computed stake before evaluating the opening threshold', () => {
    expect(schemeMinSingleBetAmount({ betUnit: 0.001 }, 176)).toBe(0.17)
    expect(schemeMinSingleBetAmount({ betUnit: 0.001 }, 179)).toBe(0.17)
  })
})
