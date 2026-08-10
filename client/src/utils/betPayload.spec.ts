import { describe, expect, it } from 'vitest'
import { validateLhcRenyiDuipengContent } from './betPayload'

describe('validateLhcRenyiDuipengContent', () => {
  it('accepts ten unique numbers and preserves A|B', () => {
    expect(validateLhcRenyiDuipengContent('01,02,03,04,05|06,07,08,09,10')).toEqual({
      ok: true,
      normalized: '01,02,03,04,05|06,07,08,09,10',
      betUnits: 25,
    })
  })

  it('rejects more than ten unique numbers across both zones', () => {
    expect(validateLhcRenyiDuipengContent('01,02,03,04,05,06|07,08,09,10,11')).toEqual({
      ok: false,
      message: '任意对碰：A区和B区合计最多选择 10 个号码',
    })
  })
})
