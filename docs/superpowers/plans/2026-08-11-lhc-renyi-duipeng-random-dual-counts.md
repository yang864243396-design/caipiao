# 二全中任意对碰随机出号双区数量 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让二全中任意对碰仅在随机出号模式中分别配置并使用 A/B 区随机号码数量。

**Architecture:** 前端把 `randomDraw.counts` 对任意对碰解释为 `[aCount,bCount]`，并兼容旧单总数配置；后端优先识别 `renyi_dp`，从一份 01–49 随机排列连续切出两个区。

**Tech Stack:** Vue 3、TypeScript、Vitest、Go、Go testing。

## Global Constraints

- 仅 `runTypeId=random_draw` 且 `betMode=renyi_dp` 使用双区数量。
- A、B 均不少于 1，`A+B` 不得超过 10。
- 两区取自同一随机排列，禁止跨区重复。
- 旧 `[n]`（2–10）拆为 `[floor(n/2),n-floor(n/2)]`；无效值回退 `[1,1]`。
- 开某投某、冷热出号和固定选号不变。

---

### Task 1: 前端随机双区工具与单元测试

**Files:**
- Modify: `client/src/utils/lhcRenyiDuipengRandom.ts`
- Modify: `client/src/utils/lhcRenyiDuipengRandom.spec.ts`

**Interfaces:**
- Produces: `normalizeLhcRenyiDuipengRandomCounts(raw: unknown): [number, number]`.
- Produces: `randomLhcRenyiDuipengContentForCounts(aCount: number, bCount: number, random?: () => number): string`.

- [ ] **Step 1: Write failing compatibility and non-overlap tests**

```ts
expect(normalizeLhcRenyiDuipengRandomCounts([5])).toEqual([2, 3])
expect(normalizeLhcRenyiDuipengRandomCounts([9, 9])).toEqual([5, 5])
const content = randomLhcRenyiDuipengContentForCounts(4, 6, () => 0)
const sides = parseLhcRenyiDuipengSides(content)!
expect([sides.a.length, sides.b.length]).toEqual([4, 6])
expect(new Set([...sides.a, ...sides.b]).size).toBe(10)
```

- [ ] **Step 2: Run test and verify RED**

Run: `npm.cmd test -- lhcRenyiDuipengRandom.spec.ts`.

Expected: missing helper exports.

- [ ] **Step 3: Implement the helpers**

```ts
const a = Math.min(9, Math.max(1, Math.trunc(Number(values[0])) || 1))
const b = Math.min(10 - a, Math.max(1, Math.trunc(Number(values[1])) || 1))
return [a, b]
```

For a one-item legacy array, preserve its total by splitting it into floor/remaining halves. For generation, shuffle 01–49 once, slice A then B, and format `A|B`.

- [ ] **Step 4: Run test and verify GREEN**

Run: `npm.cmd test -- lhcRenyiDuipengRandom.spec.ts`.

### Task 2: 随机出号面板、回填和预览

**Files:**
- Modify: `client/src/views/play/AdvancedSchemeEditView.vue`
- Test: `client/src/utils/lhcRenyiDuipengRandom.spec.ts`

**Interfaces:**
- Consumes: both Task 1 helpers.
- Produces: `randomDraw.counts=[aCount,bCount]` for random-draw `renyi_dp` only.

- [ ] **Step 1: Add the specialized mode predicate and count normalization**

```ts
const isRdLhcRenyiDuipeng = computed(() =>
  runTypeId.value === 'random_draw' && isLhcRenyiDuipengConfig(schemePlayConfig.value),
)

if (isRdLhcRenyiDuipeng.value) {
  rdCounts.value = [...normalizeLhcRenyiDuipengRandomCounts(rdCounts.value)]
  return
}
```

Use the branch in `ensureRdCounts` and in `applyRandomDrawFromConfig`; do not call it for other run types.

- [ ] **Step 2: Render two bounded steppers and generate the A|B preview**

