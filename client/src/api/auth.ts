import { requestApi, setAccessToken } from './client'
import type { AuthTokenPayload } from './types'

export async function loginClient(account: string, password: string): Promise<AuthTokenPayload> {
  const data = await requestApi<AuthTokenPayload>('/client/auth/login', {
    method: 'POST',
    auth: false,
    body: { account, password },
  })
  setAccessToken(data.accessToken)
  return data
}

/** 退出登录：清除本地 token */
export function logoutClient(): void {
  setAccessToken(null)
}

/**
 * 会话占位：登录改由登录页统一入口，这里不再静默自动登录。
 * 保留该函数以兼容各 API 模块的调用；无 token 时交由路由守卫拦截到登录页。
 */
export async function ensureClientSession(): Promise<void> {
  return
}
