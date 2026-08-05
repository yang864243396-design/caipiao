import { describe, expect, it } from 'vitest'
import type { PlayTypeNode, SubPlayNode } from '@/types/playCatalog'
import { isSscDanshiLikeConfig } from './betPayload'
import { resolvePlayConfigFromTree } from './playConfig'
import {
  commitSchemeGroupContentOnBlur,
  schemeGroupUsesDanshiTextInput,
  schemeGroupUsesDigitInput,
  schemeGroupUsesPickPanel,
  schemeGroupUsesTextInputPanel,
} from './pickPanelOptions'

describe('前后三直选单式失焦校验', () => {
  const typeNode = {
    typeId: 'g012',
    label: '前后三',
    sortOrder: 12,
    enabled: true,
  } as PlayTypeNode
  const subNode = {
    subId: '90',
    label: '直选单式',
    sortOrder: 2,
    enabled: true,
    segmentRule: {
      guajiGroup: '前后三',
      guajiTeam: '前后三直选',
      guajiFullName: '前后三直选单式',
      guajiRuleId: '90',
    },
  } as SubPlayNode

  const cfg = resolvePlayConfigFromTree('ssc_std', typeNode, subNode)

  it('识别为单式整注文本面板（非复式按位号池）', () => {
    expect(cfg.betMode).toBe('danshi')
    expect(cfg.inputMode).toBe('danshi')
    expect(cfg.segmentLen).toBe(3)
    expect(isSscDanshiLikeConfig(cfg)).toBe(true)
    expect(schemeGroupUsesPickPanel(cfg)).toBe(false)
    expect(schemeGroupUsesDigitInput(cfg)).toBe(false)
    expect(schemeGroupUsesDanshiTextInput(cfg)).toBe(true)
    expect(schemeGroupUsesTextInputPanel(cfg)).toBe(true)
  })

  it('失焦：去重、剔非法位数；全废票清空', () => {
    expect(commitSchemeGroupContentOnBlur('1234,012,12,345', cfg)).toBe('012,345')
    expect(commitSchemeGroupContentOnBlur('012,012,345', cfg)).toBe('012,345')
    expect(commitSchemeGroupContentOnBlur('1234,5678', cfg)).toBe('')
    expect(commitSchemeGroupContentOnBlur('01', cfg)).toBe('01')
  })
})
