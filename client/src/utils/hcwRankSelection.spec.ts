import { describe, expect, it } from 'vitest'
import { shouldDirectSwitchHcwRank, toggleSingleHcwRank } from './hcwRankSelection'

describe('toggleSingleHcwRank', () => {
  it('replaces the existing rank when a different option is clicked', () => {
    expect(toggleSingleHcwRank([2], 5)).toEqual([5])
  })

  it('clears the rank when the selected option is clicked again', () => {
    expect(toggleSingleHcwRank([2], 2)).toEqual([])
  })
})

describe('shouldDirectSwitchHcwRank', () => {
  it('enables direct switching for cap-one tail parity and lhc guoguan', () => {
    expect(shouldDirectSwitchHcwRank(1, false, true)).toBe(true)
    expect(shouldDirectSwitchHcwRank(1, true, false)).toBe(true)
  })

  it('keeps sum parity and per-position cap-one selections on their existing policy', () => {
    expect(shouldDirectSwitchHcwRank(1, false, false)).toBe(false)
  })

  it('does not enable direct switching when the cap is not one', () => {
    expect(shouldDirectSwitchHcwRank(2, true, true)).toBe(false)
    expect(shouldDirectSwitchHcwRank(null, true, true)).toBe(false)
  })
})
