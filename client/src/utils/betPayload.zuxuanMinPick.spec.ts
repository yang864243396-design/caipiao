import { describe, expect, it } from 'vitest'
import {
  isSixingZu6PlayConfig,
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

  it('四星/任四组选6：最少 2 码（区别于三星组六）', () => {
    const sixing = cfg({
      betMode: 'zu6',
      subPlayId: '145',
      catalogSubId: '145',
      playMethodLabel: '任选四组选6',
      playTypeId: 'g011',
      renPositionCount: 4,
    })
    expect(isSixingZu6PlayConfig(sixing)).toBe(true)
    expect(zuxuanPoolMinPick(sixing)).toBe(2)
    expect(zuxuanPoolMinPickMessage(sixing)).toBe('组选6至少选择 2 个号码')
    expect(validateGroupContent(sixing, '万,千,百,十\n1,2').ok).toBe(true)
    expect(validateGroupContent(sixing, '万,千,百,十\n1').ok).toBe(false)
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
