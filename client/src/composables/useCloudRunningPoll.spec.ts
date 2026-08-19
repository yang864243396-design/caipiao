import { describe, expect, it } from 'vitest'

import { cloudRunningPollMs } from './useCloudRunningPoll'

describe('cloud running REST fallback interval', () => {
  it('uses fast fallback until websocket is actually connected', () => {
    expect(cloudRunningPollMs(false)).toBe(15_000)
  })

  it('uses slow safety polling only while websocket is connected', () => {
    expect(cloudRunningPollMs(true)).toBe(60_000)
  })

  it('preserves caller fallback interval while disconnected', () => {
    expect(cloudRunningPollMs(false, 7_500)).toBe(7_500)
  })
})
