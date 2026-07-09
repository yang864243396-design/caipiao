/** 投注模式：展示中文，存储为金额数值（厘 0.001 / 分 0.01 / 角 0.1 / 元 1） */
export const BET_MODE_OPTIONS = [
  { label: '1厘', value: '0.001' },
  { label: '2厘', value: '0.002' },
  { label: '1分', value: '0.01' },
  { label: '2分', value: '0.02' },
  { label: '1角', value: '0.1' },
  { label: '2角', value: '0.2' },
  { label: '1元', value: '1' },
  { label: '2元', value: '2' },
] as const

export function betModeLabelOf(value: string): string {
  const n = Number(value)
  const hit = BET_MODE_OPTIONS.find((o) => Number(o.value) === n)
  return hit ? hit.label : value || '—'
}

export function isBetUnitValue(raw: string): boolean {
  const t = raw.trim()
  if (!t) return false
  return BET_MODE_OPTIONS.some((o) => o.value === t || Number(o.value) === Number(t))
}

export function normalizeBetUnitValue(raw: string): string {
  const t = raw.trim()
  if (!t) return '2'
  const hit = BET_MODE_OPTIONS.find((o) => o.value === t || Number(o.value) === Number(t))
  if (hit) return hit.value
  const n = Number(t)
  if (Number.isFinite(n) && n > 0) return String(n)
  return '2'
}

export function betUnitFromSchemeConfig(cfg: Record<string, unknown>): string {
  const unitRaw = cfg.betUnit != null ? String(cfg.betUnit).trim() : ''
  if (unitRaw) return normalizeBetUnitValue(unitRaw)
  const legacy = cfg.betMode != null ? String(cfg.betMode).trim() : ''
  if (isBetUnitValue(legacy)) return normalizeBetUnitValue(legacy)
  return '2'
}
