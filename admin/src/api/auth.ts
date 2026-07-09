import { requestApi, setAccessToken } from './client'

export interface LoginResult {
  accessToken: string
  expiresAt: string
  account: string
  displayName: string
  roleId: string
}

export interface AdminSession {
  account: string
  displayName: string
  roleId: string
}

export async function loginAdmin(account: string, password: string): Promise<LoginResult> {
  const result = await requestApi<LoginResult>('/admin/auth/login', {
    method: 'POST',
    auth: false,
    body: { account, password },
  })
  setAccessToken(result.accessToken, Date.parse(result.expiresAt))
  return result
}

export async function fetchAdminSession(): Promise<AdminSession> {
  return requestApi<AdminSession>('/admin/auth/session')
}

export function logoutAdmin(): void {
  setAccessToken(null)
}
