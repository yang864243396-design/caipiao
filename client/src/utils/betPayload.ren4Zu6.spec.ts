import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  bareConfigForRenxuanPicks,
  countBetUnits,
  groupContentPlaceholder,
  isSixingZu6PlayConfig,
  validateGroupContent,
  zuxuanPoolMinPick,
  zuxuanPoolMinPickMessage,
} from './betPayload'
import { groupDigitInputHint } from './pickPanelOptions'

const ren4Zu6 = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '145',
  catalogSubId: '145',
  betMode: 'zu6',
  playMethodLabel: '任选四组选6',
  inputMode: 'pool',
  segmentLen: 1,
  segmentLabels: ['选号'],
  renPositionCount: 4,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('任四组选6 号池 placeholder / 校验 / 计注', () => {
  it('识别为四星/任四组选6，最少 2 码', () => {
    expect(isSixingZu6PlayConfig(ren4Zu6)).toBe(true)
    expect(zuxuanPoolMinPick(ren4Zu6)).toBe(2)
    expect(zuxuanPoolMinPickMessage(ren4Zu6)).toBe('组选6至少选择 2 个号码')
  })

  it('placeholder 要求选位 + 两个及以上号码', () => {
    expect(groupContentPlaceholder(ren4Zu6)).toBe(
      '从万、千、百、十、个中勾选至少 4 个、最多 5 个位置，再输入两个及以上的0-9的号码，多选用逗号分隔，如1,2',
    )
  })

  it('剥位后号池提示为两个及以上', () => {
    const bare = bareConfigForRenxuanPicks(ren4Zu6)
    expect(groupDigitInputHint(bare)).toContain('两个及以上')
    expect(groupDigitInputHint(bare)).not.toContain('三个及以上')
  })

  it('至少 2 码：1 码拒、2 码过；计注 C(n,2)×C(选位数,4)', () => {
    const bad = validateGroupContent(ren4Zu6, '万,千,百,十\n1')
    expect(bad.ok).toBe(false)
    if (!bad.ok) expect(bad.message).toContain('组选6至少选择 2')

    const ok = validateGroupContent(ren4Zu6, '万,千,百,十\n1,2')
    expect(ok.ok).toBe(true)
    if (ok.ok) expect(ok.betUnits).toBe(1) // C(2,2 号码)×C(4,4 选位)=1

    expect(countBetUnits(ren4Zu6, '万,千,百,十\n1,2,3')).toBe(3) // C(3,2)×1
    expect(countBetUnits(ren4Zu6, '万,千,百,十,个\n1,2')).toBe(5) // C(5,4)×1
  })

  it('三星组六仍最少 3 码', () => {
    const zhong3Zu6 = {
      playTemplate: 'ssc_std',
      playTypeId: 'g002',
      subPlayId: '261',
      betMode: 'zu6',
      playMethodLabel: '中三组六',
      segmentLen: 1,
      segmentLabels: ['选号'],
    } as PlayConfig
    expect(isSixingZu6PlayConfig(zhong3Zu6)).toBe(false)
    expect(zuxuanPoolMinPick(zhong3Zu6)).toBe(3)
  })

  it('目录文案为「组六」+ ren4 / 剥位后仍按组选6≥2', () => {
    const shortLabel = {
      ...ren4Zu6,
      playMethodLabel: '组六',
      subPlayId: 'zuxuan_fs',
      catalogSubId: '145',
    } as PlayConfig
    expect(isSixingZu6PlayConfig(shortLabel)).toBe(true)
    const bare = bareConfigForRenxuanPicks(shortLabel)
    expect(isSixingZu6PlayConfig(bare)).toBe(true)
    expect(zuxuanPoolMinPick(bare)).toBe(2)
    expect(zuxuanPoolMinPickMessage(bare)).toBe('组选6至少选择 2 个号码')
    expect(validateGroupContent(shortLabel, '万,千,百,十\n1,2').ok).toBe(true)
  })

  it('无 catalogSubId 时靠 renPositionCount≥4 + zu6 识别', () => {
    const noCat = {
      playTemplate: 'ssc_std',
      playTypeId: 'g011',
      playTypeLabel: '任选',
      guajiGroup: '任选',
      subPlayId: 'zuxuan_fs',
      betMode: 'zu6',
      playMethodLabel: '组六',
      inputMode: 'pool',
      segmentLen: 1,
      segmentLabels: ['选号'],
      renPositionCount: 4,
    } as PlayConfig
    expect(isSixingZu6PlayConfig(noCat)).toBe(true)
    // 剥位后 renPositionCount 被清掉，须仍能靠其它信号识别（否则会误报组六≥3）
    const bare = bareConfigForRenxuanPicks(noCat)
    expect(isSixingZu6PlayConfig(bare)).toBe(true)
    expect(validateGroupContent(noCat, '万,千,百,十\n1,2').ok).toBe(true)
  })
})
