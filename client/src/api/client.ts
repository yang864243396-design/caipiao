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
  /**
   * 写接口连点限制（默认 true）：同一 method+url+body 在 1s 内重复调用会抛 RequestThrottledError。
   * GET 不受限。传 false 可关闭（特殊场景）。
   */
  throttle?: boolean
}

/** 写接口 1s 连点限制（与按钮层配合，防止漏网重复提交） */
const MUTATE_THROTTLE_MS = 1000

export class RequestThrottledError extends ApiError {
  constructor(message = '操作过于频繁，请稍后再试') {
    super(message, 429, 42900)
    this.name = 'RequestThrottledError'
  }
}

export function isRequestThrottledError(err: unknown): err is RequestThrottledError {
  return err instanceof RequestThrottledError
}

const mutateThrottleMap = new Map<string, number>()

function isMutatingMethod(method: HttpMethod): boolean {
  return method === 'POST' || method === 'PUT' || method === 'PATCH' || method === 'DELETE'
}

function mutateThrottleKey(method: HttpMethod, url: string, body: unknown): string {
  let bodyKey = ''
  if (body !== undefined && body !== null) {
    if (typeof body === 'string') bodyKey = body
    else if (body instanceof FormData) bodyKey = '[form-data]'
    else {
      try {
        bodyKey = JSON.stringify(body)
      } catch {
        bodyKey = String(body)
      }
    }
  }
  return `${method}\n${url}\n${bodyKey}`
}

function assertMutateThrottle(method: HttpMethod, url: string, body: unknown): void {
  const key = mutateThrottleKey(method, url, body)
  const now = Date.now()
  const prev = mutateThrottleMap.get(key)
  if (prev !== undefined && now - prev < MUTATE_THROTTLE_MS) {
    throw new RequestThrottledError()
  }
  mutateThrottleMap.set(key, now)
  if (mutateThrottleMap.size > 200) {
    for (const [k, at] of mutateThrottleMap) {
      if (now - at >= MUTATE_THROTTLE_MS) mutateThrottleMap.delete(k)
    }
  }
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
  const { method = 'GET', headers = {}, body, auth = true, throttle = true } = opts
  const url = buildUrl(path, opts.query)
  if (throttle && isMutatingMethod(method)) {
    assertMutateThrottle(method, url, body)
  }
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
    if (isRequestThrottledError(err)) {
      throw err
    }
    if (isSessionExpiredError(err)) {
      return hangAfterSessionExpired<T>()
    }
    throw err
  }
}
