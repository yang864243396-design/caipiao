/**
 * 投注列表 → 详情返回：缓存优先 + 锚点回退。
 * 缓存命中则瞬时还原；失效时由页面用 anchorRecordNo 发一次请求定位。
 *
 * 列表行数据放模块级内存（组件 remount 不丢）；滚动/锚点放 sessionStorage。
 */

export interface BetListRestoreState {
  scrollY: number
  anchorRecordNo: string
  filters?: Record<string, unknown>
  savedAt: number
}

const TTL_MS = 30 * 60 * 1000

/** 模块级列表快照：避免进详情卸载后 Map 被清空导致返回必重拉 */
const listMemoryCache = new Map<string, unknown>()

export function setBetListMemoryCache<T>(key: string, value: T): void {
  listMemoryCache.set(key, value)
}

export function getBetListMemoryCache<T>(key: string): T | undefined {
  const v = listMemoryCache.get(key)
  return v === undefined ? undefined : (v as T)
}

export function clearBetListMemoryCache(key?: string): void {
  if (key == null) listMemoryCache.clear()
  else listMemoryCache.delete(key)
}

function storageKey(pageKey: string): string {
  return `bet-list-restore:${pageKey}`
}

export function saveBetListRestore(pageKey: string, state: Omit<BetListRestoreState, 'savedAt'>): void {
  try {
    const payload: BetListRestoreState = { ...state, savedAt: Date.now() }
    sessionStorage.setItem(storageKey(pageKey), JSON.stringify(payload))
  } catch {
    /* ignore quota */
  }
}

export function takeBetListRestore(pageKey: string): BetListRestoreState | null {
  try {
    const raw = sessionStorage.getItem(storageKey(pageKey))
    if (!raw) return null
    sessionStorage.removeItem(storageKey(pageKey))
    const parsed = JSON.parse(raw) as BetListRestoreState
    if (!parsed || typeof parsed !== 'object') return null
    if (!parsed.savedAt || Date.now() - parsed.savedAt > TTL_MS) return null
    return parsed
  } catch {
    return null
  }
}

export function peekBetListRestore(pageKey: string): BetListRestoreState | null {
  try {
    const raw = sessionStorage.getItem(storageKey(pageKey))
    if (!raw) return null
    const parsed = JSON.parse(raw) as BetListRestoreState
    if (!parsed?.savedAt || Date.now() - parsed.savedAt > TTL_MS) return null
    return parsed
  } catch {
    return null
  }
}
