import { ElMessageBox } from 'element-plus'

import { logoutClient } from './auth'

/** 标记 token 失效错误，供 request / requestApi 识别 */
export class SessionExpiredError extends Error {
  readonly sessionExpired = true

  constructor() {
    super('登录已过期')
    this.name = 'SessionExpiredError'
  }
}

export function isSessionExpiredError(err: unknown): boolean {
  return (
    err instanceof SessionExpiredError ||
    (err instanceof Error && (err as SessionExpiredError).sessionExpired === true)
  )
}

/** 并发 401 共用同一处理 Promise，避免第二次直接 return 导致无人跳转 */
let handling: Promise<void> | null = null

function buildLoginHref(redirect: string): string {
  const base = import.meta.env.BASE_URL || '/'
  const prefix = base.endsWith('/') ? base.slice(0, -1) : base
  const qs = new URLSearchParams({ redirect, expired: '1' })
  return `${prefix}/login?${qs.toString()}`
}

function goLoginPage(redirect: string): void {
  const href = buildLoginHref(redirect)
  // 会话失效必须整页进入登录：软路由 replace 可能被挂起的导航/组件卸载拖死，
  // 表现为点「重新登录」后仍停在原页，再点其它入口才被守卫送去登录。
  window.location.assign(href)
}

/**
 * token 失效：清除会话、弹窗提示并跳转登录页（并发 401 仅处理一次）。
 */
export function handleSessionExpired(): Promise<void> {
  if (handling) return handling

  handling = (async () => {
    try {
      const { router } = await import('@/router')
      const current = router.currentRoute.value
      if (current.name === 'login' || current.path === '/login') {
        logoutClient()
        return
      }

      const redirect = current.fullPath && current.fullPath !== '/login' ? current.fullPath : '/'
      logoutClient()

      try {
        await ElMessageBox.alert('登录状态已失效，请重新登录', '登录过期', {
          confirmButtonText: '重新登录',
          type: 'warning',
          closeOnClickModal: false,
          closeOnPressEscape: false,
          showClose: false,
          appendTo: document.body,
          // 避免被全局连点节流 stopImmediatePropagation 误伤确认按钮
          customClass: 'session-expired-msgbox',
        })
      } catch {
        // 关闭/异常也继续去登录页
      }

      // 弹窗期间若有逻辑写回 token，跳转前再清一次
      logoutClient()
      goLoginPage(redirect)
    } finally {
      handling = null
    }
  })()

  return handling
}

/** 401 后阻止业务层继续抛错 / 重复 toast */
export function hangAfterSessionExpired<T>(): Promise<T> {
  return new Promise(() => {})
}
