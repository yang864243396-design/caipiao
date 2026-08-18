import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type {
  WsCloudStatsSnapshotPayload,
  WsEnvelope,
  WsSchemeInstancesSnapshotPayload,
} from '@shared/types/ws'
import type { CloudSchemeCard } from '@/api/cloud/center'

const api = vi.hoisted(() => ({
  fetchRunningSchemesByIds: vi.fn(),
  fetchCloudCenterStats: vi.fn(),
}))

const ws = vi.hoisted(() => ({
  onEvent: undefined as ((event: WsEnvelope) => void) | undefined,
  onConnected: undefined as (() => void) | undefined,
  onDisconnected: undefined as (() => void) | undefined,
  stop: vi.fn(),
}))

const token = vi.hoisted(() => ({ value: 'token' }))

type Subject = typeof import('./useCloudRunningPoll')

const stats = (value: number, generatedAt?: string) => ({
  ...(generatedAt ? { generatedAt } : {}),
  formal: {
    totalTurnover: value,
    totalSessionPnl: value,
    runningSessionPnl: value,
  },
  sim: {
    totalTurnover: value,
    totalSessionPnl: value,
    runningSessionPnl: value,
  },
  simQuota: {
    todayStarts: value,
    todayStartsLimit: 5,
    running: value,
    runningLimit: 5,
  },
})

const scheme = (id: string, updatedAt: string, turnover = 1) => ({
  id,
  lotteryName: '分分彩',
  schemeName: `方案-${id}`,
  status: 'running' as const,
  statusLabel: '运行中',
  turnover,
  countdownSec: 10,
  pnl: 0,
  runTimeSec: 1,
  lookbackPnl: 0,
  sessionPnl: 0,
  multiplier: 1,
  simBet: false,
  updatedAt,
})

function envelope<T>(name: string, payload: T): WsEnvelope<T> {
  return {
    type: 'event',
    name,
    ts: '2026-08-18T00:00:00.000Z',
    payload,
  }
}

function snapshot(
  items: ReturnType<typeof scheme>[],
  removedIds: string[] = [],
): WsEnvelope<WsSchemeInstancesSnapshotPayload> {
  return envelope('client.scheme.instances.snapshot', {
    schemaVersion: 1,
    generatedAt: '2026-08-18T00:00:00.000Z',
    items,
    removedIds,
  })
}

function statsSnapshot(
  value: number,
  generatedAt = '2026-08-18T00:00:00.000Z',
): WsEnvelope<WsCloudStatsSnapshotPayload> {
  return envelope('client.cloud.stats.snapshot', {
    schemaVersion: 1,
    generatedAt,
    stats: stats(value),
  })
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => {
    resolve = done
  })
  return { promise, resolve }
}

async function loadSubject(options?: {
  realtimeEnabled?: boolean
  wsEnabled?: boolean
  tokenValue?: string
}): Promise<Subject> {
  vi.resetModules()
  token.value = options?.tokenValue ?? 'token'
  ws.onEvent = undefined
  ws.onConnected = undefined
  ws.onDisconnected = undefined

  vi.doMock('@/api/config', () => ({
    WS_CLIENT_ENABLED: options?.wsEnabled ?? true,
    WS_CLIENT_BASE: (options?.wsEnabled ?? true) ? 'ws://client' : '',
    CLOUD_REALTIME_CLIENT_ENABLED: options?.realtimeEnabled ?? true,
  }))
  vi.doMock('@/api/client', () => ({
    getAccessToken: () => token.value,
  }))
  vi.doMock('@/api/cloud/center', async (importOriginal) => {
    const actual = await importOriginal<typeof import('@/api/cloud/center')>()
    return {
      ...actual,
      fetchRunningSchemesByIds: api.fetchRunningSchemesByIds,
      fetchCloudCenterStats: api.fetchCloudCenterStats,
    }
  })
  vi.doMock('@/composables/ws/useClientWs', () => ({
    connectClientWs: (
      _url: string,
      _accessToken: string,
      onEvent: (event: WsEnvelope) => void,
      lifecycle?: { onConnected?: () => void; onDisconnected?: () => void },
    ) => {
      ws.onEvent = onEvent
      ws.onConnected = lifecycle?.onConnected
      ws.onDisconnected = lifecycle?.onDisconnected
      return ws.stop
    },
    isCloudRefreshEvent: (event: WsEnvelope) =>
      event.name === 'client.scheme.instance.updated' ||
      event.name === 'client.wallet.updated',
  }))

  return import('./useCloudRunningPoll')
}

