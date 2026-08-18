# Cloud Center NATS Realtime Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Cloud Center scheme-card and statistics polling with account-isolated NATS Core events delivered through WebSocket, while preserving REST initialization/reconnect reconciliation and database-backed recovery.

**Architecture:** Scheme mutations mark `(memberId, instanceId)` dirty in a bounded in-process publisher. The publisher coalesces changes, bulk-loads current database projections, groups snapshots by member, and publishes them through NATS Core; API nodes dynamically subscribe only for members connected to that node and forward full snapshots through the existing WebSocket protocol. PostgreSQL remains authoritative, reconnects perform one REST reconciliation, and a singleton cursor scanner republishes missed updates.

**Tech Stack:** Go 1.25.7, PostgreSQL/pgx, `github.com/nats-io/nats.go v1.53.1`, Gorilla WebSocket, Vue 3, TypeScript 5.4, Vitest 2.1.9.

## Global Constraints

- WebSocket stable online state must make zero periodic calls to `/client/cloud/schemes/running` and `/client/cloud/schemes/stats`.
- Each successful WebSocket subscription cycle may execute exactly one REST reconciliation cycle.
- Scheme events coalesce for exactly `200ms` by default; statistics events coalesce for exactly `1000ms` by default.
- The reconciliation scanner runs every `5s` by default and scans by `(updated_at, id)` cursor with bounded batches.
- PostgreSQL remains the source of truth; NATS carries replaceable state snapshots only.
- NATS, one API node, one WebSocket connection, or one member must never block scheme calculation, betting, or settlement.
- Slow WebSocket connections are closed and reconciled; realtime frames are never silently dropped.
- Message routing is keyed by numeric `memberId`; account names are display/authentication data and are not NATS subjects.
- Existing card order, server-side search, cursor pagination, local one-second countdown, and user-action responses remain unchanged.
- Preserve all pre-existing uncommitted workspace changes. Do not reset, discard, or silently include unrelated files in a task commit.
- Migration `00151_scheme_instances_running_lottery_idx.sql` already exists in the working tree; this feature starts at migration `00152`.
- Do not remove legacy `client.scheme.instance.updated` compatibility until the new client has been deployed and observed.

---

## File Structure

New backend responsibilities:

- `backend/internal/realtimebus/bus.go`: transport-neutral publish/subscribe and diagnostics interfaces.
- `backend/internal/realtimebus/memory.go`: same-process development/test implementation.
- `backend/internal/realtimebus/nats.go`: reconnecting NATS Core implementation.
- `backend/internal/schemeevents/marker.go`: dependency-light scheme dirty-marker interface used by mutation packages.
- `backend/internal/db/sqlcdb/cloud_realtime_ext.go`: focused bulk projection, batch statistics, and cursor scan queries.
- `backend/internal/schemes/cloud_realtime_snapshot.go`: maps database rows to the existing `schemes.Instance`/stats wire shape.
- `backend/internal/cloudrealtime/contracts.go`: NATS message schemas and account-scoped subject builders.
- `backend/internal/cloudrealtime/publisher.go`: bounded 200ms scheme and 1s stats coalescers.
- `backend/internal/cloudrealtime/reconciler.go`: advisory-lock leader and `(updated_at,id)` compensation scanner.
- `backend/internal/cloudrealtime/wsbridge/bridge.go`: dynamic member subscription adapter from NATS to `ws.Envelope`.
- `backend/internal/cloudrealtime/diagnostics.go`: immutable runtime counters/snapshot.
- `backend/internal/handler/admin_cloud_realtime_diagnostics.go`: admin read-only diagnostics endpoint.
- `backend/migrations/00152_scheme_instances_updated_id_idx.sql`: cursor-scan index.

Existing backend integration points:

- `backend/internal/config/config.go`, `backend/.env.example`, `backend/go.mod`, `backend/go.sum`: configuration and NATS dependency.
- `backend/internal/ws/{handler,hub,conn,envelope}.go`: numeric member identity, dynamic subscription refcounts, snapshot events, and slow-client closure.
- `backend/internal/schemes/{service,share_add_to_cloud,instance_lifecycle,delete_definition,worker,worker_notify,instance_cloud_limit,instance_session_limit}.go`: change markers at committed mutations.
- `backend/internal/{cloudlimits,schemelimits}/pause.go`, `backend/internal/guaji/accountsvc/payout_sync.go`, `backend/internal/member/{service,admin_ops}.go`: non-handler mutation publishers.
- `backend/internal/handler/{handler,cloud}.go`, `backend/internal/server/server.go`: dependency injection, lifecycle, diagnostics route, and compatibility removal.

Frontend responsibilities:

- `shared/types/ws.ts`: snapshot/statistics event contracts.
- `client/src/composables/ws/useClientWs.ts`: subscription-ready connection lifecycle.
- `client/src/composables/useCloudRunningPoll.ts`: retained filename, replaced implementation: snapshots plus one reconnect reconciliation.
- `client/src/api/config.ts`: frontend rollout switch for snapshot mode versus legacy polling.
- `client/src/api/types.ts`, `client/src/api/cloud/center.ts`: preserve `updatedAt` through card mapping.
- `client/src/views/cloud/CloudCenterView.vue`: apply snapshots directly and remove countdown/action-driven REST refreshes.
- Focused `*.spec.ts` files next to these modules test ordering, buffering, reconnection, and absence of timers.

Operational documentation:

- `backend/docs/cloud-realtime-nats.md`: NATS cluster, credentials, rollout, rollback, and runtime checks.
- `backend/docs/websocket.md`: new events and reconnect contract.

---

### Task 1: Realtime Bus Foundation and Configuration

