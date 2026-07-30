import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  saveBetListRestore,
  takeBetListRestore,
  peekBetListRestore,
  setBetListMemoryCache,
  getBetListMemoryCache,
  clearBetListMemoryCache,
  type BetListRestoreState,
} from './useBetListRestore'

// 从投注列表点进详情、再返回时用它还原滚动位置和筛选条件。
// 三个容易出错的点：take 是一次性的（读完即清，否则第二次进列表会莫名跳位）、
// 超过 30 分钟的缓存要作废（数据早变了，按旧锚点定位会错）、
// 存储层抛异常必须吞掉（隐私模式下 sessionStorage 会直接 throw，
// 还原失败顶多是不还原，不能让整个页面崩掉）。

const KEY = 'cloud-bet-records'
const TTL_MS = 30 * 60 * 1000

function stateOf(over: Partial<Omit<BetListRestoreState, 'savedAt'>> = {}) {
  return { scrollY: 1200, anchorRecordNo: 'CB1785210232593016', ...over }
}

describe('useBetListRestore', () => {
  beforeEach(() => {
    sessionStorage.clear()
    clearBetListMemoryCache()
  })

  afterEach(() => {
    vi.useRealTimers()
    clearBetListMemoryCache()
  })

  describe('模块级列表快照', () => {
    it('跨「组件卸载」仍可取回', () => {
      setBetListMemoryCache(KEY, { rows: [{ recordNo: 'A' }], hasMore: false })
      expect(getBetListMemoryCache<{ rows: { recordNo: string }[] }>(KEY)?.rows[0]?.recordNo).toBe('A')
    })

    it('可按 key 清除', () => {
      setBetListMemoryCache('a', 1)
      setBetListMemoryCache('b', 2)
      clearBetListMemoryCache('a')
      expect(getBetListMemoryCache('a')).toBeUndefined()
      expect(getBetListMemoryCache('b')).toBe(2)
    })
  })

  describe('存取往返', () => {
    it('存进去能原样取出来', () => {
      saveBetListRestore(KEY, stateOf())
      const got = takeBetListRestore(KEY)
      expect(got).toMatchObject({ scrollY: 1200, anchorRecordNo: 'CB1785210232593016' })
      expect(got?.savedAt).toBeTypeOf('number')
    })

    it('筛选条件一并保留', () => {
      const filters = { lotteryCode: 'tron_ffc_1m', mode: 'sim', page: 3 }
      saveBetListRestore(KEY, stateOf({ filters }))
      expect(takeBetListRestore(KEY)?.filters).toEqual(filters)
    })

    it('不同页面各存各的，互不干扰', () => {
      saveBetListRestore('cloud', stateOf({ scrollY: 100 }))
      saveBetListRestore('member', stateOf({ scrollY: 900 }))
      expect(takeBetListRestore('cloud')?.scrollY).toBe(100)
      expect(takeBetListRestore('member')?.scrollY).toBe(900)
    })

    it('重复保存以最后一次为准', () => {
      saveBetListRestore(KEY, stateOf({ scrollY: 100 }))
      saveBetListRestore(KEY, stateOf({ scrollY: 500 }))
      expect(takeBetListRestore(KEY)?.scrollY).toBe(500)
    })
  })

  describe('take 是一次性的', () => {
    it('取过一次之后就没了', () => {
      saveBetListRestore(KEY, stateOf())
      expect(takeBetListRestore(KEY)).not.toBeNull()
      expect(takeBetListRestore(KEY)).toBeNull()
    })

    it('取完底层存储也要清干净', () => {
      saveBetListRestore(KEY, stateOf())
      takeBetListRestore(KEY)
      expect(sessionStorage.getItem(`bet-list-restore:${KEY}`)).toBeNull()
    })

    it('没存过时返回 null', () => {
      expect(takeBetListRestore('never-saved')).toBeNull()
    })
  })

  describe('peek 不消费', () => {
    it('连看两次都还在', () => {
      saveBetListRestore(KEY, stateOf())
      expect(peekBetListRestore(KEY)?.scrollY).toBe(1200)
      expect(peekBetListRestore(KEY)?.scrollY).toBe(1200)
    })

    it('peek 之后仍可 take', () => {
      saveBetListRestore(KEY, stateOf())
      peekBetListRestore(KEY)
      expect(takeBetListRestore(KEY)).not.toBeNull()
    })
  })

  describe('30 分钟过期', () => {
    it('刚好卡在 TTL 之内仍有效', () => {
      vi.useFakeTimers()
      saveBetListRestore(KEY, stateOf())
      vi.advanceTimersByTime(TTL_MS - 1000)
      expect(takeBetListRestore(KEY)).not.toBeNull()
    })

    it('超过 TTL 后 take 返回 null', () => {
      vi.useFakeTimers()
      saveBetListRestore(KEY, stateOf())
      vi.advanceTimersByTime(TTL_MS + 1000)
      expect(takeBetListRestore(KEY)).toBeNull()
    })

    it('超过 TTL 后 peek 也返回 null', () => {
      vi.useFakeTimers()
      saveBetListRestore(KEY, stateOf())
      vi.advanceTimersByTime(TTL_MS + 1000)
      expect(peekBetListRestore(KEY)).toBeNull()
    })
  })

  describe('脏数据一律当作没有', () => {
    it.each([
      { name: '不是 JSON', raw: 'not-json{' },
      { name: 'JSON 但不是对象', raw: '"a string"' },
      { name: 'null', raw: 'null' },
      { name: '缺 savedAt', raw: JSON.stringify({ scrollY: 1, anchorRecordNo: 'x' }) },
      { name: 'savedAt 为 0', raw: JSON.stringify({ scrollY: 1, savedAt: 0 }) },
    ])('$name', ({ raw }) => {
      sessionStorage.setItem(`bet-list-restore:${KEY}`, raw)
      expect(takeBetListRestore(KEY)).toBeNull()
      expect(peekBetListRestore(KEY)).toBeNull()
    })
  })

  describe('存储层异常不能把页面搞崩', () => {
    it('写入抛异常（配额满/隐私模式）时静默失败', () => {
      const spy = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
        throw new DOMException('QuotaExceededError')
      })
      expect(() => saveBetListRestore(KEY, stateOf())).not.toThrow()
      spy.mockRestore()
    })

    it('读取抛异常时返回 null 而不是往上抛', () => {
      const spy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
        throw new DOMException('SecurityError')
      })
      expect(takeBetListRestore(KEY)).toBeNull()
      expect(peekBetListRestore(KEY)).toBeNull()
      spy.mockRestore()
    })
  })
})
