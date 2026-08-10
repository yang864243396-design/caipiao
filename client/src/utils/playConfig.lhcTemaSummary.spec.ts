import { describe, expect, it } from 'vitest'
import type { PlayTypeNode, SubPlayNode } from '@/types/playCatalog'
import { playConfigSummary } from './betPayload'
import { resolvePlayConfigFromTree } from './playConfig'

describe('六合彩特码A 方案内容摘要', () => {
  it('玩法树解析为 特码 · 特码A，而非前三·272', () => {
    const typeNode: PlayTypeNode = {
      typeId: 'g001',
      label: '特码',
      sortOrder: 1,
      subPlays: [],
    }
    const subNode: SubPlayNode = {
      subId: '272',
      label: '特码A',
      sortOrder: 1,
      betMode: 'tema',
      outboundPlayCode: '272',
      segmentRule: { guajiGroup: '特码', guajiFullName: '特码A', guajiRuleId: '272' },
    }
    typeNode.subPlays = [subNode]
    const cfg = resolvePlayConfigFromTree('lhc_std', typeNode, subNode)
    expect(cfg.playTypeLabel).toBe('特码')
    expect(cfg.playMethodLabel).toBe('特码A')
    expect(playConfigSummary(cfg)).toBe('特码 · 特码A')
    expect(playConfigSummary(cfg)).not.toMatch(/前三/)
  })
})
