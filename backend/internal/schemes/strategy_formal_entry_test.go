package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/playrules"
)

func TestFormalEvaluationCommitsAwaitingTargetBeforeProviderAvailability(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", time.Now().UTC())
	if err := f.process(); err != nil {
		t.Fatalf("ProcessStrategyReady() error = %v", err)
	}
	s := f.snapshot()
	if s.StateVersion != 1 || s.RoundIndex != 1 {
		t.Fatalf("strategy state version=%d round=%d, want 1/1", s.StateVersion, s.RoundIndex)
	}
	if s.DecisionCount != 1 || s.DecisionStatus != "awaiting_target" || !s.TargetDeadlineAt.Valid {
		t.Fatalf("decision count=%d status=%q deadline=%+v", s.DecisionCount, s.DecisionStatus, s.TargetDeadlineAt)
	}
	wantDeadline := f.drawnAt.Add(15*time.Second - guajiPlaceCloseSafety)
	if !s.TargetDeadlineAt.Time.Equal(wantDeadline) {
		t.Fatalf("deadline=%s want=%s", s.TargetDeadlineAt.Time, wantDeadline)
	}
	if s.EvaluationStatus != "completed" || !s.StrategyEvaluatedAt.Valid || s.OutboxCount != 0 {
		t.Fatalf("evaluation=%q cloudMarker=%+v outbox=%d", s.EvaluationStatus, s.StrategyEvaluatedAt, s.OutboxCount)
	}
}

func TestDuplicateFormalEvaluationSequentialRedeliveryAdvancesOnce(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", time.Now().UTC())
	if err := f.process(); err != nil {
		t.Fatal(err)
	}
	if err := f.process(); err != nil {
		t.Fatal(err)
	}
	s := f.snapshot()
	if s.StateVersion != 1 || s.RoundIndex != 1 || s.DecisionCount != 1 || s.EvaluationCount != 1 {
		t.Fatalf("state=%d round=%d decisions=%d evaluations=%d, want 1/1/1/1", s.StateVersion, s.RoundIndex, s.DecisionCount, s.EvaluationCount)
	}
}

func TestDuplicateFormalEvaluationConcurrentCallsAdvanceOnce(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", time.Now().UTC())
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- f.process()
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent ProcessStrategyReady() error = %v", err)
		}
	}
	s := f.snapshot()
	if s.StateVersion != 1 || s.RoundIndex != 1 || s.DecisionCount != 1 || s.EvaluationCount != 1 {
		t.Fatalf("state=%d round=%d decisions=%d evaluations=%d, want 1/1/1/1", s.StateVersion, s.RoundIndex, s.DecisionCount, s.EvaluationCount)
	}
}

func TestFormalEvaluationRejectsIncompletePreseededDecision(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", time.Now().UTC())
	f.preseedIncompleteDecision()
	err := f.process()
	if !errors.Is(err, ErrFormalPhaseOneInconsistentState) {
		t.Fatalf("ProcessStrategyReady() error = %v, want ErrFormalPhaseOneInconsistentState", err)
	}
	s := f.snapshot()
	if s.StateVersion != 0 || s.RoundIndex != 0 || s.DecisionCount != 1 || s.EvaluationCount != 0 || s.StrategyEvaluatedAt.Valid {
		t.Fatalf("state=%d round=%d decisions=%d evaluations=%d cloudMarker=%+v, want unchanged incomplete phase", s.StateVersion, s.RoundIndex, s.DecisionCount, s.EvaluationCount, s.StrategyEvaluatedAt)
	}
}

func TestFormalEvaluationExpiredDeadlineCommitsThenMisses(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", time.Now().UTC().Add(-30*time.Second))
	if err := f.process(); err != nil {
		t.Fatalf("ProcessStrategyReady() error = %v", err)
	}
	s := f.snapshot()
	if s.StateVersion != 1 || s.RoundIndex != 1 || s.DecisionStatus != "missed_contiguous_period" {
		t.Fatalf("state=%d round=%d decision=%q, want committed phase then terminal miss", s.StateVersion, s.RoundIndex, s.DecisionStatus)
	}
	if s.EvaluationStatus != "completed" || !s.StrategyEvaluatedAt.Valid || s.ChainState != "blocked_requires_rearm" || s.OutboxCount != 0 {
		t.Fatalf("evaluation=%q marker=%+v chain=%q outbox=%d", s.EvaluationStatus, s.StrategyEvaluatedAt, s.ChainState, s.OutboxCount)
	}
}

