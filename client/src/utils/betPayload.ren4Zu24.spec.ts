import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import {
  bareConfigForRenxuanPicks,
  countBetUnits,
  groupContentPlaceholder,
  validateGroupContent,
  zuxuanPoolMinPick,
} from './betPayload'
import { groupDigitInputHint } from './pickPanelOptions'

const ren4Zu24 = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '143',
  catalogSubId: '143',
  betMode: 'zu24',
  playMethodLabel: '任四组选24',
  inputMode: 'pool',
  segmentLen: 1,
  segmentLabels: ['选号'],
  renPositionCount: 4,
  numberPoolMin: 0,
  numberPoolMax: 9,
} as PlayConfig

describe('任四组选24 号池 placeholder / 校验 / 计注', () => {
  it('placeholder 含选位与至少 4 码', () => {
    const tip = groupContentPlaceholder(ren4Zu24)
    expect(tip).toBe(
      '从万、千、百、十、个中勾选至少 4 个，再输入4个及以上0-9的号码，多选用逗号分隔，如：1,3,5,7',
    )
  })

  it('剥位后号池 UI 保持 segmentLen=1，提示为逗号号池', () => {
    const bare = bareConfigForRenxuanPicks(ren4Zu24)
    expect(bare.segmentLen).toBe(1)
    expect(groupDigitInputHint(bare)).toContain('4个及以上')
    expect(groupDigitInputHint(bare)).not.toContain('每一位置皆要输入')
  })

  it('至少 4 码：3 码拒、4 码过；计注 C(n,4)', () => {
    expect(zuxuanPoolMinPick(ren4Zu24)).toBe(4)
    const bad = validateGroupContent(ren4Zu24, '万,千,百,十\n1,2,3')
    expect(bad.ok).toBe(false)
    if (!bad.ok) expect(bad.message).toContain('组选24至少选择 4')

    const ok = validateGroupContent(ren4Zu24, '万,千,百,十\n1,3,5,7')
    expect(ok.ok).toBe(true)
    if (ok.ok) expect(ok.betUnits).toBe(1)

    expect(countBetUnits(ren4Zu24, '万,千,百,十\n1,2,3,4,5')).toBe(5) // C(5,4)=5
    expect(countBetUnits(ren4Zu24, '万,千,百,十,个\n1,2,3,4')).toBe(5) // C(5,4)×1
  })
})
