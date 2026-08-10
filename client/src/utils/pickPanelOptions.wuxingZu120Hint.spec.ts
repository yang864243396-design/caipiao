import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  countBetUnits,
  groupContentPlaceholder,
  validateGroupContent,
  zuxuanPoolMinPick,
} from './betPayload'
import {
  groupDigitInputHint,
  poolUsesCommaSeparatedInput,
  schemeGroupInputBoxToContent,
} from './pickPanelOptions'

const wuxingZu120 = {
  playTemplate: 'ssc_std',
  playTypeId: 'g015',
  playTypeLabel: '五星',
  guajiGroup: '五星',
  subPlayId: '156',
  catalogSubId: '156',
  betMode: 'zu120',
  playMethodLabel: '组选120',
  inputMode: 'pool',
  segmentLen: 1,
  segmentLabels: ['选号'],
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('五星组选120 号池提示与逗号录入', () => {
  it('placeholder / hint 为至少 5 码逗号分隔', () => {
    const tip = groupDigitInputHint(wuxingZu120)
    expect(tip).toBe('输入5个及以上0-9的号码，多选用逗号分隔，如：1,3,5,7,9')
    expect(groupContentPlaceholder(wuxingZu120)).toBe(tip)
    expect(poolUsesCommaSeparatedInput(wuxingZu120)).toBe(true)
  })

  it('连写 13579 失焦按逗号号池拆开；计注 C(5,5)=1', () => {
    expect(schemeGroupInputBoxToContent('1,3,5,7,9', wuxingZu120)).toBe('1,3,5,7,9')
    expect(schemeGroupInputBoxToContent('13579', wuxingZu120)).toBe('1,3,5,7,9')
    expect(zuxuanPoolMinPick(wuxingZu120)).toBe(5)
    const bad = validateGroupContent(wuxingZu120, '1,3,5,7')
    expect(bad.ok).toBe(false)
    if (!bad.ok) expect(bad.message).toContain('组选120至少选择 5')
    const ok = validateGroupContent(wuxingZu120, '1,3,5,7,9')
    expect(ok.ok).toBe(true)
    if (ok.ok) expect(ok.betUnits).toBe(1)
    expect(countBetUnits(wuxingZu120, '1,3,5,7,9,0')).toBe(6) // C(6,5)
  })
})
