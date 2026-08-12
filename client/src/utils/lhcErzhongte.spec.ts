import { describe, expect, it } from 'vitest'
import { validateGroupContent } from '@/utils/betPayload'

const erzhongteFushi = {
  playTemplate: 'lhc_std',
  playTypeId: 'erzhongte',
  catalogSubId: '285',
	 subPlayId: '285',
  betMode: 'fushi',
  inputMode: 'lhc_num' as const,
  playTypeLabel: '二中特',
  playMethodLabel: '复式',
	 segmentLen: 1,
	 segmentLabels: ['选号'],
}

describe('二中特方案内容', () => {
	 it('复式沿用六合彩号码池并以二中特名称校验', () => {
    const invalid = validateGroupContent(erzhongteFushi, '01')
    expect(invalid.ok).toBe(false)
	 if (invalid.ok) throw new Error('single number must be invalid')
    expect(invalid.message).toContain('二中特复式')

    const valid = validateGroupContent(erzhongteFushi, '01,13')
    expect(valid).toMatchObject({ ok: true, normalized: '01,13', betUnits: 1 })
  })
})

describe('连码其余子玩法方案内容', () => {
  it('特串复式使用二全中同样的 01–49 号码池校验', () => {
    const config = { ...erzhongteFushi, playTypeId: 'g003', catalogSubId: '291', subPlayId: '291', playTypeLabel: '连码', playMethodLabel: '特串复式' }
    const invalid = validateGroupContent(config, '01')
    expect(invalid.ok).toBe(false)
    if (invalid.ok) throw new Error('single number must be invalid')
    expect(invalid.message).toContain('特串复式')
  })

  it('三中二复式至少选择三个号码', () => {
    const config = { ...erzhongteFushi, playTypeId: 'g003', catalogSubId: '297', subPlayId: '297', playTypeLabel: '连码', playMethodLabel: '三中二复式' }
    const invalid = validateGroupContent(config, '01,02')
    expect(invalid.ok).toBe(false)
    if (invalid.ok) throw new Error('two numbers must be invalid')
    expect(invalid.message).toContain('三中二复式')

    expect(validateGroupContent(config, '01,02,03')).toMatchObject({ ok: true, betUnits: 1 })
  })
})
