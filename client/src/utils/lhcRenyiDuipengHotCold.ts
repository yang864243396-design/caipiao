const LHC_RENYI_DUIPENG_HCW_MAX_RANKS = 10

export type LhcRenyiDuipengHotColdRanks = {
  a: number[]
  b: number[]
  valid: boolean
}

function normalizeZone(raw: unknown, orderLength: number): number[] {
  if (!Array.isArray(raw) || orderLength <= 0) return []
  const out: number[] = []
  const seen = new Set<number>()
  for (const value of raw) {
    const rank = Math.trunc(Number(value))
    if (!Number.isInteger(rank) || rank < 0 || rank >= orderLength || seen.has(rank)) continue
    seen.add(rank)
    out.push(rank)
  }
  return out
}

/** 任意对碰冷热配置：ranks[0] 为 A区，ranks[1] 为 B区。 */
export function normalizeLhcRenyiDuipengHotColdRanks(
  ranks: unknown,
  orderLength: number,
): LhcRenyiDuipengHotColdRanks {
  const rows = Array.isArray(ranks) ? ranks : []
  const a = normalizeZone(rows[0], orderLength)
  const b = normalizeZone(rows[1], orderLength)
  const aSet = new Set(a)
  const overlap = b.some((rank) => aSet.has(rank))
  return {
    a,
    b,
    valid: a.length > 0 && b.length > 0 && !overlap && a.length + b.length <= LHC_RENYI_DUIPENG_HCW_MAX_RANKS,
  }
}

/** 用候选名次替换一个分区，自动排除另一分区名次并保留总计十个的上限。 */
export function replaceLhcRenyiDuipengHotColdRanks(
  ranks: unknown,
  zone: 0 | 1,
  candidates: unknown,
  orderLength: number,
): number[][] {
  const rows = Array.isArray(ranks) ? ranks : []
  const otherZone: 0 | 1 = zone === 0 ? 1 : 0
  const other = normalizeZone(rows[otherZone], orderLength)
  const otherSet = new Set(other)
  const capacity = Math.max(0, LHC_RENYI_DUIPENG_HCW_MAX_RANKS - other.length)
  const selected = normalizeZone(candidates, orderLength)
    .filter((rank) => !otherSet.has(rank))
    .slice(0, capacity)
  return zone === 0 ? [selected, other] : [other, selected]
}
