import { getSchemeDefinition } from '@/api/schemes/definitions'
import { fetchLotteryRouteStatus, type LotteryRouteStatus } from '@/api/games/lotteries'
import { isLegacyLotteryCode } from '@/constants/legacyLotteryCodes'
import type { RouteLocationNormalized } from 'vue-router'

export type LotteryRouteBlock = 'offline' | 'maintenance' | null

const STATUS_CACHE_TTL_MS = 30_000

type CachedStatus = { status: LotteryRouteStatus; at: number }

const statusCache = new Map<string, CachedStatus>()

export function extractLotteryCodeFromRoute(to: RouteLocationNormalized): string {
  if (to.name === 'play-detail') {
    return String(to.query.lotteryCode ?? '').trim()
  }
  if (to.name === 'advanced-scheme-edit' || to.name === 'advanced-scheme-rounds') {
    return String(to.query.lottery ?? '').trim()
  }
  return ''
}

export function extractSchemeDefinitionIdFromRoute(to: RouteLocationNormalized): string {
  if (to.name === 'scheme-detail') {
    return String(to.params.definitionId ?? to.query.definitionId ?? '').trim()
  }
  return ''
}

export function routeNeedsLotteryGuard(to: RouteLocationNormalized): boolean {
  return !!extractLotteryCodeFromRoute(to) || !!extractSchemeDefinitionIdFromRoute(to)
}

function readCachedStatus(code: string): LotteryRouteStatus | null {
  const entry = statusCache.get(code)
  if (!entry) return null
  if (Date.now() - entry.at > STATUS_CACHE_TTL_MS) {
    statusCache.delete(code)
    return null
  }
  return entry.status
}

export async function resolveLotteryRouteBlock(code: string): Promise<LotteryRouteBlock> {
  const trimmed = code.trim()
  if (!trimmed) return null
  if (isLegacyLotteryCode(trimmed)) return 'offline'

  let status = readCachedStatus(trimmed)
  if (!status) {
    status = await fetchLotteryRouteStatus(trimmed)
    statusCache.set(trimmed, { status, at: Date.now() })
  }
  if (status.legacy || !status.exists) return 'offline'
  if (status.saleStatus === 'maintenance') return 'maintenance'
  return null
}

export async function resolveRouteLotteryBlock(
  to: RouteLocationNormalized,
): Promise<LotteryRouteBlock> {
  const code = extractLotteryCodeFromRoute(to)
  if (code) return resolveLotteryRouteBlock(code)

  const definitionId = extractSchemeDefinitionIdFromRoute(to)
  if (!definitionId) return null

  try {
    const def = await getSchemeDefinition(definitionId)
    if (!def.lotteryCode?.trim()) return null
    return resolveLotteryRouteBlock(def.lotteryCode)
  } catch {
    return null
  }
}

export function lotteryRouteToastMessage(block: LotteryRouteBlock): string {
  if (block === 'maintenance') return '该彩种维护中'
  if (block === 'offline') return '该彩种已下线'
  return ''
}
