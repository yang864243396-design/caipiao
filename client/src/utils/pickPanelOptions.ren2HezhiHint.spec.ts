import { describe, expect, it } from 'vitest'
import type { PlayConfig } from './betPayload'
import { bareConfigForRenxuanPicks, groupContentPlaceholder } from './betPayload'
import { groupDigitInputHint } from './pickPanelOptions'

const ren2Hezhi = {
  playTemplate: 'ssc_std',
  playTypeId: 'g011',
  playTypeLabel: '任选',
  guajiGroup: '任选',
  subPlayId: '76',
  catalogSubId: '76',
  betMode: 'hezhi',
  playMethodLabel: '直选和值',
  inputMode: 'pool',
  segmentLen: 1,
  renPositionCount: 2,
  numberPoolMin: 0,
  numberPoolMax: 18,
  segmentLabels: ['选号'],
} as PlayConfig

const ren2ZuxuanFs = {
  ...ren2Hezhi,
  subPlayId: '77',
  catalogSubId: '77',
  betMode: 'zuxuan_fs',
  playMethodLabel: '组选复式',
  numberPoolMax: 9,
} as PlayConfig

describe('任二直选和值 / 组选复式录入提示', () => {
  it('任二直选和值提示为和值区间，而非「3 个及以上」', () => {
    const bare = bareConfigForRenxuanPicks(ren2Hezhi)
    const tip = groupDigitInputHint(bare)
    expect(tip).toMatch(/和值/)
    expect(tip).toMatch(/0/)
    expect(tip).toMatch(/18/)
    expect(tip).not.toMatch(/及以上/)
    expect(groupContentPlaceholder(ren2Hezhi)).toMatch(/和值/)
  })

  it('任二组选复式剥位后提示至少 2 个号', () => {
    const bare = bareConfigForRenxuanPicks(ren2ZuxuanFs)
    const tip = groupDigitInputHint(bare)
    expect(tip).toMatch(/2 个及以上/)
    expect(tip).not.toMatch(/3 个及以上/)
  })
})
