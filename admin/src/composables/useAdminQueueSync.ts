import { WS_ADMIN_ENABLED, WS_ADMIN_BASE } from '@/api/config'
import { getAccessToken } from '@/api/client'
import { connectAdminWs } from '@/composables/ws/useAdminWs'
import { WS_EVENTS } from '@shared/types/ws'

const FALLBACK_POLL_MS = 15_000

/** 提现审批/出款：监听 admin.withdraw.queue */
export function startWithdrawQueueSync(onRefresh: () => void, pollMs = FALLBACK_POLL_MS) {
  let stopped = false
  let pollTimer: ReturnType<typeof setInterval> | null = null
  let stopWs: (() => void) | null = null

  function refresh() {
    if (stopped) return
    onRefresh()
  }

  function stopPoll() {
    if (pollTimer) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function startPoll() {
    stopPoll()
    pollTimer = window.setInterval(refresh, pollMs)
  }

  refresh()

  const token = getAccessToken()
  if (WS_ADMIN_ENABLED && WS_ADMIN_BASE && token) {
    stopWs = connectAdminWs(WS_ADMIN_BASE, token, (env) => {
      if (env.name === WS_EVENTS.withdrawQueueChanged) refresh()
    })
    startPoll()
  } else {
    startPoll()
  }

  return () => {
    stopped = true
    stopWs?.()
    stopPoll()
  }
}

/** 方案监控：监听 admin.scheme.monitor */
export function startSchemeMonitorSync(onRefresh: () => void, pollMs = FALLBACK_POLL_MS) {
  let stopped = false
  let pollTimer: ReturnType<typeof setInterval> | null = null
  let stopWs: (() => void) | null = null

  function refresh() {
    if (stopped) return
    onRefresh()
  }

  function stopPoll() {
    if (pollTimer) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function startPoll() {
    stopPoll()
    pollTimer = window.setInterval(refresh, pollMs)
  }

  refresh()

  const token = getAccessToken()
  if (WS_ADMIN_ENABLED && WS_ADMIN_BASE && token) {
    stopWs = connectAdminWs(WS_ADMIN_BASE, token, (env) => {
      if (env.name === WS_EVENTS.schemeMonitorChanged) refresh()
    })
    startPoll()
  } else {
    startPoll()
  }

  return () => {
    stopped = true
    stopWs?.()
    stopPoll()
  }
}

/** 仪表盘 KPI：监听提现队列 + 方案监控 + KPI 变更（WS-4 + 降级轮询） */
export function startDashboardKpiSync(onRefresh: () => void, pollMs = FALLBACK_POLL_MS) {
  let stopped = false
  let pollTimer: ReturnType<typeof setInterval> | null = null
  let stopWs: (() => void) | null = null

  function refresh() {
    if (stopped) return
    onRefresh()
  }

  function stopPoll() {
    if (pollTimer) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function startPoll() {
    stopPoll()
    pollTimer = window.setInterval(refresh, pollMs)
  }

  refresh()

  const token = getAccessToken()
  if (WS_ADMIN_ENABLED && WS_ADMIN_BASE && token) {
    stopWs = connectAdminWs(WS_ADMIN_BASE, token, (env) => {
      if (
        env.name === WS_EVENTS.withdrawQueueChanged
        || env.name === WS_EVENTS.schemeMonitorChanged
        || env.name === WS_EVENTS.dashboardKpiChanged
      ) {
        refresh()
      }
    })
    startPoll()
  } else {
    startPoll()
  }

  return () => {
    stopped = true
    stopWs?.()
    stopPoll()
  }
}
