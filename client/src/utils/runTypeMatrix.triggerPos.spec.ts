import { describe, expect, it } from 'vitest'
import { isZhixuanDanshiPerPosPlay, supportsAdvTriggerPerPosColumns } from './runTypeMatrix'

describe('supportsAdvTriggerPerPosColumns', () => {
  it('中三直选单式按千/百/十三位分列', () => {
    expect(
      supportsAdvTriggerPerPosColumns({
        betMode: 'danshi',
        playTypeId: 'g002',
        subPlayId: '2',
        playMethodLabel: '直选单式',
        playTypeLabel: '中三',
        inputMode: 'danshi',
        segmentLen: 3,
        segmentLabels: ['千', '百', '十'],
      }),
    ).toBe(true)
  })

  it('中三直选单式无位标签仍按段长分列（随机出号）', () => {
    expect(
      isZhixuanDanshiPerPosPlay({
        betMode: 'danshi',
        playTypeId: 'g002',
        playMethodLabel: '直选单式',
        segmentLen: 3,
      }),
    ).toBe(true)
  })

  it('任选单式（仅选号）不分列', () => {
    expect(
      supportsAdvTriggerPerPosColumns({
        betMode: 'danshi',
        playTypeId: 'g011',
        playMethodLabel: '任二直选单式',
        playTypeLabel: '任选',
        inputMode: 'danshi',
        segmentLen: 2,
        segmentLabels: ['选号'],
      }),
    ).toBe(false)
  })

  it('前三直选复式仍按位分列', () => {
    expect(
      supportsAdvTriggerPerPosColumns({
        betMode: 'fushi',
        playTypeId: 'g001',
        subPlayId: 'zhixuan_fs',
        playMethodLabel: '直选复式',
        inputMode: 'multiline',
        segmentLen: 3,
        segmentLabels: ['万', '千', '百'],
      }),
    ).toBe(true)
  })

  it('中三混合组选与直选复式同按位分列（千/百/十）', () => {
    expect(
      supportsAdvTriggerPerPosColumns({
        betMode: 'hunhe',
        playTypeId: 'g002',
        subPlayId: '23',
        playMethodLabel: '混合组选',
        playTypeLabel: '中三',
        inputMode: 'danshi',
        segmentLen: 3,
        segmentLabels: ['千', '百', '十'],
      }),
    ).toBe(true)
  })
})
