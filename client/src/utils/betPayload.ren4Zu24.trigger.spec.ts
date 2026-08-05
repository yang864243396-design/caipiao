import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  buildRenxuanPositionContent,
  countBetUnits,
  validateGroupContent,
} from './betPayload'
import {
  isRenxuanPerPosTriggerPlay,
  supportsAdvTriggerPerPosColumns,
} from './runTypeMatrix'

const ren4Zu24 = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '143',
  catalogSubId: '143',
  betMode: 'zu24',
  playMethodLabel: '任四组选24',
  inputMode: 'pool',
  segmentLen: 1,
  segmentLabels: ['选号'],
  renPositionCount: 4,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('任四组选24 · 开某投某正反投', () => {
  it('不按位分列（整行号池）', () => {
    expect(isRenxuanPerPosTriggerPlay(ren4Zu24)).toBe(false)
    expect(supportsAdvTriggerPerPosColumns(ren4Zu24)).toBe(false)
  })

  it('正/反投 1,2,3,4 补选位后应通过', () => {
    const cell = '1,2,3,4'
    const wrapped = buildRenxuanPositionContent(['万', '千', '百', '十'], cell)
    const r = validateGroupContent(ren4Zu24, wrapped)
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBeGreaterThan(0)
    expect(countBetUnits(ren4Zu24, wrapped)).toBe(1)
  })

  it('未补选位的裸号池也至少 4 码通过内层计注口径', () => {
    // 开某投某格子本身无位名；若未 wrap 会在选位校验处失败，但不应误报「至少 4 码」
    const r = validateGroupContent(ren4Zu24, '1,2,3,4')
    expect(r.ok).toBe(false)
    if (!r.ok) {
      expect(r.message).not.toContain('组选24至少选择')
      expect(r.message).toMatch(/勾选|位置/)
    }
  })
})
