/** 单选冷热格：再次点击已选项则取消，点击其他项直接覆盖原选项。 */
export function toggleSingleHcwRank(current: number[], next: number): number[] {
  return current.includes(next) ? [] : [next]
}

/** 仅过关和尾数单双的单选格允许点击另一项时直接替换。 */
export function shouldDirectSwitchHcwRank(
  cap: number | null,
  isLhcGuoguan: boolean,
  isTailParity: boolean,
): boolean {
  return cap === 1 && (isLhcGuoguan || isTailParity)
}
