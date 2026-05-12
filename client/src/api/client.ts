import { API_BASE } from './config'

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
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

/**
 * 预留的通用请求器：将来替换为带 token 拦截器、刷新等
 */
export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = 'GET', headers = {}, body } = opts
  const url = buildUrl(path, opts.query)
  const isJson = body !== undefined && body !== null && typeof body === 'object' && !(body instanceof FormData)
  const res = await fetch(url, {
    method,
    headers: {
      ...(isJson ? { 'Content-Type': 'application/json' } : {}),
      ...headers,
    },
    body: isJson ? JSON.stringify(body) : (body as BodyInit | undefined),
  })
  const text = await res.text()
  const data = text ? (JSON.parse(text) as unknown) : null
  if (!res.ok) {
    throw new ApiError(res.statusText || 'Request failed', res.status, data)
  }
  return data as T
}
