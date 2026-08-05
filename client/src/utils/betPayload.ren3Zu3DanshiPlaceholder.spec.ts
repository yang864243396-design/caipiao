import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  countBetUnits,
  groupContentPlaceholder,
  isZu3DanshiConfig,
  isZu3DigitTicket,
  randomZu3DanshiTickets,
  validateGroupContent,
  ZU3_DANSHI_FORM_COUNT,
  ZU3_DANSHI_PATTERN_MSG,
} from './betPayload'

const ren3Zu3Ds = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '84',
  catalogSubId: '84',
  betMode: 'danshi',
  playMethodLabel: '任三组三单式',
  inputMode: 'danshi',
  segmentLen: 3,
  segmentLabels: ['选号'],
  renPositionCount: 3,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('任三组三单式 placeholder / 校验 / 注数', () => {
  it('识别为组三单式', () => {
    expect(isZu3DanshiConfig(ren3Zu3Ds)).toBe(true)
  })

  it('提示两同号+一异号，示例 112,223', () => {
    const tip = groupContentPlaceholder(ren3Zu3Ds)
    expect(tip).toContain('两个号相同号码和一个不同号码')
    expect(tip).toContain('112,223')
    expect(tip).not.toContain('012,345')
  })

  it('112 通过且计 1 注；121 与 112 同形态去重', () => {
    const r = validateGroupContent(ren3Zu3Ds, '万,千,个\n112')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(1)
    expect(countBetUnits(ren3Zu3Ds, '万,千,个\n112')).toBe(1)

    const dup = validateGroupContent(ren3Zu3Ds, '万,千,个\n112,121')
    expect(dup.ok).toBe(true)
    if (dup.ok) expect(dup.betUnits).toBe(1)
  })

  it('012 组六形态拒绝；111 豹子拒绝', () => {
    const six = validateGroupContent(ren3Zu3Ds, '万,千,个\n012')
    expect(six.ok).toBe(false)
    if (!six.ok) expect(six.message).toBe(ZU3_DANSHI_PATTERN_MSG)
    expect(countBetUnits(ren3Zu3Ds, '万,千,个\n012')).toBe(0)

    const bao = validateGroupContent(ren3Zu3Ds, '万,千,个\n111')
    expect(bao.ok).toBe(false)
    if (!bao.ok) expect(bao.message).toBe(ZU3_DANSHI_PATTERN_MSG)
  })

  it('多选位：C(4,3)×有效组三票', () => {
    const r = validateGroupContent(ren3Zu3Ds, '万,千,百,个\n112,223')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(4 * 2)
  })

  it('随机出号：生成的整注均为组三形态且形态去重', () => {
    const raw = randomZu3DanshiTickets(20)
    const parts = raw.split(',').filter(Boolean)
    expect(parts.length).toBe(20)
    const forms = new Set<string>()
    for (const t of parts) {
      expect(isZu3DigitTicket(t)).toBe(true)
      forms.add([...t].sort().join(''))
    }
    expect(forms.size).toBe(parts.length)

    const full = randomZu3DanshiTickets(ZU3_DANSHI_FORM_COUNT + 10)
    expect(full.split(',').filter(Boolean).length).toBe(ZU3_DANSHI_FORM_COUNT)
  })
})
