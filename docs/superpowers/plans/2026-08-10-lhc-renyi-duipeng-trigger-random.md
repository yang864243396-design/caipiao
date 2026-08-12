# 任意对碰高级开某投某随机出号 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让二全中任意对碰在高级开某投某的“全部随机”操作中生成总数 2–10、两区均非空且跨区无重复的 `A|B` 号码。

**Architecture:** 新增一个与 Vue 状态无关的前端纯函数，负责抽取、切分和格式化 01–49 号码。编辑页仅识别任意对碰玩法、配置专用步进器上下限，并在现有“全部随机”分支调用该函数。

**Tech Stack:** Vue 3、TypeScript、Vitest。

## Global Constraints

- 只作用于 `adv_trigger_bet` 的全部随机填充；不改变 `random_draw` 运行类型。
- 总随机号码必须被钳制在 2–10。
- 生成的每个 `A|B` 内容必须通过既有 `validateLhcRenyiDuipengContent` 校验。
- 号码范围为 01–49，区内及跨区均不重复，A/B 各至少一号。

---

### Task 1: 任意对碰随机生成器

**Files:**
- Create: `client/src/utils/lhcRenyiDuipengRandom.ts`
- Create: `client/src/utils/lhcRenyiDuipengRandom.spec.ts`

**Interfaces:**
- Produces: `randomLhcRenyiDuipengContent(total: number, random?: () => number): string`
- Consumes: `formatLhcRenyiDuipengContent` from `client/src/utils/betPayload.ts`

- [ ] **Step 1: Write the failing test**

```ts
it('clamps to 2 and puts one distinct number in each zone', () => {
  const content = randomLhcRenyiDuipengContent(1, () => 0)
  const result = validateLhcRenyiDuipengContent(content)
  expect(result.ok).toBe(true)
  if (result.ok) expect(result.normalized.split(/[|,]/)).toHaveLength(2)
})

it('keeps 3–10 picks distinct and valid', () => {
  for (const total of [3, 10]) {
    const result = validateLhcRenyiDuipengContent(randomLhcRenyiDuipengContent(total))
    expect(result.ok).toBe(true)
    if (result.ok) expect(result.normalized.split(/[|,]/)).toHaveLength(total)
  }
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd run test -- src/utils/lhcRenyiDuipengRandom.spec.ts`

Expected: FAIL because the generator module does not exist.

- [ ] **Step 3: Write minimal implementation**

```ts
export function randomLhcRenyiDuipengContent(total: number, random = Math.random): string {
  const count = Math.min(10, Math.max(2, Math.trunc(total) || 2))
  const pool = Array.from({ length: 49 }, (_, i) => i + 1)
  shuffle(pool, random)
  const split = 1 + Math.floor(random() * (count - 1))
  return formatLhcRenyiDuipengContent(pool.slice(0, split), pool.slice(split, count))
}
```

Use Fisher-Yates shuffling with the supplied random source, format every number as two digits, and keep the helper private.

- [ ] **Step 4: Run test to verify it passes**

Run: `npm.cmd run test -- src/utils/lhcRenyiDuipengRandom.spec.ts`

Expected: PASS.

### Task 2: 高级开某投某接入

**Files:**
- Modify: `client/src/views/play/AdvancedSchemeEditView.vue:1291-1456`
- Test: `client/src/utils/lhcRenyiDuipengRandom.spec.ts`

**Interfaces:**
- Consumes: `isLhcRenyiDuipengConfig` and `randomLhcRenyiDuipengContent(total)`.
- Produces: 正投、反投格的合法 `A|B` 内容。

- [ ] **Step 1: Extend the failing tests**

```ts
it('clamps totals above ten while preserving two non-empty zones', () => {
  const result = validateLhcRenyiDuipengContent(randomLhcRenyiDuipengContent(99))
  expect(result.ok).toBe(true)
  if (result.ok) expect(result.normalized.split(/[|,]/)).toHaveLength(10)
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd run test -- src/utils/lhcRenyiDuipengRandom.spec.ts`

Expected: FAIL until the generator clamps its input to 10.

- [ ] **Step 3: Integrate the generator**

```ts
const renyiDp = isLhcRenyiDuipengConfig(schemePlayConfig.value)
// triggerRandomMin returns 2 for renyiDp; triggerRandomMax returns 10.
// In randomFillTrigger, fill row.pos and row.neg with randomLhcRenyiDuipengContent(count).
```

Leave the existing生尾对碰、组选、整注和普通号码分支 unchanged. Update the completion message to identify the A/B double-zone result.

- [ ] **Step 4: Verify tests and production build**

Run: `npm.cmd run test -- src/utils/lhcRenyiDuipengRandom.spec.ts; npm.cmd run build`

Expected: tests PASS and build exits 0.

### Task 3: Regression checks

**Files:**
- Modify: `client/src/utils/lhcRenyiDuipengRandom.spec.ts`

- [ ] **Step 1: Add repeated generation coverage**

```ts
it('always returns a valid cross-zone-distinct pair', () => {
  for (let i = 0; i < 100; i++) {
    expect(validateLhcRenyiDuipengContent(randomLhcRenyiDuipengContent(7)).ok).toBe(true)
  }
})
```

- [ ] **Step 2: Run focused and existing validation tests**

Run: `npm.cmd run test -- src/utils/lhcRenyiDuipengRandom.spec.ts src/utils/betPayload.spec.ts`

Expected: PASS.

- [ ] **Step 3: Inspect changed files**

Run: `git diff --check; git diff -- client/src/utils/lhcRenyiDuipengRandom.ts client/src/utils/lhcRenyiDuipengRandom.spec.ts client/src/views/play/AdvancedSchemeEditView.vue`

Expected: no whitespace errors; changes limited to the random generator and advanced trigger integration.
