import { flushPromises, shallowMount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { CloudRunningScheme } from '@/api/types'
import type {
  CloudCenterStatsDto,
  CloudSchemeCard,
} from '@/api/cloud/center'

const realtime = vi.hoisted(() => ({
  getLoadedIds: null as null | (() => string[]),
  handlers: null as null | {
    onSchemes(cards: CloudSchemeCard[], removedIds: string[]): void
    onStats(stats: CloudCenterStatsDto): void
  },
  stop: vi.fn(),
  reconcile: vi.fn<() => Promise<void>>(),
  refresh: vi.fn<() => Promise<void>>(),
  start: vi.fn(),
}))

const cloudApi = vi.hoisted(() => ({
  fetchCloudCenterStats: vi.fn(),
  fetchCloudGlobalSettings: vi.fn(),
  fetchLookbackSettings: vi.fn(),
  fetchRunningSchemesPage: vi.fn(),
  startCloudInstance: vi.fn(),
  stopCloudInstance: vi.fn(),
}))

const definitionsApi = vi.hoisted(() => ({
  getSchemeDefinition: vi.fn(),
}))

const dialogs = vi.hoisted(() => ({
  confirm: vi.fn(),
}))

vi.mock('@/composables/useCloudRunningPoll', () => ({
  startCloudRunningSync: realtime.start,
  cloudRunningPollMs: () => 15_000,
}))

vi.mock('@/api/cloud/center', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/cloud/center')>()
  return {
    ...actual,
    fetchCloudCenterStats: cloudApi.fetchCloudCenterStats,
    fetchCloudGlobalSettings: cloudApi.fetchCloudGlobalSettings,
    fetchLookbackSettings: cloudApi.fetchLookbackSettings,
    fetchRunningSchemesPage: cloudApi.fetchRunningSchemesPage,
    startCloudInstance: cloudApi.startCloudInstance,
    stopCloudInstance: cloudApi.stopCloudInstance,
  }
})

vi.mock('@/api/schemes/definitions', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/schemes/definitions')>()
  return {
    ...actual,
    getSchemeDefinition: definitionsApi.getSchemeDefinition,
  }
})

