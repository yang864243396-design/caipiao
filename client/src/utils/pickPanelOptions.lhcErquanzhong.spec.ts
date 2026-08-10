import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { countBetUnits, validateGroupContent } from './betPayload'
import {
  groupDigitInputHint,
  schemeGroupUsesDigitInput,
  schemeGroupUsesPickPanel,
  schemeGroupUsesTextInputPanel,
  schemeGroupInputBoxToContent,
  schemeGroupContentToInputBox,
} from './pickPanelOptions'

const er2Fushi = {
  playTemplate: 'lhc_std',
  playTypeId: 'erquanzhong',
  catalogSubId: '279',
  subPlayId: 'fushi',
  betMode: 'fushi',
  playTypeLabel: '二全中',
  playMethodLabel: '复式',
  segmentLen: 1,
  segmentLabels: ['选号'],
  inputMode: 'lhc_num',
  numberPoolMin: 1,
  numberPoolMax: 49,
} as PlayConfig

const er2Tuotou = {
  ...er2Fushi,
  catalogSubId: '280',
  subPlayId: 'tuotou',
  betMode: 'tuotou',
  playMethodLabel: '拖头',
} as PlayConfig

describe('二全中复式 方案内容输入框', () => {
  it('走输入框而非 chip 点选', () => {
    expect(schemeGroupUsesPickPanel(er2Fushi)).toBe(false)
    expect(schemeGroupUsesDigitInput(er2Fushi)).toBe(true)
    expect(schemeGroupUsesTextInputPanel(er2Fushi)).toBe(true)
  })

  it('提示 2–10 个 01–49', () => {
    expect(groupDigitInputHint(er2Fushi)).toMatch(/2–10|2-10/)
    expect(groupDigitInputHint(er2Fushi)).toMatch(/01–49|01-49/)
  })

  it('失焦规范化为补零逗号串，超过 10 个截断', () => {
    expect(schemeGroupInputBoxToContent('1,13,25', er2Fushi)).toBe('01,13,25')
    expect(schemeGroupContentToInputBox('01,13,25', er2Fushi)).toBe('01,13,25')
    const eleven = Array.from({ length: 11 }, (_, i) => String(i + 1).padStart(2, '0')).join(',')
    const capped = schemeGroupInputBoxToContent(eleven, er2Fushi).split(',')
    expect(capped).toHaveLength(10)
  })

  it('校验：少于 2 或超过 10 失败；2–10 通过并计组合注', () => {
    expect(validateGroupContent(er2Fushi, '07').ok).toBe(false)
    const eleven = Array.from({ length: 11 }, (_, i) => String(i + 1).padStart(2, '0')).join(',')
    expect(validateGroupContent(er2Fushi, eleven).ok).toBe(false)
    const ok = validateGroupContent(er2Fushi, '7,13,25')
    expect(ok.ok).toBe(true)
    if (ok.ok) {
      expect(ok.normalized).toBe('07,13,25')
      expect(ok.betUnits).toBe(countBetUnits(er2Fushi, '07,13,25'))
      expect(ok.betUnits).toBe(3) // C(3,2)
    }
    const ten = Array.from({ length: 10 }, (_, i) => String(i + 1).padStart(2, '0')).join(',')
    expect(validateGroupContent(er2Fushi, ten).ok).toBe(true)
  })
})

describe('二全中拖头 方案内容输入框', () => {
  it('走输入框而非 chip 点选（与复式一致）', () => {
    expect(schemeGroupUsesPickPanel(er2Tuotou)).toBe(false)
    expect(schemeGroupUsesDigitInput(er2Tuotou)).toBe(true)
    expect(schemeGroupUsesTextInputPanel(er2Tuotou)).toBe(true)
  })

  it('提示 2–10 个 01–49，并说明首个为胆', () => {
    expect(groupDigitInputHint(er2Tuotou)).toMatch(/2–10|2-10/)
    expect(groupDigitInputHint(er2Tuotou)).toMatch(/01–49|01-49/)
    expect(groupDigitInputHint(er2Tuotou)).toMatch(/胆/)
  })

  it('失焦规范化；旧 胆|拖 展成逗号扁选', () => {
    expect(schemeGroupInputBoxToContent('1,13,25', er2Tuotou)).toBe('01,13,25')
    expect(schemeGroupContentToInputBox('01|13,25', er2Tuotou)).toBe('01,13,25')
    const eleven = Array.from({ length: 11 }, (_, i) => String(i + 1).padStart(2, '0')).join(',')
    expect(schemeGroupInputBoxToContent(eleven, er2Tuotou).split(',')).toHaveLength(10)
  })

  it('校验：2–10；注数按首胆余拖（非复式组合）', () => {
    expect(validateGroupContent(er2Tuotou, '07').ok).toBe(false)
    const eleven = Array.from({ length: 11 }, (_, i) => String(i + 1).padStart(2, '0')).join(',')
    expect(validateGroupContent(er2Tuotou, eleven).ok).toBe(false)
    const ok = validateGroupContent(er2Tuotou, '7,13,25')
    expect(ok.ok).toBe(true)
    if (ok.ok) {
      expect(ok.normalized).toBe('07,13,25')
      // 胆 07 × 拖 C(2,1) = 2
      expect(ok.betUnits).toBe(2)
      expect(ok.betUnits).toBe(countBetUnits(er2Tuotou, '07,13,25'))
    }
    // 兼容旧 胆|拖
    const legacy = validateGroupContent(er2Tuotou, '07|13,25')
    expect(legacy.ok).toBe(true)
    if (legacy.ok) {
      expect(legacy.normalized).toBe('07,13,25')
      expect(legacy.betUnits).toBe(2)
    }
  })
})
