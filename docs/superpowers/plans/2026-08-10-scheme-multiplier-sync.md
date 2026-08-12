# Scheme Multiplier Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep a scheme definition's `multCoeff` and its cloud instance `multiplier` synchronized, while retaining the instance multiplier as the runtime betting input.

**Architecture:** The definition config remains the edit-page representation and the cloud instance remains the runtime representation. Synchronize definition-to-instance on configuration save and initial cloud creation, and instance-to-definition on the cloud-card edit path. Worker multiple calculation continues reading `scheme_instances.multiplier`.

**Tech Stack:** Go, PostgreSQL, sqlc, Vue/Vite.

## Global Constraints

- `multCoeff`, card multiplier, and runtime multiplier are positive integers.
- No change to the global plan multiplier or per-round bet-multiplier rules.
- No automated real-bet request is used for verification.

---

### Task 1: Add synchronization regression coverage

**Files:**

- Create: `backend/internal/schemes/instance_multiplier_sync_test.go`
- Test: `backend/internal/schemes/instance_multiplier_sync_test.go`

**Interfaces:**

- Consumes: `Service.UpdateDefinition`, `Service.UpdateInstanceMultiplier`, `Service.AddDefinitionToCloud`.
- Produces: coverage proving definition saves, initial cloud creation, and card edits preserve one multiplier value.

- [ ] **Step 1: Write the failing test**

```go
func TestDefinitionMultiplierAndCloudInstanceStayInSync(t *testing.T) {
    // Create an isolated simulated definition and cloud instance.
    // Save multCoeff=3 and assert the persisted instance multiplier is 3.
    // Update the card instance multiplier to 5 and assert config.multCoeff is "5".
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/schemes -run TestDefinitionMultiplierAndCloudInstanceStayInSync -count=1`

Expected: FAIL because configuration and cloud-instance values are currently updated independently.

- [ ] **Step 3: Write minimal implementation**

```go
// On definition multCoeff save, update its owned instance multiplier.
// On cloud-card instance multiplier save, update config.multCoeff in the same transaction.
// Initialize a newly created cloud instance from config.multCoeff.
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/schemes -run TestDefinitionMultiplierAndCloudInstanceStayInSync -count=1`

Expected: PASS.

### Task 2: Verify runtime input and deploy

**Files:**

- Modify: `backend/internal/schemes/worker.go`
- Verify: `backend/internal/schemes/worker.go:369`

**Interfaces:**

- Consumes: persisted `scheme_instances.multiplier`.
- Produces: no runtime source change; `combinedBaseCoef(inst.Multiplier, planMult)` remains the multiplier used to build a bet.

- [ ] **Step 1: Run focused regression and package tests**

Run: `go test ./internal/schemes -run 'TestDefinitionMultiplierAndCloudInstanceStayInSync|TestWorker' -count=1`

Expected: PASS.

- [ ] **Step 2: Build and restart services**

Run: `go build ./cmd/server`, restart backend and the client development server, then check their HTTP endpoints.

Expected: backend health endpoint returns HTTP 200; no scheme-start or bet endpoint is invoked.
