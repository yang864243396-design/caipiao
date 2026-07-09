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

export function isSessionExpiredError(err: unknown): err is SessionExpiredError {
  return err instanceof SessionExpiredError || (err instanceof Error && (err as SessionExpiredError).sessionExpired === true)
}

let handling = false

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

    const redirect = current.fullPath
    logoutClient()

    await ElMessageBox.alert('登录状态已失效，请重新登录', '登录过期', {
      confirmButtonText: '重新登录',
      type: 'warning',
      closeOnClickModal: false,
      closeOnPressEscape: false,
      showClose: false,
    })

    await router.replace({ name: 'login', query: { redirect } })
  } finally {
    handling = false
  }
}

/** 401 后阻止业务层继续抛错 / 重复 toast */
export function hangAfterSessionExpired<T>(): Promise<T> {
  return new Promise(() => {})
}
