/** 高级倍投 · 局次规则（平台模板与会员方案模板共用） */
export interface SchemeRoundRule {
  mult: number
  afterHit: number
  afterMiss: number
}

export const SCHEME_ROUND_MULT_CAP = 200_000

export function defaultSchemeRoundRules(): SchemeRoundRule[] {
  // 默认挂翻倍（1-based 局号）：未中进下一局、命中回第 1 局；倍数为正整数
  return [
    { mult: 1, afterHit: 1, afterMiss: 2 },
    { mult: 2, afterHit: 1, afterMiss: 3 },
    { mult: 4, afterHit: 1, afterMiss: 1 },
  ]
}

export function normalizeSchemeRoundRules(raw: unknown): SchemeRoundRule[] | null {
  if (!Array.isArray(raw) || raw.length === 0) return null
  const parsed = raw
    .map((item) => {
      if (item == null || typeof item !== 'object') return null
      const row = item as Record<string, unknown>
      const mult = Number(row.mult)
      const afterHit = Number(row.afterHit)
      const afterMiss = Number(row.afterMiss)
      if (!Number.isFinite(mult) || !Number.isFinite(afterHit) || !Number.isFinite(afterMiss)) {
        return null
      }
      return { mult, afterHit, afterMiss }
    })
    .filter((r): r is SchemeRoundRule => r != null)
  return parsed.length > 0 ? parsed : null
}

export function schemeRoundRulesFromConfig(config?: { rounds?: unknown }): SchemeRoundRule[] {
  return normalizeSchemeRoundRules(config?.rounds) ?? defaultSchemeRoundRules()
}

export function validateSchemeRoundRules(rows: SchemeRoundRule[]): string | null {
  if (rows.length === 0) return '请至少配置一局'
  const bad = rows.some(
    (r) => !Number.isFinite(r.mult) || r.mult < 1 || r.mult > SCHEME_ROUND_MULT_CAP,
  )
  if (bad) return `倍数须为 1～${SCHEME_ROUND_MULT_CAP} 的正数，且不能为 0`
  return null
}
