import { MOCK_APPENDIX } from './appendixMock'
import { playIdsForCopyHallMethod } from './copyHallPlayIds'

/** 跟单大厅榜单类型（对齐 client CopyHallView Tab） */
export type CopyHallBoardKind = 'master' | 'contrary'

/** 单榜名次条目（固定 Top 10） */
export interface CopyHallRankSlot {
  rank: number
  schemeId: string
  schemeName: string
  /** 玩法展示，如「定位胆万位」 */
  playMethod: string
  /** 玩法段 ID（优先于 playMethod 中文解析） */
  playTypeId?: string
  /** 子玩法 ID */
  subPlayId?: string
  /** 上榜方案所属彩种 */
  lotteryCode?: string
  lotteryLabel?: string
}

export interface CopyHallLotteryBoards {
  lotteryLabel: string
  lotteryCode: string
  master: CopyHallRankSlot[]
  contrary: CopyHallRankSlot[]
}

export interface CopyHallRankingsState {
  boards: CopyHallLotteryBoards[]
}

export const COPY_HALL_LOTTERIES = [
  { code: 'tron_ffc_1m', label: '波场1分彩' },
  { code: 'hash_ffc_1m', label: '哈希1分彩' },
  { code: 'eth_ffc_1m', label: '以太坊1分彩' },
  { code: 'bnb_ffc_1m', label: '币安1分彩' },
  { code: 'tron_jisu', label: '波场极速彩' },
  { code: 'tron_syxw', label: '波场11选5' },
] as const

export type CopyHallLotteryLabel = (typeof COPY_HALL_LOTTERIES)[number]['label']

export const STORAGE_COPY_HALL_RANKINGS = 'mock_copy_hall_rankings'

const LEGACY_COPY_HALL_RANKINGS = 'admin_mock_copy_hall_rankings'

const MASTER_DEFAULT_NAMES = [
  '太乙后二',
  '紫燕万位',
  '莺凤十位',
  '宛天个位',
  '路线6000+',
  '打狗前二',
  '邯肖任四',
  '关冲70+',
  '猎豹后二',
  '青衫万位',
] as const

const CONTRARY_DEFAULT_NAMES = [
  '逆锋万位',
  '反打后二',
  '折戟十位',
  '回风个位',
  '暗线3000-',
  '退守前三',
  '虚晃任四',
  '蛰伏50-',
  '裂空后一',
  '寒江千位',
] as const

const PLAY_METHODS = [
  '定位胆万位',
  '定位胆后二',
  '定位胆十位',
  '定位胆个位',
  '组选六',
  '定位胆前三',
  '任选四',
  '定位胆后一',
  '定位胆千位',
  '定位胆任二',
] as const

function buildSlots(names: readonly string[], idPrefix: string): CopyHallRankSlot[] {
  return names.map((schemeName, i) => {
    const playMethod = PLAY_METHODS[i] ?? '定位胆万位'
    const ids = playIdsForCopyHallMethod(playMethod)
    return {
      rank: i + 1,
      schemeId: i === 0 ? MOCK_APPENDIX.schemeCopyId : `${idPrefix}_${3001 + i}`,
      schemeName,
      playMethod,
      playTypeId: ids.playTypeId,
      subPlayId: ids.subPlayId,
    }
  })
}

function cloneBoards(boards: CopyHallLotteryBoards[]): CopyHallLotteryBoards[] {
  return boards.map((b) => ({
    ...b,
    master: b.master.map((s) => ({ ...s })),
    contrary: b.contrary.map((s) => ({ ...s })),
  }))
}

/** 客户端 CopyHallView 演示同源默认 Top 10 */
export function defaultCopyHallRankingsState(): CopyHallRankingsState {
  const master = buildSlots(MASTER_DEFAULT_NAMES, 'copy_demo')
  const contrary = buildSlots(CONTRARY_DEFAULT_NAMES, 'copy_contrary')

  return {
    boards: COPY_HALL_LOTTERIES.map((lot) => ({
      lotteryLabel: lot.label,
      lotteryCode: lot.code,
      master: master.map((s) => ({ ...s })),
      contrary: contrary.map((s) => ({ ...s })),
    })),
  }
}

function readCookie(name: string): string | null {
  if (typeof document === 'undefined') return null
  const m = document.cookie.match(
    new RegExp(`(?:^|; )${encodeURIComponent(name).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}=([^;]*)`),
  )
  return m ? decodeURIComponent(m[1]) : null
}

function writeCookie(name: string, value: string, maxAgeSec = 7 * 86400) {
  if (typeof document === 'undefined') return
  document.cookie = `${encodeURIComponent(name)}=${encodeURIComponent(value)}; path=/; max-age=${maxAgeSec}`
}

function readStorage(key: string, legacy?: string): string | null {
  try {
    const v = localStorage.getItem(key)
    if (v !== null) return v
    if (legacy) return localStorage.getItem(legacy)
  } catch {
    /* SSR / 隐私模式 */
  }
  return null
}

