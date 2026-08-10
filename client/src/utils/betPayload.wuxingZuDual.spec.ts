import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  countBetUnits,
  countZu20BetUnits,
  countZu30BetUnits,
  countZu60BetUnits,
  countZu10BetUnits,
  countZu5BetUnits,
  validateGroupContent,
  validateZu20Content,
  validateZu30Content,
  validateZu60Content,
  validateZu10Content,
  validateZu5Content,
  zuDualFormatHint,
  zuDualMetaOf,
  zuxuanPoolMinPick,
} from './betPayload'

function cfg(bm: string, label: string): PlayConfig {
  return {
    playTemplate: 'ssc_std',
    playTypeId: 'g015',
    playTypeLabel: '五星',
    subPlayId: bm,
    catalogSubId: bm,
    playMethodLabel: label,
    betMode: bm,
    inputMode: 'pool',
    segmentLen: 1,
    segmentLabels: [],
  } as PlayConfig
}

describe('五星组选60 双区', () => {
  const c = cfg('zu60', '组选60')
  it('meta / hint', () => {
    expect(zuDualMetaOf(c)?.minTail).toBe(3)
    expect(zuDualFormatHint(c)).toContain('二重')
    expect(zuDualFormatHint(c)).toContain('1,234')
  })
  it('计注：1,234=1；12,345=2；1,1234=1', () => {
    expect(countZu60BetUnits('1,234')).toBe(1)
    expect(countZu60BetUnits('12,345')).toBe(2)
    expect(countZu60BetUnits('1,1234')).toBe(1)
    expect(countZu60BetUnits('12,3')).toBe(0)
    expect(countBetUnits(c, '12,3456')).toBe(8)
  })
  it('校验', () => {
    expect(zuxuanPoolMinPick(c)).toBeNull()
    expect(validateZu60Content('1,234').ok).toBe(true)
    expect(validateZu60Content('12,3').ok).toBe(false)
    const v = validateGroupContent(c, '1,234')
    expect(v.ok).toBe(true)
    if (!v.ok) return
    expect(v.betUnits).toBe(1)
  })
})

describe('五星组选30 双区', () => {
  const c = cfg('zu30', '组选30')
  it('meta / hint', () => {
    expect(zuDualMetaOf(c)?.minHead).toBe(3)
    expect(zuDualMetaOf(c)?.minTail).toBe(1)
    expect(zuDualFormatHint(c)).toContain('3个及以上的二重号')
    expect(zuDualFormatHint(c)).toContain('123,1')
    expect(zuxuanPoolMinPick(c)).toBeNull()
  })
  it('计注：123,1=1；123,45=6；1234,5=6；1234,56=12', () => {
    expect(countZu30BetUnits('123,1')).toBe(1)
    expect(countZu30BetUnits('123,45')).toBe(6)
    expect(countZu30BetUnits('1234,5')).toBe(6)
    expect(countZu30BetUnits('1234,56')).toBe(12)
    expect(countZu30BetUnits('12,3')).toBe(0)
    expect(countZu30BetUnits('123,12')).toBe(2)
    expect(countBetUnits(c, '123,45')).toBe(6)
  })
  it('校验', () => {
    expect(validateZu30Content('123,1').ok).toBe(true)
    expect(validateZu30Content('12,3').ok).toBe(false)
    expect(validateGroupContent(c, '123,1').ok).toBe(true)
  })
})

describe('五星组选20 双区（个数须相同且各≥2）', () => {
  const c = cfg('zu20', '组选20')
  it('meta / hint', () => {
    expect(zuDualMetaOf(c)?.equalCounts).toBe(true)
    expect(zuDualMetaOf(c)?.minHead).toBe(2)
    expect(zuDualMetaOf(c)?.minTail).toBe(2)
    expect(zuDualFormatHint(c)).toContain('须相同')
    expect(zuDualFormatHint(c)).toContain('至少各 2 个')
    expect(zuDualFormatHint(c)).toContain('12,34')
  })
  it('计注：12,34=2；123,345=7；123,456=9；各1非法', () => {
    expect(countZu20BetUnits('12,34')).toBe(2)
    expect(countZu20BetUnits('123,345')).toBe(7)
    expect(countZu20BetUnits('123,456')).toBe(9)
    expect(countZu20BetUnits('1234,5678')).toBe(24)
    expect(countZu20BetUnits('12,345')).toBe(0)
    expect(countZu20BetUnits('1,2')).toBe(0)
    expect(countBetUnits(c, '12,34')).toBe(2)
    expect(countBetUnits(c, '123,345')).toBe(7)
    expect(countBetUnits(c, '123,456')).toBe(9)
  })
  it('校验：各1或个数不同报错', () => {
    expect(validateZu20Content('12,34').ok).toBe(true)
    expect(validateZu20Content('123,345').ok).toBe(true)
    expect(validateZu20Content('123,456').ok).toBe(true)
    expect(validateZu20Content('12,345').ok).toBe(false)
    expect(validateZu20Content('1,2').ok).toBe(false)
  })
})

describe('五星组选10/5 双区', () => {
  it('组选10：三重×二重', () => {
    const c = cfg('zu10', '组选10')
    expect(zuDualMetaOf(c)?.tailLabel).toBe('二重号')
    expect(countZu10BetUnits('1,2')).toBe(1)
    expect(countZu10BetUnits('12,34')).toBe(4)
    expect(countZu10BetUnits('12,12')).toBe(2)
    expect(validateZu10Content('1,1').ok).toBe(false)
    expect(countBetUnits(c, '1,23')).toBe(2)
  })
  it('组选5：四重×单号', () => {
    const c = cfg('zu5', '组选5')
    expect(zuDualMetaOf(c)?.headLabel).toBe('四重号')
    expect(countZu5BetUnits('1,2')).toBe(1)
    expect(countZu5BetUnits('12,34')).toBe(4)
    expect(validateZu5Content('1,2').ok).toBe(true)
  })
})
