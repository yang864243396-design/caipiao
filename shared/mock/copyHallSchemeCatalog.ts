/** 跟单大厅可选方案（源自全站方案监控「运行中」实例） */
import type { CopyHallRankSlot } from './copyHallRankings'

export interface CopyHallSchemeCandidate {
  /** 业务主键 refId，与全站方案监控「业务主键」列同源 */
  schemeId: string
  schemeName: string
  playMethod: string
  playTypeId?: string
  subPlayId?: string
  lotteryCode: string
  lotteryLabel: string
  /** 会员 / 发布者 */
  publisherName: string
  /** 全站方案监控实例 ID */
  instanceId?: string
  kind?: string
  status?: '待开启' | '运行中' | '已暂停' | '已封停'
  simBet?: boolean
  createdAt?: string
  /** 计算时间区间内投注次数 */
  betCount?: number
  /** 计算时间区间内胜率（%） */
  winRate?: number
}

export interface CopyHallFilterOpts {
  lotteryCode?: string
  keyword?: string
  /** 弹窗筛选：schemeName | snapshotId | instanceId；未指定时沿用宽泛关键词匹配 */
  searchField?: 'schemeName' | 'snapshotId' | 'instanceId'
  status?: string
  createdStart?: string
  createdEnd?: string
}

export function filterCopyHallSchemes(
  catalog: CopyHallSchemeCandidate[],
  opts: CopyHallFilterOpts,
): CopyHallSchemeCandidate[] {
  let rows = catalog
  if (opts.lotteryCode) {
    rows = rows.filter((s) => s.lotteryCode === opts.lotteryCode)
  }
  if (opts.status) {
    rows = rows.filter((s) => s.status === opts.status)
  }
  if (opts.createdStart || opts.createdEnd) {
    rows = rows.filter((s) => {
      if (!s.createdAt) return false
      const t = new Date(s.createdAt).getTime()
      if (opts.createdStart && t < new Date(opts.createdStart).getTime()) return false
      if (opts.createdEnd) {
        const end = new Date(opts.createdEnd)
        end.setHours(23, 59, 59, 999)
        if (t > end.getTime()) return false
      }
      return true
    })
  }
  const q = opts.keyword?.trim()
  if (q) {
    if (opts.searchField === 'instanceId') {
      rows = rows.filter((s) => s.instanceId?.includes(q) ?? false)
    } else if (opts.searchField === 'snapshotId') {
      rows = rows.filter((s) => s.schemeId.includes(q) || (s.instanceId?.includes(q) ?? false))
    } else if (opts.searchField === 'schemeName') {
      rows = rows.filter((s) => s.schemeName.includes(q))
    } else {
      rows = rows.filter(
        (s) =>
          s.schemeName.includes(q) ||
          s.schemeId.includes(q) ||
          s.playMethod.includes(q) ||
          s.publisherName.includes(q) ||
          (s.instanceId?.includes(q) ?? false) ||
          (s.kind?.includes(q) ?? false),
      )
    }
  }
  return rows
}

export function findCopyHallSchemeIn(
  catalog: CopyHallSchemeCandidate[],
  schemeId: string,
): CopyHallSchemeCandidate | undefined {
  return catalog.find((s) => s.schemeId === schemeId)
}

export function candidateToRankSlot(
  candidate: CopyHallSchemeCandidate,
  rank: number,
): CopyHallRankSlot {
  return {
    rank,
    schemeId: candidate.schemeId,
    schemeName: candidate.schemeName,
    playMethod: candidate.playMethod,
    playTypeId: candidate.playTypeId,
    subPlayId: candidate.subPlayId,
    lotteryCode: candidate.lotteryCode,
    lotteryLabel: candidate.lotteryLabel,
  }
}
