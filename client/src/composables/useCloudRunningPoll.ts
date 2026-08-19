import type {
  WsCloudStatsSnapshotPayload,
  WsEnvelope,
  WsSchemeInstancesSnapshotPayload,
} from '@shared/types/ws'
import { WS_EVENTS } from '@shared/types/ws'

import {
  CLOUD_REALTIME_CLIENT_ENABLED,
  WS_CLIENT_BASE,
  WS_CLIENT_ENABLED,
} from '@/api/config'
import { getAccessToken } from '@/api/client'
import {
  fetchCloudCenterStats,
  fetchRunningSchemesByIds,
  instanceToDisplay,
  markCloudSchemePatch,
  type CloudCenterStatsDto,
  type CloudSchemeCard,
} from '@/api/cloud/center'
import { connectClientWs, isCloudRefreshEvent } from '@/composables/ws/useClientWs'

const FALLBACK_POLL_MS = 15_000
const WS_CONNECTED_POLL_MS = 60_000

export interface CloudRunningSyncHandlers {
  onSchemes(cards: CloudSchemeCard[], removedIds: string[]): void
  onStats(stats: CloudCenterStatsDto): void
}

export interface CloudRunningSync {
  stop(): void
  reconcile(): Promise<void>
  /** @deprecated 使用 reconcile */
  refresh(): Promise<void>
}

type BufferedSnapshot =
  | { kind: 'schemes'; payload: WsSchemeInstancesSnapshotPayload }
  | { kind: 'stats'; payload: WsCloudStatsSnapshotPayload }

const RFC3339_UTC_NANO_PATTERN =
  /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(\d{1,9}))?Z$/

function normalizeRFC3339UtcNano(value: string): string | null {
  const match = RFC3339_UTC_NANO_PATTERN.exec(value)
  if (!match) return null

  const wholeSecond = match[1]
  if (!Number.isFinite(Date.parse(`${wholeSecond}Z`))) return null

  return `${wholeSecond}.${(match[2] ?? '').padEnd(9, '0')}`
}

function compareVersions(candidate: string, current: string): number {
  const candidateCanonical = normalizeRFC3339UtcNano(candidate)
  const currentCanonical = normalizeRFC3339UtcNano(current)
  if (candidateCanonical && currentCanonical) {
    if (candidateCanonical === currentCanonical) return 0
    return candidateCanonical < currentCanonical ? -1 : 1
  }

  const candidateTime = Date.parse(candidate)
  const currentTime = Date.parse(current)
  if (Number.isFinite(candidateTime) && Number.isFinite(currentTime)) {
    if (candidateTime === currentTime) return 0
    return candidateTime < currentTime ? -1 : 1
  }

  if (candidate === current) return 0
  return candidate < current ? -1 : 1
}

function isVersionOlder(candidate: string, current: string): boolean {
  if (!candidate || !current) return false
  return compareVersions(candidate, current) < 0
}

function isVersionNewer(candidate: string, current: string): boolean {
  if (!candidate) return false
  if (!current) return true
  return compareVersions(candidate, current) > 0
}

function isSchemeSnapshot(
  event: WsEnvelope,
): event is WsEnvelope<WsSchemeInstancesSnapshotPayload> {
  const payload = event.payload as Partial<WsSchemeInstancesSnapshotPayload> | undefined
  return event.name === WS_EVENTS.schemeInstancesSnapshot &&
    payload?.schemaVersion === 1 &&
    Array.isArray(payload.items) &&
    Array.isArray(payload.removedIds)
}

function isStatsSnapshot(
  event: WsEnvelope,
): event is WsEnvelope<WsCloudStatsSnapshotPayload> {
  const payload = event.payload as Partial<WsCloudStatsSnapshotPayload> | undefined
  return event.name === WS_EVENTS.cloudStatsSnapshot &&
    payload?.schemaVersion === 1 &&
    payload.stats != null
}

/**
 * 云端中心运行卡片同步：版本化 WS 快照为主，每个订阅周期仅做一次 REST 对账。
 * 旧回调签名暂时保留到 CloudCenterView 完成迁移。
 */
