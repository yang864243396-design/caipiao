import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  groupDigitInputHint,
  poolUsesCommaSeparatedInput,
  schemeGroupContentToInputBox,
  schemeGroupInputBoxToContent,
} from './pickPanelOptions'

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

describe('kuadu comma-separated input box', () => {
  it('uses comma-separated pool input', () => {
    expect(poolUsesCommaSeparatedInput(kuaduConfig)).toBe(true)
  })

  it('shows stored content with commas in the input box', () => {
    expect(schemeGroupContentToInputBox('0,3,9', kuaduConfig)).toBe('0,3,9')
    expect(schemeGroupContentToInputBox('039', kuaduConfig)).toBe('0,3,9')
  })

  it('parses comma-separated or glued box into content', () => {
    expect(schemeGroupInputBoxToContent('0,3,9', kuaduConfig)).toBe('0,3,9')
    expect(schemeGroupInputBoxToContent('039', kuaduConfig)).toBe('0,3,9')
  })

  it('hint tells user to separate each digit with commas', () => {
    expect(groupDigitInputHint(kuaduConfig)).toMatch(/逗号/)
  })
})
