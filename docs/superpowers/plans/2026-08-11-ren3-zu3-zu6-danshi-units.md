# 任三组三组六单式注数 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让任选任三组三/组六单式保留整注形态并正确计算注数。

**Architecture:** 在 `validateGroupContent` 的组选号码池预校验处排除组三和组六单式，使其进入下方已有的任选单式路线。该路线已负责解析选位、形态去重和 `C(n,3)` 乘数。

**Tech Stack:** Vue 3、TypeScript、Vitest。

## Global Constraints

- 仅修改任三组三/组六单式的路由条件。
- 不改变号码形态规则或其他组选号码池玩法。

---

### Task 1: 修复任选组三/组六单式计注路由

**Files:**
- Modify: `client/src/utils/betPayload.ts:3467-3498`
- Test: `client/src/utils/betPayload.ren3Zu3DanshiPlaceholder.spec.ts`
- Test: `client/src/utils/betPayload.ren3Zu6DanshiPlaceholder.spec.ts`

- [ ] **Step 1: 运行既有失败用例**

Run: `npm.cmd test -- betPayload.ren3Zu3DanshiPlaceholder.spec.ts betPayload.ren3Zu6DanshiPlaceholder.spec.ts`

Expected: 三组与六组单式案例因提前号码池归一化而得到 0 注或错误提示。

- [ ] **Step 2: 排除单式玩法的号码池预校验**

```ts
const zuxuanMin = isZu3DanshiConfig(config) || isZu6DanshiConfig(config)
  ? null
  : zuxuanPoolMinPick(config)
```

- [ ] **Step 3: 运行回归用例**

Run: `npm.cmd test -- betPayload.ren3Zu3DanshiPlaceholder.spec.ts betPayload.ren3Zu6DanshiPlaceholder.spec.ts`

Expected: 12 项通过；单票 1 注、四选位两票 8 注、形态不符仍拒绝。

- [ ] **Step 4: 运行前端构建**

Run: `npm.cmd run build`

Expected: `vue-tsc --noEmit && vite build` exit 0。
