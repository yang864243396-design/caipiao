import type { WsEnvelope } from '@shared/types/ws'

import { WS_EVENTS, WS_TOPICS } from '@shared/types/ws'



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



/** 会员 WS：鉴权后订阅 client.* topic（WS-2/WS-3） */

export function connectClientWs(

  url: string,

  accessToken: string,

  onEvent: WsEventHandler,

  options?: {
    extraTopics?: string[]
    /** WS 鉴权完成、可收事件时 */
    onConnected?: () => void
    /** 连接断开（含重连前） */
    onDisconnected?: () => void
  },

): () => void {

  let stopped = false

  let socket: WebSocket | null = null

  let backoff = 1000

  let pingTimer: ReturnType<typeof setInterval> | null = null

  let authed = false

  let subscribedExtra = false

  let readyNotified = false



  const wsUrl = accessToken ? `${url}?token=${encodeURIComponent(accessToken)}` : url

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

    authed = false

    subscribedExtra = false

    readyNotified = false

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



  function maybeSubscribeExtra() {

    if (!socket || subscribedExtra || !extraTopics.length) return

    if (!authed) return

    sendSubscribe(socket, extraTopics)

    subscribedExtra = true

  }

  function maybeNotifyReady(frame: WsEnvelope) {
    if (!authed || readyNotified) return
    const payload = frame.payload as { topics?: unknown } | undefined
    if (!Array.isArray(payload?.topics)) return
    const topics = payload.topics.filter((topic): topic is string => typeof topic === 'string')
    if (
      !topics.includes(WS_TOPICS.clientSchemeInstance) ||
      !topics.includes(WS_TOPICS.clientCloudStats)
    ) {
      return
    }
    readyNotified = true
    options?.onConnected?.()
  }



  function connect() {

    cleanup()

    if (stopped) return

    socket = new WebSocket(wsUrl)

    socket.onopen = () => {

      backoff = 1000

      if (!accessToken) sendAuth()

      else {

        authed = true

        maybeSubscribeExtra()

      }

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

          maybeSubscribeExtra()

          return

        }

        if (frame.type === 'system' && frame.name === 'system.subscribed') {

          maybeNotifyReady(frame)

          return

        }

        if (frame.type === 'event') onEvent(frame)

      } catch {

        /* ignore */

      }

    }

    socket.onclose = () => {

      options?.onDisconnected?.()

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



export function isCloudRefreshEvent(env: WsEnvelope): boolean {

  return (

    env.name === WS_EVENTS.schemeInstanceUpdated ||

    env.name === WS_EVENTS.walletUpdated

  )

}


