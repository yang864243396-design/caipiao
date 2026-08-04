import { describe, expect, it } from 'vitest'
import {
  defaultRenxuanHcwOpenPositionIdxs,
  defaultRenxuanTriggerPositionIdxs,
  isRenxuanZhixuanDanshiTriggerPlay,
  isZhixuanDanshiPerPosPlay,
  supportsAdvTriggerPerPosColumns,
  supportsAdvTriggerPositionPicker,
} from './runTypeMatrix'

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

  it('任选直选单式显示选位芯片，并按投注选位分列正反投（默认投注万千）', () => {
    const cfg = {
      betMode: 'danshi',
      playTypeId: 'g011',
      playMethodLabel: '任二直选单式',
      playTypeLabel: '任选',
      guajiGroup: '任选',
      inputMode: 'danshi' as const,
      segmentLen: 2,
      segmentLabels: ['选号'],
    }
    expect(isRenxuanZhixuanDanshiTriggerPlay(cfg)).toBe(true)
    expect(supportsAdvTriggerPerPosColumns(cfg)).toBe(true)
    expect(supportsAdvTriggerPositionPicker(cfg)).toBe(true)
    expect(defaultRenxuanTriggerPositionIdxs(2)).toEqual([0, 1])
  })

  it('任选组选复式须选位芯片，但不按位分列正反投', () => {
    const cfg = {
      betMode: 'zuxuan_fs',
      playTypeId: 'g011',
      playMethodLabel: '任二组选复式',
      playTypeLabel: '任选',
      guajiGroup: '任选',
      inputMode: 'pool' as const,
      segmentLen: 2,
      segmentLabels: ['选号'],
    }
    expect(supportsAdvTriggerPositionPicker(cfg)).toBe(true)
    expect(supportsAdvTriggerPerPosColumns(cfg)).toBe(false)
    expect(isRenxuanZhixuanDanshiTriggerPlay(cfg)).toBe(false)
  })

  it('任选直选复式不显示选位芯片', () => {
    const cfg = {
      betMode: 'fushi',
      playTypeId: 'g011',
      playMethodLabel: '任二直选复式',
      playTypeLabel: '任选',
      guajiGroup: '任选',
      inputMode: 'multiline' as const,
      segmentLen: 2,
      segmentLabels: ['万', '千', '百', '十', '个'],
    }
    expect(supportsAdvTriggerPositionPicker(cfg)).toBe(false)
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

  it('任选冷热开奖选位默认前 k 位（任二万千）', () => {
    expect(defaultRenxuanHcwOpenPositionIdxs(2)).toEqual([0, 1])
    expect(defaultRenxuanHcwOpenPositionIdxs(3)).toEqual([0, 1, 2])
    expect(defaultRenxuanHcwOpenPositionIdxs(4)).toEqual([0, 1, 2, 3])
    expect(defaultRenxuanTriggerPositionIdxs(3)).toEqual([0, 1, 4])
  })
})
