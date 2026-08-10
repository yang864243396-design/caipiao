import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { countBetUnits, validateGroupContent, buildGroupContent, parseGroupPicks } from './betPayload'
import {
  poolMaxPicksForConfig,
  schemeGroupUsesPickPanel,
  schemeGroupUsesDigitInput,
  lhcTailChipLabel,
  lhcTailChipSub,
  togglePoolPick,
} from './pickPanelOptions'
import { isLhcWsDuipengConfig, LHC_TAIL_NUMBERS } from '@/constants/lhcPlay'

const wsDpCfg = {
  playTemplate: 'lhc_std',
  playTypeId: 'g003',
  catalogSubId: '282',
  subPlayId: 'ws_dp',
  betMode: 'ws_dp',
  playTypeLabel: '连码',
  playMethodLabel: '二全中尾数对碰',
  guajiGroup: '二全中',
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'lhc_tail',
  numberPoolMin: 1,
  numberPoolMax: 49,
  poolMaxPicks: 2,
} as PlayConfig

describe('二全中尾数对碰', () => {
  it('识别为尾数对碰，走点选面板而非 01–49 输入框', () => {
    expect(isLhcWsDuipengConfig(wsDpCfg)).toBe(true)
    expect(schemeGroupUsesPickPanel(wsDpCfg)).toBe(true)
    expect(schemeGroupUsesDigitInput(wsDpCfg)).toBe(false)
    expect(poolMaxPicksForConfig(wsDpCfg)).toBe(2)
  })

  it('chip 主文案为 N尾，副文案含固定号码', () => {
    expect(lhcTailChipLabel('0')).toBe('0尾')
    expect(lhcTailChipLabel('1')).toBe('1尾')
    expect(lhcTailChipLabel('0尾')).toBe('0尾')
    expect(lhcTailChipSub('0')).toBe('10,20,30,40')
    expect(lhcTailChipSub('1')).toBe('01,11,21,31,41')
    expect(LHC_TAIL_NUMBERS['0']).toEqual(['10', '20', '30', '40'])
  })

  it('最多选 2 个，保选择序', () => {
    let sel = togglePoolPick([], '0', 2)
    sel = togglePoolPick(sel, '1', 2)
    expect(sel).toEqual(['0', '1'])
    const blocked = togglePoolPick(sel, '2', 2)
    expect(blocked).toEqual(['0', '1'])
  })

  it('落库/校验为 尾A|尾B；0×1=20 注、1×2=25 注', () => {
    expect(buildGroupContent(wsDpCfg, { digits: ['0', '1'] })).toBe('0|1')
    expect(parseGroupPicks(wsDpCfg, '0|1').digits).toEqual(['0', '1'])
    const withZero = validateGroupContent(wsDpCfg, '0,1')
    expect(withZero.ok).toBe(true)
    if (withZero.ok) {
      expect(withZero.normalized).toBe('0|1')
      expect(withZero.betUnits).toBe(20)
    }
    const noZero = validateGroupContent(wsDpCfg, '1|2')
    expect(noZero.ok).toBe(true)
    if (noZero.ok) {
      expect(noZero.betUnits).toBe(25)
    }
    expect(validateGroupContent(wsDpCfg, '0').ok).toBe(false)
    expect(countBetUnits(wsDpCfg, '0|1')).toBe(20)
    expect(countBetUnits(wsDpCfg, '1|2')).toBe(25)
  })
})
