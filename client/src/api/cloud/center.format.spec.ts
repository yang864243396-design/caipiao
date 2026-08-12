import { describe, expect, it } from 'vitest'
import { formatCloudBetTurnover, formatCloudSchemeTurnover } from './center'
import { calcBetAmount } from '@/utils/betPayload'

describe('formatCloudSchemeTurnover', () => {
  it('keeps two decimal places for scheme card turnover', () => {
    expect(formatCloudSchemeTurnover(0.15)).toBe('0.15')
    expect(formatCloudSchemeTurnover(0.1)).toBe('0.10')
    expect(formatCloudSchemeTurnover(0.176)).toBe('0.17')
    expect(formatCloudSchemeTurnover(0.179)).toBe('0.17')
    expect(formatCloudBetTurnover(0.179)).toBe('0.17')
  })

  it('uses the third-party truncated amount before client preflight', () => {
    expect(calcBetAmount(176, 1, 0.001)).toBe(0.17)
    expect(calcBetAmount(179, 1, 0.001)).toBe(0.17)
  })
})
