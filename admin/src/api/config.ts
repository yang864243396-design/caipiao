/** 后端 REST 基址：
 *  - 显式配置 VITE_API_BASE_URL 时优先使用；
 *  - 开发态未配置时，按当前访问的主机名推导 http://<host>:8080/api/v1，
 *    便于局域网内任意 IP / 设备直接联调（无需为每个 IP 改配置）；
 *  - 生产态未配置则走同源（由反向代理处理）。
 */
const ENV_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '')
export const API_BASE = ENV_BASE
  ? ENV_BASE
  : import.meta.env.DEV && typeof window !== 'undefined'
    ? `${window.location.protocol}//${window.location.hostname}:8080/api/v1`
    : ''

function httpBaseToWs(httpBase: string): string {
  return httpBase.replace(/^http/i, 'ws')
}

function resolveWsHttpBase(): string {
  if (API_BASE) return API_BASE
  if (import.meta.env.PROD && typeof window !== 'undefined') {
    return `${window.location.origin}/api/v1`
  }
  return ''
}

/** Admin WS 端点；未配置时由 API_BASE 或当前页面 origin 推导 */
export const WS_ADMIN_BASE = (() => {
  const explicit = (import.meta.env.VITE_WS_ADMIN_URL as string | undefined)?.replace(/\/$/, '')
  if (explicit) return explicit
  const base = resolveWsHttpBase()
  if (!base) return ''
  return httpBaseToWs(base) + '/ws/admin'
})()

export const WS_ADMIN_ENABLED =
  import.meta.env.VITE_WS_ENABLED !== 'false' && WS_ADMIN_BASE.length > 0
