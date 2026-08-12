# 浜屽叏涓换鎰忓纰板喎鐑弻鍖哄嚭鍙?Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 璁╀换鎰忓纰板喎鐑柟妗堝垎鍒€夋嫨 A銆丅 鍖哄喎鐑悕娆★紝骞跺湪杩愯鏃剁敓鎴愭湁鏁堢殑 `A|B` 鎶曟敞鍐呭銆俓n
**Architecture:** 澶嶇敤 `hotColdWarm.ranks` 鐨勪簩缁存暟缁勶紝`ranks[0]`銆乣ranks[1]` 鍒嗗埆琛ㄧず A銆丅 鍖恒€傚悗绔负浠绘剰瀵圭澧炲姞涓撶敤鍐风儹鏋勯€犲櫒锛屼粠鍚屼竴 01鈥?9 鎺掑悕涓彇涓ゅ尯鍙风爜锛涘墠绔互涓ょ粍鍏变韩缁熻鐨勯€夊彿闈㈡澘缂栬緫杩欎袱琛屽悕娆″苟鍦ㄦ彁浜ゅ墠寮哄埗鍙屽尯绾︽潫銆俓n
**Tech Stack:** Vue 3 + TypeScript + Vitest锛汫o + Go testing銆俓n
## Global Constraints

- A銆丅 涓ゅ尯鍚勮嚦灏戜竴涓悕娆★紝鍚嶆涓嶅彲璺ㄥ尯閲嶅锛屽悎璁℃渶澶氬崄涓€俓n- 涓ゅ尯浣跨敤鍚屼竴鍐风儹鎺掑悕锛岃繍琛屾椂鍐呭蹇呴』涓?`A鍙风爜|B鍙风爜`銆俓n- 鏃т换鎰忓纰板崟琛?ranks 涓嶄綔鐚滄祴锛岃繍琛屾椂璺宠繃锛屼笉鐢熸垚闈炴硶鎶曟敞銆俓n- 涓嶆敼鏁版嵁搴?schema锛涘叾瀹冪帺娉曠殑 ranks 璇箟涓嶅彉銆俓n
---

### Task 1: 鍚庣浠绘剰瀵圭鍐风儹鏋勯€犱笌闃插尽鏍￠獙

**Files:**
- Modify: `backend/internal/schemes/worker_pick.go:1504-1600`
- Modify: `backend/internal/schemes/scheme_audit_api.go:142-197`
- Create: `backend/internal/schemes/lhc_renyi_dp_run_modes_test.go`

**Interfaces:**
- Consumes: `hotColdWarmCfg.Ranks [][]int`銆乣isLHCRenyiDuipengPlayRule`銆乣validateLHCRenyiDuipengBetContent`銆俓n- Produces: `buildLHCRenyiDuipengHotColdPickContent(cfg, draws) (string, bool)`锛涜繑鍥?`ok=true` 鏃跺唴瀹逛负绌鸿〃绀洪厤缃笉瀹屾暣锛岃皟鐢ㄦ柟涓嶅緱鍥為€€鍒伴€氱敤鍗曞尯鍑哄彿銆俓n
- [ ] **Step 1: 鍐欏け璐ユ祴璇?*

```go
func TestLHCRenyiDuipengHotColdPickBuildsTwoZones(t *testing.T) {
    cfg := pickTestConfig(t, `{"runTypeId":"hot_cold_warm","playTemplate":"lhc_std","typeId":"g003","subId":"284","betMode":"renyi_dp","hotColdWarm":{"ranks":[[0,1],[2]]}}`)
    got := buildHotColdPickContent(cfg, [][]string{{"1", "2", "3", "4", "5", "6", "7"}, {"1", "2", "3", "8", "9", "10", "11"}})
    if got != "1,2|3" { t.Fatalf("content=%q want 1,2|3", got) }
    if vs := validateLHCRenyiDuipengBetContent(got); len(vs) != 0 { t.Fatalf("invalid=%+v", vs) }
}
```

- [ ] **Step 2: 楠岃瘉澶辫触**

Run: `go test ./internal/schemes -run TestLHCRenyiDuipengHotColdPickBuildsTwoZones -count=1`
Expected: FAIL锛屽洜涓洪€氱敤鍐风儹鍒嗘敮灏氭湭杩斿洖 `A|B`銆俓n
- [ ] **Step 3: 鏈€灏忓疄鐜?*

```go
func buildLHCRenyiDuipengHotColdPickContent(cfg parsedSchemeConfig, draws [][]string) (string, bool) {
    if !isLHCRenyiDuipengPlayRule(cfg.Play) { return "", false }
    aRanks, bRanks, ok := lhcRenyiDuipengHotColdRanks(cfg.HotCold, 49)
    if !ok { return "", true }
    hot, cold := hotColdWarmTiersOverallForPositions(draws, cfg.Play, playNumberPool(cfg.Play), nil)
    ordered := append(hot, cold...)
    a := pickHotColdByRanks(ordered, aRanks)
    b := pickHotColdByRanks(ordered, bRanks)
    content := strings.Join(sortHotColdBetTokens(a), ",") + "|" + strings.Join(sortHotColdBetTokens(b), ",")
    if len(validateLHCRenyiDuipengBetContent(content)) != 0 { return "", true }
    return content, true
}
```

Call it before the attribute, dual-zone group, overall and positional generic branches. `lhcRenyiDuipengHotColdRanks` must require exactly two nonempty rows, normalize valid ranks, reject cross-row duplicate ranks and reject a total above ten. Add the equivalent rule in `validateHotColdWarmConfig` so malformed requests are rejected before persistence.

- [ ] **Step 4: 楠岃瘉閫氳繃**

Run: `go test ./internal/schemes -run 'TestLHCRenyiDuipengHotColdPick|TestValidateSchemeBetContent_lhcRenyiDuipeng' -count=1`
Expected: PASS.

- [ ] **Step 5: 鏍煎紡鍖栦笌鎻愪氦妫€鏌?*

Run: `gofmt -w internal/schemes/worker_pick.go internal/schemes/scheme_audit_api.go internal/schemes/lhc_renyi_dp_run_modes_test.go && git diff --check`

### Task 2: 鍓嶇鍙屽尯鍚嶆鐘舵€併€佸睍绀哄拰浜や簰绾︽潫

**Files:**
- Modify: `client/src/views/play/AdvancedSchemeEditView.vue:2018-2886, 4014-4032, 4563-4664, 6060-6153`
- Create: `client/src/utils/lhcRenyiDuipengHotCold.ts`
- Create: `client/src/utils/lhcRenyiDuipengHotCold.spec.ts`

**Interfaces:**
- Produces `normalizeLhcRenyiDuipengHotColdRanks(ranks, orderLength): { a: number[]; b: number[]; valid: boolean }` and `selectLhcRenyiDuipengHotColdRank(ranks, zone, rank, orderLength): number[][]`.
- Consumes `isLhcRenyiDuipengConfig` and `hcwRanks`; `ranks[0]` maps to A 鍖猴紝`ranks[1]` maps to B 鍖恒€俓n
- [ ] **Step 1: 鍐欏け璐ユ祴璇?*

```ts
it('keeps two non-overlapping zones and limits their combined ranks to ten', () => {
  const result = normalizeLhcRenyiDuipengHotColdRanks([[0, 1], [2]], 49)
  expect(result).toEqual({ a: [0, 1], b: [2], valid: true })
  expect(normalizeLhcRenyiDuipengHotColdRanks([[0], [0]], 49).valid).toBe(false)
  expect(normalizeLhcRenyiDuipengHotColdRanks([Array.from({ length: 6 }, (_, i) => i), Array.from({ length: 5 }, (_, i) => i + 6)], 49).valid).toBe(false)
})
```

- [ ] **Step 2: 楠岃瘉澶辫触**

Run: `npm.cmd run test -- src/utils/lhcRenyiDuipengHotCold.spec.ts`
Expected: FAIL锛屽洜涓哄伐鍏锋ā鍧楀皻涓嶅瓨鍦ㄣ€俓n
- [ ] **Step 3: 鏈€灏忓疄鐜?*

鍒涘缓绾嚱鏁板伐鍏锋ā鍧楋紝鍙帴鍙楁湁鏁堟暣鏁板悕娆★紝鍖哄唴鍘婚噸鎺掑簭锛涗笉闈欓粯鍚堝苟璺ㄥ尯閲嶅椤广€備慨鏀圭紪杈戦〉锛歕n
```ts
const isHcwRenyiDuipeng = computed(() => isLhcRenyiDuipengConfig(schemePlayConfig.value))
function hcwDimCount(): number {
  if (isHcwRenyiDuipeng.value) return 2
  // existing branches
}
const hcwGroupLabels = computed(() => {
  if (isHcwRenyiDuipeng.value) return ['A鍖?, 'B鍖?]
  // existing branches
})
```

For this play, duplicate the same 01鈥?9 frequency/tier result into both panes. When toggling or applying quick picks, use the pure selector so the other zone鈥檚 ranks are excluded and remaining capacity is `10 - otherZone.length`. Use the exact same helper before saving and before building the `schemeGroups` placeholder; show a focused A/B validation message when invalid.

- [ ] **Step 4: 楠岃瘉閫氳繃**

Run: `npm.cmd run test -- src/utils/lhcRenyiDuipengHotCold.spec.ts src/utils/lhcRenyiDuipengRandom.spec.ts src/utils/betPayload.spec.ts`
Expected: PASS.

- [ ] **Step 5: 鏋勫缓妫€鏌?*

Run: `npm.cmd run build && git diff --check`
Expected: exit 0; Vite existing dynamic-import warnings may remain.

### Task 3: 浜ゅ弶灞傚洖褰掗獙璇乗n
**Files:**
- Verify: `backend/internal/schemes/lhc_renyi_dp_run_modes_test.go`
- Verify: `client/src/utils/lhcRenyiDuipengHotCold.spec.ts`

- [ ] **Step 1: 杩愯鍚庣鐩爣娴嬭瘯**

Run: `go test ./internal/schemes -run 'TestLHCRenyiDuipengHotColdPick|TestValidateSchemeBetContent_lhcRenyiDuipeng' -count=1`
Expected: PASS锛屼袱涓?ranks 琛屼骇鐢熷悎娉曚笖涓嶉噸鍙犵殑 `A|B`銆俓n
- [ ] **Step 2: 杩愯鍓嶇鐩爣娴嬭瘯涓庢瀯寤?*

Run: `npm.cmd run test -- src/utils/lhcRenyiDuipengHotCold.spec.ts src/utils/lhcRenyiDuipengRandom.spec.ts src/utils/betPayload.spec.ts && npm.cmd run build`
Expected: PASS銆俓n
- [ ] **Step 3: 妫€鏌ュ彉鏇磋寖鍥?*

Run: `git diff --check && git status --short`
Expected: no whitespace errors; preserve pre-existing unrelated worktree changes.
