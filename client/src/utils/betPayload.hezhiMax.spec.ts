import { describe, expect, it } from 'vitest'
import {
  countBetUnits,
  greedyHezhiKuaduPicksUnderMax,
  hezhiKuaduMaxBetUnits,
  maxHezhiKuaduRandomCount,
  REN2_ZHIXUAN_HEZHI_MAX_BET_UNITS,
  renxuanZhixuanFushiMaxBetUnits,
  validateGroupContent,
  type PlayConfig,
} from './betPayload'

const qian2Hezhi = {
  playTemplate: 'ssc_std',
  playTypeId: 'g004',
  playTypeLabel: '前二',
  subPlayId: '41',
  betMode: 'hezhi',
  segmentLen: 1, // UI 单档；上限须按文案推断为二星
  segmentLabels: ['和值'],
  inputMode: 'pool',
  playMethodLabel: '前二直选和值',
  numberPoolMin: 0,
  numberPoolMax: 18,
} as PlayConfig

const qian3Hezhi = {
  playTemplate: 'ssc_std',
  playTypeId: 'g001',
  playTypeLabel: '前三',
  subPlayId: '3',
  betMode: 'hezhi',
  segmentLen: 1,
  segmentLabels: ['和值'],
  inputMode: 'pool',
  playMethodLabel: '前三直选和值',
  numberPoolMin: 0,
  numberPoolMax: 27,
} as PlayConfig

describe('和值最大注数', () => {
  it('前二上限 90', () => {
    expect(hezhiKuaduMaxBetUnits(qian2Hezhi)).toBe(90)
  })

  it('前三上限 900', () => {
    expect(hezhiKuaduMaxBetUnits(qian3Hezhi)).toBe(900)
  })

  it('任二直选和值上限 900', () => {
    const ren2 = {
      playTemplate: 'ssc_std',
      playTypeId: 'g011',
      playTypeLabel: '任选',
      guajiGroup: '任选',
      subPlayId: '76',
      catalogSubId: '76',
      betMode: 'hezhi',
      segmentLen: 1,
      segmentLabels: ['选号'],
      inputMode: 'pool',
      playMethodLabel: '任二直选和值',
      renPositionCount: 2,
      numberPoolMin: 0,
      numberPoolMax: 18,
    } as PlayConfig
    expect(hezhiKuaduMaxBetUnits(ren2)).toBe(REN2_ZHIXUAN_HEZHI_MAX_BET_UNITS)
    expect(REN2_ZHIXUAN_HEZHI_MAX_BET_UNITS).toBe(900)
    // 万千 + 满选 0–18 = 100 注，可通过
    const ok2 = validateGroupContent(ren2, '万,千\n0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18')
    expect(ok2.ok).toBe(true)
    if (ok2.ok) expect(ok2.betUnits).toBe(100)
    // 三选位 → C(3,2)×100=300 ≤ 900，可通过
    const ok3 = validateGroupContent(ren2, '万,千,个\n0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18')
    expect(ok3.ok).toBe(true)
    if (ok3.ok) expect(ok3.betUnits).toBe(300)
    // 五选位 → C(5,2)×100=1000 > 900
    const over = validateGroupContent(
      ren2,
      '万,千,百,十,个\n0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18',
    )
    expect(over.ok).toBe(false)
    if (!over.ok) expect(over.message).toBe('投注注数超过最大投注注数:900')
  })

  it('任三直选和值上限 9000（勿套前三 900）', () => {
    const ren3 = {
      playTemplate: 'ssc_std',
      playTypeId: 'g011',
      playTypeLabel: '任选',
      guajiGroup: '任选',
      subPlayId: '82',
      catalogSubId: '82',
      betMode: 'hezhi',
      segmentLen: 1,
      segmentLabels: ['选号'],
      inputMode: 'pool',
      playMethodLabel: '任三直选和值',
      renPositionCount: 3,
      numberPoolMin: 0,
      numberPoolMax: 27,
    } as PlayConfig
    expect(renxuanZhixuanFushiMaxBetUnits(ren3)).toBe(9000)
    expect(hezhiKuaduMaxBetUnits(ren3)).toBe(9000)
    // 三选位满选 0–27：内层注数通常 >900，但总注 ≤9000 时应通过（旧逻辑误报 900）
    const full = Array.from({ length: 28 }, (_, i) => String(i)).join(',')
    const r = validateGroupContent(ren3, `万,千,个\n${full}`)
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.betUnits).toBeGreaterThan(900)
      expect(r.betUnits).toBeLessThanOrEqual(9000)
    }
  })

  it('前二满选 0–18 拦截', () => {
    const r = validateGroupContent(qian2Hezhi, '0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18')
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.message).toBe('投注注数超过最大投注注数:90')
  })

  it('前二单和值通过', () => {
    const r = validateGroupContent(qian2Hezhi, '9')
    expect(r.ok).toBe(true)
  })

  it('前二随机个数上限为 18（非号池 19）', () => {
    const uni = Array.from({ length: 19 }, (_, i) => String(i))
    expect(maxHezhiKuaduRandomCount(qian2Hezhi, uni)).toBe(18)
  })

  it('贪心 18 个前二和值总注 ≤ 90', () => {
    const uni = Array.from({ length: 19 }, (_, i) => String(i))
    const picks = greedyHezhiKuaduPicksUnderMax(qian2Hezhi, 18, uni)
    const r = validateGroupContent(qian2Hezhi, picks.join(','))
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBeLessThanOrEqual(90)
  })

  it('前中后三直选和值 0–9 = 220×3 = 660（勿按四星 715×3=2145）', () => {
    const cfg = {
      playTemplate: 'ssc_std',
      playTypeId: 'g007',
      playTypeLabel: '前中后三',
      guajiGroup: '前中后三',
      subPlayId: '103',
      catalogSubId: '103',
      betMode: 'hezhi',
      segmentLen: 1,
      segmentLabels: ['和值'],
      inputMode: 'pool',
      playMethodLabel: '直选和值',
      numberPoolMin: 0,
      numberPoolMax: 27,
    } as PlayConfig
    expect(countBetUnits(cfg, '0,1,2,3,4,5,6,7,8,9')).toBe(660)
  })

  it('仅 playTypeId=g007 时按前中后三计注', () => {
    const cfg = {
      playTemplate: 'ssc_std',
      playTypeId: 'g007',
      betMode: 'hezhi',
      segmentLen: 1,
      inputMode: 'pool',
      numberPoolMin: 0,
      numberPoolMax: 27,
    } as PlayConfig
    // g007 是前中后三，三段均计入：220 × 3 = 660。
    expect(countBetUnits(cfg, '0,1,2,3,4,5,6,7,8,9')).toBe(660)
  })
})
