import type { WsEnvelope } from '@shared/types/ws'

export type WsEventHandler = (env: WsEnvelope) => void

const MAX_BACKOFF_MS = 30_000

/** Admin WS：鉴权后订阅 admin.* topic（WS-4） */
export function connectAdminWs(
  url: string,
  accessToken: string,
  onEvent: WsEventHandler,
): () => void {
  let stopped = false
  let socket: WebSocket | null = null
  let backoff = 1000
  let pingTimer: ReturnType<typeof setInterval> | null = null
  let authed = false

  const wsUrl = accessToken ? `${url}?token=${encodeURIComponent(accessToken)}` : url

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
    authed = false
  }

  function scheduleReconnect() {
    if (stopped) return
    window.setTimeout(() => {
      if (!stopped) connect()
    }, backoff)
    backoff = Math.min(backoff * 2, MAX_BACKOFF_MS)
  }

  function sendAuth() {
    if (!socket || socket.readyState !== WebSocket.OPEN || authed || !accessToken) return
    socket.send(
      JSON.stringify({
        type: 'command',
        name: 'auth',
        ts: new Date().toISOString(),
        payload: { accessToken },
      }),
    )
  }

  function connect() {
    cleanup()
    if (stopped) return
    socket = new WebSocket(wsUrl)
    socket.onopen = () => {
      backoff = 1000
      if (!accessToken) sendAuth()
      else authed = true
      pingTimer = setInterval(() => {
        if (socket?.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({ type: 'command', name: 'ping', ts: new Date().toISOString() }))
        }
      }, 30_000)
    }
    socket.onmessage = (ev) => {
      try {
        const frame = JSON.parse(String(ev.data)) as WsEnvelope
        if (frame.name === 'system.auth.ok') {
          authed = true
          return
        }
        if (frame.name === 'system.subscribed') {
          return
        }
        if (frame.type === 'event') onEvent(frame)
      } catch {
        /* ignore */
      }
    }
    socket.onclose = () => {
      cleanup()
      scheduleReconnect()
    }
    socket.onerror = () => {
      /* onclose follows */
    }
  }

  connect()

  return () => {
    stopped = true
    cleanup()
  }
}
