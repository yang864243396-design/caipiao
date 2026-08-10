import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { countBetUnits, validateGroupContent, buildGroupContent, parseGroupPicks } from './betPayload'
import {
  poolMaxPicksForConfig,
  schemeGroupUsesPickPanel,
  schemeGroupUsesDigitInput,
  lhcZodiacChipSub,
  lhcTailChipLabel,
  lhcTailChipSub,
  toggleSwDuipengPick,
  textPickOptionsForConfig,
} from './pickPanelOptions'
import { isLhcSwDuipengConfig, pickRandomLhcSwDuipengPair, LHC_ZODIACS, LHC_TAIL_OPTIONS } from '@/constants/lhcPlay'

const swDpCfg = {
  playTemplate: 'lhc_std',
  playTypeId: 'g003',
  catalogSubId: '283',
  subPlayId: 'sw_dp',
  betMode: 'sw_dp',
  playTypeLabel: '连码',
  playMethodLabel: '二全中生尾对碰',
  guajiGroup: '二全中',
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'lhc_attr',
  numberPoolMin: 1,
  numberPoolMax: 49,
  poolMaxPicks: 2,
} as PlayConfig

describe('二全中生尾对碰', () => {
  it('识别为生尾对碰，走点选面板', () => {
    expect(isLhcSwDuipengConfig(swDpCfg)).toBe(true)
    expect(schemeGroupUsesPickPanel(swDpCfg)).toBe(true)
    expect(schemeGroupUsesDigitInput(swDpCfg)).toBe(false)
    expect(poolMaxPicksForConfig(swDpCfg)).toBe(2)
    expect(textPickOptionsForConfig(swDpCfg)).toHaveLength(22)
  })

  it('chip 文案：生肖号码 + 尾数 N尾', () => {
    expect(lhcZodiacChipSub('马')).toBe('01,13,25,37,49')
    expect(lhcTailChipLabel('0')).toBe('0尾')
    expect(lhcTailChipSub('0')).toBe('10,20,30,40')
  })

  it('点选：肖/尾各最多 1，点同侧替换', () => {
    let sel = toggleSwDuipengPick([], '马')
    sel = toggleSwDuipengPick(sel, '0')
    expect(sel).toEqual(['马', '0'])
    sel = toggleSwDuipengPick(sel, '蛇')
    expect(sel).toEqual(['蛇', '0'])
    sel = toggleSwDuipengPick(sel, '1')
    expect(sel).toEqual(['蛇', '1'])
    sel = toggleSwDuipengPick(sel, '蛇')
    expect(sel).toEqual(['1'])
  })

  it('betMode=sw_dp 时二中特/特串也识别（勿卡 typeId=g003）', () => {
    expect(
      isLhcSwDuipengConfig({
        playTemplate: 'lhc_std',
        betMode: 'sw_dp',
        playTypeId: 'g004',
        playTypeLabel: '二中特',
        playMethodLabel: '生尾对碰',
      }),
    ).toBe(true)
    expect(
      isLhcSwDuipengConfig({
        playTemplate: 'lhc_std',
        betMode: 'sw_dp',
        playTypeId: 'g005',
        playTypeLabel: '特串',
        playMethodLabel: '生尾对碰',
      }),
    ).toBe(true)
  })

  it('随机出号固定 1 肖 + 1 尾', () => {
    for (let i = 0; i < 40; i++) {
      const [z, t] = pickRandomLhcSwDuipengPair()
      expect(LHC_ZODIACS).toContain(z)
      expect(LHC_TAIL_OPTIONS).toContain(t)
    }
  })

  it('落库/校验为 肖|尾；注数扣共有号码（马×0=20、鼠×0=16、马×1=24、狗×5=19）', () => {
    expect(buildGroupContent(swDpCfg, { digits: ['马', '0'] })).toBe('马|0')
    expect(parseGroupPicks(swDpCfg, '马|0').digits).toEqual(['马', '0'])
    expect(parseGroupPicks(swDpCfg, '0|马').digits).toEqual(['马', '0'])
    const ok = validateGroupContent(swDpCfg, '马,0')
    expect(ok.ok).toBe(true)
    if (ok.ok) {
      expect(ok.normalized).toBe('马|0')
      expect(ok.betUnits).toBe(20)
    }
    expect(countBetUnits(swDpCfg, '鼠|0')).toBe(16)
    expect(countBetUnits(swDpCfg, '马|1')).toBe(24)
    expect(countBetUnits(swDpCfg, '狗|5')).toBe(19)
    expect(validateGroupContent(swDpCfg, '马').ok).toBe(false)
    expect(validateGroupContent(swDpCfg, '马|蛇').ok).toBe(false)
    expect(validateGroupContent(swDpCfg, '0|1').ok).toBe(false)
  })
})
