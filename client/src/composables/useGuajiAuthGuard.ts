import {
  fetchGuajiAuthStatus,
  type GuajiAuthStatus,
} from '@/api/guaji/accounts'
import type { RouteLocationNormalized } from 'vue-router'

const GUAJI_AUTH_BIND = '/member/auth/bind'
const GUAJI_AUTH_LIST = '/member/auth/list'

const GUAJI_WHITELIST = new Set([GUAJI_AUTH_BIND, GUAJI_AUTH_LIST, '/login'])

let cached: { status: GuajiAuthStatus; at: number } | null = null
const CACHE_TTL_MS = 15_000

export function guajiAuthWhitelistPaths(): readonly string[] {
  return [GUAJI_AUTH_BIND, GUAJI_AUTH_LIST]
}

export function routeNeedsGuajiGuard(to: RouteLocationNormalized): boolean {
  if (to.meta.public) return false
  return !GUAJI_WHITELIST.has(to.path)
}

export function guajiRouteRedirect(to: RouteLocationNormalized, status: GuajiAuthStatus): string {
  if (status.bindingCount === 0) return GUAJI_AUTH_BIND
  if (!status.hasActiveGuajiAuth) return GUAJI_AUTH_LIST
  return typeof to.fullPath === 'string' ? to.fullPath : '/'
}

export function guajiGateToastMessage(status: GuajiAuthStatus): string {
  if (status.bindingCount === 0) return '请先绑定第三方授权账号'
  if (status.activeAuthExpired) return '授权已失效，请重新授权'
  return '请先启用授权账号'
}

export async function resolveGuajiAuthStatus(force = false): Promise<GuajiAuthStatus> {
  if (!force && cached && Date.now() - cached.at < CACHE_TTL_MS) {
    return cached.status
  }
  const status = await fetchGuajiAuthStatus()
  cached = { status, at: Date.now() }
  return status
}

export function invalidateGuajiAuthCache(): void {
  cached = null
}

let pendingGuajiToast: string | null = null

export function consumeGuajiGateToast(): string | null {
  const msg = pendingGuajiToast
  pendingGuajiToast = null
  return msg
}

export function setPendingGuajiGateToast(msg: string): void {
  pendingGuajiToast = msg
}

export const guajiAuthRoutes = {
  bind: GUAJI_AUTH_BIND,
  list: GUAJI_AUTH_LIST,
} as const