func TestFormalEvaluationNonPositiveIntervalTerminatesConfiguration(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "", time.Now().UTC())
	if err := f.process(); err != nil {
		t.Fatalf("ProcessStrategyReady() error = %v", err)
	}
	s := f.snapshot()
	if s.StateVersion != 1 || s.RoundIndex != 1 || s.DecisionStatus != "chain_broken" || s.ChainState != "blocked_requires_rearm" {
		t.Fatalf("state=%d round=%d decision=%q chain=%q, want terminal configuration failure", s.StateVersion, s.RoundIndex, s.DecisionStatus, s.ChainState)
	}
	if s.EvaluationStatus != "completed" || !s.StrategyEvaluatedAt.Valid || s.OutboxCount != 0 {
		t.Fatalf("evaluation=%q marker=%+v outbox=%d", s.EvaluationStatus, s.StrategyEvaluatedAt, s.OutboxCount)
	}
}

type formalStrategyEntryFixture struct {
	t           *testing.T
	ctx         context.Context
	pool        *db.Pool
	processor   *StrategyProcessor
	memberID    int64
	definition  string
	schemeID    string
	lotteryCode string
	periodNo    string
	recordID    int64
	drawnAt     time.Time
}

type formalStrategyEntrySnapshot struct {
	StateVersion        int64
	RoundIndex          int32
	ChainState          string
	DecisionCount       int
	DecisionStatus      string
	TargetDeadlineAt    pgtype.Timestamptz
	EvaluationCount     int
	EvaluationStatus    string
	StrategyEvaluatedAt pgtype.Timestamptz
	OutboxCount         int
}

func newFormalStrategyEntryFixture(t *testing.T, drawInterval string, drawnAt time.Time) *formalStrategyEntryFixture {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, 6, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `SELECT target_deadline_at, target_period_no, failure_reason FROM scheme_period_decisions WHERE FALSE`); err != nil {
		t.Skipf("migration 177 not applied: %v", err)
	}
	if _, err := pool.Exec(ctx, `SELECT chain_block_reason FROM scheme_instances WHERE FALSE`); err != nil {
		t.Skipf("migration 177 not applied: %v", err)
	}

	stamp := time.Now().UnixNano()
	f := &formalStrategyEntryFixture{
		t: t, ctx: ctx, pool: pool, definition: fmt.Sprintf("formal-def-%d", stamp),
		schemeID: fmt.Sprintf("formal-inst-%d", stamp), lotteryCode: fmt.Sprintf("fp_%d", stamp),
		periodNo: fmt.Sprintf("p%d", stamp), drawnAt: drawnAt.UTC(),
	}
	t.Cleanup(f.cleanup)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
INSERT INTO lottery_catalog
    (code, display_name, category_code, play_template, ball_count, draw_interval,
     sort_order, on_sale, sale_status, outbound_lottery_code)
