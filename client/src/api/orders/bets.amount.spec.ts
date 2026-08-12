import { describe, expect, it } from 'vitest'
import { formatBetAmount, toBetDisplayRow } from './bets'

describe('actual bet amount display', () => {
  it('truncates order stake to the amount accepted by Guaji', () => {
    expect(formatBetAmount(0.179)).toBe('0.17')
    expect(toBetDisplayRow({
      time: '2026-08-12 12:00:00',
      game: '分分彩',
      orderId: 'BO1',
      amount: 0.176,
      returnAmount: 0,
      status: 'pending',
    }).amount).toBe('0.17')
  })
})