```vue
<el-input-number v-model="rdCounts[0]" :min="1" :max="10 - rdCounts[1]" />
<el-input-number v-model="rdCounts[1]" :min="1" :max="10 - rdCounts[0]" />
```

Render these rows before generic single/per-position branches. In `generateRdPreview`, call the Task 1 content helper and display the split result as one `A|B` preview.

- [ ] **Step 3: Test and build the client**

Run: `npm.cmd test -- lhcRenyiDuipengRandom.spec.ts` and then `npm.cmd run build` in `client`.

Expected: compatibility tests and Vue typecheck/build pass.

### Task 3: 后端双区随机内容生成

**Files:**
- Create: `backend/internal/schemes/lhc_renyi_dp_random.go`
- Create: `backend/internal/schemes/lhc_renyi_dp_random_test.go`
- Modify: `backend/internal/schemes/worker_pick.go`

**Interfaces:**
- Produces: `lhcRenyiDuipengRandomCounts(cfg parsedSchemeConfig) (a, b int, ok bool)`.
- Produces: `randomLHCRenyiDuipengContent(a, b int) string`.
- Consumes: `cfg.Random.Counts`, `isLHCRenyiDuipengPlayRule`, and `validateLHCRenyiDuipengBetContent`.

- [ ] **Step 1: Write failing Go tests**

```go
cfg := parsedSchemeConfig{Play: playRule{BetMode: "renyi_dp"}, Random: &randomDrawCfg{Counts: []int{4, 6}}}
got := generateRandomDrawContent(cfg, 0)
if vs := validateLHCRenyiDuipengBetContent(got); len(vs) != 0 { t.Fatal(vs) }
if a, b := countSides(got); a != 4 || b != 6 { t.Fatalf("A=%d B=%d", a, b) }
```

Cover legacy `[5] -> (2,3)` and invalid `[0,1]` / `[6,5]` rejection.

- [ ] **Step 2: Run focused test and verify RED**

Run: `go test ./internal/schemes -run 'Test(RandomLHCRenyiDuipengContentUsesSeparateDistinctZones|LHCRenyiDuipengRandomCountsRejectInvalidTotals)' -count=1`.

- [ ] **Step 3: Implement the special generator branch**

```go
if isLHCRenyiDuipengPlayRule(cfg.Play) {
  a, b, ok := lhcRenyiDuipengRandomCounts(cfg)
  if !ok { return "" }
  return randomLHCRenyiDuipengContent(shrinkCount(a, scale, 1), shrinkCount(b, scale, 1))
}
```

Generate one `rand.Perm(49)`, zero-pad its values, slice both zones consecutively, and return empty if `validateLHCRenyiDuipengBetContent` rejects the result.

- [ ] **Step 4: Run focused Go regressions and build**

Run: `go test ./internal/schemes -run 'RenyiDuipeng|RandomDraw' -count=1` and `go build -o server.verify.exe ./cmd/server` in `backend`.

Expected: targeted tests and backend build pass; remove the exact temporary `server.verify.exe` after verification.

### Task 4: Final scope verification

**Files:**
- Modify: only the Task 1–3 files.

- [ ] **Step 1: Confirm saved and legacy configuration behavior**

Save A=4/B=6 and confirm `randomDraw.counts` is `[4,6]`; re-open legacy `[7]` and confirm `[3,4]`. Confirm trigger and hot/cold modes do not show dual random steppers.

- [ ] **Step 2: Check final diff and commit only feature files**

Run: `git diff --check HEAD -- client/src/utils/lhcRenyiDuipengRandom.ts client/src/utils/lhcRenyiDuipengRandom.spec.ts client/src/views/play/AdvancedSchemeEditView.vue backend/internal/schemes/lhc_renyi_dp_random.go backend/internal/schemes/lhc_renyi_dp_random_test.go backend/internal/schemes/worker_pick.go`.
