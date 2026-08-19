import { beforeEach, describe, expect, it } from 'vitest'

import {
  acquireBetRequestIdentity,
  releaseBetRequestIdentity,
} from './betRequestIdentity'

describe('bet request identity', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('reuses the same request id for the same unresolved bet', () => {
    const input = { issueNo: '20260819001', amount: 2, multiplier: 1, betPayload: { content: '1,2' } }
    const first = acquireBetRequestIdentity('lottery-a', input)
    const retry = acquireBetRequestIdentity('lottery-a', input)

    expect(retry.requestId).toBe(first.requestId)
    expect(first.requestId.length).toBeLessThanOrEqual(76)
  })

  it('uses a new request id after a definitive completion', () => {
    const input = { issueNo: '20260819001', amount: 2, multiplier: 1 }
    const first = acquireBetRequestIdentity('lottery-a', input)
    releaseBetRequestIdentity(first)
    const next = acquireBetRequestIdentity('lottery-a', input)

    expect(next.requestId).not.toBe(first.requestId)
  })

  it('does not share request ids across different bet payloads', () => {
    const first = acquireBetRequestIdentity('lottery-a', { issueNo: '1', amount: 2 })
    const second = acquireBetRequestIdentity('lottery-a', { issueNo: '1', amount: 4 })

    expect(second.requestId).not.toBe(first.requestId)
  })
})
