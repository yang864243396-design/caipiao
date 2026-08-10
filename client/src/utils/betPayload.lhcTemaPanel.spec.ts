import { describe, expect, it } from 'vitest'
import { LHC_TEMA_QUICK_OPTIONS, lhcTemaHcwUniverse } from '@/constants/lhcPlay'
import {
  countBetUnits,
  isLhcTemaPlayConfig,
  lhcTemaInvalidTokens,
  normalizeLhcTemaContent,
  normalizeLhcTemaFlatContent,
  parseLhcTemaContentTokens,
  parseLhcTemaParts,
  validateGroupContent,
  type PlayConfig,
} from './betPayload'
import { schemeGroupUsesLhcTemaPanel, schemeGroupUsesPickPanel } from './pickPanelOptions'

const temaCfg = {
  playTemplate: 'lhc_std',
  playTypeId: 'g001',
  catalogSubId: '272',
  subPlayId: 'tema',
  betMode: 'tema',
  playTypeLabel: '特码',
  playMethodLabel: '特码A',
  segmentLen: 1,
  segmentLabels: ['特码A'],
  inputMode: 'lhc_num',
  numberPoolMin: 1,
  numberPoolMax: 49,
} as PlayConfig

describe('特码A 方案内容面板', () => {
  it('提供 19 个快捷选项（红波/绿波）', () => {
    expect(LHC_TEMA_QUICK_OPTIONS).toHaveLength(19)
    expect(LHC_TEMA_QUICK_OPTIONS).toContain('红波')
    expect(LHC_TEMA_QUICK_OPTIONS).toContain('绿波')
    expect(LHC_TEMA_QUICK_OPTIONS).not.toContain('洪波')
    expect(LHC_TEMA_QUICK_OPTIONS).not.toContain('绿播')
  })

  it('冷热出号宇宙为 68 项（号码+属性+波色）', () => {
    const uni = lhcTemaHcwUniverse()
    expect(uni).toHaveLength(68)
    expect(uni[0]).toBe('01')
    expect(uni[48]).toBe('49')
    expect(uni).toContain('大')
    expect(uni).toContain('红波')
    expect(uni).toContain('绿波')
  })

  it('随机出号识别特码为属性宇宙（上限 68）', () => {
    expect(isLhcTemaPlayConfig(temaCfg)).toBe(true)
    expect(lhcTemaHcwUniverse()).toHaveLength(68)
    expect(countBetUnits(temaCfg, lhcTemaHcwUniverse().join(','))).toBe(68)
  })

  it('随机出号宇宙上限为 68', () => {
    expect(lhcTemaHcwUniverse()).toHaveLength(68)
    expect(countBetUnits(temaCfg, lhcTemaHcwUniverse().join(','))).toBe(68)
  })

  it('走特码面板而非 49 码点选', () => {
    expect(schemeGroupUsesLhcTemaPanel(temaCfg)).toBe(true)
    expect(schemeGroupUsesPickPanel(temaCfg)).toBe(false)
  })

  it('规范化为 号码|属性|波色，并纠正洪波/绿播', () => {
    expect(parseLhcTemaParts('大,洪波,7,13,绿播')).toEqual({
      nums: ['07', '13'],
      attrs: ['大'],
      waves: ['红波', '绿波'],
    })
    expect(normalizeLhcTemaContent('大,洪波,7,13,绿播')).toBe('07,13|大|红波,绿波')
    expect(normalizeLhcTemaContent('0,49,大')).toBe('49|大|')
    expect(normalizeLhcTemaContent('00,01,02|大|')).toBe('01,02|大|')
    expect(normalizeLhcTemaContent('07||')).toBe('07||')
    expect(normalizeLhcTemaContent('07||,13||')).toBe('07,13||')
  })

  it('全选三段顺序与第三方示例一致', () => {
    const allNums = Array.from({ length: 49 }, (_, i) => String(i + 1).padStart(2, '0')).join(',')
    const attrs =
      '尾双,尾单,尾小,尾大,总分大,总分小,合小,合大,大,小,单,双,合双,合单,总分单,总分双'
    const waves = '红波,蓝波,绿波'
    const raw = `${allNums},${attrs},${waves}`
    expect(normalizeLhcTemaContent(raw)).toBe(`${allNums}|${attrs}|${waves}`)
  })

  it('计注与校验', () => {
    expect(countBetUnits(temaCfg, '大,红波,07')).toBe(3)
    expect(countBetUnits(temaCfg, '07,13|大|红波')).toBe(4)
    const r = validateGroupContent(temaCfg, '合大,蓝波,1')
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.normalized).toBe('01|合大|蓝波')
      expect(r.betUnits).toBe(3)
    }
    expect(parseLhcTemaContentTokens('01|大|红波')).toEqual(['01', '大', '红波'])
  })

  it('只选属性（大/小/单）可保存', () => {
    const r = validateGroupContent(temaCfg, '|大,小,单|')
    expect(r.ok).toBe(true)
    if (r.ok) {
      expect(r.normalized).toBe('|大,小,单|')
      expect(r.betUnits).toBe(3)
    }
  })

  it('开某投某 flat 混选校验并合成三段 wire', () => {
    expect(normalizeLhcTemaFlatContent('01,02,大,03,蓝波')).toBe('01,02,大,03,蓝波')
    expect(normalizeLhcTemaContent('01,02,大,03,蓝波')).toBe('01,02,03|大|蓝波')
    expect(lhcTemaInvalidTokens('01,foo,大,xx')).toEqual(['foo', 'xx'])
    const bad = validateGroupContent(temaCfg, '01,foo,大')
    expect(bad.ok).toBe(false)
    const ok = validateGroupContent(temaCfg, '01,02,大,蓝波')
    expect(ok.ok).toBe(true)
    if (ok.ok) {
      expect(ok.normalized).toBe('01,02|大|蓝波')
      expect(ok.betUnits).toBe(4)
    }
  })
})
