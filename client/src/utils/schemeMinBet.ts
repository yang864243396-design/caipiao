/**
 * 第三方单次投注最低金额：投注单位 × 方案倍数系数 × 模式最低倍率 × 注数
 * USDT 最低 0.1，其他币种最低 1
 */

import { betUnitFromSchemeConfig } from '@/constants/betModeOptions'
import { countBetUnits, resolvePlayConfig } from '@/utils/betPayload'

export const MIN_SINGLE_BET_AMOUNT_USDT = 0.1
export const MIN_SINGLE_BET_AMOUNT_OTHER = 1

/** @deprecated 兼容旧引用，等同 USDT 阈值 */
export const MIN_SINGLE_BET_AMOUNT = MIN_SINGLE_BET_AMOUNT_USDT

export function minSingleBetAmountForCurrency(currency?: string | null): number {
  return String(currency ?? '').trim().toUpperCase() === 'USDT'
    ? MIN_SINGLE_BET_AMOUNT_USDT
    : MIN_SINGLE_BET_AMOUNT_OTHER
}

export function minBetOpenMessage(currency?: string | null): string {
  const minAmt = minSingleBetAmountForCurrency(currency)
  const label = Number.isInteger(minAmt) ? String(minAmt) : String(minAmt)
  return `单次投注金额低于${label} 请提高投注单位、倍数系数、倍投倍率或注数后再开启`
}

/** @deprecated 兼容旧引用；实际文案请用 minBetOpenMessage(currency) */
export const MIN_BET_OPEN_MSG = minBetOpenMessage('USDT')

function toPositiveNumber(raw: unknown, fallback = 1): number {
  const n = Number(raw)
  if (!Number.isFinite(n) || n <= 0) return fallback
  return n
}

function collectModeMultipliers(cfg: Record<string, unknown>): number[] {
  const out: number[] = []

  const rounds = cfg.rounds
  if (Array.isArray(rounds)) {
    for (const row of rounds) {
      if (!row || typeof row !== 'object') continue
      const m = toPositiveNumber((row as Record<string, unknown>).mult, 0)
      if (m > 0) out.push(m)
    }
  }
  if (out.length) return out

  const bm = cfg.betMultiplier
  if (!bm || typeof bm !== 'object') return out
  const payload = bm as Record<string, unknown>
  const kind = String(payload.kind ?? '')

  const pushFromProfitTable = (section: unknown) => {
    if (!section || typeof section !== 'object') return
    const table = (section as Record<string, unknown>).profitTable
    if (!Array.isArray(table)) return
    for (const row of table) {
      if (!row || typeof row !== 'object') continue
      const m = toPositiveNumber((row as Record<string, unknown>).mult, 0)
      if (m > 0) out.push(m)
    }
  }

  if (kind === '0') pushFromProfitTable(payload.newbie)
  else if (kind === '1') pushFromProfitTable(payload.oneclick)
  else if (kind === '2') {
    const simple = payload.simple
    if (simple && typeof simple === 'object') {
      const multiples = String((simple as Record<string, unknown>).multiples ?? '')
      for (const part of multiples.split(/[,，\s\n]+/)) {
        const m = toPositiveNumber(part, 0)
        if (m > 0) out.push(m)
      }
    }
  } else if (kind === '3') {
    const adv = payload.advanced
    if (adv && typeof adv === 'object') {
      const advRounds = (adv as Record<string, unknown>).rounds
      if (Array.isArray(advRounds)) {
        for (const row of advRounds) {
          if (!row || typeof row !== 'object') continue
          const m = toPositiveNumber((row as Record<string, unknown>).mult, 0)
          if (m > 0) out.push(m)
        }
      }
    }
  }

  return out
}

/** 方案模式中的最低有效倍率；无配置时按 1 */
export function schemeMinModeMultiplier(cfg: Record<string, unknown> | undefined | null): number {
  if (!cfg) return 1
  const mults = collectModeMultipliers(cfg)
  if (!mults.length) return 1
  return Math.min(...mults)
}

/** 方案各组中的最低注数；无有效内容时按 1 */
export function schemeMinBetUnits(cfg: Record<string, unknown> | undefined | null): number {
  if (!cfg) return 1
  const playConfig = resolvePlayConfig({
    playTypeId: String(cfg.typeId ?? cfg.playTypeId ?? ''),
    subPlayId: String(cfg.subId ?? cfg.subPlayId ?? ''),
    betMode: isBetModeLike(cfg.betMode) ? String(cfg.betMode) : '',
  })
  const groups = Array.isArray(cfg.schemeGroups)
    ? cfg.schemeGroups.map((g) => String(g ?? ''))
    : []
  let minUnits = 0
  for (const g of groups) {
    const u = countBetUnits(playConfig, g)
    if (u <= 0) continue
    if (minUnits === 0 || u < minUnits) minUnits = u
  }
  return minUnits > 0 ? minUnits : 1
}

function isBetModeLike(raw: unknown): boolean {
  const s = String(raw ?? '').trim()
  if (!s) return false
  return !/^[0-9.]+$/.test(s)
}

/** 最低单次投注金额 = 投注单位 × 倍数系数 × 模式最低倍率 × 注数 */
export function schemeMinSingleBetAmount(
  cfg: Record<string, unknown> | undefined | null,
  multiplierCoef: string | number | null | undefined,
): number {
  const unit = toPositiveNumber(betUnitFromSchemeConfig(cfg ?? {}), 2)
  const coef = toPositiveNumber(multiplierCoef, 1)
  const modeMult = schemeMinModeMultiplier(cfg)
  const betUnits = schemeMinBetUnits(cfg)
  return Math.round(unit * coef * modeMult * betUnits * 100) / 100
}

/** 开启前校验；通过返回 null，否则返回提示文案 */
export function schemeMinBetOpenError(
  cfg: Record<string, unknown> | undefined | null,
  multiplierCoef: string | number | null | undefined,
  currency?: string | null,
): string | null {
  const amount = schemeMinSingleBetAmount(cfg, multiplierCoef)
  const minAmt = minSingleBetAmountForCurrency(currency)
  if (amount + 1e-9 < minAmt) return minBetOpenMessage(currency)
  return null
}
