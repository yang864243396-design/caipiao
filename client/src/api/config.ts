/** 后端 REST 基址：
 *  - 显式配置 VITE_API_BASE_URL 时优先使用；
 *  - 开发态未配置时，按当前访问的主机名推导 http://<host>:8080/api/v1，
 *    便于局域网内任意 IP / 设备直接联调（无需为每个 IP 改配置）；
 *  - 生产态未配置则走同源，由反向代理将 /api/v1 转发到后端。
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

/** 推导 WebSocket 用的 REST 基址；生产同源时 REST 可走相对路径，WS 仍需绝对 wss 地址 */
function resolveWsHttpBase(): string {
  if (API_BASE) return API_BASE
  if (import.meta.env.PROD && typeof window !== 'undefined') {
    return `${window.location.origin}/api/v1`
  }
  return ''
}

/** WebSocket 公共端点；未配置时由 API_BASE 或当前页面 origin 推导 */
export const WS_PUBLIC_BASE = (() => {
  const explicit = (import.meta.env.VITE_WS_PUBLIC_URL as string | undefined)?.replace(/\/$/, '')
  if (explicit) return explicit
  const base = resolveWsHttpBase()
  if (!base) return ''
  return httpBaseToWs(base) + '/ws/public'
})()

export const WS_ENABLED =
  import.meta.env.VITE_WS_ENABLED !== 'false' && WS_PUBLIC_BASE.length > 0

/** 会员 WS 端点；未配置时由 API_BASE 或当前页面 origin 推导 */
export const WS_CLIENT_BASE = (() => {
  const explicit = (import.meta.env.VITE_WS_CLIENT_URL as string | undefined)?.replace(/\/$/, '')
  if (explicit) return explicit
  const base = resolveWsHttpBase()
  if (!base) return ''
  return httpBaseToWs(base) + '/ws/client'
})()

export const WS_CLIENT_ENABLED =
  import.meta.env.VITE_WS_ENABLED !== 'false' && WS_CLIENT_BASE.length > 0
