import { requestApi } from './client'
import type { CopyHallBoardKind, CopyHallRankSlot } from '@shared/mock/copyHallRankings'

export interface AdminCopyHallRankingsResult {
  board: CopyHallBoardKind
  slots: CopyHallRankSlot[]
}

export async function fetchCopyHallRankingsState(
  board: CopyHallBoardKind,
): Promise<AdminCopyHallRankingsResult> {
  const query = new URLSearchParams({ board })
  return requestApi<AdminCopyHallRankingsResult>(`/admin/copy-hall/rankings?${query}`)
}

export async function saveCopyHallBoard(
  boardKind: CopyHallBoardKind,
  slots: CopyHallRankSlot[],
): Promise<void> {
  await requestApi(`/admin/copy-hall/boards/${encodeURIComponent(boardKind)}`, {
    method: 'PUT',
    body: { slots },
  })
}
