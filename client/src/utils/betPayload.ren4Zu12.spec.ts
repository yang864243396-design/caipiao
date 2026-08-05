import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  countBetUnits,
  countZu12BetUnits,
  groupContentPlaceholder,
  parseZu12Zones,
  randomZu12DualContent,
  validateGroupContent,
  zuxuanPoolMinPick,
} from './betPayload'
import { groupDigitInputHint } from './pickPanelOptions'

const ren4Zu12 = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '144',
  catalogSubId: '144',
  betMode: 'zu12',
  playMethodLabel: '任四组选12',
  inputMode: 'pool',
  segmentLen: 1,
  segmentLabels: ['选号'],
  renPositionCount: 4,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('任四组选12 双区 placeholder / 校验 / 计注', () => {
  it('placeholder 为选位 + 二重/单号双区', () => {
    expect(groupContentPlaceholder(ren4Zu12)).toBe(
      '从万、千、百、十、个中勾选至少 4 个，再从0-9中，输入1个及以上的二重号码，2个及以上的单号，两个位置由逗号分隔，如：12,3234',
    )
    expect(groupDigitInputHint(ren4Zu12)).toContain('二重')
    expect(zuxuanPoolMinPick(ren4Zu12)).toBeNull()
  })

  it('解析双区：区内去重、跨区重叠保留；按二重分别计注', () => {
    expect(parseZu12Zones('12,3234')?.normalized).toBe('12,324')
    expect(parseZu12Zones('1,234')?.doubles).toEqual(['1'])
    expect(parseZu12Zones('1,2')).toBeNull()
    expect(parseZu12Zones('1,2,3,4')).toBeNull()
    expect(countZu12BetUnits('12,34')).toBe(2) // C(2,2)+C(2,2)
    expect(countZu12BetUnits('1,234')).toBe(3) // C(3,2)
    expect(countZu12BetUnits('1,12')).toBe(0) // 选1后单号仅剩1码
    expect(countZu12BetUnits('2,123')).toBe(1) // 选2后单号{1,3}
    // 23,123：选2→{1,3} 1注；选3→{1,2} 1注；内容保持重叠
    expect(parseZu12Zones('23,123')?.normalized).toBe('23,123')
    expect(countZu12BetUnits('23,123')).toBe(2)
    // 12,3234 → 12,324：选1→C(3,2)=3；选2→C(2,2)=1 → 4
    expect(countZu12BetUnits('12,3234')).toBe(4)
  })

  it('带选位校验：合法过、非法拒；跨区重叠可投且不改内容', () => {
    const ok = validateGroupContent(ren4Zu12, '万,千,百,十\n12,34')
    expect(ok.ok).toBe(true)
    if (ok.ok) expect(ok.betUnits).toBe(2)

    const bad = validateGroupContent(ren4Zu12, '万,千,百,十\n1,2')
    expect(bad.ok).toBe(false)
    if (!bad.ok) expect(bad.message).toContain('组选12')

    const overlap = validateGroupContent(ren4Zu12, '万,千,百,十\n1,12')
    expect(overlap.ok).toBe(false)
    if (!overlap.ok) expect(overlap.message).toContain('二重')

    const partial = validateGroupContent(ren4Zu12, '万,千,百,十\n23,123')
    expect(partial.ok).toBe(true)
    if (partial.ok) {
      expect(partial.betUnits).toBe(2)
      expect(partial.normalized).toContain('23,123')
    }

    const one = validateGroupContent(ren4Zu12, '万,千,百,十\n2,123')
    expect(one.ok).toBe(true)
    if (one.ok) {
      expect(one.betUnits).toBe(1)
      expect(one.normalized).toContain('2,123')
    }

    expect(countBetUnits(ren4Zu12, '万,千,百,十\n12,34')).toBe(2)
    expect(countBetUnits(ren4Zu12, '万,千,百,十\n23,123')).toBe(2)
  })

  it('随机双区始终至少 1 注', () => {
    for (const doubles of [1, 2, 3, 4, 9]) {
      for (const singles of [2, 3, 5, 10]) {
        for (let i = 0; i < 20; i++) {
          const raw = randomZu12DualContent(doubles, singles)
          const zones = parseZu12Zones(raw)
          expect(zones, `d=${doubles} s=${singles} raw=${raw}`).not.toBeNull()
          expect(zones!.doubles.length, `d=${doubles} s=${singles} raw=${raw}`).toBe(doubles)
          expect(zones!.singles.length, `d=${doubles} s=${singles} raw=${raw}`).toBe(singles)
          expect(countZu12BetUnits(raw), `d=${doubles} s=${singles} raw=${raw}`).toBeGreaterThan(0)
        }
      }
    }
  })
})
