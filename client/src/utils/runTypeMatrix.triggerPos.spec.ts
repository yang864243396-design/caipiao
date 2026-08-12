import { describe, expect, it } from 'vitest'
import type { PlayTypeNode } from '@/types/playCatalog'
import {
  defaultRenxuanHcwOpenPositionIdxs,
  defaultRenxuanTriggerPositionIdxs,
  isHcwZu12DualPlay,
  isHcwZu4DualPlay,
  isHcwZuDualPlay,
  isRenxuanHcwZu4Play,
  isRenxuanHcwOpenPosPlay,
  isRenxuanHcwZu12Play,
  isRenxuanPerPosTriggerPlay,
  isRenxuanZhixuanDanshiTriggerPlay,
  isZhixuanDanshiPerPosPlay,
  isTailParityPlayConfig,
  lotteryHasAdvTriggerPlay,
  supportsAdvTriggerBet,
  supportsAdvTriggerPerPosColumns,
  supportsAdvTriggerPositionPicker,
} from './runTypeMatrix'

describe('isTailParityPlayConfig', () => {
  it('recognizes tail parity but not sum parity', () => {
    expect(isTailParityPlayConfig({ playMethodLabel: '尾数单双' })).toBe(true)
    expect(isTailParityPlayConfig({ playMethodLabel: '和值单双' })).toBe(false)
    expect(isTailParityPlayConfig({ playTypeId: 'g017', subPlayId: '267' })).toBe(true)
    expect(isTailParityPlayConfig({ playTypeId: 'g017', subPlayId: '387' })).toBe(true)
  })
})

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

  it('任三混合组选开某投某：一行正/反投整注，不按位分列', () => {
    const cfg = {
      betMode: 'hunhe',
      playTypeId: 'g011',
      playMethodLabel: '任三混合组选',
      playTypeLabel: '任选',
      guajiGroup: '任选',
      inputMode: 'danshi' as const,
      segmentLen: 3,
      segmentLabels: ['选号'],
    }
    expect(supportsAdvTriggerPositionPicker(cfg)).toBe(true)
    expect(supportsAdvTriggerPerPosColumns(cfg)).toBe(false)
    expect(isRenxuanPerPosTriggerPlay(cfg)).toBe(false)
    // 冷热仍启用开奖选位（须选 3），与任三直选单式同口径
    expect(isRenxuanHcwOpenPosPlay(cfg)).toBe(true)
  })

  it('中三混合组选冷热不走任选开奖选位', () => {
    expect(
      isRenxuanHcwOpenPosPlay({
        betMode: 'hunhe',
        playTypeId: 'g002',
        subPlayId: '23',
        playMethodLabel: '混合组选',
        playTypeLabel: '中三',
        segmentLen: 3,
      }),
    ).toBe(false)
  })

  it('任选冷热开奖选位默认前 k 位（任二万千）', () => {
    expect(defaultRenxuanHcwOpenPositionIdxs(2)).toEqual([0, 1])
    expect(defaultRenxuanHcwOpenPositionIdxs(3)).toEqual([0, 1, 2])
    expect(defaultRenxuanHcwOpenPositionIdxs(4)).toEqual([0, 1, 2, 3])
    expect(defaultRenxuanTriggerPositionIdxs(3)).toEqual([0, 1, 4])
  })

  it('四星组选12 冷热走二重/单号双池；任选才带投注选位', () => {
    const sixing = {
      betMode: 'zu12',
      playTypeId: 'sixing',
      playMethodLabel: '组选12',
      playTypeLabel: '四星',
      segmentLen: 4,
      segmentLabels: ['千', '百', '十', '个'],
    }
    expect(isHcwZu12DualPlay(sixing)).toBe(true)
    expect(isRenxuanHcwZu12Play(sixing)).toBe(false)
    const ren4 = {
      betMode: 'zu12',
      playTypeId: 'g011',
      playMethodLabel: '任四组选12',
      renPositionCount: 4,
      segmentLen: 1,
    }
    expect(isHcwZu12DualPlay(ren4)).toBe(true)
    expect(isRenxuanHcwZu12Play(ren4)).toBe(true)
  })

  it('四星组选4 冷热走三重/单号双池；任选才带投注选位', () => {
    const sixing = {
      betMode: 'zu4',
      playTypeId: 'sixing',
      playMethodLabel: '组选4',
      playTypeLabel: '四星',
      segmentLen: 4,
      segmentLabels: ['千', '百', '十', '个'],
    }
    expect(isHcwZu4DualPlay(sixing)).toBe(true)
    expect(isHcwZuDualPlay(sixing)).toBe(true)
    expect(isRenxuanHcwZu4Play(sixing)).toBe(false)
    const ren4 = {
      betMode: 'zu4',
      playTypeId: 'g011',
      playMethodLabel: '任四组选4',
      renPositionCount: 4,
      segmentLen: 1,
    }
    expect(isHcwZu4DualPlay(ren4)).toBe(true)
    expect(isRenxuanHcwZu4Play(ren4)).toBe(true)
  })
})