**Files:**
- Create: `backend/internal/realtimebus/bus.go`
- Create: `backend/internal/realtimebus/memory.go`
- Create: `backend/internal/realtimebus/memory_test.go`
- Create: `backend/internal/realtimebus/nats.go`
- Create: `backend/internal/realtimebus/nats_integration_test.go`
- Create: `backend/internal/schemeevents/marker.go`
- Modify: `backend/internal/config/config.go:14-73`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/.env.example:47-51`
- Modify: `backend/go.mod`
- Modify: `backend/go.sum`

**Interfaces:**
- Produces: `realtimebus.Bus`, `realtimebus.Subscription`, `realtimebus.Diagnostics`, `realtimebus.NewMemory()`, and `realtimebus.NewNATS(config)`.
- Produces: `schemeevents.Marker` with `MarkScheme(memberID int64, instanceID string)`.
- Consumes: no feature-specific interfaces.

- [ ] **Step 1: Write failing configuration and memory-bus tests**

Add table tests that set all realtime environment variables and assert exact defaults and overrides:

```go
func TestLoadCloudRealtimeDefaults(t *testing.T) {
	t.Setenv("CLOUD_REALTIME_ENABLED", "")
	t.Setenv("CLOUD_REALTIME_BUS", "")
	t.Setenv("NATS_URL", "")
	cfg := Load()
	if !cfg.CloudRealtimeEnabled || cfg.CloudRealtimeBus != "nats" {
		t.Fatalf("enabled=%v bus=%q", cfg.CloudRealtimeEnabled, cfg.CloudRealtimeBus)
	}
	if cfg.NATSURL != "nats://127.0.0.1:4222" || cfg.NATSSubjectPrefix != "caipiao" {
		t.Fatalf("url=%q prefix=%q", cfg.NATSURL, cfg.NATSSubjectPrefix)
	}
	if cfg.CloudRealtimeCoalesce != 200*time.Millisecond || cfg.CloudStatsCoalesce != time.Second {
		t.Fatalf("coalesce=%s stats=%s", cfg.CloudRealtimeCoalesce, cfg.CloudStatsCoalesce)
	}
	if cfg.CloudReconcileInterval != 5*time.Second || cfg.CloudReconcileBatch != 500 {
		t.Fatalf("interval=%s batch=%d", cfg.CloudReconcileInterval, cfg.CloudReconcileBatch)
	}
}
```

Add a memory-bus test proving exact-subject isolation and unsubscribe behavior:

```go
func TestMemoryBusIsolatesSubjectsAndUnsubscribes(t *testing.T) {
	bus := NewMemory()
	got := make(chan string, 2)
	sub, err := bus.Subscribe("caipiao.client.7.scheme", func(_ string, b []byte) { got <- string(b) })
	if err != nil { t.Fatal(err) }
	if err := bus.Publish(context.Background(), "caipiao.client.8.scheme", []byte("wrong")); err != nil { t.Fatal(err) }
	if err := bus.Publish(context.Background(), "caipiao.client.7.scheme", []byte("right")); err != nil { t.Fatal(err) }
	if value := <-got; value != "right" { t.Fatalf("got %q", value) }
	if err := sub.Unsubscribe(); err != nil { t.Fatal(err) }
	if err := bus.Publish(context.Background(), "caipiao.client.7.scheme", []byte("late")); err != nil { t.Fatal(err) }
	select { case value := <-got: t.Fatalf("unexpected %q", value); default: }
}
```

- [ ] **Step 2: Run the focused tests and verify they fail**

Run:

```powershell
cd backend
go test ./internal/config ./internal/realtimebus -run "CloudRealtime|MemoryBus" -count=1
```

Expected: compilation fails because the new config fields and `realtimebus` package do not exist.

- [ ] **Step 3: Add the transport interfaces and dependency**

Run:

```powershell
cd backend
go get github.com/nats-io/nats.go@v1.53.1
```

Define the transport boundary exactly as:

```go
package realtimebus

type Handler func(subject string, payload []byte)

type Subscription interface {
	Unsubscribe() error
}

type Diagnostics struct {
	Kind             string    `json:"kind"`
	Connected        bool      `json:"connected"`
	LastConnectedAt  time.Time `json:"lastConnectedAt,omitempty"`
	LastDisconnectedAt time.Time `json:"lastDisconnectedAt,omitempty"`
	Reconnects       uint64    `json:"reconnects"`
	Published        uint64    `json:"published"`
	PublishErrors    uint64    `json:"publishErrors"`
	Subscriptions    int64     `json:"subscriptions"`
	LastError        string    `json:"lastError,omitempty"`
}

