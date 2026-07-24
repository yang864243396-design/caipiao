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

let handling = false

function buildLoginHref(redirect: string): string {
  const base = import.meta.env.BASE_URL || '/'
  const prefix = base.endsWith('/') ? base.slice(0, -1) : base
  const qs = new URLSearchParams({ redirect, expired: '1' })
  return `${prefix}/login?${qs.toString()}`
}

/**
 * token 失效：清除会话、弹窗提示并跳转登录页（并发 401 仅处理一次）。
 */
export async function handleSessionExpired(): Promise<void> {
  if (handling) return
  handling = true

  try {
    const { router } = await import('@/router')
    const current = router.currentRoute.value
    if (current.name === 'login') {
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
      })
    } catch {
      // 关闭/异常也继续去登录页
    }

    // 弹窗期间若有逻辑写回 token，跳转前再清一次，避免守卫把已登录用户从登录页弹回原页
    logoutClient()

    try {
      await router.replace({
        path: '/login',
        query: { redirect, expired: '1' },
      })
    } catch {
      // 导航失败走硬跳转
    }

    // 软路由未切走时强制整页进入登录页（焦点陷阱/导航中止等边缘情况）
    const now = router.currentRoute.value
    if (now.name !== 'login' && now.path !== '/login') {
      window.location.assign(buildLoginHref(redirect))
    }
  } finally {
    handling = false
  }
}

/** 401 后阻止业务层继续抛错 / 重复 toast */
export function hangAfterSessionExpired<T>(): Promise<T> {
  return new Promise(() => {})
}
