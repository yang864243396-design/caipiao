package schemes

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/lottery"
)

func TestResolveAwaitingTargetRequiresExactSourceAndImmediateSuccessor(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wsCurrent string
		wsNext    string
		want      contiguousTargetResolution
	}{
		{
			name:   "exact source accepts immediate successor",
			source: "100", wsCurrent: "100", wsNext: "101",
			want: contiguousTargetResolved,
		},
		{
			name:   "advanced websocket current misses chain instead of skipping",
			source: "100", wsCurrent: "101", wsNext: "102",
			want: contiguousTargetMissed,
		},
		{
			name:   "non immediate websocket target misses chain",
			source: "100", wsCurrent: "100", wsNext: "102",
			want: contiguousTargetMissed,
		},
		{
			name:   "unrelated older boundary leaves wait recoverable",
			source: "100", wsCurrent: "99", wsNext: "100",
			want: contiguousTargetWaiting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyContiguousTargetBoundary(tt.source, tt.wsCurrent, tt.wsNext); got != tt.want {
				t.Fatalf("classifyContiguousTargetBoundary(%q, %q, %q) = %v, want %v", tt.source, tt.wsCurrent, tt.wsNext, got, tt.want)
			}
		})
	}
}

func TestImmediateContiguousSuccessorRejectsSkippedAndAmbiguousIssues(t *testing.T) {
	tests := []struct {
		source string
		target string
		want   bool
	}{
		{source: "100", target: "101", want: true},
		{source: "P0099", target: "P0100", want: true},
		{source: "100", target: "102"},
		{source: "P99", target: "Q100"},
		{source: "P99", target: "P099"},
	}
	for _, tt := range tests {
		if got := isImmediateContiguousSuccessor(tt.source, tt.target); got != tt.want {
			t.Fatalf("isImmediateContiguousSuccessor(%q, %q) = %v, want %v", tt.source, tt.target, got, tt.want)
		}
	}
}

// This schema-gated test proves the resolver's terminal branches use its
// locked database transaction: neither a passed source nor an expired wait
// can create an outbox or advance strategy state. It never applies migrations.
func TestResolveAwaitingTargetTerminatesGapAndDeadlineWithoutOutbox(t *testing.T) {
	f := newResolverTerminalFixture(t)
	t.Run("gap", func(t *testing.T) {
		decisionID := f.seed("100", f.databaseNow().Add(time.Minute))
		if !lottery.UpdatePeriodState("tron_ffc_6s", "101", "102", time.Now().UTC(), 6) {
			t.Fatal("set websocket boundary")
		}
		if err := f.processor.ResolveAwaitingTarget(f.ctx, decisionID); err != nil {
			t.Fatal(err)
		}
		f.assertMissedWithoutOutbox(decisionID)
	})
	t.Run("deadline", func(t *testing.T) {
		decisionID := f.seed("200", f.databaseNow().Add(-time.Millisecond))
		if err := f.processor.ResolveAwaitingTarget(f.ctx, decisionID); err != nil {
			t.Fatal(err)
		}
		f.assertMissedWithoutOutbox(decisionID)
	})
}

type resolverTerminalFixture struct {
	t          *testing.T
	ctx        context.Context
	pool       *db.Pool
	processor  *StrategyProcessor
	memberID   int64
	definition string
	schemeID   string
}

func newResolverTerminalFixture(t *testing.T) *resolverTerminalFixture {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 4, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), `SELECT target_deadline_at, target_period_no, failure_reason, shard_no FROM scheme_period_decisions WHERE FALSE`); err != nil {
		t.Skipf("migration 177 not applied: %v", err)
	}
	stamp := time.Now().UnixNano()
	f := &resolverTerminalFixture{t: t, ctx: context.Background(), pool: pool, processor: NewStrategyProcessor(pool), definition: fmt.Sprintf("resolver-def-%d", stamp), schemeID: fmt.Sprintf("resolver-inst-%d", stamp)}
	t.Cleanup(f.cleanup)
	if err := pool.QueryRow(f.ctx, `INSERT INTO members (account, password_hash, display_name, status) VALUES ($1, 'test', 'resolver test', 'active') RETURNING id`, fmt.Sprintf("resolver-%d", stamp)).Scan(&f.memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(f.ctx, `INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config) VALUES ($1, $2, 'custom', 'resolver test', 'tron_ffc_6s', 'test', 'private', '{}'::jsonb)`, f.definition, f.memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(f.ctx, `INSERT INTO scheme_instances (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status, sim_bet, betting_owner, strict_chain_state, chain_id, chain_seq, state_version) VALUES ($1, $2, $3, 'custom', 'resolver test', 'tron_ffc_6s', 'test', 'running', false, 'event', 'active', 'resolver-chain', 4, 8)`, f.schemeID, f.definition, f.memberID); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *resolverTerminalFixture) databaseNow() time.Time {
	f.t.Helper()
	var now time.Time
	if err := f.pool.QueryRow(f.ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		f.t.Fatal(err)
	}
	return now
}

func (f *resolverTerminalFixture) seed(source string, deadline time.Time) int64 {
	f.t.Helper()
	var decisionID int64
	if err := f.pool.QueryRow(f.ctx, `INSERT INTO scheme_period_decisions (scheme_id, lottery_code, source_period_no, state_version_before, state_version_after, status, target_deadline_at, shard_no) VALUES ($1, 'tron_ffc_6s', $2, 7, 8, 'awaiting_target', $3, 3) RETURNING id`, f.schemeID, source, deadline).Scan(&decisionID); err != nil {
		f.t.Fatal(err)
	}
	return decisionID
}

func (f *resolverTerminalFixture) assertMissedWithoutOutbox(decisionID int64) {
	f.t.Helper()
	var status, instanceStatus, reason, chainState string
	var stateVersion int64
	var outboxCount int
	if err := f.pool.QueryRow(f.ctx, `SELECT d.status, i.status, i.status_reason, i.strict_chain_state, i.state_version, (SELECT COUNT(*)::int FROM scheme_bet_outbox o WHERE o.decision_id = d.id) FROM scheme_period_decisions d JOIN scheme_instances i ON i.id = d.scheme_id WHERE d.id = $1`, decisionID).Scan(&status, &instanceStatus, &reason, &chainState, &stateVersion, &outboxCount); err != nil {
		f.t.Fatal(err)
	}
	if status != "missed_contiguous_period" || instanceStatus != "paused" || reason != "bet_failed" || chainState != "blocked_requires_rearm" || stateVersion != 8 || outboxCount != 0 {
		f.t.Fatalf("decision=%q instance=%q reason=%q chain=%q state=%d outbox=%d", status, instanceStatus, reason, chainState, stateVersion, outboxCount)
	}
}

func (f *resolverTerminalFixture) cleanup() {
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM scheme_bet_outbox WHERE scheme_id = $1`, f.schemeID)
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM scheme_period_decisions WHERE scheme_id = $1`, f.schemeID)
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM scheme_instances WHERE id = $1`, f.schemeID)
	_, _ = f.pool.Exec(context.Background(), `DELETE FROM scheme_definitions WHERE id = $1`, f.definition)
	if f.memberID > 0 {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM members WHERE id = $1`, f.memberID)
	}
}
