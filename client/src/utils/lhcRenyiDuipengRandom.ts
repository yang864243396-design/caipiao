import { formatLhcRenyiDuipengContent, validateLhcRenyiDuipengContent } from './betPayload'

const LHC_RENYI_DUIPENG_MIN_PICKS = 2
const LHC_RENYI_DUIPENG_MAX_PICKS = 10
const LHC_RENYI_DUIPENG_NUMBERS = Array.from({ length: 49 }, (_, i) => String(i + 1).padStart(2, '0'))

function normalizedRandom(random: () => number): number {
  const value = Number(random())
  if (!Number.isFinite(value)) return 0
  return Math.min(1 - Number.EPSILON, Math.max(0, value))
}

function shuffle<T>(items: T[], random: () => number): void {
  for (let i = items.length - 1; i > 0; i--) {
    const j = Math.floor(normalizedRandom(random) * (i + 1))
    ;[items[i], items[j]] = [items[j]!, items[i]!]
  }
}

/**
 * 任意对碰高级开某投某行落库前的专用规范化。
 * 合法 A|B 保持双区结构；不合法内容不丢弃，交由既有前后端校验返回具体提示。
 */
export function normalizeLhcRenyiDuipengTriggerContent(raw: string): string {
  const text = String(raw ?? '').replace(/，/g, ',').trim()
  const validation = validateLhcRenyiDuipengContent(text)
  return validation.ok ? validation.normalized : text
}

/**
 * 任意对碰专用随机内容：总数钳制到 2–10，A/B 各至少一个，跨区无重复。
 */
export function randomLhcRenyiDuipengContent(total: number, random: () => number = Math.random): string {
  const count = Math.min(
    LHC_RENYI_DUIPENG_MAX_PICKS,
    Math.max(LHC_RENYI_DUIPENG_MIN_PICKS, Math.trunc(Number(total) || LHC_RENYI_DUIPENG_MIN_PICKS)),
  )
  const pool = [...LHC_RENYI_DUIPENG_NUMBERS]
  shuffle(pool, random)
  const aCount = 1 + Math.floor(normalizedRandom(random) * (count - 1))
  return formatLhcRenyiDuipengContent(pool.slice(0, aCount), pool.slice(aCount, count))
}
