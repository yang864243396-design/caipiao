/**
 * 后端基址：开发默认空（走同源或 mock），上线前在 .env 中配置 VITE_API_BASE_URL
 */
export const API_BASE =
  (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') ?? ''

export const USE_MOCK = import.meta.env.VITE_USE_MOCK !== 'false'
