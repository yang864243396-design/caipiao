import { API_BASE } from './config'
import {
  handleSessionExpired,
  hangAfterSessionExpired,
  isSessionExpiredError,
  SessionExpiredError,
} from './authSession'
import type { ApiEnvelope } from './types'

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

const TOKEN_KEY = 'client_access_token'

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: number,
    public body?: unknown,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export type RequestOptions = {
  method?: HttpMethod
  headers?: Record<string, string>
  body?: unknown
  query?: Record<string, string | number | boolean | undefined>
  /** 默认 true；登录等公共接口传 false */
  auth?: boolean
}

function buildUrl(path: string, query?: RequestOptions['query']): string {
  const base = path.startsWith('http') ? path : `${API_BASE}${path.startsWith('/') ? path : `/${path}`}`
  if (!query || !Object.keys(query).length) return base
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined) q.set(k, String(v))
  }
  const s = q.toString()
  return s ? `${base}${base.includes('?') ? '&' : '?'}${s}` : base
}

export function getAccessToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

export function setAccessToken(token: string | null): void {
  try {
    if (token) localStorage.setItem(TOKEN_KEY, token)
    else localStorage.removeItem(TOKEN_KEY)
  } catch {
    /* ignore */
  }
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = 'GET', headers = {}, body, auth = true } = opts
  const url = buildUrl(path, opts.query)
  const isJson = body !== undefined && body !== null && typeof body === 'object' && !(body instanceof FormData)
  const reqHeaders: Record<string, string> = {
    ...(isJson ? { 'Content-Type': 'application/json' } : {}),
    ...headers,
  }
  if (auth) {
    const token = getAccessToken()
    if (token) reqHeaders.Authorization = `Bearer ${token}`
  }
  const res = await fetch(url, {
    method,
    headers: reqHeaders,
    body: isJson ? JSON.stringify(body) : (body as BodyInit | undefined),
  })
  const text = await res.text()
  const data = text ? (JSON.parse(text) as unknown) : null
  if (!res.ok) {
    const env = data as ApiEnvelope | null
    if (res.status === 401 && auth) {
      await handleSessionExpired()
      throw new SessionExpiredError()
    }
    throw new ApiError(env?.message || res.statusText || 'Request failed', res.status, env?.code, data)
  }
  return data as T
}

/** 解包 `{ code, message, data }`，code !== 0 抛 ApiError */
export async function requestApi<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  try {
    const env = await request<ApiEnvelope<T>>(path, opts)
    if (env.code !== 0) {
      // 40100 亦用于第三方授权/密码错误（HTTP 200），不可当作平台 token 失效
      throw new ApiError(env.message || '业务错误', 200, env.code, env)
    }
    return env.data
  } catch (err) {
    if (isSessionExpiredError(err)) {
      return hangAfterSessionExpired<T>()
    }
    throw err
  }
}
