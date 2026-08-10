import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  countBetUnits,
  groupContentPlaceholder,
  isWuxingQuweiDigitPlayConfig,
  parseWuxingQuweiDigits,
  validateGroupContent,
  wuxingQuweiFormatHint,
  wuxingQuweiMaxPicks,
} from './betPayload'
import {
  poolMaxPicksForConfig,
  poolUsesCommaSeparatedInput,
  schemeGroupContentToInputBox,
  schemeGroupInputBoxToContent,
  textPickOptionsForConfig,
} from './pickPanelOptions'

function cfg(overrides: Partial<PlayConfig> = {}): PlayConfig {
  return {
    playTypeId: 'g015',
    subPlayId: '162',
    catalogSubId: '162',
    segmentLen: 1,
    segmentLabels: ['选号'],
    inputMode: 'pool',
    betMode: 'teshu',
    playMethodLabel: '一帆风顺',
    playTypeLabel: '五星',
    playTemplate: 'ssc_std',
    numberPoolMin: 0,
    numberPoolMax: 9,
    poolMaxPicks: 2,
    ...overrides,
  }
}

describe('五星一帆风顺 数字池', () => {
  it('识别趣味玩法，最多 2 码，非文字特殊号', () => {
    const c = cfg()
    expect(isWuxingQuweiDigitPlayConfig(c)).toBe(true)
    expect(wuxingQuweiMaxPicks(c)).toBe(2)
    expect(poolMaxPicksForConfig(c)).toBe(2)
    expect(textPickOptionsForConfig(c)).toEqual([])
    expect(wuxingQuweiFormatHint(c)).toMatch(/1.?2/)
    expect(wuxingQuweiFormatHint(c)).toContain('0,3')
    expect(groupContentPlaceholder(c)).toContain('0,3')
  })

  it('计注与校验：0,3 → 2 注；超过 2 码拒绝', () => {
    const c = cfg()
    expect(parseWuxingQuweiDigits('0,3')).toEqual(['0', '3'])
    expect(countBetUnits(c, '0,3')).toBe(2)
    const v = validateGroupContent(c, '0,3')
    expect(v.ok).toBe(true)
    if (!v.ok) return
    expect(v.normalized).toBe('0,3')
    expect(v.betUnits).toBe(2)
    expect(validateGroupContent(c, '0,3,9').ok).toBe(false)
  })

  it('空选号或文字特殊号非法', () => {
    const c = cfg()
    expect(validateGroupContent(c, '').ok).toBe(false)
    expect(validateGroupContent(c, '豹子').ok).toBe(false)
  })

  it('输入框按逗号展示；录入超限截到 2 码', () => {
    const c = cfg()
    expect(poolUsesCommaSeparatedInput(c)).toBe(true)
    expect(schemeGroupContentToInputBox('0,3', c)).toBe('0,3')
    expect(schemeGroupContentToInputBox('039', c)).toBe('0,3,9')
    expect(schemeGroupInputBoxToContent('0,3', c)).toBe('0,3')
    expect(schemeGroupInputBoxToContent('0,3,9', c)).toBe('0,3')
    expect(schemeGroupInputBoxToContent('039', c)).toBe('0,3')
  })
})
