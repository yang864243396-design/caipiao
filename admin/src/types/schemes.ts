export interface SchemeOptionItem {
  label: string
  value: string
}

export const SCHEME_LOTTERY_OPTIONS: SchemeOptionItem[] = [
  { label: '波场1分彩', value: 'tron_ffc_1m' },
  { label: '哈希1分彩', value: 'hash_ffc_1m' },
  { label: '以太坊1分彩', value: 'eth_ffc_1m' },
  { label: '币安1分彩', value: 'bnb_ffc_1m' },
  { label: '波场极速彩', value: 'tron_jisu' },
  { label: '波场11选5', value: 'tron_syxw' },
]

export const SCHEME_RUN_TYPE_OPTIONS: SchemeOptionItem[] = [
  { label: '定码轮换', value: 'fixed_rotate' },
  { label: '高级定码轮换', value: 'adv_fixed_rotate' },
  { label: '高级开某投某', value: 'adv_trigger_bet' },
  { label: '冷热出号', value: 'hot_cold_warm' },
  { label: '随机出号', value: 'random_draw' },
  { label: '内置计划', value: 'builtin_plan' },
  { label: '固定号码', value: 'fixed_number' },
]

/** 废弃运行类型：存量数据已由后端迁移映射到 adv_fixed_rotate，仅用于旧值展示兜底 */
export const SCHEME_RUN_TYPE_LEGACY_LABELS: Record<string, string> = {
  batch_fixed: '高级定码轮换(迁移)',
  dynamic_chase: '高级定码轮换(迁移)',
  plan_follow: '高级定码轮换(迁移)',
}

export function labelOfRunType(value: string): string {
  return (
    SCHEME_RUN_TYPE_OPTIONS.find((o) => o.value === value)?.label ??
    SCHEME_RUN_TYPE_LEGACY_LABELS[value] ??
    value
  )
}

export const SCHEME_PLAY_TYPE_OPTIONS: SchemeOptionItem[] = [
  { label: '定位胆', value: 'dingwei' },
  { label: '前三', value: 'qian3' },
  { label: '中三', value: 'zhong3' },
  { label: '后三', value: 'hou3' },
  { label: '任选', value: 'renxuan' },
]

export const SCHEME_SUB_PLAY_OPTIONS: SchemeOptionItem[] = [
  { label: '万位', value: 'dingwei_wan' },
  { label: '前三直选复式', value: 'qian3_zhixuan_fs' },
  { label: '中三组六', value: 'zhong3_zu6' },
  { label: '任二直选复式', value: 'ren2_zhixuan_fs' },
]

export function labelOfOption(list: SchemeOptionItem[], value: string): string {
  return list.find((o) => o.value === value)?.label ?? value
}

export interface SchemeCustomSettings {
  schemeName: string
  lotteryId: string
  runTypeId: string
  playTypeId: string
  subPlayId: string
}

export const SCHEME_SETTING_FIELD_LABELS: Record<keyof SchemeCustomSettings, string> = {
  schemeName: '方案名称',
  lotteryId: '彩种',
  runTypeId: '运行类型',
  playTypeId: '玩法类型',
  subPlayId: '子玩法',
}

export type SchemeKind = '自创' | '反买' | '跟单'
export type SchemeInstanceStatus = '待开启' | '运行中' | '已暂停' | '已封停'

export function schemeSimBetLabel(simBet: boolean): string {
  return simBet ? '模拟' : '正式'
}

export interface SchemeInstanceRow {
  id: string
  memberId: string
  memberName: string
  kind: SchemeKind
  lotteryCode: string
  lotteryLabel: string
  refId: string
  status: SchemeInstanceStatus
  /** false=正式盘，true=模拟盘 */
  simBet: boolean
  /** 运行类型展示名（仅自创方案，后端下发） */
  runTypeLabel?: string
  /** 玩法类型展示名（后端下发） */
  playTypeLabel?: string
  createdAt: string
  settings: SchemeCustomSettings
  updatedAt: string
}

export interface SchemeChangeLogRow {
  id: string
  schemeInstanceId: string
  field: keyof SchemeCustomSettings
  fieldLabel: string
  oldValue: string
  newValue: string
  changedAt: string
  operator: string
}

export type SchemeShareSnapshotKind = '自创'

export interface SchemeShareSnapshotRow {
  id: string
  kind: SchemeShareSnapshotKind
  schemeName: string
  lotteryCode: string
  lotteryLabel: string
  playMethod?: string
  fundYuan?: number
  config?: Record<string, unknown>
  settings: SchemeCustomSettings
  publishedAt: string
  updatedAt: string
}

export interface SchemeBetExecutionRow {
  id: string
  schemeInstanceId: string
  time: string
  betAt: string
  schemeName: string
  numbers: string
  period: string
  draw: string
  win: boolean
}

export interface SchemePlanTrendRow {
  id: string
  schemeInstanceId: string
  period: string
  win: boolean
}

export interface SchemeDrawHistoryRow {
  id: string
  schemeInstanceId: string
  periodShort: string
  time: string
  balls: string[]
  sum: number
}

export interface SchemeBetRecordRow {
  id: string
  schemeInstanceId: string
  period: string
  betAt: string
  playMethod: string
  multiplier: string
  round: string
  amount: string
  profitLoss: number
  status: string
}

/** 投注执行 + 投注记录合并行 */
export interface SchemeBetHistoryItem {
  id: string
  schemeInstanceId: string
  time: string
  betAt: string
  schemeName: string
  numbers: string
  period: string
  draw: string
  playMethod: string
  multiplier: string
  round: string
  amount: string
  profitLoss: number
  status: string
  /** 已结算为 true/false；待开奖/撤单为 null */
  win: boolean | null
}