type Bus interface {
	Publish(ctx context.Context, subject string, payload []byte) error
	Subscribe(subject string, handler Handler) (Subscription, error)
	OnConnectionChange(func(connected bool))
	Diagnostics() Diagnostics
	Close() error
}
```

Define `NATSConfig` with `URL`, `Name`, `User`, `Password`, `Token`, `CredentialsFile`, and `ReconnectWait`. Configure `nats.RetryOnFailedConnect(true)`, `nats.MaxReconnects(-1)`, disconnected/reconnected/closed handlers, and a bounded reconnect buffer. Authentication precedence is credentials file, then token, then username/password. Never call `Flush` for each publication.

- [ ] **Step 4: Add exact configuration fields and environment parsing**

Add these `config.Config` fields and defaults:

```go
CloudRealtimeEnabled   bool
CloudRealtimeBus       string
NATSURL                string
NATSUser               string
NATSPassword           string
NATSToken              string
NATSCredentialsFile    string
NATSSubjectPrefix      string
CloudRealtimeCoalesce  time.Duration
CloudStatsCoalesce     time.Duration
CloudReconcileInterval time.Duration
CloudReconcileBatch    int
```

Parse milliseconds/seconds through a new bounded helper:

```go
func envDurationMS(key string, fallback time.Duration) time.Duration {
	n := envInt(key, int(fallback/time.Millisecond))
	if n <= 0 { return fallback }
	return time.Duration(n) * time.Millisecond
}
```

Document the corresponding variables in `.env.example`; credential values remain blank.

- [ ] **Step 5: Verify unit tests and optional real-NATS integration**

Run:

```powershell
cd backend
go test ./internal/config ./internal/realtimebus -count=1
```

Expected: PASS.

Create an integration test guarded by `NATS_TEST_URL`; when set, two independent NATS bus clients must exchange one member-scoped message and expose `Connected=true`. When unset, it calls `t.Skip("NATS_TEST_URL not set")`.

- [ ] **Step 6: Commit the foundation**

```powershell
git add -- backend/go.mod backend/go.sum backend/.env.example backend/internal/config/config.go backend/internal/config/config_test.go backend/internal/realtimebus backend/internal/schemeevents
git commit -m "feat: add cloud realtime message bus"
```

---

### Task 2: Bulk Scheme and Statistics Projections

**Files:**
- Create: `backend/internal/db/sqlcdb/cloud_realtime_ext.go`
- Create: `backend/internal/schemes/cloud_realtime_snapshot.go`
- Create: `backend/internal/schemes/cloud_realtime_snapshot_test.go`
- Modify: `backend/internal/schemes/share_add_to_cloud.go:36-66`
- Modify: `backend/internal/schemes/cloud_center_stats.go:30-63`

**Interfaces:**
- Produces: `schemes.RealtimeInstanceRef`.
- Produces: `(*schemes.Service).LoadRealtimeSchemeSnapshots(ctx, refs) (schemes.RealtimeSchemeSnapshotResult, error)`.
- Produces: `(*schemes.Service).LoadRealtimeStats(ctx, memberIDs) (map[int64]schemes.CloudCenterStats, error)`.
- Consumes: existing `enrichInstanceListItem`, `CloudCenterStats`, `Instance`, and pgx pool.

- [ ] **Step 1: Write pure grouping tests before database code**

Test cross-member rejection, missing-row tombstones, deterministic ordering, and `UpdatedAt` retention:

```go
func TestGroupRealtimeSchemeSnapshotsIsolatesMembersAndMarksMissing(t *testing.T) {
	refs := []RealtimeInstanceRef{
		{MemberID: 7, InstanceID: "inst-a"},
		{MemberID: 7, InstanceID: "inst-gone"},
		{MemberID: 8, InstanceID: "inst-b"},
	}
	rows := []sqlcdb.SchemeInstance{
		{ID: "inst-a", MemberID: 7, UpdatedAt: pgtype.Timestamptz{Time: time.Unix(20, 0), Valid: true}},
		{ID: "inst-b", MemberID: 999, UpdatedAt: pgtype.Timestamptz{Time: time.Unix(30, 0), Valid: true}},
	}
	got := groupRealtimeSchemeSnapshots(refs, rows, nil, time.Unix(40, 0))
	if len(got.ItemsByMember[7]) != 1 || got.ItemsByMember[7][0].ID != "inst-a" { t.Fatalf("items=%v", got.ItemsByMember) }
	if !reflect.DeepEqual(got.RemovedByMember[7], []string{"inst-gone"}) { t.Fatalf("removed=%v", got.RemovedByMember) }
	if !reflect.DeepEqual(got.RemovedByMember[8], []string{"inst-b"}) { t.Fatalf("removed=%v", got.RemovedByMember) }
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```powershell
cd backend
go test ./internal/schemes -run RealtimeSchemeSnapshots -count=1
```

Expected: compilation fails because realtime projection types and helpers do not exist.

- [ ] **Step 3: Add focused pgx bulk queries**

In `cloud_realtime_ext.go`, add concrete methods that make one query per batch:

```go
type CloudRealtimeDefinitionMeta struct {
	ID, RunType, SchemeCurrency string
	MemberID int64
}

func (q *Queries) ListSchemeInstancesRealtimeByIDs(ctx context.Context, ids []string) ([]SchemeInstance, error)
func (q *Queries) ListSchemeDefinitionRealtimeMeta(ctx context.Context, ids []string) ([]CloudRealtimeDefinitionMeta, error)
func (q *Queries) ListCloudRealtimeStats(ctx context.Context, memberIDs []int64, today time.Time) ([]CloudRealtimeStatsRow, error)
func (q *Queries) ListSchemeRealtimeChanges(ctx context.Context, after time.Time, afterID string, limit int) ([]SchemeRealtimeChange, error)
```

`ListCloudRealtimeStats` returns one row per requested member and uses aggregate `FILTER` clauses for formal/sim totals, running simulated count, and the member's Shanghai-date start counter. It must not execute a query inside a member loop.

- [ ] **Step 4: Implement projection mapping with existing display rules**

Add the field `MemberID int64` with struct tag `json:"-"` to `schemes.Instance`. Implement these exact types:

```go
type RealtimeInstanceRef struct {
	MemberID int64
	InstanceID string
}

type RealtimeSchemeSnapshotResult struct {
	ItemsByMember map[int64][]Instance
	RemovedByMember map[int64][]string
}
```

`LoadRealtimeSchemeSnapshots` deduplicates IDs, bulk-loads instances and definition metadata, verifies every returned row matches the marked member, calls `enrichInstanceListItem`, and sorts each member's items/removals by ID for stable tests. `LoadRealtimeStats` maps the one-row-per-member result through the same truncation/rounding functions used by `GetCloudCenterStats`.

Refactor `GetCloudCenterStats` to resolve the account once and call `LoadRealtimeStats(ctx, []int64{m.ID})`, so REST and realtime statistics cannot drift.

- [ ] **Step 5: Run scheme projection and existing stats tests**

Run:

```powershell
cd backend
go test ./internal/schemes -run "Realtime|CloudCenterStats" -count=1
```

Expected: PASS, including existing amount truncation expectations.

- [ ] **Step 6: Commit bulk projections**

```powershell
git add -- backend/internal/db/sqlcdb/cloud_realtime_ext.go backend/internal/schemes/cloud_realtime_snapshot.go backend/internal/schemes/cloud_realtime_snapshot_test.go backend/internal/schemes/share_add_to_cloud.go backend/internal/schemes/cloud_center_stats.go
git commit -m "feat: add cloud realtime projections"
```

---

### Task 3: Coalesced Member Snapshot Publisher

**Files:**
- Create: `backend/internal/cloudrealtime/contracts.go`
- Create: `backend/internal/cloudrealtime/diagnostics.go`
- Create: `backend/internal/cloudrealtime/publisher.go`
- Create: `backend/internal/cloudrealtime/publisher_test.go`

**Interfaces:**
- Consumes: `realtimebus.Bus`, `schemes.LoadRealtimeSchemeSnapshots`, `schemes.LoadRealtimeStats`.
- Produces: `cloudrealtime.Publisher`, which implements `schemeevents.Marker`.
- Produces: `SchemeSnapshotMessage`, `StatsSnapshotMessage`, `SchemeSubject(prefix, memberID)`, and `StatsSubject(prefix, memberID)`.

- [ ] **Step 1: Write deterministic coalescing and isolation tests**

Use a fake source and memory bus; call the package-private `flushSchemes` method directly instead of sleeping. Assert three marks for one instance produce one source batch and one publication, two members produce different subjects, and a source failure leaves entries dirty for the next flush:

```go
func TestPublisherCoalescesLatestSchemeMarksByMemberAndInstance(t *testing.T) {
	source := &fakeSource{schemeResult: schemes.RealtimeSchemeSnapshotResult{
		ItemsByMember: map[int64][]schemes.Instance{7: {{ID: "inst-a", UpdatedAt: "2026-08-18T00:00:01Z"}}},
	}}
	bus := realtimebus.NewMemory()
	p := NewPublisher(source, bus, Config{SubjectPrefix: "caipiao", SchemeCoalesce: 200*time.Millisecond, StatsCoalesce: time.Second})
	p.MarkScheme(7, "inst-a")
	p.MarkScheme(7, "inst-a")
	p.MarkScheme(7, "inst-a")
	if err := p.flushSchemes(context.Background()); err != nil { t.Fatal(err) }
	if source.schemeCalls != 1 { t.Fatalf("calls=%d", source.schemeCalls) }
	want := []schemes.RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-a"}}
	if !reflect.DeepEqual(source.lastRefs, want) { t.Fatalf("refs=%v want=%v", source.lastRefs, want) }
}
```

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```powershell
cd backend
go test ./internal/cloudrealtime -run Publisher -count=1
```

Expected: compilation fails because the package does not exist.

- [ ] **Step 3: Define versioned messages and safe subjects**

Use exact wire fields:

```go
const SchemaVersion = 1

type SchemeSnapshotMessage struct {
	SchemaVersion int                `json:"schemaVersion"`
	GeneratedAt   string             `json:"generatedAt"`
	Items         []schemes.Instance `json:"items"`
	RemovedIDs    []string           `json:"removedIds"`
}

type StatsSnapshotMessage struct {
	SchemaVersion int                      `json:"schemaVersion"`
	GeneratedAt   string                   `json:"generatedAt"`
	Stats         schemes.CloudCenterStats `json:"stats"`
}
```

Define the source boundary exactly as:

```go
type SnapshotSource interface {
	LoadRealtimeSchemeSnapshots(context.Context, []schemes.RealtimeInstanceRef) (schemes.RealtimeSchemeSnapshotResult, error)
	LoadRealtimeStats(context.Context, []int64) (map[int64]schemes.CloudCenterStats, error)
}
```

Subject builders trim dots from the configured prefix, reject non-positive IDs, and return exactly `caipiao.client.<memberId>.scheme` or `.cloud_stats`.

- [ ] **Step 4: Implement bounded latest-wins publisher loops**

`MarkScheme` must be non-blocking. Store dirty keys in a mutex-protected bounded map, wake one loop through a size-one channel, and let the loop own flush scheduling. A successful scheme flush also marks the affected member's stats dirty. A failed load or publish restores only the affected keys and increments diagnostics; it never waits on the betting goroutine.

Expose:

```go
func NewPublisher(source SnapshotSource, bus realtimebus.Bus, cfg Config) *Publisher
func (p *Publisher) Run(ctx context.Context)
func (p *Publisher) MarkScheme(memberID int64, instanceID string)
func (p *Publisher) MarkStats(memberID int64)
func (p *Publisher) Diagnostics() Diagnostics
```

Diagnostics include queue sizes, accepted marks, coalesced marks, scheme/stat publish counts, errors, last success, last error, and last publish latency.

- [ ] **Step 5: Run race-enabled focused tests**

Run:

```powershell
cd backend
go test -race ./internal/cloudrealtime -run "Publisher|Subject" -count=1
```

Expected: PASS without races or goroutine leaks.

- [ ] **Step 6: Commit the publisher**

```powershell
git add -- backend/internal/cloudrealtime
git commit -m "feat: publish coalesced cloud snapshots"
```

---

### Task 4: Mark Every Existing Scheme Mutation Path

**Files:**
- Modify: `backend/internal/schemes/service.go:20-48`
- Modify: `backend/internal/schemes/instance_lifecycle.go`
- Modify: `backend/internal/schemes/delete_definition.go`
- Modify: `backend/internal/schemes/worker.go:43-81`
- Modify: `backend/internal/schemes/worker_notify.go`
- Modify: `backend/internal/schemes/instance_cloud_limit.go`
- Modify: `backend/internal/schemes/instance_session_limit.go`
- Modify: `backend/internal/cloudlimits/pause.go`
- Modify: `backend/internal/schemelimits/pause.go`
- Modify: `backend/internal/guaji/accountsvc/payout_sync.go:52-72,671-708`
- Modify: `backend/internal/member/service.go`
- Modify: `backend/internal/member/admin_ops.go:307-343`
- Modify: `backend/internal/handler/handler.go:36-100`
- Modify: `backend/internal/handler/cloud.go:338-360`
- Create: `backend/internal/schemes/realtime_marker_test.go`

**Interfaces:**
- Consumes: `schemeevents.Marker` from Task 1.
- Produces: setters on schemes Service/Worker, payout worker, and member Service.
- Preserves: local admin-monitor and wallet WebSocket events; only scheme-card invalidation is replaced.

- [ ] **Step 1: Write a recording marker test for committed mutations**

Use a thread-safe fake:

```go
type recordingMarker struct {
	mu sync.Mutex
	refs []schemes.RealtimeInstanceRef
}
func (m *recordingMarker) MarkScheme(memberID int64, instanceID string) {
	m.mu.Lock(); defer m.mu.Unlock()
	m.refs = append(m.refs, schemes.RealtimeInstanceRef{MemberID: memberID, InstanceID: instanceID})
}
```

Add these concrete tests: `TestWorkerNotifySchemeInstanceMarksRealtime`, `TestCloudActionPublisherMarksCommittedInstance`, `TestPayoutSettlementMarksRealtime`, `TestCloudLimitPauseMarksEveryChangedInstance`, `TestSchemeLimitPauseMarksChangedInstance`, `TestAdminPauseMarksEveryChangedInstance`, and `TestDeleteDefinitionMarksRemovedInstance`. The worker test is a pure unit test:

```go
func TestWorkerNotifySchemeInstanceMarksRealtime(t *testing.T) {
	marker := &recordingMarker{}
	w := &Worker{realtime: marker}
	w.notifySchemeInstance(context.Background(), 7, "inst-a", "real", "running", "cloud_active")
	marker.mu.Lock()
	defer marker.mu.Unlock()
	want := []RealtimeInstanceRef{{MemberID: 7, InstanceID: "inst-a"}}
	if !reflect.DeepEqual(marker.refs, want) { t.Fatalf("refs=%v want=%v", marker.refs, want) }
}
```

Database-backed tests use the existing environment-gated fixture pattern, perform the real committed mutation, and assert the marker only after the service method returns success. The delete test captures the instance before cascade deletion and expects one marker containing its original member/instance IDs.

- [ ] **Step 2: Run focused tests and verify at least the first mutation fails**

Run:

```powershell
cd backend
go test ./internal/schemes ./internal/cloudlimits ./internal/schemelimits ./internal/guaji/accountsvc ./internal/member -run RealtimeMarker -count=1
```

Expected: compilation fails because marker setters and calls do not exist.

- [ ] **Step 3: Inject the marker without changing business constructors**

Add nil-safe setters. The schemes Service marks add-to-cloud, definition edits that synchronize an instance, and deletion. The HTTP handler marks start/stop/pause/resume/multiplier/sim-bet after receiving a committed Service result, so those actions are not marked twice:

```go
func (s *Service) SetRealtimeMarker(m schemeevents.Marker) { if s != nil { s.realtime = m } }
func (w *Worker) SetRealtimeMarker(m schemeevents.Marker) { if w != nil { w.realtime = m } }
func (s *Service) SetRealtimeMarker(m schemeevents.Marker) // in package member
func (w *PayoutSyncWorker) SetRealtimeMarker(m schemeevents.Marker)
```

Keep existing constructor signatures to limit unrelated churn. Add `markScheme(memberID, instanceID)` helpers that return immediately for nil markers, invalid member IDs, or blank instance IDs.

- [ ] **Step 4: Replace direct scheme-card invalidations after successful commits**

Rules for every call site:

- Mark only after transaction commit or a successful single-statement mutation.
- One scheme mark automatically dirties member statistics in the publisher.
- `worker_notify.go` keeps `ws.PublishSchemeMonitor` for administrators but removes the member-account lookup and direct `PublishSchemeInstance` call.
- `payout_sync.go` marks the scheme after the settlement commit and limit checks.
- `cloudlimits` marks every returned paused instance; `schemelimits` marks the paused instance.
- `member/admin_ops.go` marks every administratively paused instance.
- `DeleteDefinition` loads `{memberID, instanceID}` before deletion and marks it after deletion, producing `removedIds` when projection reload finds no row.
- User action handlers keep applying their HTTP response; `publishSchemeInstanceWS` becomes a marker call and no longer emits `refresh_running_list` locally.

- [ ] **Step 5: Run mutation tests plus worker/account settlement tests**

Run:

```powershell
cd backend
go test ./internal/schemes ./internal/cloudlimits ./internal/schemelimits ./internal/guaji/accountsvc ./internal/member -count=1
```

Expected: PASS. Existing wallet and administrator events remain covered.

- [ ] **Step 6: Commit mutation coverage**

```powershell
git add -- backend/internal/schemes backend/internal/cloudlimits/pause.go backend/internal/schemelimits/pause.go backend/internal/guaji/accountsvc/payout_sync.go backend/internal/member/service.go backend/internal/member/admin_ops.go backend/internal/handler/handler.go backend/internal/handler/cloud.go
git commit -m "feat: mark cloud scheme state changes"
```

---

### Task 5: Dynamic NATS-to-WebSocket Member Routing

**Files:**
- Create: `backend/internal/cloudrealtime/wsbridge/bridge.go`
- Create: `backend/internal/cloudrealtime/wsbridge/bridge_test.go`
- Modify: `backend/internal/ws/envelope.go:20-75`
- Modify: `backend/internal/ws/hub.go`
- Modify: `backend/internal/ws/conn.go:21-220`
- Modify: `backend/internal/ws/handler.go:13-83`
- Create: `backend/internal/ws/hub_member_test.go`
- Create: `backend/internal/ws/conn_overflow_test.go`

**Interfaces:**
- Consumes: Task 3 subjects/messages and Task 1 bus.
- Produces: `ws.MemberEventSource` and `ws.Hub.SetMemberEventSource`.
- Produces: `ws.ClientIdentity{Account string, MemberID int64}` and `ws.Server.ResolveClientIdentity`.

- [ ] **Step 1: Write member subscription-refcount and overflow tests**

Use a fake source that records subscriptions and cancellation:

```go
func TestHubSubscribesOncePerMemberAndReleasesLastConnection(t *testing.T) {
	source := &fakeMemberEventSource{}
	h := NewHub()
	h.SetMemberEventSource(source)
	c1 := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	c2 := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	h.Register(c1); h.Register(c2)
	if source.subscribeCalls != 1 { t.Fatalf("subscribe=%d", source.subscribeCalls) }
	h.Unregister(c1)
	if source.cancelCalls != 0 { t.Fatalf("cancel=%d", source.cancelCalls) }
	h.Unregister(c2)
	if source.cancelCalls != 1 { t.Fatalf("cancel=%d", source.cancelCalls) }
}
```

Define the test helper as a package-local synthetic connection with a buffered send channel and no network socket:

```go
func newTestClientConn(identity ClientIdentity) *Conn {
	return &Conn{
		kind: KindClient, authenticated: true,
		account: identity.Account, memberID: identity.MemberID,
		topics: make(map[string]struct{}), send: make(chan Envelope, 4),
	}
}
```

Fill a size-one send channel, call `TrySend`, and assert the connection's injected close function is called once and `TrySend` returns false.

- [ ] **Step 2: Run focused tests and verify they fail**

Run:

```powershell
cd backend
go test ./internal/ws ./internal/cloudrealtime/wsbridge -run "Member|Overflow|Bridge" -count=1
```

Expected: compilation fails because identities, event source, bridge, and close-on-overflow do not exist.

- [ ] **Step 3: Add numeric identity and lifecycle hooks**

Change authentication to return:

```go
type ClientIdentity struct {
	Account string
	MemberID int64
}
```

`ws.Server.ResolveClientIdentity(ctx, claims.Subject)` resolves active clients through an injected callback. Query-token and command-auth paths both call `Hub.BindClientIdentity`; registration with an already authenticated identity acquires immediately. Unregister decrements the member refcount and cancels the member source only at zero.

- [ ] **Step 4: Implement bridge subscriptions and compatibility events**

`Bridge.SubscribeMember` subscribes to exactly the scheme and stats subjects for that member. It unmarshals schema version 1 and emits:

```go
NameSchemeInstancesSnapshot = "client.scheme.instances.snapshot"
NameCloudStatsSnapshot       = "client.cloud.stats.snapshot"
TopicClientCloudStats        = "client.cloud.stats"
```

Add `TopicClientCloudStats` to `Conn.subscribeClientTopics`, so `system.subscribed` acknowledges both scheme and statistics topics before the browser declares the connection ready.

For each scheme batch it also emits legacy `client.scheme.instance.updated` hints during the compatibility phase. The legacy payload carries only instance ID/status plus `hint=refresh_running_list`; the new client ignores it after receiving a schema-versioned snapshot.

- [ ] **Step 5: Close slow or unverifiable connections**

Add `Conn.Close(code int, reason string)` guarded by `sync.Once`. `TrySend` on a full buffer increments a Hub metric and asynchronously closes the socket with application code `4010` and reason `realtime_backpressure`; it never closes the send channel from the producer goroutine. The read/write pumps remain responsible for unregistering and final cleanup.

Add `Hub.CloseClientConnections(code, reason)` for NATS disconnect handling. It snapshots client connections under the read lock and closes them after releasing the lock.

- [ ] **Step 6: Run race-enabled WS and bridge tests**

Run:

```powershell
cd backend
go test -race ./internal/ws ./internal/cloudrealtime/wsbridge -count=1
```

Expected: PASS with one subscription per member per API node, no cross-member delivery, and no send-on-closed-channel race.

- [ ] **Step 7: Commit clustered WebSocket routing**

```powershell
git add -- backend/internal/ws backend/internal/cloudrealtime/wsbridge
git commit -m "feat: route cloud snapshots to member websockets"
```

---

### Task 6: Singleton Database Compensation Scanner

**Files:**
- Create: `backend/migrations/00152_scheme_instances_updated_id_idx.sql`
- Modify: `backend/internal/db/sqlcdb/cloud_realtime_ext.go`
- Create: `backend/internal/cloudrealtime/reconciler.go`
- Create: `backend/internal/cloudrealtime/reconciler_test.go`

**Interfaces:**
- Consumes: `schemeevents.Marker`, pgx pool, and Task 2 cursor query.
- Produces: `cloudrealtime.Reconciler.Run(ctx)` and reconciliation diagnostics.

- [ ] **Step 1: Write fake-session cursor and leader tests**

Cover these exact cases:

```go
func TestReconcilerOnlyLeaderScansAndAdvancesCompositeCursor(t *testing.T)
func TestReconcilerMarksEveryRowBeforeAdvancingCursor(t *testing.T)
func TestReconcilerRetriesSameCursorAfterQueryFailure(t *testing.T)
func TestReconcilerYieldsAfterConfiguredBatchBudget(t *testing.T)
```

The first test supplies rows with equal timestamps and IDs `a`, `b`; the next query must receive `(sameTimestamp,"b")`, proving no equal-time row is skipped.

- [ ] **Step 2: Run focused tests and verify they fail**

Run:

```powershell
cd backend
go test ./internal/cloudrealtime -run Reconciler -count=1
```

Expected: compilation fails because `Reconciler` does not exist.

- [ ] **Step 3: Add the cursor index migration**

Migration content:

```sql
-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scheme_instances_updated_id
    ON scheme_instances (updated_at ASC, id ASC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_scheme_instances_updated_id;
```

The `NO TRANSACTION` directive must remain at the top of the file because Goose otherwise wraps SQL migrations in a transaction; see the [official Goose migration documentation](https://github.com/pressly/goose).

- [ ] **Step 4: Implement advisory-lock leadership and bounded scans**

Use one acquired pgx connection and a stable 64-bit advisory key. The leader starts its cursor at `time.Now().UTC().Add(-5*time.Minute)` on process start, scans up to `CloudReconcileBatch` rows per query, processes at most four consecutive full batches per tick, and then yields. It calls `MarkScheme` for each row before advancing to that row's cursor.

On connection loss or scan failure, release the connection, retain the last in-memory cursor, and retry on the next interval. No transaction is held while publishing markers.

- [ ] **Step 5: Verify tests and inspect the query plan**

Run:

```powershell
cd backend
go test ./internal/cloudrealtime -run Reconciler -count=1
go test ./internal/db/sqlcdb -count=1
```

Against a migrated development database, run:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT member_id, id, updated_at
FROM scheme_instances
WHERE (updated_at, id) > (now() - interval '5 minutes', '')
ORDER BY updated_at, id
LIMIT 500;
```

Expected: `idx_scheme_instances_updated_id` is eligible; no sequential full-table scan on a populated table.

- [ ] **Step 6: Commit compensation recovery**

```powershell
git add -- backend/migrations/00152_scheme_instances_updated_id_idx.sql backend/internal/db/sqlcdb/cloud_realtime_ext.go backend/internal/cloudrealtime/reconciler.go backend/internal/cloudrealtime/reconciler_test.go
git commit -m "feat: reconcile missed cloud scheme events"
```

---

### Task 7: Server Wiring, Failure Isolation, and Admin Diagnostics

**Files:**
- Modify: `backend/internal/server/server.go:42-51,105-226,428-435`
- Modify: `backend/internal/handler/handler.go:36-100`
- Create: `backend/internal/handler/admin_cloud_realtime_diagnostics.go`
- Create: `backend/internal/handler/admin_cloud_realtime_diagnostics_test.go`
- Modify: `backend/internal/handler/handler.go:115-150`

**Interfaces:**
- Consumes: bus, publisher, reconciler, bridge, marker setters, member identity callback.
- Produces: `GET /admin/diagnostics/cloud-realtime`.
- Preserves: server startup and scheme workers when NATS is initially unavailable.

- [ ] **Step 1: Write diagnostic response and degraded-start tests**

The handler test installs a fake provider and asserts the response contains bus, publisher, Hub, and scanner sections without credentials:

```go
type fakeRealtimeDiagnostics struct { snapshot map[string]any }
func (f fakeRealtimeDiagnostics) Snapshot() map[string]any { return maps.Clone(f.snapshot) }

func TestAdminCloudRealtimeDiagnosticsReturnsReadOnlySnapshot(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/admin/diagnostics/cloud-realtime", nil)
	h := &Handler{cloudRealtimeDiagnostics: fakeRealtimeDiagnostics{snapshot: map[string]any{
		"bus": map[string]any{"connected": false, "lastError": "dial timeout"},
	}}}
	h.AdminCloudRealtimeDiagnostics(w, r)
	if w.Code != http.StatusOK { t.Fatalf("status=%d body=%s", w.Code, w.Body.String()) }
	if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "token") { t.Fatal("secret leaked") }
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
cd backend
go test ./internal/handler ./internal/server -run CloudRealtime -count=1
```

Expected: compilation fails because the diagnostics dependency and route do not exist.

- [ ] **Step 3: Wire services in dependency order**

In `server.New`:

1. Create the local Hub.
2. Create NATS or memory bus from config; NATS initial failure enters reconnecting state and does not return a fatal server error.
3. Create schemes Service.
4. Create/start Publisher and Reconciler with `workerCtx`.
5. Inject the Publisher marker into schemes Service, scheme Worker, member Service, payout worker, limit call paths, and handlers.
6. Create the WS bridge and install it on Hub.
7. Set `ws.Server.ResolveClientIdentity` to `memberSvc.GetByAccount(ctx, account)` and return its numeric ID.
8. Register a bus connection callback: on `connected=false`, call `wsHub.CloseClientConnections(1012, "realtime_bus_unavailable")`.

Store bus and realtime closers in `Server`; `Close` cancels workers, closes the bus, then closes the DB pool.

- [ ] **Step 4: Expose health and administrator diagnostics**

Register:

```go
api.Handle("GET /admin/diagnostics/cloud-realtime", adminAuth(http.HandlerFunc(s.handler.AdminCloudRealtimeDiagnostics)))
```

The diagnostics response includes bus status/reconnects/publications, publisher dirty sizes/coalescing/errors, Hub connections/subscribed members/backpressure closes, and scanner leader/cursor/lag/errors. `/health` reports this component as degraded when disconnected but does not return a failing application status solely because NATS is down.

- [ ] **Step 5: Run server, handler, and backend build verification**

Run:

```powershell
cd backend
go test ./internal/server ./internal/handler ./internal/ws ./internal/cloudrealtime -count=1
go build ./cmd/server
```

Expected: PASS and build success both with `CLOUD_REALTIME_ENABLED=false` and with an unreachable `NATS_URL`.

- [ ] **Step 6: Commit runtime wiring**

```powershell
git add -- backend/internal/server/server.go backend/internal/handler/handler.go backend/internal/handler/admin_cloud_realtime_diagnostics.go backend/internal/handler/admin_cloud_realtime_diagnostics_test.go
git commit -m "feat: wire cloud realtime runtime"
```

---

### Task 8: Frontend Contracts and Subscription-Ready WebSocket Lifecycle

**Files:**
- Modify: `shared/types/ws.ts`
- Modify: `backend/contracts/ws.ts`
- Modify: `client/src/composables/ws/useClientWs.ts`
- Create: `client/src/composables/ws/useClientWs.spec.ts`

**Interfaces:**
- Produces: `WsSchemeInstancesSnapshotPayload` and `WsCloudStatsSnapshotPayload`.
- Changes: `onConnected` means authenticated default topics are subscribed, not merely TCP/WebSocket open.
- Consumes: backend event names from Task 5.

- [ ] **Step 1: Write a deterministic fake-WebSocket lifecycle test**

The fake records constructed sockets and can emit `open`, `message`, and `close`. Assert `onConnected` is zero after `open`, one after `system.subscribed` includes `client.scheme.instance`, and one more only after a new socket reaches subscribed state:

```ts
it('announces readiness once per subscribed connection cycle', () => {
  const connected = vi.fn()
  connectClientWs('ws://test', 'token', vi.fn(), { onConnected: connected })
  const first = FakeWebSocket.instances[0]
  first.emitOpen()
  expect(connected).not.toHaveBeenCalled()
  first.emitMessage({ type: 'system', name: 'system.subscribed', ts: now, payload: { topics: ['client.scheme.instance', 'client.cloud.stats'] } })
  expect(connected).toHaveBeenCalledTimes(1)
  first.emitClose()
  vi.runOnlyPendingTimers()
  const second = FakeWebSocket.instances[1]
  second.emitOpen()
  second.emitMessage({ type: 'system', name: 'system.subscribed', ts: now, payload: { topics: ['client.scheme.instance', 'client.cloud.stats'] } })
  expect(connected).toHaveBeenCalledTimes(2)
})
```

- [ ] **Step 2: Run the test and verify failure**

Run:

```powershell
cd client
npm.cmd test -- src/composables/ws/useClientWs.spec.ts
```

Expected: FAIL because the current implementation calls `onConnected` during `onopen`.

- [ ] **Step 3: Add shared versioned payload contracts**

Add the exact event constants and payloads:

```ts
export interface WsCloudSchemeSnapshotItem {
  id: string
  definitionId?: string
  lotteryCode?: string
  lotteryName?: string
  lotteryLabel?: string
  schemeName: string
  status: 'pending' | 'running' | 'paused' | 'soft_stopped'
  statusReason?: string
  statusLabel: string
  turnover: number
  countdownSec: number
  countdownEndTime?: string
  countdownCloseAt?: string
  countdownPeriod?: string
  countdownWindowSec?: number
  countdownLabel?: string
  pnl: number
  runTimeSec: number
  lookbackPnl: number
  sessionPnl: number
  multiplier: number
  simBet: boolean
  schemeCurrency?: string
  runTypeId?: string
  runTypeLabel?: string
  updatedAt: string
}

export interface WsCloudCenterChannelStats {
  totalTurnover: number
  totalSessionPnl: number
  runningSessionPnl: number
}

export interface WsCloudCenterStats {
  formal: WsCloudCenterChannelStats
  sim: WsCloudCenterChannelStats
  simQuota: {
    todayStarts: number
    todayStartsLimit: number
    running: number
    runningLimit: number
  }
}

export interface WsSchemeInstancesSnapshotPayload {
  schemaVersion: 1
  generatedAt: string
  items: WsCloudSchemeSnapshotItem[]
  removedIds: string[]
}

export interface WsCloudStatsSnapshotPayload {
  schemaVersion: 1
  generatedAt: string
  stats: WsCloudCenterStats
}

schemeInstancesSnapshot: 'client.scheme.instances.snapshot'
cloudStatsSnapshot: 'client.cloud.stats.snapshot'
```

Keep these wire DTOs in `shared/types/ws.ts`; do not import client API modules into `shared`.

- [ ] **Step 4: Move readiness to subscription acknowledgement**

Track `readyNotified` per socket. Parse `system.subscribed.payload.topics`; call `onConnected` only when authenticated default client topics include both `client.scheme.instance` and `client.cloud.stats`. Reset the flag in cleanup. Query-token authentication must not bypass this acknowledgement.

- [ ] **Step 5: Run WS tests and TypeScript build**

Run:

```powershell
cd client
npm.cmd test -- src/composables/ws/useClientWs.spec.ts
npm.cmd run build
```

Expected: PASS.

- [ ] **Step 6: Commit frontend protocol readiness**

```powershell
git add -- shared/types/ws.ts backend/contracts/ws.ts client/src/composables/ws/useClientWs.ts client/src/composables/ws/useClientWs.spec.ts
git commit -m "feat: add cloud snapshot websocket contracts"
```

---

### Task 9: Snapshot State Machine and One-Shot Reconciliation

**Files:**
- Modify: `client/src/api/types.ts:36-72`
- Modify: `client/src/api/config.ts`
- Modify: `client/src/api/cloud/center.ts:29-71,600-655`
- Rewrite: `client/src/composables/useCloudRunningPoll.ts`
- Rewrite: `client/src/composables/useCloudRunningPoll.spec.ts`

**Interfaces:**
- Consumes: Task 8 payloads and existing `fetchRunningSchemesByIds`/`fetchCloudCenterStats`.
- Produces: `startCloudRunningSync(getLoadedIds, handlers, options)`.
- Preserves: exported `cloudRunningPollMs` only for other screens that still use it; Cloud Center no longer calls it.

- [ ] **Step 1: Write tests for no polling, buffering, versions, and reconnect count**

Use fake timers and mocked API/WS modules. Include these tests:

```ts
it('does not create a REST interval while websocket is online')
it('runs exactly one reconciliation for each subscribed connection cycle')
it('shares one in-flight reconciliation across duplicate readiness events')
it('buffers snapshots during reconciliation and applies them afterward')
it('ignores an older updatedAt snapshot')
it('removes only loaded cards named in removedIds')
it('uses legacy invalidation only until the first versioned snapshot')
it('keeps legacy polling only when client websocket is disabled')
```

The reconnect-count assertion is concrete:

```ts
expect(fetchRunningSchemesByIds).toHaveBeenCalledTimes(2)
expect(fetchCloudCenterStats).toHaveBeenCalledTimes(2)
expect(setInterval).not.toHaveBeenCalled()
```

after two completed subscription cycles.

- [ ] **Step 2: Run tests and verify current polling behavior fails**

Run:

```powershell
cd client
npm.cmd test -- src/composables/useCloudRunningPoll.spec.ts
```

Expected: FAIL because current code starts 15s/60s intervals and refreshes for every legacy event.

- [ ] **Step 3: Preserve server versions through card mapping**

Add required `updatedAt: string` to `CloudRunningScheme`. `instanceToDisplay` returns `updatedAt`; `mergeCloudSchemesStable` must retain the newer value. REST rows and WS snapshots therefore use the same comparison field.

- [ ] **Step 4: Implement the explicit synchronization state machine**

Use this public shape:

```ts
export interface CloudRunningSyncHandlers {
  onSchemes(cards: CloudSchemeCard[], removedIds: string[]): void
  onStats(stats: CloudCenterStatsDto): void
}

export function startCloudRunningSync(
  getLoadedIds: () => string[],
  handlers: CloudRunningSyncHandlers,
  options?: { legacyFallbackMs?: number },
): { stop(): void; reconcile(): Promise<void> }
```

Add `CLOUD_REALTIME_CLIENT_ENABLED = import.meta.env.VITE_CLOUD_REALTIME_ENABLED !== 'false'` in `client/src/api/config.ts`. Maintain `versions: Map<string,string>`, `reconcilePromise`, `bufferedSchemeMessages`, and `hasVersionedSnapshot`. A connection-ready callback starts one reconciliation promise that fetches currently loaded IDs and stats concurrently. Snapshot messages received during it are appended, then replayed in arrival order with `updatedAt` checks. No timer exists when both WebSocket and realtime-client mode are enabled, whether connected or reconnecting. If WebSocket or realtime-client mode is disabled at build time, retain the existing 15-second disconnected/60-second connected legacy polling behavior for rollback compatibility.

- [ ] **Step 5: Verify focused tests and client build**

Run:

```powershell
cd client
npm.cmd test -- src/composables/useCloudRunningPoll.spec.ts src/composables/ws/useClientWs.spec.ts
npm.cmd run build
```

Expected: PASS with no implicit-any or stale import errors.

- [ ] **Step 6: Commit the synchronization state machine**

```powershell
git add -- client/src/api/types.ts client/src/api/config.ts client/src/api/cloud/center.ts client/src/composables/useCloudRunningPoll.ts client/src/composables/useCloudRunningPoll.spec.ts
git commit -m "feat: sync cloud cards from websocket snapshots"
```

---

### Task 10: Cloud Center View Integration and REST Removal

**Files:**
- Modify: `client/src/views/cloud/CloudCenterView.vue:1-114,171-245,302-370,641-644,833-858`
- Create: `client/src/views/cloud/CloudCenterView.realtime.spec.ts`

**Interfaces:**
- Consumes: Task 9 synchronization handlers.
- Produces: direct card/stat updates without background REST refreshes.

- [ ] **Step 1: Write view behavior tests**

Mock `startCloudRunningSync` and API methods. Mount the view and assert:

```ts
expect(startCloudRunningSync).toHaveBeenCalledTimes(1)
expect(fetchCloudCenterStats).toHaveBeenCalledTimes(1) // initial load only
```

Advance the one-second countdown beyond zero and assert neither `sync.reconcile` nor `fetchCloudCenterStats` is called again. Invoke captured `onSchemes` and verify the original card order is retained; invoke `onStats` and verify the header uses pushed values.

- [ ] **Step 2: Run the view test and verify failure**

Run:

```powershell
cd client
npm.cmd test -- src/views/cloud/CloudCenterView.realtime.spec.ts
```

Expected: FAIL because countdown expiry and waiting state currently call `cloudRefresh`, which fetches both endpoints.

- [ ] **Step 3: Remove timer/action-driven REST refreshes**

Make `tickSchemeLiveFields` update only `runTimeSec`, `countdownSec`, and `countdownLabel`. Delete `cloudRefresh` and `lastWaitingRefreshAt`. Remove `refreshCloudStats()` after start/stop/bulk-start; realtime stats follows the committed mutation within the one-second stats window.

Keep `refreshCloudStats` for initial `loadCloudData` only. User action API responses continue to call `patchSchemeCard` immediately.

- [ ] **Step 4: Start realtime synchronization after initial REST state exists**

Mount order:

```ts
onMounted(async () => {
  countdownTimer = window.setInterval(tickSchemeLiveFields, 1000)
  await loadCloudData()
  const sync = startCloudRunningSync(
    () => runningSchemes.value.map((s) => s.id),
    {
      onSchemes: (cards, removedIds) => applyRealtimeSchemes(cards, removedIds),
      onStats: (stats) => { centerStats.value = stats },
    },
  )
  stopCloudPoll = sync.stop
})
```

`applyRealtimeSchemes` filters updates to loaded IDs, calls `applyRunningSchemes(cards, true)`, removes `removedIds`, decrements `listTotal` only for cards actually removed, and never reorders or appends search results.

- [ ] **Step 5: Run the view, composable, and production build checks**

Run:

```powershell
cd client
npm.cmd test -- src/views/cloud/CloudCenterView.realtime.spec.ts src/composables/useCloudRunningPoll.spec.ts
npm.cmd run build
```

Expected: PASS.

- [ ] **Step 6: Commit Cloud Center integration**

```powershell
git add -- client/src/views/cloud/CloudCenterView.vue client/src/views/cloud/CloudCenterView.realtime.spec.ts
git commit -m "perf: remove cloud center background polling"
```

---

### Task 11: Cluster Documentation, Compatibility Smoke, and Final Verification

**Files:**
- Create: `backend/docs/cloud-realtime-nats.md`
- Modify: `backend/docs/websocket.md`
- Create: `backend/cmd/cloud-realtime-smoke/main.go`
- Create: `backend/internal/cloudrealtime/cluster_integration_test.go`
- Modify: `scripts/restart-prod.sh`

**Interfaces:**
- Consumes: all previous tasks.
- Produces: an operator smoke command and deployment/runbook evidence.

- [ ] **Step 1: Add a two-client NATS cluster integration test**

Guard it with `NATS_TEST_URL`. Create two bus clients to represent Worker/API nodes, subscribe the API side for member 7, publish member 7 and member 8 snapshots from the Worker side, and assert only member 7 arrives. Disconnect/reconnect the API client and assert its connection callback requests browser closure; after resubscribe, one new snapshot arrives.

- [ ] **Step 2: Add an operator smoke command**

`cloud-realtime-smoke` accepts `-nats`, `-prefix`, and `-member-id`, subscribes to the member's scheme/stat subjects, prints schema version/event counts without payload secrets, publishes a synthetic diagnostic event only when `-publish` is explicitly supplied, and exits non-zero on timeout.

Example read-only run:

```bash
cd /opt/caipiao/backend
go run ./cmd/cloud-realtime-smoke -nats nats://10.0.0.11:4222 -member-id 7 -timeout 15s
```

- [ ] **Step 3: Document production topology and phased rollout**

The runbook must include:

- a three-node NATS cluster example with routes;
- firewall ports `4222` for clients and `6222` for NATS routes;
- dedicated credentials for caipiao publishers/subscribers;
- systemd environment variables for API-only and Worker nodes;
- backend-first, frontend-second rollout order;
- verification of `/admin/diagnostics/cloud-realtime`;
- browser Network-panel proof that stable WS produces zero `/running?ids` and `/stats` background calls;
- rollback by setting backend `CLOUD_REALTIME_ENABLED=false`, rebuilding the client with `VITE_CLOUD_REALTIME_ENABLED=false`, and thereby restoring legacy polling;
- removal criteria for legacy invalidation after all clients are upgraded.

Update `restart-prod.sh` to print a post-restart warning when realtime is enabled but `NATS_URL` is absent; do not install or mutate NATS automatically.

- [ ] **Step 4: Run targeted race, frontend, and build suites**

Run:

```powershell
cd backend
go test -race ./internal/realtimebus ./internal/cloudrealtime ./internal/cloudrealtime/wsbridge ./internal/ws ./internal/schemes ./internal/handler -count=1
go build ./cmd/server ./cmd/cloud-realtime-smoke
cd ..\client
npm.cmd test -- src/composables/ws/useClientWs.spec.ts src/composables/useCloudRunningPoll.spec.ts src/views/cloud/CloudCenterView.realtime.spec.ts
npm.cmd run build
```

Expected: all targeted checks PASS.

- [ ] **Step 5: Run full repository regression checks and classify pre-existing failures**

Run:

```powershell
cd backend
go test ./... -count=1
cd ..\client
npm.cmd test
npm.cmd run build
cd ..\admin
npm.cmd run build
```

Expected: no new failure attributable to realtime synchronization. Any pre-existing failure must be recorded with its exact test name and reproduced against the pre-feature commit before being classified as pre-existing.

- [ ] **Step 6: Verify the acceptance metrics on a two-API/two-Worker environment**

Record evidence for:

1. Stable WebSocket for five minutes: zero background `/schemes/running` and `/schemes/stats` requests.
2. One forced socket close: exactly one reconciliation cycle after subscription recovery.
3. 100 rapid updates to one instance inside 200ms: one scheme snapshot publication.
4. NATS restart while schemes run: betting/settlement continues; browser sockets close and reconcile after recovery.
5. A slow WebSocket test client: only that connection closes with code `4010`.
6. Two members on different API nodes: no cross-member event.
7. Snapshot delivery P95 under one second from mutation commit to browser handler.

- [ ] **Step 7: Commit operational delivery**

```powershell
git add -- backend/docs/cloud-realtime-nats.md backend/docs/websocket.md backend/cmd/cloud-realtime-smoke backend/internal/cloudrealtime/cluster_integration_test.go scripts/restart-prod.sh
git commit -m "docs: add cloud realtime cluster runbook"
```

---

## Final Review Checklist

- [ ] `git diff --check` reports no whitespace errors.
- [ ] Every task commit contains only files listed in that task or a documented dependency lockfile change.
- [ ] NATS credentials never appear in logs, diagnostics, tests, or committed configuration.
- [ ] All NATS subjects contain numeric member IDs and all deliveries are checked against authenticated connection identity.
- [ ] Publisher and scanner failures are observable but cannot return errors into betting/settlement transactions.
- [ ] REST and realtime use the same scheme/stat projection functions.
- [ ] Browser snapshot application is idempotent and rejects stale `updatedAt` values.
- [ ] Countdown expiry performs no network request.
- [ ] Slow WebSocket clients are disconnected rather than silently losing frames.
- [ ] Reconnect performs one reconciliation only after default client topics are subscribed.
- [ ] Legacy invalidation remains available during phased rollout and is ignored by new clients after the first versioned snapshot.
