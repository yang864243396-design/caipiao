import { describe, expect, it } from 'vitest'
import {
  normalizeLhcRenyiDuipengHotColdRanks,
  replaceLhcRenyiDuipengHotColdRanks,
} from './lhcRenyiDuipengHotCold'

describe('任意对碰冷热双区名次', () => {
  it('accepts two non-overlapping zones and rejects overlap or more than ten ranks', () => {
    expect(normalizeLhcRenyiDuipengHotColdRanks([[0, 1], [2]], 49)).toEqual({
      a: [0, 1],
      b: [2],
      valid: true,
    })
    expect(normalizeLhcRenyiDuipengHotColdRanks([[0], [0]], 49).valid).toBe(false)
    expect(
      normalizeLhcRenyiDuipengHotColdRanks(
        [Array.from({ length: 6 }, (_, i) => i), Array.from({ length: 5 }, (_, i) => i + 6)],
        49,
      ).valid,
    ).toBe(false)
  })

  it('replaces one zone without selecting the other zone ranks or exceeding ten in total', () => {
    expect(replaceLhcRenyiDuipengHotColdRanks([[0], [1]], 0, [1, 2, 3], 49)).toEqual([[2, 3], [1]])
    expect(
      replaceLhcRenyiDuipengHotColdRanks(
        [[0], [1, 2, 3, 4, 5, 6, 7, 8, 9]],
        0,
        [10, 11],
        49,
      ),
    ).toEqual([[10], [1, 2, 3, 4, 5, 6, 7, 8, 9]])
  })
})
