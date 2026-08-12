/** 单选冷热格：再次点击已选项则取消，点击其他项直接覆盖原选项。 */
export function toggleSingleHcwRank(current: number[], next: number): number[] {
  return current.includes(next) ? [] : [next]
}
