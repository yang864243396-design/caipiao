import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { countBetUnits, validateGroupContent } from './betPayload'
import {
  poolMaxPicksForConfig,
  textPickOptionsForConfig,
  togglePoolPick,
} from './pickPanelOptions'

const luckyZhuangXian = {
  playTemplate: 'fast_ssc_std',
  playTypeId: 'g017',
  catalogSubId: '388',
  playTypeLabel: '哈希玩法',
  playMethodLabel: '幸运庄闲',
  betMode: 'zhuangxian',
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'pool',
} as PlayConfig

describe('幸运庄闲：庄、和、闲单选', () => {
  it('“和”可选，且新选择会替换旧选择', () => {
    expect(textPickOptionsForConfig(luckyZhuangXian)).toEqual(['庄', '和', '闲'])
    expect(poolMaxPicksForConfig(luckyZhuangXian)).toBe(1)
    expect(togglePoolPick(['庄'], '和', poolMaxPicksForConfig(luckyZhuangXian))).toEqual(['和'])
  })

  it('“和”是一注，多个选项会被校验拒绝', () => {
    expect(validateGroupContent(luckyZhuangXian, '和')).toMatchObject({
      ok: true,
      normalized: '和',
      betUnits: 1,
    })
    expect(countBetUnits(luckyZhuangXian, '和')).toBe(1)
    expect(validateGroupContent(luckyZhuangXian, '庄,和')).toMatchObject({ ok: false })
  })
})
