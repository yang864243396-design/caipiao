/** 旧 9 彩种 purge 键（与 backend LegacyLotteryCodes 一致） */
export const LEGACY_LOTTERY_CODES = [
  'tencent_ffc',
  'tencent_10',
  'qiqu_tencent',
  'us_ffc',
  'cq_ssc',
  'xj_ssc',
  'tj_ssc',
  'fc_3d',
  'pl3',
] as const

export function isLegacyLotteryCode(code: string): boolean {
  const c = code.trim()
  return LEGACY_LOTTERY_CODES.includes(c as (typeof LEGACY_LOTTERY_CODES)[number])
}
