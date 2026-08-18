import { beforeEach, describe, expect, it, vi } from 'vitest'

const dependencies = vi.hoisted(() => ({
  ensureClientSession: vi.fn(),
  requestApi: vi.fn(),
}))

vi.mock('@/api/auth', () => ({
  ensureClientSession: dependencies.ensureClientSession,
}))

vi.mock('@/api/client', () => ({
  requestApi: dependencies.requestApi,
}))

import { fetchCloudCenterStats } from './center'

describe('fetchCloudCenterStats', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('preserves the server generated stats version', async () => {
    dependencies.requestApi.mockResolvedValue({
      generatedAt: '2026-08-18T00:00:02.000Z',
      formal: { totalTurnover: 2, totalSessionPnl: 2, runningSessionPnl: 2 },
      sim: { totalTurnover: 0, totalSessionPnl: 0, runningSessionPnl: 0 },
      simQuota: { todayStarts: 0, todayStartsLimit: 5, running: 0, runningLimit: 5 },
    })

    const got = await fetchCloudCenterStats()

    expect(got.generatedAt).toBe('2026-08-18T00:00:02.000Z')
  })
})
