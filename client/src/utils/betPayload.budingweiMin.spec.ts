import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  budingweiMinPicks,
  budingweiMinPicksMessage,
  validateGroupContent,
} from './betPayload'

const qian3Erma = {
  playTemplate: 'ssc_std',
  playTypeId: 'g009',
  playTypeLabel: '不定位',
  guajiGroup: '不定位',
  subPlayId: '114',
  catalogSubId: '114',
  betMode: 'budingwei',
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'pool',
  playMethodLabel: '前三二码不定位',
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('前三二码不定位最少选号', () => {
  it('最少 2 个号', () => {
    expect(budingweiMinPicks(qian3Erma)).toBe(2)
    expect(budingweiMinPicksMessage(qian3Erma)).toBe('投注数字不能低于两个')
  })

  it('仅 1 个号保存失败', () => {
    const r = validateGroupContent(qian3Erma, '5')
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.message).toBe('投注数字不能低于两个')
  })

  it('2 个号保存通过且为 1 注', () => {
    const r = validateGroupContent(qian3Erma, '1,2')
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.betUnits).toBe(1)
      expect(r.normalized).toBe('1,2')
    }
  })

  it('3 个号为 C(3,2)=3 注', () => {
    const r = validateGroupContent(qian3Erma, '1,2,3')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(3)
  })
})

const wuxingErma = {
  ...qian3Erma,
  subPlayId: '151',
  catalogSubId: '151',
  playMethodLabel: '五星二码不定位',
} as PlayConfig

describe('五星二码不定位最少选号', () => {
  it('最少 4 个号（含仅目录 id）', () => {
    expect(budingweiMinPicks(wuxingErma)).toBe(4)
    expect(budingweiMinPicks({ ...wuxingErma, playMethodLabel: '不定位' })).toBe(4)
    expect(budingweiMinPicksMessage(wuxingErma)).toBe('五星二码不定位：至少选择 4 个号码')
  })

  it('少于 4 个号保存失败', () => {
    expect(validateGroupContent(wuxingErma, '1,2').ok).toBe(false)
    const r = validateGroupContent(wuxingErma, '1,2,3')
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.message).toBe('五星二码不定位：至少选择 4 个号码')
  })

  it('4 个号为 C(4,2)=6 注', () => {
    const r = validateGroupContent(wuxingErma, '1,2,3,4')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(6)
  })
})
