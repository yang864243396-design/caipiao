declare module '@client/utils/betPayload' {
  export interface PlayConfig {
    playTypeId: string
    subPlayId: string
    segmentLen: number
    segmentLabels: string[]
    inputMode: string
    betMode?: string
    catalogSubId?: string
    numberPoolMin?: number
    numberPoolMax?: number
    playTypeLabel?: string
    playMethodLabel?: string
    playTemplate?: string
    guajiGroup?: string
  }

  export interface SchemeGroupsValidation {
    ok: boolean
    message: string
    invalidIndexes: number[]
    normalized: string[]
  }

  export function validateSchemeGroups(config: PlayConfig, groups: string[]): SchemeGroupsValidation
  export function countBetUnits(config: PlayConfig, groupContent: string): number
  export function groupContentPlaceholder(config: PlayConfig): string
  export function resolvePlayConfig(input: { playTypeId?: string; subPlayId?: string }): PlayConfig
  export function buildGroupContent(
    config: PlayConfig,
    picks: { digits: string[]; lines: string[][] },
  ): string
  export function parseGroupPicks(
    config: PlayConfig,
    content: string,
  ): { digits: string[]; lines: string[][] }
}

declare module '@client/constants/lhcPlay' {
  export const LHC_NUMBERS: readonly string[]
  export const LHC_TAIL_OPTIONS: readonly string[]
  export const LHC_ZODIACS: readonly string[]
  export function lhcAttrOptions(betMode: string, mode: string): readonly string[]
}

declare module '@client/utils/pickPanelOptions' {
  import type { PlayConfig } from '@client/utils/betPayload'
  export function schemeGroupUsesPickPanel(config: PlayConfig): boolean
  export function digitOptionsForConfig(config: PlayConfig): string[]
  export function textPickOptionsForConfig(config: PlayConfig): string[]
  export function useCompactPickChips(config: PlayConfig): boolean
}

declare module '@client/utils/playConfig' {
  import type { PlayConfig } from '@client/utils/betPayload'

  export function defaultPlaySelection(tree: unknown): { typeId: string; subId: string }
  export function findSubPlay(
    tree: unknown,
    typeId: string,
    subId: string,
  ): { typeNode: unknown; subNode: unknown } | null
  export function resolvePlayConfigFromTree(
    playTemplate: string,
    typeNode: unknown,
    subNode: unknown,
  ): PlayConfig
}

declare module '@client/utils/betMultiplierPlan' {
  export interface PlanTableRow {
    period: string
    mult: string
    curBet: string
    totalBet: string
    prize: string
    profit: string
    margin: string
  }

  export type CalcType = 'rate' | 'fixed' | 'step' | 'free'
  export type AdvanceMode = 'on_lose' | 'on_win'

  export const DEFAULT_SIDES_PRESET: readonly number[]
  export const AGGRESSIVE_PRESET: readonly number[]

  export function canGenerateNewbiePlan(input: Record<string, unknown>): string | null
  export function canGenerateOneclickPlan(input: Record<string, unknown>): string | null
  export function generateNewbiePlan(input: Record<string, unknown>): PlanTableRow[] | null
  export function generateOneclickPlan(input: Record<string, unknown>): PlanTableRow[] | null
  export function applyPresetTimes(
    preset: readonly number[],
    money: number,
    number: number,
    mode: number,
  ): PlanTableRow[]
}
