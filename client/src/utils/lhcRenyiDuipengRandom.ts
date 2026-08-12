import { formatLhcRenyiDuipengContent, validateLhcRenyiDuipengContent } from './betPayload'

const LHC_RENYI_DUIPENG_MIN_PICKS = 2
const LHC_RENYI_DUIPENG_MAX_PICKS = 10
const LHC_RENYI_DUIPENG_NUMBERS = Array.from({ length: 49 }, (_, i) => String(i + 1).padStart(2, '0'))

/**
 * 历史方案可能缺少可供玩法识别器使用的完整六合彩模板字段；随机出号仍应按
 * 持久化的 betMode 进入 A/B 双区分支。
 */
export function isRandomDrawLhcRenyiDuipengConfig(config: {
  betMode?: unknown
  playTemplate?: unknown
}): boolean {
  return String(config.betMode ?? '').toLowerCase() === 'renyi_dp'
}

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
 * 任意对碰随机出号数量：新配置为 [A区个数, B区个数]。
 * 兼容旧的单总数配置，且始终收敛到两区各至少一个、合计最多十个。
 */
export function normalizeLhcRenyiDuipengRandomCounts(raw: unknown): [number, number] {
  const values = Array.isArray(raw) ? raw.map((value) => Math.trunc(Number(value))) : []
  if (values.length === 1) {
    const total = Math.min(
      LHC_RENYI_DUIPENG_MAX_PICKS,
      Math.max(LHC_RENYI_DUIPENG_MIN_PICKS, values[0] || LHC_RENYI_DUIPENG_MIN_PICKS),
    )
    const aCount = Math.floor(total / 2)
    return [aCount, total - aCount]
  }

  const aCount = Math.min(9, Math.max(1, values[0] || 1))
  const bCount = Math.min(LHC_RENYI_DUIPENG_MAX_PICKS - aCount, Math.max(1, values[1] || 1))
  return [aCount, bCount]
}

/** 任意对碰随机出号：按指定 A/B 数量从同一 01–49 号池依次抽取，跨区不重复。 */
export function randomLhcRenyiDuipengContentForCounts(
  aCount: number,
  bCount: number,
  random: () => number = Math.random,
): string {
  const [a, b] = normalizeLhcRenyiDuipengRandomCounts([aCount, bCount])
  const pool = [...LHC_RENYI_DUIPENG_NUMBERS]
  shuffle(pool, random)
  return formatLhcRenyiDuipengContent(pool.slice(0, a), pool.slice(a, a + b))
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
  const aCount = 1 + Math.floor(normalizedRandom(random) * (count - 1))
  return randomLhcRenyiDuipengContentForCounts(aCount, count - aCount, random)
}
