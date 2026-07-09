import { ElMessage } from 'element-plus'

import { useAuthStore } from '@/stores/auth'

/** 标记 token 失效，供 requestApi 识别 */
export class SessionExpiredError extends Error {
  readonly sessionExpired = true

  constructor() {
    super('登录已过期')
    this.name = 'SessionExpiredError'
  }
}

export function isSessionExpiredError(err: unknown): err is SessionExpiredError {
  return (
    err instanceof SessionExpiredError ||
    (err instanceof Error && (err as SessionExpiredError).sessionExpired === true)
  )
}

let handling = false

/** token 失效：清会话并跳转登录（并发 401 仅处理一次） */
export async function handleSessionExpired(): Promise<void> {
  if (handling) return
  handling = true

  try {
    const { router } = await import('@/router')
    const auth = useAuthStore()
    const current = router.currentRoute.value

    if (current.path === '/login') {
      auth.logout()
      return
    }

    const redirect = current.fullPath
    auth.logout()
    ElMessage.warning('登录已失效，请重新登录')
    await router.replace({ path: '/login', query: { redirect } })
  } finally {
    handling = false
  }
}

/** 401 后阻止业务层继续抛错 / 重复 toast */
export function hangAfterSessionExpired<T>(): Promise<T> {
  return new Promise(() => {})
}
