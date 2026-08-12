import { describe, expect, it } from 'vitest'
import { countBetUnits, type PlayConfig, validateGroupContent } from './betPayload'
import { lhcGuoguanAttrsForNumber, randomLhcGuoguanContent } from '@/constants/lhcPlay'

const guoguanCfg = {
  playTemplate: 'lhc_std',
  playTypeId: 'g004',
  catalogSubId: 'guoguan',
  subPlayId: 'guoguan',
  betMode: 'guoguan',
  playTypeLabel: '过关',
  playMethodLabel: '过关',
  segmentLen: 1,
  segmentLabels: ['过关'],
  inputMode: 'lhc_attr',
} as PlayConfig

describe('六合彩过关方案内容', () => {
  it('按后端相同口径计算正码属性', () => {
    expect(lhcGuoguanAttrsForNumber(26)).toEqual(['大', '双', '蓝波'])
    expect(lhcGuoguanAttrsForNumber(49)).toEqual(['大', '单', '绿波'])
  })

  it('保留六个正码空位并按一注计数', () => {
    const content = '大,单,,红波,,绿波'
    const result = validateGroupContent(guoguanCfg, content)

    expect(result).toEqual({ ok: true, normalized: content, betUnits: 1 })
    expect(countBetUnits(guoguanCfg, content)).toBe(1)
  })

  it('至少选择两个位置且每位只能是允许的属性', () => {
    expect(validateGroupContent(guoguanCfg, '大,,,,,')).toEqual({
      ok: false,
      message: '过关：至少选择 2 个正码位置',
    })
    expect(validateGroupContent(guoguanCfg, '大,单,,豹子,,双')).toEqual({
      ok: false,
      message: '过关：第 4 个位置只能选择大/小/单/双/红波/蓝波/绿波',
    })
  })

  it('高级开某投某随机可只填一个正码位置，并始终保留六位格式', () => {
    const content = randomLhcGuoguanContent(1, () => 0)
    const positions = content.split(',')

    expect(positions).toHaveLength(6)
    expect(positions.filter(Boolean)).toHaveLength(1)
    expect(positions.find(Boolean)).toBe('大')
  })
})
