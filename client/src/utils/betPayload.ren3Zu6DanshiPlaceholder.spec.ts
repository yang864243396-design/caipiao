import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  countBetUnits,
  groupContentPlaceholder,
  isZu6DanshiConfig,
  isZu6DigitTicket,
  validateGroupContent,
  ZU6_DANSHI_PATTERN_MSG,
} from './betPayload'

const ren3Zu6Ds = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '86',
  catalogSubId: '86',
  betMode: 'danshi',
  playMethodLabel: '任三组六单式',
  inputMode: 'danshi',
  segmentLen: 3,
  segmentLabels: ['选号'],
  renPositionCount: 3,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('任三组六单式 placeholder / 校验 / 注数', () => {
  it('识别为组六单式', () => {
    expect(isZu6DanshiConfig(ren3Zu6Ds)).toBe(true)
  })

  it('提示三位互不相同，示例 012,345', () => {
    const tip = groupContentPlaceholder(ren3Zu6Ds)
    expect(tip).toContain('三个各不相同的3个号码')
    expect(tip).toContain('012,345')
    expect(tip).not.toContain('112,223')
  })

  it('012 通过且计 1 注；210 与 012 同形态去重', () => {
    const r = validateGroupContent(ren3Zu6Ds, '万,千,个\n012')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(1)
    expect(countBetUnits(ren3Zu6Ds, '万,千,个\n012')).toBe(1)

    const dup = validateGroupContent(ren3Zu6Ds, '万,千,个\n012,210')
    expect(dup.ok).toBe(true)
    if (dup.ok) expect(dup.betUnits).toBe(1)
  })

  it('112 组三形态拒绝；111 豹子拒绝', () => {
    const zu3 = validateGroupContent(ren3Zu6Ds, '万,千,个\n112')
    expect(zu3.ok).toBe(false)
    if (!zu3.ok) expect(zu3.message).toBe(ZU6_DANSHI_PATTERN_MSG)
    expect(countBetUnits(ren3Zu6Ds, '万,千,个\n112')).toBe(0)

    const bao = validateGroupContent(ren3Zu6Ds, '万,千,个\n111')
    expect(bao.ok).toBe(false)
    if (!bao.ok) expect(bao.message).toBe(ZU6_DANSHI_PATTERN_MSG)
  })

  it('多选位：C(4,3)×有效组六票', () => {
    const r = validateGroupContent(ren3Zu6Ds, '万,千,百,个\n012,345')
    expect(r.ok).toBe(true)
    if (r.ok) expect(r.betUnits).toBe(4 * 2)
  })

  it('isZu6DigitTicket', () => {
    expect(isZu6DigitTicket('012')).toBe(true)
    expect(isZu6DigitTicket('112')).toBe(false)
    expect(isZu6DigitTicket('111')).toBe(false)
  })
})