vi.mock('@/utils/confirmDialog', () => ({
  confirmDialog: dialogs.confirm,
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

import { instanceToDisplay } from '@/api/cloud/center'
import CloudCenterView from './CloudCenterView.vue'

function stats(seed: number): CloudCenterStatsDto {
  return {
    formal: {
      totalTurnover: seed + 0.12,
      totalSessionPnl: seed + 0.3,
      runningSessionPnl: seed + 0.4,
    },
    sim: {
      totalTurnover: seed + 10.12,
      totalSessionPnl: seed + 10.3,
      runningSessionPnl: seed + 10.4,
    },
    simQuota: {
      todayStarts: 1,
      todayStartsLimit: 5,
      running: 2,
      runningLimit: 5,
    },
  }
}

function scheme(
  id: string,
  schemeName: string,
  status: CloudRunningScheme['status'] = 'running',
  overrides: Partial<CloudRunningScheme> = {},
): CloudRunningScheme {
  return {
    id,
    updatedAt: '2026-08-18T00:00:00.000Z',
    lotteryCode: 'fast',
    lotteryName: `Lottery ${id}`,
    schemeName,
    status,
    statusLabel: status,
    turnover: 1,
    countdownSec: 10,
    pnl: 0,
    runTimeSec: 1,
    lookbackPnl: 0,
    sessionPnl: 0,
    multiplier: 1,
    simBet: true,
    schemeCurrency: 'USDT',
    ...overrides,
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => {
    resolve = done
  })
  return { promise, resolve }
}

function page(rows: CloudRunningScheme[], total = rows.length) {
  return {
    items: rows,
    total,
    page: { hasMore: false },
  }
}

function mountView(): VueWrapper {
  return shallowMount(CloudCenterView, {
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        ContentDialog: true,
        ElButton: true,
        ElDialog: true,
        ElInput: true,
        ElOption: true,
        ElSelect: true,
        ElSwitch: true,
        ElTag: true,
      },
    },
  })
}

async function mountedView(rows: CloudRunningScheme[], total = rows.length): Promise<VueWrapper> {
  cloudApi.fetchRunningSchemesPage.mockResolvedValueOnce(page(rows, total))
  const wrapper = mountView()
  await flushPromises()
  return wrapper
}

describe('CloudCenterView realtime integration', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-18T00:00:00.000Z'))
    realtime.getLoadedIds = null
    realtime.handlers = null
    realtime.stop.mockReset()
    realtime.reconcile.mockReset().mockResolvedValue()
    realtime.refresh.mockReset().mockResolvedValue()
    realtime.start.mockReset().mockImplementation((getLoadedIds, handlers) => {
      realtime.getLoadedIds = getLoadedIds
      realtime.handlers = handlers
      return {
        stop: realtime.stop,
        reconcile: realtime.reconcile,
        refresh: realtime.refresh,
      }
    })
    cloudApi.fetchCloudCenterStats.mockReset().mockResolvedValue(stats(0))
    cloudApi.fetchCloudGlobalSettings.mockReset().mockResolvedValue({
      totalStopLoss: 0,
      totalTakeProfit: 0,
      planMultiplier: 1,
      breakPeriodStop: false,
    })
    cloudApi.fetchLookbackSettings.mockReset().mockResolvedValue({
      applyFormal: false,
      applySim: false,
      runModes: [],
      judgment: '',
      singleProfitThreshold: 0,
      singleLossThreshold: 0,
      overallProfitThreshold: 0,
      overallLossThreshold: 0,
      schemeWinsMin: 0,
      schemeWinsMax: 0,
      periodProfit: 0,
      periodLoss: 0,
    })
    cloudApi.fetchRunningSchemesPage.mockReset()
    cloudApi.startCloudInstance.mockReset()
    cloudApi.stopCloudInstance.mockReset()
    definitionsApi.getSchemeDefinition.mockReset().mockRejectedValue(new Error('not needed'))
    dialogs.confirm.mockReset().mockResolvedValue(true)

    class FakeIntersectionObserver {
      observe() {}
      disconnect() {}
    }
    vi.stubGlobal('IntersectionObserver', FakeIntersectionObserver)
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      callback(0)
      return 1
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('finishes the one initial REST load before starting realtime sync', async () => {
    const response = deferred<ReturnType<typeof page>>()
    cloudApi.fetchRunningSchemesPage.mockReturnValueOnce(response.promise)
    const wrapper = mountView()

    await flushPromises()
    expect(cloudApi.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    expect(realtime.start).not.toHaveBeenCalled()

    response.resolve(page([scheme('a', 'Alpha')]))
    await flushPromises()

    expect(realtime.start).toHaveBeenCalledTimes(1)
    expect(realtime.getLoadedIds?.()).toEqual(['a'])
    wrapper.unmount()
  })

  it('keeps countdown ticks local after the timer passes zero', async () => {
    const wrapper = await mountedView([
      scheme('a', 'Alpha', 'running', {
        countdownSec: 1,
        countdownEndTime: '2026-08-18T00:00:01.000Z',
      }),
    ])

    expect(cloudApi.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    expect(realtime.reconcile).not.toHaveBeenCalled()
    expect(realtime.refresh).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(3_000)
    await flushPromises()

    expect(cloudApi.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    expect(realtime.reconcile).not.toHaveBeenCalled()
    expect(realtime.refresh).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('applies only loaded snapshot cards in place and removes only loaded tombstones', async () => {
    const wrapper = await mountedView([
      scheme('a', 'Alpha'),
      scheme('b', 'Beta'),
      scheme('c', 'Gamma'),
    ], 7)

    expect(realtime.handlers).toEqual(expect.objectContaining({
      onSchemes: expect.any(Function),
      onStats: expect.any(Function),
    }))

    realtime.handlers!.onSchemes([
      instanceToDisplay(scheme('c', 'Gamma updated', 'running', {
        updatedAt: '2026-08-18T00:00:01.000Z',
      })),
      instanceToDisplay(scheme('d', 'Delta', 'running', {
        updatedAt: '2026-08-18T00:00:01.000Z',
      })),
    ], ['b', 'missing'])
    await nextTick()

    expect(wrapper.findAll('.cc-v--ellipsis').map((node) => node.text())).toEqual([
      'Alpha',
      'Gamma updated',
    ])
    expect(wrapper.text()).toContain('共 6 个方案')
    expect(realtime.getLoadedIds?.()).toEqual(['a', 'c'])
    wrapper.unmount()
  })

  it('renders pushed statistics without another REST request', async () => {
    const wrapper = await mountedView([])
    const pushed = stats(40)
    pushed.formal.totalTurnover = 40.5
    pushed.sim.totalTurnover = 50.5

    expect(realtime.handlers).toEqual(expect.objectContaining({ onStats: expect.any(Function) }))
    realtime.handlers!.onStats(pushed)
    await nextTick()

    expect(wrapper.text()).toContain('40.50')
    expect(wrapper.text()).toContain('40.3')
    expect(wrapper.text()).toContain('50.50')
    expect(cloudApi.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('keeps action response patches but does not refetch statistics', async () => {
    const pending = scheme('a', 'Alpha', 'pending')
    cloudApi.startCloudInstance.mockResolvedValueOnce(scheme('a', 'Alpha', 'running', {
      updatedAt: '2026-08-18T00:00:01.000Z',
    }))
    const wrapper = await mountedView([pending])

    await wrapper.find('.cc-start-btn').trigger('click')
    await flushPromises()

    expect(cloudApi.startCloudInstance).toHaveBeenCalledWith('a')
    expect(wrapper.find('.cc-start-btn').text()).toContain('停止')
    expect(cloudApi.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('keeps stop response patches but does not refetch statistics', async () => {
    cloudApi.stopCloudInstance.mockResolvedValueOnce(scheme('a', 'Alpha', 'paused', {
      updatedAt: '2026-08-18T00:00:01.000Z',
    }))
    const wrapper = await mountedView([scheme('a', 'Alpha', 'running')])

    await wrapper.find('.cc-start-btn').trigger('click')
    await flushPromises()

    expect(cloudApi.stopCloudInstance).toHaveBeenCalledWith('a')
    expect(wrapper.find('.cc-start-btn').text()).toContain('开启方案')
    expect(cloudApi.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('keeps bulk-start response patches but does not refetch statistics', async () => {
    const rows = [
      scheme('a', 'Alpha', 'pending'),
      scheme('b', 'Beta', 'pending'),
    ]
    cloudApi.startCloudInstance.mockImplementation(async (id: string) => (
      scheme(id, id === 'a' ? 'Alpha' : 'Beta', 'running', {
        updatedAt: '2026-08-18T00:00:01.000Z',
      })
    ))
    const wrapper = await mountedView(rows)

    await wrapper.find('.cc-btn--primary').trigger('click')
    await flushPromises()

    expect(cloudApi.startCloudInstance).toHaveBeenCalledTimes(2)
    expect(wrapper.findAll('.cc-start-btn').map((button) => button.text())).toEqual(['停止', '停止'])
    expect(cloudApi.fetchCloudCenterStats).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('does not start realtime after an in-flight initial load is unmounted', async () => {
    const response = deferred<ReturnType<typeof page>>()
    cloudApi.fetchRunningSchemesPage.mockReturnValueOnce(response.promise)
    const wrapper = mountView()

    await flushPromises()
    wrapper.unmount()
    response.resolve(page([scheme('a', 'Alpha')]))
    await flushPromises()

    expect(realtime.start).not.toHaveBeenCalled()
    expect(realtime.stop).not.toHaveBeenCalled()
  })

  it('stops realtime sync and clears its countdown timer on unmount', async () => {
    const clearIntervalSpy = vi.spyOn(window, 'clearInterval')
    const wrapper = await mountedView([])

    wrapper.unmount()

    expect(realtime.stop).toHaveBeenCalledTimes(1)
    expect(clearIntervalSpy).toHaveBeenCalledTimes(1)
  })
})