export function startCloudRunningSync(
  getLoadedIds: () => string[],
  handlers: CloudRunningSyncHandlers | ((cards: CloudSchemeCard[]) => void),
  options?: { legacyFallbackMs?: number } | number,
): CloudRunningSync {
  const compatibilityHandler = typeof handlers === 'function' ? handlers : null
  const resolvedHandlers: CloudRunningSyncHandlers = typeof handlers === 'function'
    ? { onSchemes: (cards) => handlers(cards), onStats: () => undefined }
    : handlers
  const legacyFallbackMs = typeof options === 'number'
    ? options
    : options?.legacyFallbackMs ?? FALLBACK_POLL_MS

  let stopped = false
  let pollTimer: ReturnType<typeof setInterval> | null = null
  let stopWs: (() => void) | null = null
  let reconcilePromise: Promise<void> | null = null
  let bufferedSchemeMessages: BufferedSnapshot[] = []
  let hasVersionedSnapshot = false
  let subscribed = false
  let subscribedCycle = 0
  let startedCycle = 0
  let queuedCycle = 0
  let activeReconcileKey = ''
  let activeReconcileCycle = -1
  let lastCompletedKey = ''
  let lastCompletedCycle = -1
  const versions = new Map<string, string>()
  let statsVersion = ''
  const token = getAccessToken()

  const realtimeMode = Boolean(
    CLOUD_REALTIME_CLIENT_ENABLED && WS_CLIENT_ENABLED && WS_CLIENT_BASE && token,
  )

  function loadedIDs(): string[] {
    return Array.from(new Set(getLoadedIds().filter(Boolean)))
  }

  function loadedIDsKey(ids: string[]): string {
    return [...ids].sort().join('\u0000')
  }

  function emitSchemes(
    cards: CloudSchemeCard[],
    removedIds: string[],
    partial: boolean,
  ): void {
    if (compatibilityHandler) {
      compatibilityHandler(partial ? markCloudSchemePatch(cards, removedIds) : cards)
      return
    }
    resolvedHandlers.onSchemes(cards, removedIds)
  }

  function applySchemeSnapshot(payload: WsSchemeInstancesSnapshotPayload): void {
    if (stopped) return
    const loaded = new Set(loadedIDs())
    const cards: CloudSchemeCard[] = []

    for (const item of payload.items) {
      if (!loaded.has(item.id)) continue
      const currentVersion = versions.get(item.id)
      if (currentVersion && isVersionOlder(item.updatedAt, currentVersion)) continue
      versions.set(item.id, item.updatedAt)
      cards.push(instanceToDisplay({
        ...item,
        lotteryName: item.lotteryName || item.lotteryLabel || '',
      }))
    }

    const removedIds = payload.removedIds.filter((id) => {
      if (!loaded.has(id)) return false
      const currentVersion = versions.get(id)
      if (currentVersion && isVersionOlder(payload.generatedAt, currentVersion)) return false
      versions.delete(id)
      return true
    })

    if (cards.length || removedIds.length) {
      emitSchemes(cards, removedIds, true)
    }
  }

  function applyStats(stats: CloudCenterStatsDto, generatedAt?: string): void {
    if (stopped) return
    if (generatedAt) {
      if (!isVersionNewer(generatedAt, statsVersion)) return
      statsVersion = generatedAt
    } else if (statsVersion) {
      return
    }
    resolvedHandlers.onStats(stats)
  }

  function applyStatsSnapshot(payload: WsCloudStatsSnapshotPayload): void {
    applyStats(payload.stats, payload.generatedAt)
  }

  function applyBufferedSnapshot(message: BufferedSnapshot): void {
    if (message.kind === 'schemes') applySchemeSnapshot(message.payload)
    else applyStatsSnapshot(message.payload)
  }

  async function performReconciliation(ids: string[], includeStats: boolean): Promise<boolean> {
    try {
      const rowsPromise = fetchRunningSchemesByIds(ids)
      const [rows, stats] = includeStats
        ? await Promise.all([rowsPromise, fetchCloudCenterStats()])
        : [await rowsPromise, null]
      if (stopped) return false
      if (compatibilityHandler && loadedIDsKey(loadedIDs()) !== loadedIDsKey(ids)) {
        if (stats) applyStats(stats, stats.generatedAt)
        return true
      }

      const cards = rows.map(instanceToDisplay)
      const returnedIDs = new Set(cards.map((card) => card.id))
      const removedIds = ids.filter((id) => !returnedIDs.has(id))
      for (const card of cards) versions.set(card.id, card.updatedAt)
      for (const id of removedIds) versions.delete(id)
      emitSchemes(cards, removedIds, false)
      if (stats) applyStats(stats, stats.generatedAt)
      return true
    } catch {
      // Keep the last valid state and allow a later reconciliation to retry.
      return false
    } finally {
      const buffered = bufferedSchemeMessages
      bufferedSchemeMessages = []
      for (const message of buffered) applyBufferedSnapshot(message)
    }
  }

  function startReconciliation(includeStats = true): Promise<void> {
    if (stopped) return Promise.resolve()
    if (reconcilePromise) return reconcilePromise

    const ids = loadedIDs()
    const key = loadedIDsKey(ids)
    const cycle = subscribedCycle
    activeReconcileKey = key
    activeReconcileCycle = cycle
    const running = performReconciliation(ids, includeStats).then((completed) => {
      if (completed) {
        lastCompletedKey = key
        lastCompletedCycle = cycle
      }
    })
    reconcilePromise = running.finally(() => {
      reconcilePromise = null
      activeReconcileKey = ''
      activeReconcileCycle = -1
      if (!stopped && queuedCycle > startedCycle) {
        startedCycle = queuedCycle
        void startReconciliation()
      }
    })
    return reconcilePromise
  }

  function reconcile(): Promise<void> {
    return startReconciliation()
  }

  function compatibilityRefresh(): Promise<void> {
    if (!compatibilityHandler || !realtimeMode || subscribedCycle === 0) {
      return startReconciliation(compatibilityHandler ? false : true)
    }

    const key = loadedIDsKey(loadedIDs())
    const cycle = subscribedCycle
    if (reconcilePromise) {
      if (activeReconcileKey === key && activeReconcileCycle === cycle) {
        return reconcilePromise
      }
      return reconcilePromise.then(() => compatibilityRefresh())
    }
    if (lastCompletedKey === key && lastCompletedCycle === cycle) {
      return Promise.resolve()
    }
    return startReconciliation(false)
  }

  function requestCycleReconciliation(cycle: number): void {
    if (cycle <= startedCycle) return
    if (reconcilePromise) {
      queuedCycle = Math.max(queuedCycle, cycle)
      return
    }
    startedCycle = cycle
    void startReconciliation()
  }

  function stopPoll(): void {
    if (!pollTimer) return
    window.clearInterval(pollTimer)
    pollTimer = null
  }

  function startPoll(intervalMs: number): void {
    stopPoll()
    pollTimer = window.setInterval(() => {
      void reconcile()
    }, intervalMs)
  }

  function handleEvent(event: WsEnvelope): void {
    if (isSchemeSnapshot(event)) {
      hasVersionedSnapshot = true
      const message: BufferedSnapshot = { kind: 'schemes', payload: event.payload! }
      if (reconcilePromise) bufferedSchemeMessages.push(message)
      else applyBufferedSnapshot(message)
      return
    }
    if (isStatsSnapshot(event)) {
      hasVersionedSnapshot = true
      const message: BufferedSnapshot = { kind: 'stats', payload: event.payload! }
      if (reconcilePromise) bufferedSchemeMessages.push(message)
      else applyBufferedSnapshot(message)
      return
    }
    if (!hasVersionedSnapshot && isCloudRefreshEvent(event)) {
      void reconcile()
    }
  }

  const wsAvailable = Boolean(WS_CLIENT_ENABLED && WS_CLIENT_BASE && token)
  if (!realtimeMode) startPoll(legacyFallbackMs)

  if (wsAvailable && token) {
    stopWs = connectClientWs(WS_CLIENT_BASE, token, handleEvent, {
      onConnected: () => {
        if (stopped || subscribed) return
        subscribed = true
        if (realtimeMode) {
          subscribedCycle += 1
          requestCycleReconciliation(subscribedCycle)
        } else {
          startPoll(WS_CONNECTED_POLL_MS)
        }
      },
      onDisconnected: () => {
        subscribed = false
        if (!realtimeMode && !stopped) startPoll(legacyFallbackMs)
      },
    })
  }

  function stop(): void {
    stopped = true
    stopWs?.()
    stopWs = null
    stopPoll()
    bufferedSchemeMessages = []
  }

  return { stop, reconcile, refresh: compatibilityRefresh }
}

export function cloudRunningPollMs(
  wsConnected = false,
  fallbackMs = FALLBACK_POLL_MS,
): number {
  return wsConnected ? WS_CONNECTED_POLL_MS : fallbackMs
}

/** @deprecated 使用 startCloudRunningSync */
export { startCloudRunningSync as startCloudRunningPoll }
