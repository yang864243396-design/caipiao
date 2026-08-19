# 时时彩定码轮换预检与第三方计注一致性 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make fixed-rotate SSC start prechecks compute the same units as the Guaji wire payload for the audited five-star and Ren3 mixed-group cases.

**Architecture:** Keep `countPlayWireBetUnits` as the single precheck entry point. Add narrow parity handling for the three confirmed rule families, then validate legacy fixed-rotate group content before the amount precheck so known-invalid content cannot silently become one unit. The worker’s Guaji formatter and request format remain unchanged.

**Tech Stack:** Go, PostgreSQL-backed scheme service, `internal/schemes`, `internal/guajibet`, Go standard testing.

## Global Constraints

- Scope is only SSC fixed-rotate; do not alter hot/cold, random, trigger, UI, Guaji request payload, historical data, or scheme runtime state.
- Preserve the current third-decimal truncation rule for amounts.
- Do not automatically start, stop, retry, or migrate any existing scheme.
- Do not commit or push as part of this plan unless the user requests it separately.

---

## File Structure

- Modify `backend/internal/schemes/bet_units_cap.go`: identify explicit zero-unit content, normalize five-star Zu60/Zu30 content for precheck, and route Ren3 mixed-group content through its selected-position multiplier.
- Modify `backend/internal/schemes/plan_inverse.go`: retain fallback behaviour only for unknown rule shapes; preserve an explicit zero result for known-invalid wire content.
- Modify `backend/internal/schemes/instance_start.go`: validate each fixed-rotate group before minimum/maximum amount prechecks when an existing definition is started.
- Create `backend/internal/schemes/ssc_fixed_rotate_wire_parity_test.go`: isolated regressions that compare precheck units with `guajibet.FormatBetContentForRule` plus `guajibet.ResolveBetsNums`.
- Modify `backend/internal/schemes/instance_min_bet_test.go`: ensure a valid audited precheck amount reflects the corrected units and an invalid legacy group is not treated as one unit.

### Task 1: Add failing parity regressions for the three rule families

**Files:**
- Create: `backend/internal/schemes/ssc_fixed_rotate_wire_parity_test.go`
- Modify: `backend/internal/schemes/instance_min_bet_test.go`

**Interfaces:**
- Consumes: `parseSchemeConfig(kind, config, gameID, ruleID) parsedSchemeConfig`, `countPlayWireBetUnits(rule, content) int`, `planPickBetUnits(cfg, pick) int`.
- Consumes: `guajibet.ParseRuleMeta(...)`, `guajibet.FormatBetContentForRule(meta, content) string`, `guajibet.ResolveBetsNums(meta, wire) int`.
- Produces: regression expectations for explicit-zero content, Zu60/Zu30 normalization, and Ren3 mixed-group selected-position multiplication.

- [ ] **Step 1: Write the failing tests**

