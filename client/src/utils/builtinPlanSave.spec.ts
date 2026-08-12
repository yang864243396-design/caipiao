import { describe, expect, it } from 'vitest'
import { validateBuiltinPlanSave } from './builtinPlanSave'

describe('validateBuiltinPlanSave', () => {
  it('内置计划已选择收藏快照时跳过手工方案内容校验', () => {
    expect(validateBuiltinPlanSave('builtin_plan', ' SD10001 ')).toEqual({
      ok: true,
      skipManualContentValidation: true,
    })
  })

  it('内置计划未选择收藏快照时阻止保存', () => {
    expect(validateBuiltinPlanSave('builtin_plan', '   ')).toEqual({
      ok: false,
      skipManualContentValidation: true,
      message: '请先在方案内容中选择一个收藏方案',
    })
  })

  it('普通运行类型继续校验手工方案内容', () => {
    expect(validateBuiltinPlanSave('fixed_rotate', '')).toEqual({
      ok: true,
      skipManualContentValidation: false,
    })
  })
})
