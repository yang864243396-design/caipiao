import { requestApi } from '@/api/client'

export interface GuajiAccountRow {
  id: number
  guajiUsername: string
  isActive: boolean
  boundAt: string
  lastSyncAt?: string
  lastTokenError?: string
  /** 当前启用且第三方 Token 已过期或失效 */
  authExpired?: boolean
}

export interface GuajiAuthStatus {
  hasActiveGuajiAuth: boolean
  bindingCount: number
  activeUsername?: string
  /** 有启用账号但其 Token 已过期或失效 */
  activeAuthExpired?: boolean
}

export interface GuajiBalance {
  currency: string
  amount: number
  username?: string
}

export interface GuajiBindResult {
  account?: GuajiAccountRow
  mfaRequired?: boolean
  loginKey?: string
}

export async function fetchGuajiAuthStatus(): Promise<GuajiAuthStatus> {
  const res = await requestApi<GuajiAuthStatus>('/client/guaji/auth-status')
  return res
}

export async function fetchGuajiAccounts(): Promise<GuajiAccountRow[]> {
  const res = await requestApi<{ items: GuajiAccountRow[] }>('/client/guaji/accounts')
  return res.items ?? []
}

export async function fetchGuajiBalance(): Promise<GuajiBalance> {
  return requestApi<GuajiBalance>('/client/guaji/balance')
}

export async function bindGuajiAccount(body: {
  username: string
  password: string
  loginKey?: string
  googleCode?: string
  emailCode?: string
  phoneCode?: string
}): Promise<GuajiBindResult> {
  return requestApi<GuajiBindResult>('/client/guaji/accounts/bind', { method: 'POST', body })
}

export async function activateGuajiAccount(id: number): Promise<GuajiAccountRow> {
  return requestApi<GuajiAccountRow>(`/client/guaji/accounts/${id}/activate`, { method: 'POST' })
}

export async function reauthGuajiAccount(id: number): Promise<GuajiAccountRow> {
  return requestApi<GuajiAccountRow>(`/client/guaji/accounts/${id}/reauth`, { method: 'POST' })
}

export async function deleteGuajiAccount(id: number): Promise<void> {
  await requestApi<{ ok: boolean }>(`/client/guaji/accounts/${id}`, { method: 'DELETE' })
}

export type PrimaryCurrency = 'USDT' | 'TRX' | 'CNY'

export const PRIMARY_CURRENCIES: PrimaryCurrency[] = ['USDT', 'TRX', 'CNY']

export async function fetchPrimaryCurrency(): Promise<PrimaryCurrency> {
  const res = await requestApi<{ currency: PrimaryCurrency }>('/client/guaji/primary-currency')
  return res.currency
}

export async function setPrimaryCurrency(currency: PrimaryCurrency): Promise<PrimaryCurrency> {
  const res = await requestApi<{ currency: PrimaryCurrency }>('/client/guaji/primary-currency', {
    method: 'PUT',
    body: { currency },
  })
  return res.currency
}