```go
func TestFixedRotateWireParity_WuxingBudingweiRejectsLessThanFourDigits(t *testing.T) {
	for _, tc := range []struct{ subID, content string }{
		{"151", "0,2"},
		{"152", "0,2,4"},
	} {
		cfg := parseSchemeConfig("custom", []byte(`{"playTemplate":"ssc_std","playTypeId":"g009","subPlayId":"`+tc.subID+`","catalogSubId":"`+tc.subID+`","betMode":"budingwei"}`), 0, 0)
		if got := planPickBetUnits(cfg, tc.content); got != 0 {
			t.Fatalf("sub %s invalid content=%q units=%d want 0", tc.subID, tc.content, got)
		}
	}
}

func TestFixedRotateWireParity_WuxingZuFlatPools(t *testing.T) {
	for _, tc := range []struct{ subID, mode string; want int }{
		{"157", "zu60", 4},
		{"158", "zu30", 6},
	} {
		cfg := parseSchemeConfig("custom", []byte(`{"playTemplate":"ssc_std","playTypeId":"g015","subPlayId":"`+tc.subID+`","catalogSubId":"`+tc.subID+`","betMode":"`+tc.mode+`"}`), 0, 0)
		if got := countPlayWireBetUnits(cfg.Play, "0,2,4,6,8"); got != tc.want {
			t.Fatalf("%s flat pool units=%d want %d", tc.mode, got, tc.want)
		}
	}
}

func TestFixedRotateWireParity_Ren3HunhePositionMultiplier(t *testing.T) {
	cfg := parseSchemeConfig("custom", []byte(`{"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"87","catalogSubId":"87","betMode":"hunhe","segmentLen":3}`), 0, 0)
	for _, tc := range []struct{ content string; want int }{
		{"万,千,百,个\n345", 4},
		{"万,千,百,十,个\n658", 10},
	} {
		if got := countPlayWireBetUnits(cfg.Play, tc.content); got != tc.want {
			t.Fatalf("content=%q units=%d want %d", tc.content, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/schemes -run TestFixedRotateWireParity -count=1`

Expected: FAIL. Current behaviour is 1 for the two invalid five-star cases, 5 for both flat Zu cases, and 1 for the Ren3 mixed-group cases.

### Task 2: Implement narrow wire-parity counting

**Files:**
- Modify: `backend/internal/schemes/bet_units_cap.go:90-105,234-342`
- Modify: `backend/internal/schemes/plan_inverse.go:213-228`

**Interfaces:**
- Consumes: `playRule`, raw scheme group content, `countZu60DualZoneBetUnits`, `countZu30DualZoneBetUnits`, `countRenxuanNeedsPositionBetUnits`, `countHunhePickUnits`.
- Produces: `countPlayWireBetUnits(rule, content) int` matching the Guaji wire count for the three audited families; an internal explicit-zero classifier used by `planPickBetUnits`.

- [ ] **Step 1: Add a small explicit-zero classifier next to `countPlayWireBetUnits`**

```go
func isExplicitZeroWireContent(rule playRule, content string) bool {
	if !isWuxingBudingweiRule(rule) {
		return false
	}
	need := budingweiPickCountForRule(rule)
	if need < 2 {
		return false
	}
	return len(uniqueStringTokens(parseDigitTokens(content))) < 4
}
```

The implementation must use the project’s existing five-star and 不定位 identification helpers where available; it must not classify a non-five-star 不定位 rule as explicit zero merely because it has fewer than four digits.

- [ ] **Step 2: Normalize only flat five-star Zu60/Zu30 content before local dual-zone counting**

```go
case "zu60":
	base = countZu60DualZoneBetUnits(normalizeWuxingZu60ForPrecheck(content))
case "zu30":
	base = countZu30DualZoneBetUnits(normalizeWuxingZu30ForPrecheck(content))
```

`normalizeWuxingZu60ForPrecheck` and `normalizeWuxingZu30ForPrecheck` must reproduce Guaji’s documented split rules for flat pools while returning already-valid two-zone content unchanged:

```go
// "0,2,4,6,8" -> "0,2468"
// "0,2,4,6,8" -> "024,68"
```

Do not change `internal/guajibet/wuxing_zu_wire.go` or its payload formatting.

- [ ] **Step 3: Route Ren3 mixed-group play through the selected-position branch**

```go
if isRenxuanNeedsPositionRule(rule) {
	return countRenxuanNeedsPositionBetUnits(rule, content)
}
```

Adjust only the predicate or segment-length resolution that currently prevents rule `87` from reaching this branch. The resulting path must retain `countHunhePickUnits` for ticket validation and multiply by `combinInt(nPos, 3)`.

- [ ] **Step 4: Preserve explicit zero in `planPickBetUnits`**

```go
if wire := countPlayWireBetUnits(cfg.Play, pick); wire > 0 {
	return wire
}
if isExplicitZeroWireContent(cfg.Play, pick) {
	return 0
}
```

Keep the existing evaluator/default-one fallback after this branch so unrelated unknown rule shapes retain their established behaviour.

- [ ] **Step 5: Run the parity tests to verify they pass**

Run: `go test ./internal/schemes -run TestFixedRotateWireParity -count=1`

Expected: PASS with 0, 4, 6, 4, and 10 unit assertions.

### Task 3: Reject legacy invalid fixed-rotate groups before amount precheck

**Files:**
- Modify: `backend/internal/schemes/instance_start.go:79-103`
- Modify: `backend/internal/schemes/instance_min_bet_test.go`
- Modify: `backend/internal/schemes/ssc_fixed_rotate_wire_parity_test.go`

**Interfaces:**
- Consumes: `ValidateSchemeConfig(kind, config) []Violation`, `ValidateSchemeBetContent(kind, config, content, maxUnits) []Violation`, `parseSchemeConfig(kind, config, gameID, ruleID) parsedSchemeConfig`.
- Produces: an error returned from `StartInstance` before `validateSchemeMinBetAmount` if a historical fixed-rotate group is now invalid.

- [ ] **Step 1: Write a failing startup-validation unit test**

Construct a persisted custom definition with rule `g009/151`, fixed-rotate group `0,2`, a betting unit that would otherwise satisfy the minimum amount, and a running-eligible instance. Assert `StartInstance` returns a validation error containing `至少选择 4 个号码`, not `ErrMinBetAmountTooLow`.

- [ ] **Step 2: Run the startup-validation test to verify it fails**

Run: `go test ./internal/schemes -run TestStartInstanceRejectsLegacyInvalidFixedRotateGroup -count=1`

Expected: FAIL because the current start path goes directly from time validation to the amount precheck.

- [ ] **Step 3: Add fixed-rotate group validation immediately after loading the definition**

```go
if violations := ValidateSchemeConfig(def.Kind, def.Config); len(violations) > 0 {
	return Instance{}, fmt.Errorf("方案内容无效: %s", violations[0].Detail)
}
```

If `ValidateSchemeConfig` cannot distinguish all persisted groups in this start path, add a focused helper that parses the config and calls `ValidateSchemeBetContent` for each non-empty `schemeGroups` item. Apply it only when `runTypeId == "fixed_rotate"`; return the first violation with its group ordinal.

- [ ] **Step 4: Run the startup-validation and minimum-amount tests**

Run: `go test ./internal/schemes -run 'Test(StartInstanceRejectsLegacyInvalidFixedRotateGroup|ValidateSchemeMinBetAmount)' -count=1`

Expected: PASS. The invalid legacy group returns its content error; valid corrected Zu60/Zu30 and Ren3 mixed-group configs use their exact unit counts.

### Task 4: Verify against the worker-equivalent wire count and build

**Files:**
- Modify: `backend/internal/schemes/ssc_fixed_rotate_wire_parity_test.go`

**Interfaces:**
- Consumes: `guajibet.RuleMeta`, `FormatBetContentForRule`, `ResolveBetsNums`, `countPlayWireBetUnits`.
- Produces: tests that compare local precheck units directly to the worker-equivalent Guaji unit calculation for the five audit fixtures.

- [ ] **Step 1: Add direct Guaji parity assertions for each audited fixture**

```go
wire := guajibet.FormatBetContentForRule(meta, content)
want := guajibet.ResolveBetsNums(meta, wire)
if got := countPlayWireBetUnits(rule, content); got != want {
	t.Fatalf("content=%q wire=%q precheck=%d guaji=%d", content, wire, got, want)
}
```

Use `g009/151`, `g009/152`, `g015/157`, `g015/158`, and `g011/87`; force the same `BetMode` as the stored scheme configuration when building each metadata object.

- [ ] **Step 2: Run focused regression suites**

Run: `go test ./internal/schemes -run 'Test(FixedRotateWireParity|ValidateSchemeMinBetAmount|StartInstanceRejectsLegacyInvalidFixedRotateGroup)' -count=1`

Expected: PASS.

- [ ] **Step 3: Run package verification and build**

Run: `go test ./internal/schemes -count=1`

Expected: PASS, or report any pre-existing failures verbatim without masking them.

Run: `go build ./cmd/server`

Expected: PASS.

- [ ] **Step 4: Check the patch is clean**

Run: `git diff --check -- backend/internal/schemes`

Expected: no output.
