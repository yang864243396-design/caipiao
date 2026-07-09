import { API_BASE } from './config'
import {
  handleSessionExpired,
  hangAfterSessionExpired,
  isSessionExpiredError,
  SessionExpiredError,
} from './authSession'

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data: T
}

const TOKEN_KEY = 'admin_access_token'
const EXPIRES_KEY = 'admin_access_expires_at'

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export type RequestOptions = {
  method?: HttpMethod
  body?: unknown
  auth?: boolean
}

function buildUrl(path: string): string {
  return path.startsWith('http') ? path : `${API_BASE}${path.startsWith('/') ? path : `/${path}`}`
}

export function getAccessToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

export function setAccessToken(token: string | null, expiresAt?: number): void {
  try {
    if (token) {
      localStorage.setItem(TOKEN_KEY, token)
      if (expiresAt) localStorage.setItem(EXPIRES_KEY, String(expiresAt))
    } else {
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(EXPIRES_KEY)
    }
  } catch {
    /* ignore */
  }
}

export function getExpiresAt(): number {
  try {
    return Number(localStorage.getItem(EXPIRES_KEY)) || 0
  } catch {
    return 0
  }
}

export async function requestApi<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, auth = true } = opts
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (auth) {
    const token = getAccessToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  try {
    const res = await fetch(buildUrl(path), {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
    const text = await res.text()
    const parsed = text ? (JSON.parse(text) as ApiEnvelope<T>) : null

    if (!res.ok || !parsed) {
      if (res.status === 401 && auth) {
        await handleSessionExpired()
        throw new SessionExpiredError()
      }
      throw new ApiError(parsed?.message || res.statusText || '请求失败', res.status, parsed?.code)
    }
    if (parsed.code !== 0) {
      throw new ApiError(parsed.message || '业务错误', res.status, parsed.code)
    }
    return parsed.data
  } catch (err) {
    if (isSessionExpiredError(err)) {
      return hangAfterSessionExpired<T>()
    }
    throw err
  }
}
