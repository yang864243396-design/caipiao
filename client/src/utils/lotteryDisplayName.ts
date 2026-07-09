import type { LotteryFilterOption } from '@/api/games/lotteries'
import type { PublicLotteryRow } from '@/types/playCatalog'

const BAD_NAME = /^[\?？.\s]+$/

/** 合并公开彩种名与会员筛选项，避免 displayName 损坏时显示 ? */
export function buildLotteryNameMap(
  publicRows: PublicLotteryRow[],
  memberRows: LotteryFilterOption[] = [],
): Map<string, string> {
  const map = new Map<string, string>()
  for (const row of publicRows) {
    const name = row.displayName?.trim()
    if (name && !BAD_NAME.test(name)) map.set(row.code, name)
  }
  for (const row of memberRows) {
    const name = row.displayName?.trim()
    if (name && !BAD_NAME.test(name)) map.set(row.code, name)
  }
  return map
}

export function lotteryFilterLabel(
  code: string,
  saleStatus: LotteryFilterOption['saleStatus'] | undefined,
  nameMap: Map<string, string>,
): string {
  const base = nameMap.get(code)?.trim() || code
  return saleStatus === 'maintenance' ? `${base}（维护）` : base
}

export function resolveLotteryDisplayName(
  code: string,
  rawName: string | undefined | null,
  nameMap: Map<string, string>,
): string {
  const fromMap = nameMap.get(code)?.trim()
  if (fromMap) return fromMap
  const trimmed = rawName?.trim() ?? ''
  if (trimmed && !BAD_NAME.test(trimmed)) return trimmed
  return code
}
