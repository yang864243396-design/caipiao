/** Guaji 实际下注金额：从第三位小数直接截断到两位，绝不四舍五入。 */
export function truncateBetAmount(amount: number): number {
  if (!Number.isFinite(amount) || amount <= 0) return 0
  return Math.floor((amount + Number.EPSILON) * 100) / 100
}

/** 实际下注金额的统一展示，始终保留两位小数。 */
export function formatBetAmountFixed2(amount: number): string {
  return truncateBetAmount(amount).toFixed(2)
}
