import { describe, expect, it } from 'vitest'
import {
  countBetUnits,
  parseRenxuanPositionContent,
  validateGroupContent,
  zhixuanDanshiMaxBetUnits,
  type PlayConfig,
} from './betPayload'

function ren2DanshiConfig(): PlayConfig {
  return {
    playTemplate: 'ssc_std',
    playTypeId: 'g011',
    subPlayId: '75',
    catalogSubId: '75',
    playTypeLabel: '任选',
    playMethodLabel: '任二直选单式',
    guajiGroup: '任选',
    betMode: 'danshi',
    inputMode: 'danshi',
    segmentLen: 2,
    segmentLabels: ['万', '千', '百', '十', '个'],
    renPositionCount: 2,
    numberPoolMin: 0,
    numberPoolMax: 9,
  } as PlayConfig
}

describe('任二直选单式选位多选', () => {
  it('parse 保留多于 k 个选位', () => {
    const parsed = parseRenxuanPositionContent('万,千,百\n12,34', 2)
    expect(parsed.positions).toEqual(['万', '千', '百'])
    expect(parsed.picks).toBe('12,34')
  })

  it('恰好 2 位时注数=号码注数', () => {
    const cfg = ren2DanshiConfig()
    expect(countBetUnits(cfg, '千,个\n12,34')).toBe(2)
  })

  it('选 3 位时注数=C(3,2)×号码注数', () => {
    const cfg = ren2DanshiConfig()
    expect(countBetUnits(cfg, '万,千,百\n12')).toBe(3)
    expect(countBetUnits(cfg, '万,千,百\n12,34')).toBe(6)
  })

  it('选 5 位时注数=C(5,2)×号码注数', () => {
    const cfg = ren2DanshiConfig()
    expect(countBetUnits(cfg, '万,千,百,十,个\n12')).toBe(10)
  })

  it('校验：至少 2 位，最多 5 位', () => {
    const cfg = ren2DanshiConfig()
    expect(validateGroupContent(cfg, '万\n12').ok).toBe(false)
    const ok3 = validateGroupContent(cfg, '万,千,百\n12,34')
    expect(ok3.ok).toBe(true)
    if (ok3.ok) expect(ok3.betUnits).toBe(6)
  })

  it('上限为 900（非前二 90）；五位×00–99=1000 拒并提示正确文案', () => {
    const cfg = ren2DanshiConfig()
    expect(zhixuanDanshiMaxBetUnits(cfg)).toBe(900)
    const picks = Array.from({ length: 100 }, (_, i) => String(i).padStart(2, '0')).join(',')
    const content = `万,千,百,十,个\n${picks}`
    expect(countBetUnits(cfg, content)).toBe(1000)
    const r = validateGroupContent(cfg, content)
    expect(r.ok).toBe(false)
    if (!r.ok) expect(r.message).toBe('投注注数超过最大投注注数:900')
  })
})

function ren2ZuxuanFsConfig(): PlayConfig {
  return {
    playTemplate: 'ssc_std',
    playTypeId: 'g011',
    subPlayId: '76',
    catalogSubId: '76',
    playTypeLabel: '任选',
    playMethodLabel: '任二组选复式',
    guajiGroup: '任选',
    betMode: 'zuxuan_fs',
    inputMode: 'pool',
    segmentLen: 1,
    segmentLabels: ['选号'],
    renPositionCount: 2,
    numberPoolMin: 0,
    numberPoolMax: 9,
  } as PlayConfig
}

describe('任二组选复式选位（对齐直选单式）', () => {
  it('注数=C(选位数,2)×C(号池,2)', () => {
    const cfg = ren2ZuxuanFsConfig()
    // C(3,2)*C(3,2)=3*3=9
    expect(countBetUnits(cfg, '万,千,个\n1,2,3')).toBe(9)
  })

  it('校验须带选位前缀', () => {
    const cfg = ren2ZuxuanFsConfig()
    expect(validateGroupContent(cfg, '1,2,3').ok).toBe(false)
    const ok = validateGroupContent(cfg, '万,千\n1,2,3')
    expect(ok.ok).toBe(true)
    if (ok.ok) expect(ok.betUnits).toBe(3)
  })
})
