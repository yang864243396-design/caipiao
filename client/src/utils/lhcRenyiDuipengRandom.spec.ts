import { describe, expect, it } from 'vitest'
import { parseLhcRenyiDuipengSides, validateLhcRenyiDuipengContent } from './betPayload'
import {
  normalizeLhcRenyiDuipengRandomCounts,
  normalizeLhcRenyiDuipengTriggerContent,
  randomLhcRenyiDuipengContent,
  randomLhcRenyiDuipengContentForCounts,
  isRandomDrawLhcRenyiDuipengConfig,
} from './lhcRenyiDuipengRandom'

function expectValidRandomContent(total: number): void {
  const content = randomLhcRenyiDuipengContent(total, () => 0)
  const validation = validateLhcRenyiDuipengContent(content)
  expect(validation.ok).toBe(true)
  if (!validation.ok) return

  const sides = parseLhcRenyiDuipengSides(validation.normalized)
  expect(sides).not.toBeNull()
  expect(sides?.a.length).toBeGreaterThan(0)
  expect(sides?.b.length).toBeGreaterThan(0)
  expect((sides?.a.length ?? 0) + (sides?.b.length ?? 0)).toBe(total)
  expect(new Set([...(sides?.a ?? []), ...(sides?.b ?? [])]).size).toBe(total)
}

describe('isRandomDrawLhcRenyiDuipengConfig', () => {
  it('keeps legacy renyi_dp configurations on the A/B random-draw branch', () => {
    expect(
      isRandomDrawLhcRenyiDuipengConfig({
        playTemplate: 'legacy_lhc',
        betMode: 'renyi_dp',
      }),
    ).toBe(true)
  })
})

describe('randomLhcRenyiDuipengContent', () => {
  it('clamps one pick to two and puts one distinct number in each zone', () => {
    expectValidRandomContent(2)
    const sides = parseLhcRenyiDuipengSides(randomLhcRenyiDuipengContent(1, () => 0))
    expect(sides?.a).toHaveLength(1)
    expect(sides?.b).toHaveLength(1)
  })

  it('creates valid cross-zone-distinct content for every allowed total', () => {
    for (let total = 3; total <= 10; total++) expectValidRandomContent(total)
  })

  it('clamps totals above ten while retaining valid two-zone content', () => {
    const content = randomLhcRenyiDuipengContent(99, () => 0)
    const sides = parseLhcRenyiDuipengSides(content)
    expect((sides?.a.length ?? 0) + (sides?.b.length ?? 0)).toBe(10)
    expect(validateLhcRenyiDuipengContent(content).ok).toBe(true)
  })

  it('keeps valid A|B content when a trigger row is serialized', () => {
    expect(normalizeLhcRenyiDuipengTriggerContent('1,2|3,4')).toBe('01,02|03,04')
  })
})

describe('任意对碰随机双区数量', () => {
  it('兼容旧单总数并钳制非法双区数量', () => {
    expect(normalizeLhcRenyiDuipengRandomCounts([5])).toEqual([2, 3])
    expect(normalizeLhcRenyiDuipengRandomCounts([0, 2])).toEqual([1, 2])
    expect(normalizeLhcRenyiDuipengRandomCounts([9, 9])).toEqual([9, 1])
    expect(normalizeLhcRenyiDuipengRandomCounts([])).toEqual([1, 1])
  })

  it('按指定的 A/B 数量生成不重复的合法号码', () => {
    const content = randomLhcRenyiDuipengContentForCounts(4, 6, () => 0)
    const validation = validateLhcRenyiDuipengContent(content)
    expect(validation.ok).toBe(true)
    if (!validation.ok) return

    const sides = parseLhcRenyiDuipengSides(validation.normalized)
    expect(sides?.a).toHaveLength(4)
    expect(sides?.b).toHaveLength(6)
    expect(new Set([...(sides?.a ?? []), ...(sides?.b ?? [])]).size).toBe(10)
  })
})