describe('六合特码高级开某投某', () => {
  it('特码/正特码玩法树应开放高级开某投某运行类型', () => {
    const lhcTree: PlayTypeNode[] = [
      {
        typeId: 'g001',
        label: '特码',
        sortOrder: 1,
        subPlays: [{ subId: '272', label: '特码A', sortOrder: 1, outboundPlayCode: '272' }],
      },
    ]
    expect(lotteryHasAdvTriggerPlay(lhcTree)).toBe(true)
    expect(supportsAdvTriggerBet('g001', '272', '特码', '特码A')).toBe(true)
    expect(supportsAdvTriggerBet('tema', 'tema_a', '特码', '特码A')).toBe(true)
  })

  it('二全中复式应开放高级开某投某', () => {
    expect(supportsAdvTriggerBet('erquanzhong', 'fushi', '二全中', '复式')).toBe(true)
    expect(supportsAdvTriggerBet('g003', '279', '连码', '二全中复式')).toBe(true)
    expect(supportsAdvTriggerBet('g003', '281', '连码', '二全中生肖对碰')).toBe(true)
    expect(supportsAdvTriggerBet('erquanzhong', 'sx_dp', '二全中', '生肖对碰')).toBe(true)
    expect(supportsAdvTriggerBet('g003', '282', '连码', '二全中尾数对碰')).toBe(true)
    expect(supportsAdvTriggerBet('erquanzhong', 'ws_dp', '二全中', '尾数对碰')).toBe(true)
    expect(supportsAdvTriggerBet('g003', '283', '连码', '二全中生尾对碰')).toBe(true)
    expect(supportsAdvTriggerBet('erquanzhong', 'sw_dp', '二全中', '生尾对碰')).toBe(true)
    expect(supportsAdvTriggerBet('g003', '283', '连码', '二全中生尾对碰')).toBe(true)
    expect(supportsAdvTriggerBet('erquanzhong', 'sw_dp', '二全中', '生尾对碰')).toBe(true)
    expect(supportsAdvTriggerBet('erquanzhong', 'tuotou', '二全中', '拖头')).toBe(false)
    expect(
      lotteryHasAdvTriggerPlay([
        {
          typeId: 'erquanzhong',
          label: '二全中',
          sortOrder: 1,
          subPlays: [{ subId: 'fushi', label: '复式', sortOrder: 1, outboundPlayCode: 'fushi' }],
        },
      ]),
    ).toBe(true)
  })

  it('特码开某投某不按七位分列', () => {
    expect(
      supportsAdvTriggerPerPosColumns({
        betMode: 'tema',
        playTypeId: 'g001',
        catalogSubId: '272',
        playTypeLabel: '特码',
        playMethodLabel: '特码A',
        inputMode: 'lhc_num',
        segmentLen: 1,
      }),
    ).toBe(false)
  })
})
