# 幸运庄闲单选 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让幸运庄闲以“庄、和、闲”三个单选项贯穿所有方案内容模式。

**Architecture:** 前端以 `textPickOptionsForConfig` 作为所有文字选号面板的唯一选项源，并由 `poolMaxPicksForConfig` 强制单选。后端用同一组选项更新候选池、随机出号、注数计算、保存校验及结算判定。

**Tech Stack:** Vue 3、TypeScript、Vitest。

## Global Constraints

- 幸运庄闲仅允许一个选项。
- 保存和投注内容均为一个文字 token。
- 不修改其他玩法的选项或第三方报文。

---

### Task 1: 扩展幸运庄闲文字选项并覆盖回归测试

**Files:**
- Modify: `client/src/utils/pickPanelOptions.ts:191-245,847-865`
- Modify: `client/src/utils/betPayload.ts:1485-1753,4124-4240`
- Modify: `backend/internal/schemes/worker_pick.go:661-755,858-885`
- Modify: `backend/internal/schemes/play_api.go:220-240,312-380`
- Modify: `backend/internal/schemes/ssc_special_eval.go:25-70,136-170`
- Modify: `backend/internal/schemes/bet_units_cap.go:260-400`
- Modify: `backend/internal/schemes/fast_ssc_coverage_test.go:31-38`
- Create: `client/src/utils/pickPanelOptions.zhuangxian.spec.ts`
- Create: `backend/internal/schemes/zhuangxian_test.go`

**Interfaces:**
- Consumes: `textPickOptionsForConfig(config: PlayConfig): string[]` 和后端 `attributeUniverse(rule playRule) []string`。
- Produces: `zhuangxian` 前后端均返回 `['庄', '和', '闲']`；`togglePoolPick` 在单选上限下替换已选项；后端按一注处理“和”。

- [ ] **Step 1: 写入失败测试**

```ts
expect(textPickOptionsForConfig(zhuangxian)).toEqual(['庄', '和', '闲'])
expect(togglePoolPick(['庄'], '和', 1)).toEqual(['和'])
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `npm.cmd test -- src/utils/pickPanelOptions.zhuangxian.spec.ts`

Expected: 选项断言失败，实际值为 `['庄', '闲']`。

- [ ] **Step 3: 写入最小实现**

```ts
case 'zhuangxian':
  return ['庄', '和', '闲']
```

并将 `zhuangxian` 加入前端单选校验，以及后端候选池、随机上限、注数、保存校验和结算映射。

- [ ] **Step 4: 运行测试并确认通过**

Run: `npm.cmd test -- src/utils/pickPanelOptions.zhuangxian.spec.ts`

Expected: 全部通过。

- [ ] **Step 5: 执行前端编译校验**

Run: `npm.cmd run build`

Expected: `vue-tsc --noEmit && vite build` 成功。
