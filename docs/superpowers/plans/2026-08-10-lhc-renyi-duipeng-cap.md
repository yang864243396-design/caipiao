# 二全中任意对碰号码上限 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 对 `renyi_dp` 的 A/B 区号码在前端和后端统一执行去重后总数最多 10 个的校验。

**Architecture:** 前端继续以 `validateLhcRenyiDuipengContent` 作为保存前唯一校验入口，在跨区重复检查之后新增总数边界检查。后端在 `ValidateSchemeBetContent` 的六合彩专用分支中识别 `renyi_dp`，使用同样的 A/B 分区解析规则拒绝超过 10 个有效号码的内容；输入、回填与删除展示仍以 `A|B` 表示。

**Tech Stack:** Vue 3、TypeScript、Vitest、Go、标准库 `testing`。

## Global Constraints

- 仅适用于 `betMode=renyi_dp` 的任意对碰，不改变其它六合彩玩法。
- A/B 区各至少一个有效的 1–49 号码，区内逗号分隔，跨区号码不能重复。
- 以去重后的有效号码数计算上限，合计最大值为 10。
- 不在输入时静默丢弃第 11 个号码；保存时显示明确错误。
- 规范化内容、回填和删除展示必须保持 `A|B`。

---

### Task 1: 前端任意对碰总数校验与提示

**Files:**
- Create: `client/src/utils/betPayload.spec.ts`
- Modify: `client/src/utils/betPayload.ts:1078-1102`
- Modify: `client/src/components/schemes/SchemeLhcRenyiDuipengPanel.vue:82-111`

**Interfaces:**
- Consumes: `validateLhcRenyiDuipengContent(raw: string)` 和 `formatLhcRenyiDuipengContent(a, b)`。
- Produces: 现有校验函数在成功时返回 `{ ok: true, normalized, betUnits }`，超出上限时返回 `{ ok: false, message }`。

- [ ] **Step 1: 写入前端失败测试**

```ts
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
```

- [ ] **Step 2: 运行前端测试确认失败**

Run: `npm.cmd run test -- src/utils/betPayload.spec.ts`（工作目录 `client`）

Expected: 第二个断言失败，现有实现把 11 个号码视为有效内容。

- [ ] **Step 3: 最小化实现前端上限**

在 `validateLhcRenyiDuipengContent` 的跨区重复检查后、生成 `normalized` 前，计算 `sides.a.length + sides.b.length`；大于 10 时返回：

```ts
return { ok: false, message: '任意对碰：A区和B区合计最多选择 10 个号码' }
```

将输入面板说明更新为包含“合计最多 10 个号码”，不改动其 `formatLhcRenyiDuipengContent(a, b)` 的 `A|B` 输出。

- [ ] **Step 4: 运行前端测试确认通过**

Run: `npm.cmd run test -- src/utils/betPayload.spec.ts`（工作目录 `client`）

Expected: 两个断言通过；10 个号码仍返回 25 注，11 个号码被拒绝。

### Task 2: 后端任意对碰总数校验

**Files:**
- Create: `backend/internal/schemes/lhc_renyi_dp_validate_test.go`
- Modify: `backend/internal/schemes/play_universe_validate.go:249-320`

**Interfaces:**
- Consumes: `ValidateSchemeBetContent(kind string, config []byte, content string, maxUnits int) []Violation`。
- Produces: `renyi_dp` 内容在 10 个号码时无违规，在 11 个号码时返回包含“合计最多选择 10 个号码”的违规详情。

- [ ] **Step 1: 写入后端失败测试**

```go
func TestValidateSchemeBetContent_lhcRenyiDuipengMaxTen(t *testing.T) {
	cfg := []byte(`{"playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp","playMethodLabel":"二全中任意对碰"}`)
	if vs := ValidateSchemeBetContent("custom", cfg, "01,02,03,04,05|06,07,08,09,10", 0); len(vs) != 0 {
		t.Fatalf("10 numbers should be valid: %+v", vs)
	}
	vs := ValidateSchemeBetContent("custom", cfg, "01,02,03,04,05,06|07,08,09,10,11", 0)
	if !hasDetailContains(vs, "合计最多选择 10 个号码") {
		t.Fatalf("11 numbers should be rejected: %+v", vs)
	}
}
```

- [ ] **Step 2: 运行后端测试确认失败**

Run: `go test ./internal/schemes -run TestValidateSchemeBetContent_lhcRenyiDuipengMaxTen -count=1`（工作目录 `backend`）

Expected: 11 个号码用例失败，因为现有代码尚未为 `renyi_dp` 建立专用验证分支。

- [ ] **Step 3: 最小化实现后端验证**

在 `ValidateSchemeBetContent` 的六合彩专用分支增加 `renyi_dp` 路由。该分支将：

```go
// 仅接受两段（|，兼容旧 #）；每段按 1–49 解析并去重。
// 空区、跨区重复、任一无效号码或总数大于 10 均返回 ViolationZeroUnits。
// 上限错误详情固定为：任意对碰：A区和B区合计最多选择 10 个号码。
```

保留现有其它对碰玩法的分支和注数逻辑，不将 `renyi_dp` 落到通用 `_dp` 校验。

- [ ] **Step 4: 运行后端测试确认通过**

Run: `go test ./internal/schemes -run TestValidateSchemeBetContent_lhcRenyiDuipengMaxTen -count=1`（工作目录 `backend`）

Expected: 10 个号码通过；11 个号码返回指定上限错误。

### Task 3: 回归验证

**Files:**
- Modify: 上述文件，不增加行为范围。

- [ ] **Step 1: 运行前端相关测试与构建**

Run: `npm.cmd run test -- src/utils/betPayload.spec.ts`，随后 `npm.cmd run build`（工作目录 `client`）。

Expected: 测试和 TypeScript/Vite 构建均通过。

- [ ] **Step 2: 运行后端方案校验测试包**

Run: `go test ./internal/schemes -count=1`（工作目录 `backend`）。

Expected: 新增边界测试及既有方案校验测试通过。

- [ ] **Step 3: 审查工作区变更**

Run: `git diff --check` 和 `git status --short`（工作目录仓库根）。

Expected: 仅包含规格、计划、前端校验/提示/测试与后端校验/测试的预期改动；不暂存或覆盖已有自动生成声明和 npm 缓存。