describe('startCloudRunningSync', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    api.fetchRunningSchemesByIds.mockResolvedValue([])
    api.fetchCloudCenterStats.mockResolvedValue(stats(0))
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('does not create a REST interval while websocket is online', async () => {
    const setIntervalSpy = vi.spyOn(window, 'setInterval')
    const { startCloudRunningSync } = await loadSubject()

    const sync = startCloudRunningSync(
      () => ['a'],
      { onSchemes: vi.fn(), onStats: vi.fn() },
    )

    expect(setIntervalSpy).not.toHaveBeenCalled()
    sync.stop()
  })

  it('runs exactly one reconciliation for each subscribed connection cycle', async () => {
    const setIntervalSpy = vi.spyOn(window, 'setInterval')
    const { startCloudRunningSync } = await loadSubject()
    const sync = startCloudRunningSync(
      () => ['a'],
      { onSchemes: vi.fn(), onStats: vi.fn() },
    )

    ws.onConnected?.()
    await sync.reconcile()
    ws.onDisconnected?.()
    ws.onConnected?.()
    await sync.reconcile()

    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(2)
    expect(api.fetchCloudCenterStats).toHaveBeenCalledTimes(2)
    expect(setIntervalSpy).not.toHaveBeenCalled()
    sync.stop()
  })

  it('shares one in-flight reconciliation across duplicate readiness events', async () => {
    const schemesRequest = deferred<ReturnType<typeof scheme>[]>()
    const statsRequest = deferred<ReturnType<typeof stats>>()
    api.fetchRunningSchemesByIds.mockReturnValueOnce(schemesRequest.promise)
    api.fetchCloudCenterStats.mockReturnValueOnce(statsRequest.promise)
    const { startCloudRunningSync } = await loadSubject()
    const sync = startCloudRunningSync(
      () => ['a'],
      { onSchemes: vi.fn(), onStats: vi.fn() },
    )

    ws.onConnected?.()
    ws.onConnected?.()
    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(1)
    expect(api.fetchCloudCenterStats).toHaveBeenCalledTimes(1)

    schemesRequest.resolve([scheme('a', '2026-08-18T00:00:02.000Z')])
    statsRequest.resolve(stats(1))
    await sync.reconcile()

    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(1)
    expect(api.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    sync.stop()
  })

  it('buffers snapshots during reconciliation and applies them afterward', async () => {
    const schemesRequest = deferred<ReturnType<typeof scheme>[]>()
    const statsRequest = deferred<ReturnType<typeof stats>>()
    api.fetchRunningSchemesByIds.mockReturnValueOnce(schemesRequest.promise)
    api.fetchCloudCenterStats.mockReturnValueOnce(statsRequest.promise)
    const applied: string[] = []
    const onSchemes = vi.fn((cards: CloudSchemeCard[]) => applied.push(`schemes:${cards[0]?.id}`))
    const onStats = vi.fn((value: ReturnType<typeof stats>) =>
      applied.push(`stats:${value.formal.totalTurnover}`),
    )
    const { startCloudRunningSync } = await loadSubject()
    const sync = startCloudRunningSync(() => ['a'], { onSchemes, onStats })

    ws.onConnected?.()
    ws.onEvent?.(snapshot([scheme('a', '2026-08-18T00:00:02.000Z', 2)]))
    ws.onEvent?.(statsSnapshot(2))
    expect(applied).toEqual([])

    schemesRequest.resolve([scheme('a', '2026-08-18T00:00:01.000Z', 1)])
    statsRequest.resolve(stats(1))
    await sync.reconcile()

    expect(applied).toEqual(['schemes:a', 'stats:1', 'schemes:a', 'stats:2'])
    expect(onSchemes.mock.calls[1]?.[0][0].turnover).toBe('2.00')
    sync.stop()
  })

  it('keeps REST stats newer than buffered older or equal snapshots and applies a newer snapshot', async () => {
    const statsRequest = deferred<ReturnType<typeof stats>>()
    api.fetchCloudCenterStats.mockReturnValueOnce(statsRequest.promise)
    const applied: number[] = []
    const { startCloudRunningSync } = await loadSubject()
    const sync = startCloudRunningSync(
      () => ['a'],
      {
        onSchemes: vi.fn(),
        onStats: (value) => applied.push(value.formal.totalTurnover),
      },
    )

    ws.onConnected?.()
    ws.onEvent?.(statsSnapshot(10, '2026-08-18T00:00:01.000Z'))
    ws.onEvent?.(statsSnapshot(11, '2026-08-18T00:00:02.000Z'))
    ws.onEvent?.(statsSnapshot(30, '2026-08-18T00:00:03.000Z'))
    statsRequest.resolve(stats(20, '2026-08-18T00:00:02.000Z'))
    await sync.reconcile()

    expect(applied).toEqual([20, 30])
    sync.stop()
  })

  it('ignores an older updatedAt snapshot', async () => {
    api.fetchRunningSchemesByIds.mockResolvedValueOnce([
      scheme('a', '2026-08-18T00:00:02.000Z', 2),
    ])
    const onSchemes = vi.fn()
    const { startCloudRunningSync } = await loadSubject()
    const sync = startCloudRunningSync(
      () => ['a'],
      { onSchemes, onStats: vi.fn() },
    )

    ws.onConnected?.()
    await sync.reconcile()
    ws.onEvent?.(snapshot([scheme('a', '2026-08-18T00:00:01.000Z', 1)]))

    expect(onSchemes).toHaveBeenCalledTimes(1)
    sync.stop()
  })

  it('removes only loaded cards named in removedIds', async () => {
    const onSchemes = vi.fn()
    const { startCloudRunningSync } = await loadSubject()
    const sync = startCloudRunningSync(
      () => ['a'],
      { onSchemes, onStats: vi.fn() },
    )

    ws.onEvent?.(snapshot([], ['a', 'not-loaded']))

    expect(onSchemes).toHaveBeenCalledWith([], ['a'])
    sync.stop()
  })

  it('uses legacy invalidation only until the first versioned snapshot', async () => {
    const { startCloudRunningSync } = await loadSubject()
    const sync = startCloudRunningSync(
      () => ['a'],
      { onSchemes: vi.fn(), onStats: vi.fn() },
    )

    ws.onEvent?.(envelope('client.scheme.instance.updated', { instanceId: 'a' }))
    await sync.reconcile()
    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(1)

    ws.onEvent?.(snapshot([scheme('a', '2026-08-18T00:00:02.000Z')]))
    ws.onEvent?.(envelope('client.scheme.instance.updated', { instanceId: 'a' }))
    await Promise.resolve()

    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(1)
    sync.stop()
  })

  it('keeps legacy polling only when client websocket is disabled', async () => {
    const setIntervalSpy = vi.spyOn(window, 'setInterval')
    const { startCloudRunningSync } = await loadSubject({ wsEnabled: false })
    const sync = startCloudRunningSync(
      () => ['a'],
      { onSchemes: vi.fn(), onStats: vi.fn() },
    )

    expect(setIntervalSpy).toHaveBeenCalledTimes(1)
    expect(setIntervalSpy).toHaveBeenCalledWith(expect.any(Function), 15_000)
    await vi.advanceTimersByTimeAsync(15_000)
    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(1)
    expect(api.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    sync.stop()
  })

  it('keeps unchanged compatibility cards and applies snapshot tombstones', async () => {
    const { startCloudRunningSync } = await loadSubject()
    const { instanceToDisplay, mergeCloudSchemesStable } = await import('@/api/cloud/center')
    let displayed = [
      instanceToDisplay(scheme('a', '2026-08-18T00:00:01.000Z', 1)),
      instanceToDisplay(scheme('b', '2026-08-18T00:00:01.000Z', 1)),
      instanceToDisplay(scheme('c', '2026-08-18T00:00:01.000Z', 1)),
    ]
    const sync = startCloudRunningSync(
      () => displayed.map((card) => card.id),
      (cards) => {
        displayed = mergeCloudSchemesStable(displayed, cards)
      },
    )

    ws.onEvent?.(snapshot([scheme('a', '2026-08-18T00:00:02.000Z', 2)]))
    expect(displayed.map((card) => card.id)).toEqual(['a', 'b', 'c'])
    expect(displayed[0]?.turnover).toBe('2.00')
    expect(displayed[1]?.turnover).toBe('1.00')

    ws.onEvent?.(snapshot([], ['b']))
    expect(displayed.map((card) => card.id)).toEqual(['a', 'c'])

    api.fetchRunningSchemesByIds.mockResolvedValueOnce([
      scheme('a', '2026-08-18T00:00:03.000Z', 3),
    ])
    await sync.reconcile()
    expect(displayed.map((card) => card.id)).toEqual(['a'])
    sync.stop()
  })

  it('deduplicates compatibility refresh by loaded IDs and subscribed cycle', async () => {
    const earlySchemes = deferred<ReturnType<typeof scheme>[]>()
    const earlyStats = deferred<ReturnType<typeof stats>>()
    let loadedIds: string[] = []
    api.fetchRunningSchemesByIds
      .mockReturnValueOnce(earlySchemes.promise)
      .mockResolvedValueOnce([scheme('a', '2026-08-18T00:00:01.000Z')])
      .mockResolvedValueOnce([scheme('a', '2026-08-18T00:00:02.000Z')])
    api.fetchCloudCenterStats.mockReturnValueOnce(earlyStats.promise)
    const onUpdate = vi.fn()
    const { startCloudRunningSync } = await loadSubject()
    const sync = startCloudRunningSync(() => loadedIds, onUpdate)

    ws.onConnected?.()
    loadedIds = ['a']
    const loadedRefresh = sync.refresh()
    earlySchemes.resolve([])
    earlyStats.resolve(stats(0))
    await loadedRefresh
    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(2)
    expect(api.fetchRunningSchemesByIds).toHaveBeenNthCalledWith(1, [])
    expect(api.fetchRunningSchemesByIds).toHaveBeenNthCalledWith(2, ['a'])
    expect(api.fetchCloudCenterStats).toHaveBeenCalledTimes(1)

    expect(onUpdate).toHaveBeenCalledTimes(1)
    expect(onUpdate.mock.calls[0]?.[0].map((card: CloudSchemeCard) => card.id)).toEqual(['a'])

    await sync.refresh()
    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(2)
    expect(api.fetchCloudCenterStats).toHaveBeenCalledTimes(1)

    ws.onDisconnected?.()
    ws.onConnected?.()
    await sync.refresh()
    expect(api.fetchRunningSchemesByIds).toHaveBeenCalledTimes(3)
    expect(api.fetchCloudCenterStats).toHaveBeenCalledTimes(2)
    sync.stop()
  })
})
