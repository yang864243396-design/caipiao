import { describe, expect, it } from 'vitest'
import {
  betAmountExceedsMax,
  calcBetAmount,
  isMaxBetAmountExceededMessage,
  maxBetAmountExceededMessage,
  maxModeMultiplierFromPayload,
  MAX_SINGLE_BET_AMOUNT,
} from './betPayload'

describe('max single bet amount', () => {
  it('message matches third-party wording', () => {
    expect(maxBetAmountExceededMessage('USDT')).toBe('最高下注限额100000.00USDT')
    expect(isMaxBetAmountExceededMessage('最高下注限额100000.00USDT')).toBe(true)
  })

  it('36000×2×2 exceeds 100000', () => {
    const amount = calcBetAmount(36000, 2, 2)
    expect(amount).toBe(144000)
    expect(betAmountExceedsMax(amount)).toBe(true)
    expect(betAmountExceedsMax(MAX_SINGLE_BET_AMOUNT)).toBe(false)
  })

  it('reads max mode multiple from payload', () => {
    expect(maxModeMultiplierFromPayload({ kind: '2', simple: { multiples: '1,2,4' } })).toBe(4)
    expect(maxModeMultiplierFromPayload({ advanced: { rounds: [{ mult: 3 }, { mult: 1 }] } })).toBe(3)
  })
})
