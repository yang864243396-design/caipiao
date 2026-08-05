import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  commitSchemeGroupContentOnBlur,
  schemeGroupContentToInputBox,
  schemeGroupUsesDigitInput,
} from './pickPanelOptions'

const qian3Fushi = {
  playTemplate: 'ssc_std',
  playTypeId: 'g001',
  playTypeLabel: '前三码',
  subPlayId: '1',
  betMode: 'fushi',
  playMethodLabel: '前三直选复式',
  inputMode: 'multiline',
  segmentLen: 3,
  segmentLabels: ['万', '千', '百'],
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

const qian2Danshi = {
  playTemplate: 'ssc_std',
  playTypeId: 'g004',
  playTypeLabel: '前二',
  subPlayId: 'zhixuan_ds',
  betMode: 'danshi',
  segmentLen: 2,
  segmentLabels: ['十', '个'],
  inputMode: 'danshi',
  playMethodLabel: '前二直选单式',
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

const ren3Danshi = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  subPlayId: '81',
  catalogSubId: '81',
  betMode: 'danshi',
  segmentLen: 3,
  segmentLabels: ['选号'],
  inputMode: 'danshi',
  playMethodLabel: '任三直选单式',
  renPositionCount: 3,
  guajiGroup: '任选',
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

const kuaduConfig = {
  playTemplate: 'ssc_std',
  playTypeId: 'g002',
  subPlayId: '17',
  betMode: 'kuadu',
  playMethodLabel: '中三直选跨度',
  inputMode: 'pool',
  segmentLen: 1,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('commitSchemeGroupContentOnBlur', () => {
  it('前三直选复式：录入框往返为按位内容', () => {
    expect(schemeGroupUsesDigitInput(qian3Fushi)).toBe(true)
    const content = commitSchemeGroupContentOnBlur('12,34,5', qian3Fushi)
    expect(content).toBe('1,2\n3,4\n5')
    expect(schemeGroupContentToInputBox(content, qian3Fushi)).toBe('12,34,5')
  })

  it('前二直选单式：去重并剔除非法位数', () => {
    expect(commitSchemeGroupContentOnBlur('12,12,1', qian2Danshi)).toBe('12')
    expect(commitSchemeGroupContentOnBlur('12,34,12', qian2Danshi)).toBe('12,34')
  })

  it('前二直选单式：半截输入保留原文', () => {
    expect(commitSchemeGroupContentOnBlur('1', qian2Danshi)).toBe('1')
  })

  it('直选单式：全是超长废票则失焦清空', () => {
    expect(commitSchemeGroupContentOnBlur('1234,5678', qian2Danshi)).toBe('')
  })

  it('任三直选单式：每注 3 位，剔除 2 位票', () => {
    expect(commitSchemeGroupContentOnBlur('012,012,12', ren3Danshi)).toBe('012')
    expect(commitSchemeGroupContentOnBlur('012,345', ren3Danshi)).toBe('012,345')
  })

  it('跨度号池：粘连串拆成逗号分隔', () => {
    expect(schemeGroupUsesDigitInput(kuaduConfig)).toBe(true)
    expect(commitSchemeGroupContentOnBlur('039', kuaduConfig)).toBe('0,3,9')
    expect(commitSchemeGroupContentOnBlur('0,3,9', kuaduConfig)).toBe('0,3,9')
  })

  it('仅逗号/空白 → 空串', () => {
    expect(commitSchemeGroupContentOnBlur(',,,', qian3Fushi)).toBe('')
    expect(commitSchemeGroupContentOnBlur('  ，  ', qian2Danshi)).toBe('')
  })
})
