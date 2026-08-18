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

function isVersionOlder(candidate: string, current: string): boolean {
  const candidateTime = Date.parse(candidate)
  const currentTime = Date.parse(current)
  if (Number.isFinite(candidateTime) && Number.isFinite(currentTime)) {
    return candidateTime < currentTime
  }
  return Boolean(candidate && current && candidate < current)
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
  const versions = new Map<string, string>()
  const token = getAccessToken()

  const realtimeMode = Boolean(
    CLOUD_REALTIME_CLIENT_ENABLED && WS_CLIENT_ENABLED && WS_CLIENT_BASE && token,
  )

  function loadedIDs(): string[] {
    return Array.from(new Set(getLoadedIds().filter(Boolean)))
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
      resolvedHandlers.onSchemes(cards, removedIds)
    }
  }

  function applyStatsSnapshot(payload: WsCloudStatsSnapshotPayload): void {
    if (!stopped) resolvedHandlers.onStats(payload.stats)
  }

  function applyBufferedSnapshot(message: BufferedSnapshot): void {
    if (message.kind === 'schemes') applySchemeSnapshot(message.payload)
    else applyStatsSnapshot(message.payload)
  }

  async function performReconciliation(): Promise<void> {
    const ids = loadedIDs()
    try {
      const [rows, stats] = await Promise.all([
        fetchRunningSchemesByIds(ids),
        fetchCloudCenterStats(),
      ])
      if (stopped) return

      const cards = rows.map(instanceToDisplay)
      const returnedIDs = new Set(cards.map((card) => card.id))
      const removedIds = ids.filter((id) => !returnedIDs.has(id))
      for (const card of cards) versions.set(card.id, card.updatedAt)
      for (const id of removedIds) versions.delete(id)
      resolvedHandlers.onSchemes(cards, removedIds)
      resolvedHandlers.onStats(stats)
    } catch {
      // 保留上次有效状态；缓冲的实时快照仍在 finally 中继续应用。
    } finally {
      const buffered = bufferedSchemeMessages
      bufferedSchemeMessages = []
      for (const message of buffered) applyBufferedSnapshot(message)
    }
  }

  function startReconciliation(): Promise<void> {
    if (stopped) return Promise.resolve()
    if (reconcilePromise) return reconcilePromise

    const running = performReconciliation()
    reconcilePromise = running.finally(() => {
      reconcilePromise = null
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

  return { stop, reconcile, refresh: reconcile }
}

export function cloudRunningPollMs(): number {
  return FALLBACK_POLL_MS
}

/** @deprecated 使用 startCloudRunningSync */
export { startCloudRunningSync as startCloudRunningPoll }
