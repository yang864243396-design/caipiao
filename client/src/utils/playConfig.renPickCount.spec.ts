import { describe, expect, it } from 'vitest'
import type { PlayTypeNode, SubPlayNode } from '@/types/playCatalog'
import { resolvePlayConfigFromCatalogIds, resolvePlayConfigFromTree } from './playConfig'

describe('任选直选单式位数（renPickCount）', () => {
  it('数字 ruleId 81 → 任三：segmentLen/renPositionCount=3', () => {
    const cfg = resolvePlayConfigFromCatalogIds('renxuan', '81', 'danshi')
    expect(cfg.segmentLen).toBe(3)
    expect(cfg.renPositionCount).toBe(3)
  })

  it('数字 ruleId 75 → 任二：segmentLen/renPositionCount=2', () => {
    const cfg = resolvePlayConfigFromCatalogIds('renxuan', '75', 'danshi')
    expect(cfg.segmentLen).toBe(2)
    expect(cfg.renPositionCount).toBe(2)
  })

  it('数字 ruleId 83 → 任三组三复式：号池 segmentLen=1（勿回退五位）', () => {
    const cfg = resolvePlayConfigFromCatalogIds('renxuan', '83', '')
    expect(cfg.segmentLen).toBe(1)
    expect(cfg.inputMode).toBe('pool')
    expect(cfg.betMode).toBe('zu3')
    expect(cfg.renPositionCount).toBe(3)
  })

  it('subId ren3_zu3_fs + betMode fushi → zuxuan_fs 而非 zhixuan_fs', () => {
    const cfg = resolvePlayConfigFromCatalogIds('renxuan', 'ren3_zu3_fs', 'fushi')
    expect(cfg.segmentLen).toBe(1)
    expect(cfg.inputMode).toBe('pool')
    expect(cfg.subPlayId).toBe('zuxuan_fs')
    expect(cfg.renPositionCount).toBe(3)
  })

  it('g011 + 玩法树 任三直选单式 → 每注 3 位', () => {
    const subNode: SubPlayNode = {
      subId: '81',
      label: '任三直选单式',
      sortOrder: 0,
      betMode: 'danshi',
      outboundPlayCode: '81',
      segmentRule: {
        guajiGroup: '任选',
        guajiTeam: '任选三',
        guajiFullName: '任三直选单式',
        guajiRuleId: '81',
      },
    }
    const typeNode: PlayTypeNode = {
      typeId: 'g011',
      label: '任选',
      sortOrder: 0,
      subPlays: [subNode],
    }
    const cfg = resolvePlayConfigFromTree('ssc_std', typeNode, subNode)
    expect(cfg.segmentLen).toBe(3)
    expect(cfg.renPositionCount).toBe(3)
    expect(cfg.inputMode).toBe('danshi')
  })
})
