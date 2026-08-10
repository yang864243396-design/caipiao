import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { countBetUnits, validateGroupContent, buildGroupContent, parseGroupPicks } from './betPayload'
import {
  poolMaxPicksForConfig,
  schemeGroupUsesPickPanel,
  schemeGroupUsesDigitInput,
  lhcZodiacChipSub,
  togglePoolPick,
} from './pickPanelOptions'
import { isLhcSxDuipengConfig } from '@/constants/lhcPlay'

const sxDpCfg = {
  playTemplate: 'lhc_std',
  playTypeId: 'g003',
  catalogSubId: '281',
  subPlayId: 'sx_dp',
  betMode: 'sx_dp',
  playTypeLabel: '连码',
  playMethodLabel: '二全中生肖对碰',
  guajiGroup: '二全中',
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'lhc_zodiac',
  numberPoolMin: 1,
  numberPoolMax: 49,
  poolMaxPicks: 2,
} as PlayConfig

describe('二全中生肖对碰', () => {
  it('识别为生肖对碰，走点选面板而非 01–49 输入框', () => {
    expect(isLhcSxDuipengConfig(sxDpCfg)).toBe(true)
    expect(schemeGroupUsesPickPanel(sxDpCfg)).toBe(true)
    expect(schemeGroupUsesDigitInput(sxDpCfg)).toBe(false)
    expect(poolMaxPicksForConfig(sxDpCfg)).toBe(2)
  })

  it('chip 副文案含固定号码', () => {
    expect(lhcZodiacChipSub('马')).toBe('01,13,25,37,49')
    expect(lhcZodiacChipSub('蛇')).toBe('02,14,26,38')
  })

  it('最多选 2 个，保选择序', () => {
    let sel = togglePoolPick([], '马', 2)
    sel = togglePoolPick(sel, '蛇', 2)
    expect(sel).toEqual(['马', '蛇'])
    const blocked = togglePoolPick(sel, '龙', 2)
    expect(blocked).toEqual(['马', '蛇'])
  })

  it('落库/校验为 肖A|肖B；带马 20 注、不带马 16 注', () => {
    expect(buildGroupContent(sxDpCfg, { digits: ['马', '蛇'] })).toBe('马|蛇')
    expect(parseGroupPicks(sxDpCfg, '马|蛇').digits).toEqual(['马', '蛇'])
    const withHorse = validateGroupContent(sxDpCfg, '马,蛇')
    expect(withHorse.ok).toBe(true)
    if (withHorse.ok) {
      expect(withHorse.normalized).toBe('马|蛇')
      expect(withHorse.betUnits).toBe(20) // 5×4
    }
    const noHorse = validateGroupContent(sxDpCfg, '蛇|龙')
    expect(noHorse.ok).toBe(true)
    if (noHorse.ok) {
      expect(noHorse.betUnits).toBe(16) // 4×4
    }
    expect(validateGroupContent(sxDpCfg, '马').ok).toBe(false)
    expect(countBetUnits(sxDpCfg, '马|蛇')).toBe(20)
    expect(countBetUnits(sxDpCfg, '蛇|龙')).toBe(16)
  })
})
