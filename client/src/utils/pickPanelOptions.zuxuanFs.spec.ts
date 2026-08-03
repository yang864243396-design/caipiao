import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  groupDigitInputHint,
  poolUsesCommaSeparatedInput,
  schemeGroupContentToInputBox,
  schemeGroupInputBoxToContent,
} from './pickPanelOptions'

const qian2ZuxuanFs = {
  playTemplate: 'ssc_std',
  playTypeId: 'g004',
  subPlayId: '42',
  betMode: 'zuxuan_fs',
  playTypeLabel: '前二码',
  playMethodLabel: '前二组选复式',
  inputMode: 'pool',
  segmentLen: 1,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('前二组选复式 comma-separated input box', () => {
  it('uses comma-separated pool input', () => {
    expect(poolUsesCommaSeparatedInput(qian2ZuxuanFs)).toBe(true)
  })

  it('shows stored content with commas in the input box', () => {
    expect(schemeGroupContentToInputBox('0,1,2,3', qian2ZuxuanFs)).toBe('0,1,2,3')
    expect(schemeGroupContentToInputBox('0123', qian2ZuxuanFs)).toBe('0,1,2,3')
  })

  it('parses comma-separated or glued box into content', () => {
    expect(schemeGroupInputBoxToContent('0,1,2,3', qian2ZuxuanFs)).toBe('0,1,2,3')
    expect(schemeGroupInputBoxToContent('0123', qian2ZuxuanFs)).toBe('0,1,2,3')
  })

  it('hint tells user to separate digits with commas', () => {
    expect(groupDigitInputHint(qian2ZuxuanFs)).toMatch(/逗号/)
    expect(groupDigitInputHint(qian2ZuxuanFs)).toMatch(/2/)
  })
})