VALUES ($1, 'formal phase test', 'jisu', 'ssc_std', 5, NULLIF($2, ''), 9999, true, 'on_sale', 'formal-test')`, f.lotteryCode, drawInterval); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'formal phase entry test', 'active')
RETURNING id`, fmt.Sprintf("fpa%d", stamp)).Scan(&f.memberID); err != nil {
		t.Fatal(err)
	}
	definitionConfig := []byte(`{"runTypeId":"fixed","schemeGroups":["1"],"rounds":[{"mult":1,"afterHit":2,"afterMiss":2},{"mult":2,"afterHit":2,"afterMiss":2}]}`)
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'formal phase entry test', $3, 'test', 'private', $4::jsonb)`, f.definition, f.memberID, f.lotteryCode, definitionConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_instances
    (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status,
     sim_bet, betting_owner, strict_chain_state, chain_id, chain_seq, state_version)
VALUES ($1, $2, $3, 'custom', 'formal phase entry test', $4, 'test', 'running',
        false, 'event', 'active', $5, 1, 0)`, f.schemeID, f.definition, f.memberID, f.lotteryCode, fmt.Sprintf("formal-chain-%d", stamp)); err != nil {
		t.Fatal(err)
	}
	snapshot, err := json.Marshal(playrules.Snapshot{
		Locator:        playrules.Locator{TemplateCode: "ssc_std", TypeID: "g001", SubID: "1"},
		EvaluatorKey:   "ssc.direct",
		EvaluationSpec: []byte(`{"mode":"direct","numberMin":0,"numberMax":9,"segmentStart":0,"segmentLen":3,"betMode":"zhixuan_fs"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at)
VALUES ($1, $2, '1', '["1","2","3","4","5"]'::jsonb, 15, $3)`, f.lotteryCode, f.periodNo, f.drawnAt); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
INSERT INTO cloud_bet_records
    (record_no, member_id, sim_bet, scheme_id, scheme_name, lottery_code, period_no, play_type,
     multiplier, round_label, amount, status, bet_content, third_party_bet_id,
     rule_snapshot, rule_version, rule_snapshot_hash)
VALUES ($1, $2, false, $3, 'formal phase entry test', $4, $5, 'test',
        '1', '1/2', 1, 'hit', '1\n2\n3', $6, $7::jsonb, 1, 'formal-phase-rule')
RETURNING id`, fmt.Sprintf("FPR%d", stamp), f.memberID, f.schemeID, f.lotteryCode, f.periodNo, fmt.Sprintf("accepted-%d", stamp), snapshot).Scan(&f.recordID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	f.processor = NewStrategyProcessor(pool)
	f.processor.SetBettingMode("gray", []string{f.lotteryCode})
	return f
}

func (f *formalStrategyEntryFixture) process() error {
	return f.processor.ProcessStrategyReady(f.ctx, f.recordID, f.schemeID, f.lotteryCode, f.periodNo, 0)
}

func (f *formalStrategyEntryFixture) preseedIncompleteDecision() {
	f.t.Helper()
	deadline := f.drawnAt.Add(15*time.Second - guajiPlaceCloseSafety)
	_, created, err := sqlcdb.New(f.pool).InsertSchemePeriodDecision(f.ctx, sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: f.schemeID, LotteryCode: f.lotteryCode, SourcePeriodNo: f.periodNo,
		SourceBetRecordID: f.recordID, StateVersionBefore: 0, StateVersionAfter: 1,
		RuleVersion: pgtype.Int4{Int32: 1, Valid: true}, RuleSnapshotHash: pgtype.Text{String: "formal-phase-rule", Valid: true},
		LocalHit: true, WinningUnits: 1, Status: "awaiting_target",
		TargetDeadlineAt: pgtype.Timestamptz{Time: deadline, Valid: true},
	})
	if err != nil || !created {
		f.t.Fatalf("preseed decision created=%v err=%v", created, err)
	}
}

func (f *formalStrategyEntryFixture) snapshot() formalStrategyEntrySnapshot {
	f.t.Helper()
	var s formalStrategyEntrySnapshot
	if err := f.pool.QueryRow(f.ctx, `SELECT state_version, round_index, strict_chain_state FROM scheme_instances WHERE id = $1`, f.schemeID).Scan(&s.StateVersion, &s.RoundIndex, &s.ChainState); err != nil {
		f.t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `
SELECT COUNT(*)::int, COALESCE(MAX(status), ''), MAX(target_deadline_at)
FROM scheme_period_decisions WHERE scheme_id = $1 AND source_period_no = $2`, f.schemeID, f.periodNo).Scan(&s.DecisionCount, &s.DecisionStatus, &s.TargetDeadlineAt); err != nil {
		f.t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `
SELECT COUNT(*)::int, COALESCE(MAX(status), '')
FROM scheme_strategy_evaluations WHERE instance_id = $1 AND period_no = $2`, f.schemeID, f.periodNo).Scan(&s.EvaluationCount, &s.EvaluationStatus); err != nil {
		f.t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT strategy_evaluated_at FROM cloud_bet_records WHERE id = $1`, f.recordID).Scan(&s.StrategyEvaluatedAt); err != nil {
		f.t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*)::int FROM scheme_bet_outbox WHERE scheme_id = $1`, f.schemeID).Scan(&s.OutboxCount); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func (f *formalStrategyEntryFixture) cleanup() {
	ctx := context.Background()
	_, _ = f.pool.Exec(ctx, `DELETE FROM scheme_bet_outbox WHERE scheme_id = $1`, f.schemeID)
	_, _ = f.pool.Exec(ctx, `DELETE FROM scheme_period_decisions WHERE scheme_id = $1`, f.schemeID)
	_, _ = f.pool.Exec(ctx, `DELETE FROM scheme_strategy_evaluations WHERE instance_id = $1`, f.schemeID)
	_, _ = f.pool.Exec(ctx, `DELETE FROM cloud_bet_records WHERE scheme_id = $1`, f.schemeID)
	_, _ = f.pool.Exec(ctx, `DELETE FROM lottery_draws WHERE lottery_code = $1`, f.lotteryCode)
	_, _ = f.pool.Exec(ctx, `DELETE FROM scheme_instances WHERE id = $1`, f.schemeID)
	_, _ = f.pool.Exec(ctx, `DELETE FROM scheme_definitions WHERE id = $1`, f.definition)
	if f.memberID > 0 {
		_, _ = f.pool.Exec(ctx, `DELETE FROM members WHERE id = $1`, f.memberID)
	}
	_, _ = f.pool.Exec(ctx, `DELETE FROM lottery_catalog WHERE code = $1`, f.lotteryCode)
}
