import { resolvePlayTypeLabel } from '@client/utils/playTypeLabels'
import type { CopyHallRankSlot } from '@shared/mock/copyHallRankings'
import type { CopyHallSchemeCandidate } from '@shared/mock/copyHallSchemeCatalog'

function isBarePlayToken(value?: string): boolean {
  const t = String(value ?? '').trim()
  if (!t) return true
  if (/^\d+$/.test(t)) return true
  if (/^g\d+$/i.test(t)) return true
  return false
}

/** 跟单大厅玩法展示：避免 subPlayId / g006 等裸 ID 直接展示 */
export function resolveCopyHallPlayLabel(input: {
  playMethod?: string
  playTypeId?: string
  subPlayId?: string
}): string {
  const pm = String(input.playMethod ?? '').trim()
  if (pm && !isBarePlayToken(pm)) return pm

  const label = resolvePlayTypeLabel({
    playTypeId: input.playTypeId,
    typeId: input.playTypeId,
    betMode: input.playTypeId,
  }).trim()
  if (label && !isBarePlayToken(label)) return label

  return pm || '—'
}

export function rankSlotPlayLabel(
  slot: Pick<CopyHallRankSlot, 'playMethod' | 'playTypeId' | 'subPlayId'>,
  catalog?: Pick<CopyHallSchemeCandidate, 'playMethod' | 'playTypeId' | 'subPlayId'>,
): string {
  return resolveCopyHallPlayLabel({
    playMethod: slot.playMethod || catalog?.playMethod,
    playTypeId: slot.playTypeId || catalog?.playTypeId,
    subPlayId: slot.subPlayId || catalog?.subPlayId,
  })
}
