import { describe, expect, it } from 'vitest'
import { formatCloudSchemeTurnover } from './center'

describe('formatCloudSchemeTurnover', () => {
  it('keeps two decimal places for scheme card turnover', () => {
    expect(formatCloudSchemeTurnover(0.15)).toBe('0.15')
    expect(formatCloudSchemeTurnover(0.1)).toBe('0.10')
  })
})