function writeStorage(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* ignore */
  }
}

function normalizeSlots(slots: CopyHallRankSlot[], idPrefix: string): CopyHallRankSlot[] {
  const sorted = [...slots].sort((a, b) => a.rank - b.rank).slice(0, 10)
  while (sorted.length < 10) {
    const rank = sorted.length + 1
    const playMethod = PLAY_METHODS[rank - 1] ?? '定位胆万位'
    const ids = playIdsForCopyHallMethod(playMethod)
    sorted.push({
      rank,
      schemeId: `${idPrefix}_${3000 + rank}`,
      schemeName: `占位方案${rank}`,
      playMethod,
      playTypeId: ids.playTypeId,
      subPlayId: ids.subPlayId,
    })
  }
  return sorted.map((s, i) => ({ ...s, rank: i + 1 }))
}

function parseState(raw: string): CopyHallRankingsState | null {
  try {
    const parsed = JSON.parse(raw) as CopyHallRankingsState
    if (!parsed?.boards?.length) return null

    const defaults = defaultCopyHallRankingsState()
    const byCode = new Map(parsed.boards.map((b) => [b.lotteryCode, b]))

    return {
      boards: defaults.boards.map((def) => {
        const saved = byCode.get(def.lotteryCode)
        if (!saved) return { ...def, master: def.master.map((s) => ({ ...s })), contrary: def.contrary.map((s) => ({ ...s })) }
        return {
          lotteryLabel: def.lotteryLabel,
          lotteryCode: def.lotteryCode,
          master: normalizeSlots(saved.master ?? [], 'copy_demo'),
          contrary: normalizeSlots(saved.contrary ?? [], 'copy_contrary'),
        }
      }),
    }
  } catch {
    return null
  }
}

export function readCopyHallRankings(): CopyHallRankingsState {
  const fromCookie = readCookie(STORAGE_COPY_HALL_RANKINGS)
  if (fromCookie) {
    const parsed = parseState(fromCookie)
    if (parsed) return parsed
  }
  const raw = readStorage(STORAGE_COPY_HALL_RANKINGS, LEGACY_COPY_HALL_RANKINGS)
  if (raw) {
    const parsed = parseState(raw)
    if (parsed) return parsed
  }
  return defaultCopyHallRankingsState()
}

export function writeCopyHallRankings(state: CopyHallRankingsState) {
  const payload = JSON.stringify(state)
  writeStorage(STORAGE_COPY_HALL_RANKINGS, payload)
  writeStorage(LEGACY_COPY_HALL_RANKINGS, payload)
  writeCookie(STORAGE_COPY_HALL_RANKINGS, payload)
}

export function getBoardForLottery(
  state: CopyHallRankingsState,
  lotteryLabel: string,
): CopyHallLotteryBoards | undefined {
  return state.boards.find((b) => b.lotteryLabel === lotteryLabel)
}

export function getRankSlots(
  state: CopyHallRankingsState,
  lotteryLabel: string,
  kind: CopyHallBoardKind,
): CopyHallRankSlot[] {
  const board = getBoardForLottery(state, lotteryLabel)
  if (!board) return defaultCopyHallRankingsState().boards[0][kind]
  return board[kind]
}

/** admin/client 跨 Tab、跨端口（Cookie）同步 */
export function subscribeCopyHallRankingsSync(onChange: () => void): () => void {
  if (typeof window === 'undefined') return () => {}

  const onStorage = (e: StorageEvent) => {
    if (e.key === STORAGE_COPY_HALL_RANKINGS || e.key === LEGACY_COPY_HALL_RANKINGS) {
      onChange()
    }
  }
  window.addEventListener('storage', onStorage)
  const timer = window.setInterval(onChange, 2000)

  return () => {
    window.removeEventListener('storage', onStorage)
    window.clearInterval(timer)
  }
}

export function patchCopyHallBoard(
  state: CopyHallRankingsState,
  lotteryCode: string,
  kind: CopyHallBoardKind,
  slots: CopyHallRankSlot[],
): CopyHallRankingsState {
  const next = cloneBoards(state.boards)
  const board = next.find((b) => b.lotteryCode === lotteryCode)
  if (!board) return state
  board[kind] = normalizeSlots(slots, kind === 'master' ? 'copy_demo' : 'copy_contrary')
  return { boards: next }
}

export function moveRankSlot(
  slots: CopyHallRankSlot[],
  rank: number,
  direction: 'up' | 'down',
): CopyHallRankSlot[] {
  const sorted = [...slots].sort((a, b) => a.rank - b.rank)
  const idx = sorted.findIndex((s) => s.rank === rank)
  if (idx < 0) return slots
  const swapIdx = direction === 'up' ? idx - 1 : idx + 1
  if (swapIdx < 0 || swapIdx >= sorted.length) return slots
  ;[sorted[idx], sorted[swapIdx]] = [sorted[swapIdx], sorted[idx]]
  return sorted.map((s, i) => ({ ...s, rank: i + 1 }))
}
