/** 按授权账号 + 主币种缓存第三方余额（本地暂存，切换账号时先展示上次值） */

export interface GuajiBalanceCacheEntry {
  username: string
  currency: string
  amount: number
  updatedAt: number
}

const STORAGE_KEY = 'caipiao_guaji_balance_v1'

function cacheKey(username: string, currency: string): string {
  return `${username.trim().toLowerCase()}:${currency.trim().toUpperCase()}`
}

function readStore(): Record<string, GuajiBalanceCacheEntry> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as Record<string, GuajiBalanceCacheEntry>
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function writeStore(store: Record<string, GuajiBalanceCacheEntry>): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(store))
  } catch {
    /* 存储满或隐私模式：忽略 */
  }
}

export function readGuajiBalanceCache(
  username: string,
  currency: string,
): GuajiBalanceCacheEntry | null {
  if (!username.trim() || !currency.trim()) return null
  const entry = readStore()[cacheKey(username, currency)]
  if (!entry || typeof entry.amount !== 'number' || !Number.isFinite(entry.amount)) return null
  return entry
}

export function writeGuajiBalanceCache(entry: GuajiBalanceCacheEntry): void {
  if (!entry.username.trim() || !entry.currency.trim()) return
  const store = readStore()
  store[cacheKey(entry.username, entry.currency)] = {
    username: entry.username.trim(),
    currency: entry.currency.trim().toUpperCase(),
    amount: entry.amount,
    updatedAt: entry.updatedAt || Date.now(),
  }
  writeStore(store)
}

/** 金额是否在展示精度下相同（避免浮点抖动触发重绘） */
export function guajiAmountsEqual(a: number, b: number): boolean {
  return Math.abs(a - b) < 0.005
}
