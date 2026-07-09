import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface SchemeOptionItem {

  value: string

  label: string

}



export interface LotterySchemeOptionsDto {

  lotteryCode: string

  runTypes: SchemeOptionItem[]

  playTypes: SchemeOptionItem[]

  subPlays: SchemeOptionItem[]

}



export async function fetchLotterySchemeOptions(

  lotteryCode: string,

): Promise<LotterySchemeOptionsDto> {

  await ensureClientSession()

  return requestApi<LotterySchemeOptionsDto>(

    `/client/games/${encodeURIComponent(lotteryCode)}/scheme-options`,

  )

}

