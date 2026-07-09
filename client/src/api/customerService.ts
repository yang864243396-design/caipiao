import { requestApi } from '@/api/client'
import { ensureClientSession } from '@/api/auth'

export interface CustomerServiceAgent {
  id: string
  name: string
  tgLink: string
  workHours: string
}

export async function fetchCustomerServiceAgents(): Promise<CustomerServiceAgent[]> {
  await ensureClientSession()
  const res = await requestApi<{ items: CustomerServiceAgent[] }>('/client/customer-service/agents')
  return res.items ?? []
}

/** 将后台配置的 TG 转为可打开的 https 链接 */
export function normalizeTgHref(link: string): string {
  const t = link.trim()
  if (!t) return ''
  if (/^https?:\/\//i.test(t)) return t
  const username = t.startsWith('@') ? t.slice(1) : t
  return `https://t.me/${username}`
}

/** 列表展示用 TG 文案 */
export function tgDisplayLabel(link: string): string {
  const t = link.trim()
  if (!t) return '—'
  if (t.startsWith('@')) return t
  if (!/^https?:\/\//i.test(t)) return `@${t.replace(/^@/, '')}`
  try {
    const u = new URL(t)
    const path = u.pathname.replace(/^\//, '')
    return path ? `@${path}` : t
  } catch {
    return t
  }
}
