import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { bareConfigForRenxuanPicks, isZu6DanshiConfig } from './betPayload'
import { commitSchemeGroupContentOnBlur, schemeGroupUsesDigitInput } from './pickPanelOptions'

const ren3Zu6Ds = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '86',
  catalogSubId: '86',
  betMode: 'zuxuan_ds',
  playMethodLabel: '任三组六单式',
  inputMode: 'danshi',
  segmentLen: 3,
  renPositionCount: 3,
  numberPoolMin: 0,
  numberPoolMax: 9,
  segmentLabels: ['选号'],
} as PlayConfig

describe('任三组六单式失焦', () => {
  it('bareConfig / 完整 config 均识别组六单式', () => {
    const bare = bareConfigForRenxuanPicks(ren3Zu6Ds)
    expect(isZu6DanshiConfig(ren3Zu6Ds)).toBe(true)
    expect(isZu6DanshiConfig(bare)).toBe(true)
    expect(schemeGroupUsesDigitInput(bare)).toBe(false)
  })

  it('失焦过滤非组六形态，保留合法票', () => {
    expect(commitSchemeGroupContentOnBlur('012,112,111,210', ren3Zu6Ds)).toBe('012')
  })

  it('失焦：全是非法完整票则清空', () => {
    expect(commitSchemeGroupContentOnBlur('112,111', ren3Zu6Ds)).toBe('')
  })

  it('失焦：超长废票清空（勿当半截原样保留）', () => {
    expect(commitSchemeGroupContentOnBlur('1234,6548,3215654321', ren3Zu6Ds)).toBe('')
  })

  it('失焦：半截输入保留', () => {
    expect(commitSchemeGroupContentOnBlur('01', ren3Zu6Ds)).toBe('01')
  })

  it('失焦：合法票+超长废票只留合法', () => {
    expect(commitSchemeGroupContentOnBlur('012,1234,345', ren3Zu6Ds)).toBe('012,345')
  })
})
