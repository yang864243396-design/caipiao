import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'

import {

  COPY_HALL_LOTTERIES,

  type CopyHallBoardKind,

  type CopyHallRankSlot,

} from '@shared/mock/copyHallRankings'



export interface CopyHallRankingsResult {

  lotteryCode: string

  board: CopyHallBoardKind

  slots: CopyHallRankSlot[]

}



/** @deprecated 优先使用 usePublicLotteries().labelToCode */
export function lotteryLabelToCode(label: string): string | undefined {
  return COPY_HALL_LOTTERIES.find((l) => l.label === label)?.code
}



export async function fetchCopyHallRankings(

  lotteryCode: string,

  board: CopyHallBoardKind,

): Promise<CopyHallRankingsResult> {

  await ensureClientSession()

  return requestApi<CopyHallRankingsResult>('/client/copy-hall/rankings', {

    query: { lotteryCode, board },

  })

}

