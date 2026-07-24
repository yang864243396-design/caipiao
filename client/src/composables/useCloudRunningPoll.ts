import { WS_CLIENT_ENABLED, WS_CLIENT_BASE } from '@/api/config'

import { getAccessToken } from '@/api/client'

import {
  fetchRunningSchemesByIds,
  instanceToDisplay,
  type CloudSchemeCard,
} from '@/api/cloud/center'

import { connectClientWs, isCloudRefreshEvent } from '@/composables/ws/useClientWs'

/** REST 轮询间隔：无 WS 时兜底；有 WS 时拉长，减轻 /running?ids= 压力 */
const FALLBACK_POLL_MS = 15_000
const WS_CONNECTED_POLL_MS = 60_000

/**
 * 云端运行列表同步：按已加载实例 ID 批量刷新。
 * - WS 事件 / 手动 refresh：立即拉取
 * - REST 轮询：无 WS 15s；有 WS 60s（倒计时本地 tick，盈亏等靠 WS + 慢轮询）
 * @returns stop 停止同步；refresh 立即拉取一次（并发时合并为单次在途请求）
 */
export function startCloudRunningSync(
  getLoadedIds: () => string[],
  onUpdate: (cards: CloudSchemeCard[]) => void,
  pollMs = FALLBACK_POLL_MS,
) {
  let stopped = false
  let pollTimer: ReturnType<typeof setInterval> | null = null
  let stopWs: (() => void) | null = null
  let refreshInFlight = false
  let refreshQueued = false

  async function refresh() {
    if (stopped) return
    if (refreshInFlight) {
      refreshQueued = true
      return
    }
    const ids = getLoadedIds()
    if (ids.length === 0) return
    refreshInFlight = true
    try {
      const rows = await fetchRunningSchemesByIds(ids)
      onUpdate(rows.map(instanceToDisplay))
    } catch {
      /* 保留上次有效数据 */
    } finally {
      refreshInFlight = false
      if (refreshQueued) {
        refreshQueued = false
        void refresh()
      }
    }
  }

  function stopPoll() {
    if (pollTimer) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function startPoll(intervalMs: number) {
    if (pollTimer) return
    pollTimer = window.setInterval(() => {
      void refresh()
    }, intervalMs)
  }

  const token = getAccessToken()
  const wsAvailable = Boolean(WS_CLIENT_ENABLED && WS_CLIENT_BASE && token)

  startPoll(wsAvailable ? WS_CONNECTED_POLL_MS : pollMs)

  // 单独收窄 token / base：Boolean(...) 无法让 TS 排除 null
  if (WS_CLIENT_ENABLED && WS_CLIENT_BASE && token) {
    stopWs = connectClientWs(WS_CLIENT_BASE, token, (env) => {
      if (isCloudRefreshEvent(env)) void refresh()
    })
  }

  function stop() {
    stopped = true
    stopWs?.()
    stopPoll()
  }

  return { stop, refresh }
}

export function cloudRunningPollMs() {
  return FALLBACK_POLL_MS
}

/** @deprecated 使用 startCloudRunningSync */
export { startCloudRunningSync as startCloudRunningPoll }
