import { describe, expect, it } from 'vitest'
import { toggleSingleHcwRank } from './hcwRankSelection'

describe('toggleSingleHcwRank', () => {
  it('replaces the existing rank when a different option is clicked', () => {
    expect(toggleSingleHcwRank([2], 5)).toEqual([5])
  })

  it('clears the rank when the selected option is clicked again', () => {
    expect(toggleSingleHcwRank([2], 2)).toEqual([])
  })
})
