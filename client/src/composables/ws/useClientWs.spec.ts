import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { WsEnvelope } from '@shared/types/ws'

import { connectClientWs } from './useClientWs'

class FakeWebSocket {
  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSING = 2
  static readonly CLOSED = 3
  static instances: FakeWebSocket[] = []

  readonly url: string
  readyState = FakeWebSocket.CONNECTING
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null

  constructor(url: string | URL) {
    this.url = String(url)
    FakeWebSocket.instances.push(this)
  }

  send(_data: string | ArrayBufferLike | Blob | ArrayBufferView): void {}

  close(): void {
    this.readyState = FakeWebSocket.CLOSED
  }

  emitOpen(): void {
    this.readyState = FakeWebSocket.OPEN
    this.onopen?.(new Event('open'))
  }

  emitMessage(frame: WsEnvelope): void {
    this.onmessage?.(new MessageEvent('message', { data: JSON.stringify(frame) }))
  }

  emitClose(): void {
    this.readyState = FakeWebSocket.CLOSED
    this.onclose?.(new CloseEvent('close'))
  }
}

const subscribedFrame = (topics: string[]): WsEnvelope => ({
  type: 'system',
  name: 'system.subscribed',
  ts: '2026-08-18T00:00:00.000Z',
  payload: { topics },
})

describe('connectClientWs readiness', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    FakeWebSocket.instances = []
    vi.stubGlobal('WebSocket', FakeWebSocket as unknown as typeof WebSocket)
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('announces readiness once per subscribed connection cycle', () => {
    const connected = vi.fn()
    const stop = connectClientWs('ws://test', 'token', vi.fn(), { onConnected: connected })

    const first = FakeWebSocket.instances[0]
    expect(first?.url).toBe('ws://test?token=token')
    first?.emitOpen()
    expect(connected).not.toHaveBeenCalled()

    first?.emitMessage(subscribedFrame(['client.scheme.instance']))
    expect(connected).not.toHaveBeenCalled()

    first?.emitMessage(subscribedFrame(['client.scheme.instance', 'client.cloud.stats']))
    first?.emitMessage(subscribedFrame(['client.scheme.instance', 'client.cloud.stats']))
    expect(connected).toHaveBeenCalledTimes(1)

    first?.emitClose()
    vi.runOnlyPendingTimers()

    const second = FakeWebSocket.instances[1]
    second?.emitOpen()
    expect(connected).toHaveBeenCalledTimes(1)
    second?.emitMessage(subscribedFrame(['client.scheme.instance', 'client.cloud.stats']))
    expect(connected).toHaveBeenCalledTimes(2)

    stop()
  })

  it('does not accept a non-system frame as a subscription acknowledgement', () => {
    const connected = vi.fn()
    const stop = connectClientWs('ws://test', 'token', vi.fn(), { onConnected: connected })

    const socket = FakeWebSocket.instances[0]
    socket?.emitOpen()
    socket?.emitMessage({
      ...subscribedFrame(['client.scheme.instance', 'client.cloud.stats']),
      type: 'event',
    })

    expect(connected).not.toHaveBeenCalled()
    stop()
  })
})
