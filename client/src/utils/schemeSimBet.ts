/** 从 definition config 读取 simBet；兼容旧 runMode prod/sim/real */
export function simBetFromSchemeConfig(cfg: Record<string, unknown> | undefined | null): boolean {
  if (!cfg) return false
  if (typeof cfg.simBet === 'boolean') return cfg.simBet
  const rm = String(cfg.runMode ?? '').trim().toLowerCase()
  return rm === 'sim'
}

export function simBetLabel(simBet: boolean): string {
  return simBet ? '模拟' : '正式'
}

/** session 草稿等遗留 runMode 字段迁移 */
export function simBetFromLegacyRunMode(raw: unknown): boolean | undefined {
  if (raw === 'sim') return true
  if (raw === 'prod' || raw === 'real') return false
  return undefined
}
