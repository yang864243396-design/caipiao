import { describe, expect, it } from 'vitest'
import {
  validateGroupContent,
  zuxuanPoolMinPick,
  zuxuanPoolMinPickMessage,
  type PlayConfig,
} from './betPayload'

function cfg(partial: Partial<PlayConfig> & Pick<PlayConfig, 'betMode' | 'subPlayId'>): PlayConfig {
  return {
    playTemplate: 'ssc_std',
    playTypeId: 'g002',
    catalogSubId: '',
    segmentLen: 1,
    segmentLabels: [],
    ...partial,
  } as PlayConfig
}

describe('zuxuanPoolMinPick', () => {
  it('组三最少 2、组六最少 3', () => {
    expect(zuxuanPoolMinPick(cfg({ betMode: 'zu3', subPlayId: '19' }))).toBe(2)
    expect(zuxuanPoolMinPick(cfg({ betMode: 'zu6', subPlayId: '261' }))).toBe(3)
    expect(zuxuanPoolMinPickMessage(cfg({ betMode: 'zu3', subPlayId: '19' }))).toBe(
      '组三至少选择 2 个号码',
    )
    expect(zuxuanPoolMinPickMessage(cfg({ betMode: 'zu6', subPlayId: '261' }))).toBe(
      '组六至少选择 3 个号码',
    )
  })

  it('validateGroupContent 拦不足选号', () => {
    const zu3 = cfg({ betMode: 'zu3', subPlayId: '19', playMethodLabel: '中三组三' })
    const zu6 = cfg({ betMode: 'zu6', subPlayId: '261', playMethodLabel: '中三组六' })
    expect(validateGroupContent(zu3, '1').ok).toBe(false)
    expect(validateGroupContent(zu3, '1,2').ok).toBe(true)
    expect(validateGroupContent(zu6, '1,2').ok).toBe(false)
    expect(validateGroupContent(zu6, '1,2,6').ok).toBe(true)
  })

  it('中三组选包胆：不套组选下限，仅允许单胆', () => {
    const bd = cfg({
      betMode: 'baodan',
      subPlayId: '263',
      catalogSubId: 'zhong3_zuxuan_bd',
      playMethodLabel: '中三组选包胆',
      segmentLen: 3,
      poolMaxPicks: 1,
    })
    expect(zuxuanPoolMinPick(bd)).toBeNull()
    expect(validateGroupContent(bd, '5').ok).toBe(true)
    const multi = validateGroupContent(bd, '5,6')
    expect(multi.ok).toBe(false)
    if (!multi.ok) expect(multi.message).toMatch(/只能选择一个/)
  })
})
