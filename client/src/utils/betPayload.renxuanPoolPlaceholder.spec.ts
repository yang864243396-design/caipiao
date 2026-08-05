import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { renxuanPositionPanelPlaceholder } from './betPayload'

function base(partial: Partial<PlayConfig> & Pick<PlayConfig, 'betMode' | 'playMethodLabel' | 'renPositionCount'>): PlayConfig {
  return {
    playTemplate: 'ssc_std',
    playTypeId: 'renxuan',
    playTypeLabel: '任选',
    guajiGroup: '任选',
    subPlayId: 'x',
    catalogSubId: 'x',
    inputMode: 'pool',
    segmentLen: 1,
    segmentLabels: ['选号'],
    numberPoolMin: 0,
    numberPoolMax: 9,
    ...partial,
  } as PlayConfig
}

describe('任选选位号池 placeholder 按玩法区分', () => {
  it('任二组选复式：至少 2 码', () => {
    const tip = renxuanPositionPanelPlaceholder(
      base({ betMode: 'zuxuan_fs', playMethodLabel: '任二组选复式', renPositionCount: 2, subPlayId: 'ren2_zuxuan_fs' }),
    )
    expect(tip).toBe(
      '从万、千、百、十、个中勾选至少 2 个、最多 5 个位置，再输入两个及以上的0-9的号码，多选用逗号分隔，如1,2',
    )
    expect(tip).not.toMatch(/C\(选位数/)
    expect(tip).not.toMatch(/再选择\/输入号码/)
  })

  it('任三组三复式：至少 2 码', () => {
    const tip = renxuanPositionPanelPlaceholder(
      base({ betMode: 'zu3', playMethodLabel: '任三组三复式', renPositionCount: 3, subPlayId: 'ren3_zu3_fs' }),
    )
    expect(tip).toContain('勾选至少 3 个、最多 5 个位置')
    expect(tip).toContain('再输入两个及以上0-9的号码')
  })

  it('任三组六复式：至少 3 码', () => {
    const tip = renxuanPositionPanelPlaceholder(
      base({ betMode: 'zu6', playMethodLabel: '任三组六复式', renPositionCount: 3, subPlayId: 'ren3_zu6_fs' }),
    )
    expect(tip).toContain('再输入三个及以上0-9的号码')
  })

  it('任二直选和值：和值区间', () => {
    const tip = renxuanPositionPanelPlaceholder(
      base({
        betMode: 'hezhi',
        playMethodLabel: '任二直选和值',
        renPositionCount: 2,
        numberPoolMin: 0,
        numberPoolMax: 18,
        subPlayId: 'ren2_zhixuan_hz',
      }),
    )
    expect(tip).toBe(
      '从万、千、百、十、个中勾选至少 2 个、最多 5 个位置，再输入和值 0–18，多选用逗号分隔（如 14,15,16）',
    )
  })

  it('任四组选6：至少 2 码', () => {
    const tip = renxuanPositionPanelPlaceholder(
      base({
        betMode: 'zu6',
        playMethodLabel: '任选四组选6',
        renPositionCount: 4,
        catalogSubId: '145',
        subPlayId: '145',
      }),
    )
    expect(tip).toContain('勾选至少 4 个、最多 5 个位置')
    expect(tip).toContain('再输入两个及以上的0-9的号码')
  })
})
