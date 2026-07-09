import { WS_ENABLED, WS_PUBLIC_BASE } from '@/api/config'
import { connectPublicWs } from '@/composables/ws/usePublicWs'
import { WS_EVENTS, WS_TOPICS, type WsDrawResultPayload } from '@shared/types/ws'

const FALLBACK_POLL_MS = 15_000
/** 等待开奖时的 REST 兜底间隔（与云端中心一致，避免 1–2s 整页刷新） */
const DRAWING_POLL_MS = 5_000

export type GameDrawSyncOptions = {
  /** 兜底轮询（仅拉开奖相关，勿整页重载） */
  onPoll: () => void
  /** 平台 WS 推送开奖结果（后端 drawsync 收到第三方 WS 后广播） */
  onDrawResult?: (payload: WsDrawResultPayload) => void
}

/**
 * 游戏详情开奖同步：订阅 public.draw:{lotteryCode}，收到推送后即时更新；
 * REST 仅作降级轮询（默认 15s，等待开奖时 5s）。
 */
export function startGameDrawSync(
  lotteryCode: string,
  options: GameDrawSyncOptions,
  pollMs = FALLBACK_POLL_MS,
) {  if (!lotteryCode) {
    return {
      stop: () => {},
      refresh: () => {},
      setDrawingUrgent: (_urgent: boolean) => {},
    }
  }

  let stopped = false
  let pollTimer: ReturnType<typeof setInterval> | null = null
  let stopWs: (() => void) | null = null
  let drawingUrgent = false
  let activePollMs = pollMs

  function poll() {
    if (stopped) return
    options.onPoll()
  }

  function stopPoll() {    if (pollTimer) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function startPoll(ms = activePollMs) {
    activePollMs = ms
    stopPoll()
    pollTimer = window.setInterval(poll, activePollMs)
  }

  function setDrawingUrgent(urgent: boolean) {
    if (stopped) return
    drawingUrgent = urgent
    const nextMs = drawingUrgent ? Math.min(pollMs, DRAWING_POLL_MS) : pollMs
    if (nextMs === activePollMs && pollTimer) return
    startPoll(nextMs)
  }

  const extraTopics = [WS_TOPICS.publicDraw(lotteryCode)]
  if (WS_ENABLED && WS_PUBLIC_BASE) {
    stopWs = connectPublicWs(WS_PUBLIC_BASE, (env) => {
      if (env.name !== WS_EVENTS.drawResult) return
      const payload = env.payload as WsDrawResultPayload | undefined
      if (!payload?.lotteryCode || payload.lotteryCode !== lotteryCode) return
      options.onDrawResult?.(payload)
      poll()
    }, { extraTopics })
    startPoll()
  } else {
    startPoll()
  }

  return {
    stop: () => {
      stopped = true
      stopWs?.()
      stopPoll()
    },
    refresh: poll,
    setDrawingUrgent,
  }
}
export function gameDrawSyncPollMs() {
  return FALLBACK_POLL_MS
}

export function gameDrawSyncDrawingPollMs() {
  return DRAWING_POLL_MS
}
