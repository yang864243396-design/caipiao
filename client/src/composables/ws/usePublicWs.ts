import type { WsEnvelope, WsMaintenanceChangedPayload } from '@shared/types/ws'

export type WsEventHandler = (env: WsEnvelope) => void

const MAX_BACKOFF_MS = 30_000

function sendSubscribe(socket: WebSocket, topics: string[]) {
  if (!topics.length || socket.readyState !== WebSocket.OPEN) return
  socket.send(
    JSON.stringify({
      type: 'command',
      name: 'subscribe',
      ts: new Date().toISOString(),
      payload: { topics },
    }),
  )
}

/** 公共 WS：维护开关、开奖等 public.* topic（WS-1/WS-5） */
export function connectPublicWs(
  url: string,
  onEvent: WsEventHandler,
  options?: { extraTopics?: string[] },
): () => void {
  let stopped = false
  let socket: WebSocket | null = null
  let backoff = 1000
  let pingTimer: ReturnType<typeof setInterval> | null = null
  let subscribedExtra = false
  const extraTopics = options?.extraTopics ?? []

  function cleanup() {
    if (pingTimer) {
      clearInterval(pingTimer)
      pingTimer = null
    }
    if (socket) {
      socket.onopen = null
      socket.onmessage = null
      socket.onclose = null
      socket.onerror = null
      if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
        socket.close()
      }
      socket = null
    }
    subscribedExtra = false
  }

  function scheduleReconnect(url: string) {
    if (stopped) return
    window.setTimeout(() => {
      if (!stopped) connect(url)
    }, backoff)
    backoff = Math.min(backoff * 2, MAX_BACKOFF_MS)
  }

  function maybeSubscribeExtra() {
    if (!socket || subscribedExtra || !extraTopics.length) return
    sendSubscribe(socket, extraTopics)
    subscribedExtra = true
  }

  function connect(url: string) {
    cleanup()
    if (stopped) return
    socket = new WebSocket(url)
    socket.onopen = () => {
      backoff = 1000
      maybeSubscribeExtra()
      pingTimer = setInterval(() => {
        if (socket?.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({ type: 'command', name: 'ping', ts: new Date().toISOString() }))
        }
      }, 30_000)
    }
    socket.onmessage = (ev) => {
      try {
        const frame = JSON.parse(String(ev.data)) as WsEnvelope
        if (frame.name === 'system.subscribed') {
          maybeSubscribeExtra()
          return
        }
        if (frame.type === 'event') onEvent(frame)
      } catch {
        /* ignore malformed */
      }
    }
    socket.onclose = () => {
      cleanup()
      scheduleReconnect(url)
    }
    socket.onerror = () => {
      /* onclose follows */
    }
  }

  connect(url)

  return () => {
    stopped = true
    cleanup()
  }
}

export function parseMaintenanceChanged(env: WsEnvelope): WsMaintenanceChangedPayload | null {
  if (env.name !== 'public.maintenance.changed') return null
  return (env.payload ?? null) as WsMaintenanceChangedPayload | null
}
