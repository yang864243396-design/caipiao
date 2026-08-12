export interface BuiltinPlanSaveValidation {
  ok: boolean
  skipManualContentValidation: boolean
  message?: string
}

/** 内置计划的投注内容由收藏快照物化，不能按手工选号内容校验。 */
export function validateBuiltinPlanSave(
  runTypeId: string,
  snapshotId: string,
): BuiltinPlanSaveValidation {
  if (runTypeId !== 'builtin_plan') {
    return { ok: true, skipManualContentValidation: false }
  }
  if (!snapshotId.trim()) {
    return {
      ok: false,
      skipManualContentValidation: true,
      message: '请先在方案内容中选择一个收藏方案',
    }
  }
  return { ok: true, skipManualContentValidation: true }
}
