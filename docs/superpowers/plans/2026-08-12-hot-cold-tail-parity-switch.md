# 冷热出号尾数单双直接切换 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在冷热出号的尾数单双玩法中，点击另一项直接替换已选项。

**Architecture:** 在运行类型判定工具中新增一个仅识别“尾数单双”的纯函数。方案编辑页使用该判定决定是否调用已有的 `toggleSingleHcwRank`，因此保留现有 ranks、pool、保存和注数计算链路。

**Tech Stack:** Vue 3、TypeScript、Vitest、@vue/test-utils。

## Global Constraints

- 范围仅限冷热出号的尾数单双。
- 再次点击当前项必须取消选择。
- 不改变保存格式、ranks/pool 数据结构或注数计算。
- 不修改五星和值单双、前后二大小单双和过关玩法的现有交互。

---

### Task 1: 尾数单双玩法判定

**Files:**
- Modify: `client/src/utils/runTypeMatrix.ts`
- Modify: `client/src/utils/runTypeMatrix.triggerPos.spec.ts`

**Interfaces:**
- Produces: `isTailParityPlayConfig(config: AdvTriggerPosConfig): boolean`
- Consumes: `AdvTriggerPosConfig` 的 `playMethodLabel`、`playTypeLabel`、`guajiGroup`、`playTypeId`、`catalogSubId`、`subPlayId`。

- [ ] **Step 1: 写入失败测试**

```ts
import { isTailParityPlayConfig } from './runTypeMatrix'

it('recognizes hash tail parity but not sum parity', () => {
  expect(isTailParityPlayConfig({ playMethodLabel: '尾数单双' })).toBe(true)
  expect(isTailParityPlayConfig({ playMethodLabel: '和值单双' })).toBe(false)
  expect(isTailParityPlayConfig({ playTypeId: 'g017', subPlayId: '267' })).toBe(true)
})
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `npm.cmd test -- src/utils/runTypeMatrix.triggerPos.spec.ts`

Expected: FAIL，因为 `isTailParityPlayConfig` 尚未导出。

- [ ] **Step 3: 编写最小实现**

```ts
export function isTailParityPlayConfig(config: AdvTriggerPosConfig): boolean {
  const label = `${config.playMethodLabel ?? ''} ${config.playTypeLabel ?? ''} ${config.guajiGroup ?? ''}`
  if (/尾数单双/.test(label)) return true
  const sid = String(config.catalogSubId ?? config.subPlayId ?? '').trim()
  return String(config.playTypeId ?? '').trim() === 'g017' && ['267', '387'].includes(sid)
}
```

- [ ] **Step 4: 运行测试并确认通过**

Run: `npm.cmd test -- src/utils/runTypeMatrix.triggerPos.spec.ts`

Expected: PASS。

- [ ] **Step 5: 提交判定与测试**

```powershell
git add -- client/src/utils/runTypeMatrix.ts client/src/utils/runTypeMatrix.triggerPos.spec.ts
git commit -m "test: cover hot-cold tail parity detection"
```

### Task 2: 使用单选替换状态转换

**Files:**
- Modify: `client/src/views/play/AdvancedSchemeEditView.vue:2889-2934`
- Test: `client/src/utils/hcwRankSelection.spec.ts`

**Interfaces:**
- Consumes: `isTailParityPlayConfig(config)` 与 `toggleSingleHcwRank(current, next)`。
- Produces: 在 `toggleHcwDigit` 中，尾数单双当前项外的点击会写入唯一的新 rank。

- [ ] **Step 1: 保留既有单选状态转换测试作为回归覆盖**

```ts
import { toggleSingleHcwRank } from './hcwRankSelection'

it('switches tail parity rank from odd to even in one click', () => {
  const oddRank = 0
  const evenRank = 1
  expect(toggleSingleHcwRank([oddRank], evenRank)).toEqual([evenRank])
  expect(toggleSingleHcwRank([evenRank], evenRank)).toEqual([])
})
```

- [ ] **Step 2: 运行既有状态转换测试**

Run: `npm.cmd test -- src/utils/hcwRankSelection.spec.ts`

Expected: PASS；该工具已覆盖“替换”和“再次点击取消”。Task 1 的新增玩法判定测试提供本次变更所需的红灯。

- [ ] **Step 3: 最小修改编辑页状态分支**

```ts
const replaceSingle =
  cap === 1 && (isHcwLhcGuoguan.value || isTailParityPlayConfig(schemePlayConfig.value))
if (replaceSingle) {
  ranks.splice(0, ranks.length, ...toggleSingleHcwRank(ranks, rank))
} else if (cap != null && ranks.length >= cap) {
  ElMessage.warning(hcwPosPickCapMsg())
  return
}
```

- [ ] **Step 4: 运行目标测试并确认通过**

Run: `npm.cmd test -- src/utils/hcwRankSelection.spec.ts src/utils/runTypeMatrix.triggerPos.spec.ts`

Expected: PASS，且已有“点击同一项取消”覆盖保持通过。

- [ ] **Step 5: 提交交互修改**

```powershell
git add -- client/src/views/play/AdvancedSchemeEditView.vue client/src/utils/hcwRankSelection.spec.ts
git commit -m "fix: switch hot-cold tail parity directly"
```

### Task 3: 全量前端验证

**Files:**
- Verify only: `client/src/views/play/AdvancedSchemeEditView.vue`
- Verify only: `client/src/utils/runTypeMatrix.ts`

**Interfaces:**
- Consumes: 已完成的 Task 1 和 Task 2。
- Produces: 可构建的前端产物。

- [ ] **Step 1: 运行全量前端测试**

Run: `npm.cmd test`

Expected: PASS。

- [ ] **Step 2: 运行类型检查与生产构建**

Run: `npm.cmd run build`

Expected: PASS，`vue-tsc --noEmit` 和 `vite build` 均无错误。

- [ ] **Step 3: 检查提交内容**

```powershell
git diff --check
git status --short
git log --oneline -3
```

Expected: 不存在空白错误；仅包含本计划的提交。
