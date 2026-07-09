/** 高级倍投 · 局次规则（平台模板与会员方案模板共用） */
export interface SchemeRoundRule {
  mult: number
  afterHit: number
  afterMiss: number
}

export const SCHEME_ROUND_MULT_CAP = 200_000

export function defaultSchemeRoundRules(): SchemeRoundRule[] {
  return [
    { mult: 0, afterHit: 2, afterMiss: 1 },
    { mult: 1, afterHit: 2, afterMiss: 3 },
    { mult: 3, afterHit: 2, afterMiss: 1 },
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
  const bad = rows.some((r) => !Number.isFinite(r.mult) || r.mult > SCHEME_ROUND_MULT_CAP)
  if (bad) return `倍数须在 0～${SCHEME_ROUND_MULT_CAP} 之间`
  return null
}
