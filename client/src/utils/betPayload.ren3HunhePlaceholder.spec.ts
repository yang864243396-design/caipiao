import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  countBetUnits,
  groupContentPlaceholder,
  HUNHE_DANSHI_PATTERN_MSG,
  isHunhePlayConfig,
  validateGroupContent,
} from './betPayload'
import { commitSchemeGroupContentOnBlur } from './pickPanelOptions'

const ren3Hunhe = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '87',
  catalogSubId: '87',
  betMode: 'hunhe',
  playMethodLabel: '任三混合组选',
  inputMode: 'danshi',
  segmentLen: 3,
  segmentLabels: ['选号'],
  renPositionCount: 3,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('任三混合组选 placeholder / 校验 / 注数', () => {
  it('识别为混合组选', () => {
    expect(isHunhePlayConfig(ren3Hunhe)).toBe(true)
  })

  it('placeholder：三个号码、顺序不限', () => {
    const tip = groupContentPlaceholder(ren3Hunhe)
    expect(tip).toContain('再输入三个号码组成一注')
    expect(tip).toContain('所选3个位置的开奖号码与输入号码一致，顺序不限')
    expect(tip).toContain('012,345')
    expect(tip).not.toContain('顺序均须与开奖一致')
  })

  it('012 通过；123 与 321 同形态去重', () => {
    const r = validateGroupContent(ren3Hunhe, '万,千,个\n012')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(1)

    const dup = validateGroupContent(ren3Hunhe, '万,千,个\n123,321')
    expect(dup.ok).toBe(true)
    if (dup.ok) expect(dup.betUnits).toBe(1)
    expect(countBetUnits(ren3Hunhe, '万,千,个\n123,321')).toBe(1)
  })

  it('豹子拒绝；超长废票失焦清空', () => {
    const bao = validateGroupContent(ren3Hunhe, '万,千,个\n111')
    expect(bao.ok).toBe(false)
    if (!bao.ok) expect(bao.message).toBe(HUNHE_DANSHI_PATTERN_MSG)

    expect(commitSchemeGroupContentOnBlur('111,1234', ren3Hunhe)).toBe('')
    expect(commitSchemeGroupContentOnBlur('012,111,210', ren3Hunhe)).toBe('012')
  })

  it('多选位：C(4,3)×有效票', () => {
    const r = validateGroupContent(ren3Hunhe, '万,千,百,个\n012,345')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(8)
  })

  it('冷热按位号池先展成整注再校验（保存不报「每注须为3位」）', () => {
    const raw = '万,千,个\n0,1\n1,2\n2,3'
    const r = validateGroupContent(ren3Hunhe, raw)
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.betUnits).toBeGreaterThan(0)
      // 落库为整注，不含按位换行号池
      expect(r.normalized).toMatch(/^万,千,个\n/)
      expect(r.normalized).not.toMatch(/\n\d,\d/)
      expect(countBetUnits(ren3Hunhe, raw)).toBe(r.betUnits)
    }
  })
})
