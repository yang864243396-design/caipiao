import { describe, expect, it } from 'vitest'
import { parseLhcRenyiDuipengSides, validateLhcRenyiDuipengContent } from './betPayload'
import { normalizeLhcRenyiDuipengTriggerContent, randomLhcRenyiDuipengContent } from './lhcRenyiDuipengRandom'

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
