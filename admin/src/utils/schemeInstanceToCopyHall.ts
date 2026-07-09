import type { SchemeInstanceRow, SchemeShareSnapshotRow } from '@/types/schemes'
import type { CopyHallSchemeCandidate } from '@shared/mock/copyHallSchemeCatalog'
import type { SchemeBetStats } from '@/stores/schemeInstances'
import { playIdsForCopyHallMethod } from '@shared/mock/copyHallPlayIds'
import { resolveCopyHallPlayLabel } from '@/utils/copyHallPlayLabel'
const PLAY_BY_KIND: Record<SchemeInstanceRow['kind'], string> = {
  自创: '组选六',
  反买: '定位胆万位',
  跟单: '定位胆万位',
}

/** 全站方案监控实例 → 跟单大厅可选方案 */
export function schemeInstanceToCopyHallCandidate(
  row: SchemeInstanceRow,
  stats?: SchemeBetStats,
): CopyHallSchemeCandidate {
  const playMethod = row.lotteryLabel.includes('3D')
    ? '组选六'
    : PLAY_BY_KIND[row.kind]

  return {
    schemeId: row.refId,
    schemeName: row.settings.schemeName || `${row.memberName} · ${row.kind}`,
    playMethod,
    playTypeId: row.settings.playTypeId,
    subPlayId: row.settings.subPlayId,
    lotteryCode: row.lotteryCode,
    lotteryLabel: row.lotteryLabel,
    publisherName: row.memberName,
    instanceId: row.id,
    kind: row.kind,
    status: row.status,
    simBet: row.simBet,
    createdAt: row.createdAt,
    betCount: stats?.betCount,
    winRate: stats?.winRate,
  }
}

/** 全站方案监控分享池快照 → 跟单大厅可选方案 */
export function shareSnapshotToCopyHallCandidate(row: SchemeShareSnapshotRow): CopyHallSchemeCandidate {
  const playTypeId = row.settings.playTypeId
  const subPlayId = row.settings.subPlayId
  const playMethod = resolveCopyHallPlayLabel({
    playMethod: row.playMethod?.trim(),
    playTypeId,
    subPlayId,
  })
  const fallbackIds = playIdsForCopyHallMethod(playMethod)

  return {
    schemeId: row.id,
    schemeName: row.schemeName,
    playMethod,
    playTypeId: playTypeId || fallbackIds.playTypeId,
    subPlayId: subPlayId || fallbackIds.subPlayId,
    lotteryCode: row.lotteryCode,
    lotteryLabel: row.lotteryLabel,
    publisherName: '—',
    instanceId: row.id,
    kind: '自创',
    createdAt: row.publishedAt,
  }
}

export function runningSchemeInstancesToCandidates(
  rows: SchemeInstanceRow[],
  statsFn?: (instanceId: string) => SchemeBetStats,
): CopyHallSchemeCandidate[] {
  return rows
    .filter((r) => r.status === '运行中')
    .map((r) => schemeInstanceToCopyHallCandidate(r, statsFn?.(r.id)))
}
