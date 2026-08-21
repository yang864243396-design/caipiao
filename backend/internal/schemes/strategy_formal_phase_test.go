package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/playrules"
)

func TestContiguousTargetDeadlineUsesPersistedDrawAndSafetyWindow(t *testing.T) {
	drawnAt := time.Date(2026, time.August, 21, 10, 0, 0, 0, time.UTC)

	deadline, err := contiguousTargetDeadline(drawnAt, 6, 1200*time.Millisecond)
	if err != nil {
		t.Fatalf("contiguousTargetDeadline() error = %v", err)
	}
	want := time.Date(2026, time.August, 21, 10, 0, 4, 800000000, time.UTC)
	if !deadline.Equal(want) {
		t.Fatalf("deadline = %s, want %s", deadline, want)
	}
}

func TestContiguousTargetDeadlineRejectsNonPositiveIntervalAsConfigurationError(t *testing.T) {
	_, err := contiguousTargetDeadline(time.Date(2026, time.August, 21, 10, 0, 0, 0, time.UTC), 0, 1200*time.Millisecond)
	if !errors.Is(err, ErrContiguousTargetConfiguration) {
		t.Fatalf("error = %v, want ErrContiguousTargetConfiguration", err)
	}
}

func TestFormalEvaluationReservesDecisionBeforeStrategyAdvance(t *testing.T) {
	store := &fakeFormalPhaseOneDecisionStore{created: true, decisionID: 41}
	params := formalPhaseOneTestDecisionParams()

	decisionID, created, err := reserveFormalPhaseOneDecision(
		context.Background(), store, params,
		func() error {
			store.calls = append(store.calls, "advance")
			return nil
		},
	)
	if err != nil {
		t.Fatalf("reserveFormalPhaseOneDecision() error = %v", err)
	}
	if decisionID != 41 || !created {
		t.Fatalf("decision id=%d created=%v, want id=41 created", decisionID, created)
	}
	want := []string{"reserve", "advance"}
	if fmt.Sprint(store.calls) != fmt.Sprint(want) {
		t.Fatalf("calls = %v, want %v", store.calls, want)
	}
}

func TestDuplicateFormalEvaluationDoesNotAdvanceCompletedPhaseOne(t *testing.T) {
	params := formalPhaseOneTestDecisionParams()
	store := &fakeFormalPhaseOneDecisionStore{
		decisionID: 41,
		existing:   matchingFormalPhaseOneDecisionState(41, params),
	}
	advanced := false

	decisionID, created, err := reserveFormalPhaseOneDecision(
		context.Background(), store, params,
		func() error {
			advanced = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("reserveFormalPhaseOneDecision() error = %v", err)
	}
	if decisionID != 41 || created || advanced {
		t.Fatalf("decision id=%d created=%v advanced=%v, want existing idempotent success", decisionID, created, advanced)
	}
	want := []string{"reserve", "validate"}
	if fmt.Sprint(store.calls) != fmt.Sprint(want) {
		t.Fatalf("calls = %v, want %v", store.calls, want)
	}
}

func TestFormalEvaluationRejectsIncompleteExistingDecisionBeforeAdvance(t *testing.T) {
	params := formalPhaseOneTestDecisionParams()
	existing := matchingFormalPhaseOneDecisionState(41, params)
	existing.EvaluationStatus = pgtype.Text{String: "processing", Valid: true}
	existing.EvaluationCompletedAt = pgtype.Timestamptz{}
	existing.CloudStrategyEvaluatedAt = pgtype.Timestamptz{}
	store := &fakeFormalPhaseOneDecisionStore{decisionID: 41, existing: existing}
	advanced := false

	_, _, err := reserveFormalPhaseOneDecision(
		context.Background(), store, params,
		func() error {
			advanced = true
			return nil
		},
	)
	if !errors.Is(err, ErrFormalPhaseOneInconsistentState) {
		t.Fatalf("error = %v, want ErrFormalPhaseOneInconsistentState", err)
	}
	if advanced {
		t.Fatal("incomplete existing decision advanced strategy state")
	}
}

func TestDuplicateFormalConfigurationRequiresExactTerminalChainState(t *testing.T) {
	params := formalPhaseOneTestDecisionParams()
	params.Status = "chain_broken"
	params.Diagnostics = []byte(`{"reason":"contiguous_target_configuration"}`)
	params.TargetDeadlineAt = pgtype.Timestamptz{}

	tests := []struct {
		name        string
		chainState  string
		blockReason pgtype.Text
	}{
		{name: "chain remains active", chainState: "active"},
		{name: "wrong block reason", chainState: "blocked_requires_rearm", blockReason: pgtype.Text{String: "wrong_reason", Valid: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := matchingFormalPhaseOneDecisionState(41, params)
			existing.ExecutionChainState = tt.chainState
			existing.ExecutionChainBlockReason = tt.blockReason
			store := &fakeFormalPhaseOneDecisionStore{decisionID: 41, existing: existing}
			advanced := false

			_, _, err := reserveFormalPhaseOneDecision(context.Background(), store, params, func() error {
				advanced = true
				return nil
			})
			if !errors.Is(err, ErrFormalPhaseOneInconsistentState) {
				t.Fatalf("error = %v, want ErrFormalPhaseOneInconsistentState", err)
			}
			if advanced {
				t.Fatal("malformed terminal chain advanced strategy state")
			}
		})
	}
}

func TestDuplicateFormalConfigurationAcceptsExactTerminalChainState(t *testing.T) {
	params := formalPhaseOneTestDecisionParams()
	params.Status = "chain_broken"
	params.Diagnostics = []byte(`{"reason":"contiguous_target_configuration"}`)
	params.TargetDeadlineAt = pgtype.Timestamptz{}
	store := &fakeFormalPhaseOneDecisionStore{
		decisionID: 41,
		existing:   matchingFormalPhaseOneDecisionState(41, params),
	}

	_, created, err := reserveFormalPhaseOneDecision(context.Background(), store, params, func() error {
		t.Fatal("matching terminal duplicate advanced strategy state")
		return nil
	})
	if err != nil || created {
		t.Fatalf("created=%v error=%v, want existing terminal idempotent success", created, err)
	}
}

func TestDuplicateFormalNonConfigurationBlockUsesRecordedReason(t *testing.T) {
	params := formalPhaseOneTestDecisionParams()
	params.Status = "chain_broken"
	params.Diagnostics = []byte(`{"reason":"chain_not_active"}`)
	params.TargetDeadlineAt = pgtype.Timestamptz{}
	existing := matchingFormalPhaseOneDecisionState(41, params)
	existing.ExecutionChainBlockReason = pgtype.Text{String: "chain_not_active", Valid: true}
	store := &fakeFormalPhaseOneDecisionStore{decisionID: 41, existing: existing}

	_, created, err := reserveFormalPhaseOneDecision(context.Background(), store, params, func() error {
		t.Fatal("matching non-configuration terminal duplicate advanced strategy state")
		return nil
	})
	if err != nil || created {
		t.Fatalf("created=%v error=%v, want recorded terminal reason accepted", created, err)
	}
}

func TestDuplicateFormalAwaitingRequiresActiveUnblockedChain(t *testing.T) {
	params := formalPhaseOneTestDecisionParams()
	existing := matchingFormalPhaseOneDecisionState(41, params)
	existing.ExecutionChainState = "blocked_requires_rearm"
	existing.ExecutionChainBlockReason = pgtype.Text{String: "contiguous_target_configuration", Valid: true}
	store := &fakeFormalPhaseOneDecisionStore{decisionID: 41, existing: existing}

	_, _, err := reserveFormalPhaseOneDecision(context.Background(), store, params, func() error {
		t.Fatal("malformed awaiting duplicate advanced strategy state")
		return nil
	})
	if !errors.Is(err, ErrFormalPhaseOneInconsistentState) {
		t.Fatalf("error = %v, want ErrFormalPhaseOneInconsistentState", err)
	}
}

func TestFormalAwaitingRedeliveryRetriesConditionalMissAndPropagatesError(t *testing.T) {
	wantErr := errors.New("transient miss update failure")
	calls := 0
	evidence := formalCommittedPhaseOneEvidence{
		DecisionID:       41,
		Status:           "awaiting_target",
		TargetDeadlineAt: pgtype.Timestamptz{Time: time.Date(2026, time.August, 21, 10, 0, 4, 0, time.UTC), Valid: true},
	}

	err := retryExpiredFormalWaitTransition(context.Background(), evidence, func(context.Context, int64) (bool, error) {
		calls++
		return false, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want transient update error for redelivery", err)
	}
	if calls != 1 {
		t.Fatalf("conditional miss calls = %d, want 1", calls)
	}
}

func TestFormalAwaitingRedeliveryAcknowledgesConditionalMissResult(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprintf("changed_%v", changed), func(t *testing.T) {
			calls := 0
			evidence := formalCommittedPhaseOneEvidence{
				DecisionID:       41,
				Status:           "awaiting_target",
				TargetDeadlineAt: pgtype.Timestamptz{Time: time.Date(2026, time.August, 21, 10, 0, 4, 0, time.UTC), Valid: true},
			}
			err := retryExpiredFormalWaitTransition(context.Background(), evidence, func(context.Context, int64) (bool, error) {
				calls++
				return changed, nil
			})
			if err != nil {
				t.Fatalf("error = %v, want ACK", err)
			}
			if calls != 1 {
				t.Fatalf("conditional miss calls = %d, want 1", calls)
			}
		})
	}
}

func TestFormalTerminalRedeliveryAcknowledgesWithoutMissRetry(t *testing.T) {
	for _, status := range []string{"missed_contiguous_period", "completed", "chain_broken"} {
		t.Run(status, func(t *testing.T) {
			calls := 0
			err := retryExpiredFormalWaitTransition(context.Background(), formalCommittedPhaseOneEvidence{
				DecisionID: 41,
				Status:     status,
			}, func(context.Context, int64) (bool, error) {
				calls++
				return false, errors.New("must not be called")
			})
			if err != nil {
				t.Fatalf("error = %v, want terminal ACK", err)
			}
			if calls != 0 {
				t.Fatalf("conditional miss calls = %d, want no terminal retry storm", calls)
			}
		})
	}
}

func TestFormalAwaitingExactRedeliveryResolvesOnceAndPropagatesTransientError(t *testing.T) {
	wantErr := errors.New("transient resolver failure")
	calls := 0
	err := resolveCommittedFormalWaitTransition(context.Background(), formalCommittedPhaseOneEvidence{
		DecisionID:       41,
		Status:           "awaiting_target",
		TargetDeadlineAt: pgtype.Timestamptz{Time: time.Date(2026, time.August, 21, 10, 0, 4, 0, time.UTC), Valid: true},
	}, func(_ context.Context, decisionID int64) error {
		calls++
		if decisionID != 41 {
			t.Fatalf("decision=%d want=41", decisionID)
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want transient resolver failure", err)
	}
	if calls != 1 {
		t.Fatalf("resolver calls=%d want=1", calls)
	}
}

func TestFormalTerminalExactRedeliveryAcknowledgesWithoutResolutionStorm(t *testing.T) {
	for _, status := range []string{"completed", "missed_contiguous_period", "chain_broken"} {
		t.Run(status, func(t *testing.T) {
			calls := 0
			err := resolveCommittedFormalWaitTransition(context.Background(), formalCommittedPhaseOneEvidence{
				DecisionID: 41,
				Status:     status,
			}, func(context.Context, int64) error {
				calls++
				return errors.New("must not resolve terminal decision")
			})
			if err != nil {
				t.Fatalf("error=%v want ACK", err)
			}
			if calls != 0 {
				t.Fatalf("resolver calls=%d want=0", calls)
			}
		})
	}
}

type fakeFormalPhaseOneDecisionStore struct {
	created    bool
	decisionID int64
	existing   sqlcdb.FormalPhaseOneDecisionState
	calls      []string
}

func (f *fakeFormalPhaseOneDecisionStore) InsertSchemePeriodDecision(context.Context, sqlcdb.InsertSchemePeriodDecisionParams) (int64, bool, error) {
	f.calls = append(f.calls, "reserve")
	return f.decisionID, f.created, nil
}

func (f *fakeFormalPhaseOneDecisionStore) GetFormalPhaseOneDecisionStateForUpdate(context.Context, string, string) (sqlcdb.FormalPhaseOneDecisionState, bool, error) {
	f.calls = append(f.calls, "validate")
	return f.existing, true, nil
}

func formalPhaseOneTestDecisionParams() sqlcdb.InsertSchemePeriodDecisionParams {
	return sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: "scheme-1", LotteryCode: "lottery-1", SourcePeriodNo: "period-1",
		SourceBetRecordID: 77, DrawHash: "draw-hash", StateVersionBefore: 3, StateVersionAfter: 4,
		RuleVersion: pgtype.Int4{Int32: 2, Valid: true}, RuleSnapshotHash: pgtype.Text{String: "rule-hash", Valid: true},
		LocalHit: true, WinningUnits: 1, Status: "awaiting_target",
		TargetDeadlineAt: pgtype.Timestamptz{Time: time.Date(2026, time.August, 21, 10, 0, 4, 800000000, time.UTC), Valid: true},
	}
}

func matchingFormalPhaseOneDecisionState(decisionID int64, params sqlcdb.InsertSchemePeriodDecisionParams) sqlcdb.FormalPhaseOneDecisionState {
	state := sqlcdb.FormalPhaseOneDecisionState{
		DecisionID: decisionID, SchemeID: params.SchemeID, LotteryCode: params.LotteryCode,
		SourcePeriodNo: params.SourcePeriodNo, SourceBetRecordID: pgtype.Int8{Int64: params.SourceBetRecordID, Valid: true},
		DrawHash:           pgtype.Text{String: params.DrawHash, Valid: true},
		StateVersionBefore: params.StateVersionBefore, StateVersionAfter: params.StateVersionAfter,
		RuleVersion: params.RuleVersion, RuleSnapshotHash: params.RuleSnapshotHash,
		LocalHit: pgtype.Bool{Bool: params.LocalHit, Valid: true}, WinningUnits: pgtype.Int4{Int32: int32(params.WinningUnits), Valid: true},
		Status: params.Status, DecisionDiagnostics: params.Diagnostics,
		TargetDeadlineAt: params.TargetDeadlineAt, CurrentStateVersion: params.StateVersionAfter,
		EvaluationStatus: pgtype.Text{String: "completed", Valid: true}, EvaluationCloudBetRecordID: pgtype.Int8{Int64: params.SourceBetRecordID, Valid: true},
		EvaluationRuleVersion: params.RuleVersion, EvaluationRuleSnapshotHash: params.RuleSnapshotHash,
		EvaluationLocalHit: pgtype.Bool{Bool: params.LocalHit, Valid: true}, EvaluationWinningUnits: pgtype.Int4{Int32: int32(params.WinningUnits), Valid: true},
		EvaluationCompletedAt:    pgtype.Timestamptz{Time: time.Date(2026, time.August, 21, 10, 0, 1, 0, time.UTC), Valid: true},
		CloudStrategyEvaluatedAt: pgtype.Timestamptz{Time: time.Date(2026, time.August, 21, 10, 0, 1, 0, time.UTC), Valid: true},
	}
	if params.Status == "chain_broken" {
		state.ExecutionChainState = "blocked_requires_rearm"
		state.ExecutionChainBlockReason = pgtype.Text{String: "contiguous_target_configuration", Valid: true}
	} else {
		state.ExecutionChainState = "active"
	}
	return state
}

func TestFormalEvaluationPersistsAwaitingTargetWithoutOutbox(t *testing.T) {
	f := newFormalStrategyFixture(t)
	deadline := f.drawnAt.Add(6*time.Second - guajiPlaceCloseSafety)
	decision, err := f.processor.persistFormalAwaitingTarget(
		f.ctx, f.q, f.row, f.instance, f.definitionConfig, 0,
		schemestate.FormalRuleEvaluation{Hit: true, WinningUnits: 1}, deadline,
	)
	if err != nil {
		t.Fatalf("persistFormalAwaitingTarget() error = %v", err)
	}
	if decision.Status != "awaiting_target" || decision.DecisionID == 0 {
		t.Fatalf("decision = %+v, want persisted awaiting target", decision)
	}
	if got := f.stateVersion(); got != 1 {
		t.Fatalf("state version = %d, want 1", got)
	}
	if got := f.decisionStatus(); got != "awaiting_target" {
		t.Fatalf("decision status = %q, want awaiting_target", got)
	}
	if got := f.outboxCount(); got != 0 {
		t.Fatalf("outbox count = %d, want 0", got)
	}
}

func TestFormalAwaitingTargetDecisionIsUniqueForDuplicateSource(t *testing.T) {
	f := newFormalStrategyFixture(t)
	deadline := f.drawnAt.Add(6*time.Second - guajiPlaceCloseSafety)
	params := sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: f.row.SchemeID, LotteryCode: f.row.LotteryCode, SourcePeriodNo: f.row.PeriodNo,
		SourceBetRecordID: f.row.RecordID, StateVersionBefore: 0, StateVersionAfter: 1,
		RuleVersion: f.row.RuleVersion, RuleSnapshotHash: f.row.RuleSnapshotHash,
		Status: "awaiting_target", TargetDeadlineAt: pgtype.Timestamptz{Time: deadline, Valid: true},
	}
	first, created, err := f.q.InsertSchemePeriodDecision(f.ctx, params)
	if err != nil || !created {
		t.Fatalf("first decision id=%d created=%v err=%v", first, created, err)
	}
	second, created, err := f.q.InsertSchemePeriodDecision(f.ctx, params)
	if err != nil || created || second != first {
		t.Fatalf("duplicate decision id=%d created=%v err=%v, want existing id=%d", second, created, err, first)
	}
}

type formalStrategyFixture struct {
	t                *testing.T
	ctx              context.Context
	tx               pgx.Tx
	q                *sqlcdb.Queries
	processor        *StrategyProcessor
	row              sqlcdb.PendingFormalStrategyRow
	instance         sqlcdb.SchemeInstance
	definitionConfig []byte
	drawnAt          time.Time
}

func newFormalStrategyFixture(t *testing.T) *formalStrategyFixture {
	t.Helper()
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	if _, err := tx.Exec(ctx, `
SELECT target_deadline_at, target_period_no, failure_reason
FROM scheme_period_decisions
WHERE FALSE`); err != nil {
		t.Skipf("migration 177 not applied: %v", err)
	}
	if _, err := tx.Exec(ctx, `SELECT chain_block_reason FROM scheme_instances WHERE FALSE`); err != nil {
		t.Skipf("migration 177 not applied: %v", err)
	}

	stamp := time.Now().UnixNano()
	schemeID := fmt.Sprintf("formal-phase-instance-%d", stamp)
	definitionID := fmt.Sprintf("formal-phase-definition-%d", stamp)
	lotteryCode := fmt.Sprintf("formal-phase-lottery-%d", stamp)
	periodNo := fmt.Sprintf("formal-phase-period-%d", stamp)
	var memberID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'formal phase test', 'active')
RETURNING id`, fmt.Sprintf("formal-phase-account-%d", stamp)).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	definitionConfig := []byte(`{"runTypeId":"fixed","schemeGroups":["1"]}`)
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'formal phase test', $3, 'test', 'private', $4::jsonb)`, definitionID, memberID, lotteryCode, definitionConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_instances
    (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status,
     sim_bet, betting_owner, strict_chain_state, chain_id, chain_seq, state_version)
VALUES ($1, $2, $3, 'custom', 'formal phase test', $4, 'test', 'running',
        false, 'event', 'active', 'formal-phase-chain', 1, 0)`, schemeID, definitionID, memberID, lotteryCode); err != nil {
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
	drawnAt := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at)
VALUES ($1, $2, $2, '["1","2","3","4","5"]'::jsonb, 15, $3)`, lotteryCode, periodNo, drawnAt); err != nil {
		t.Fatal(err)
	}
	var recordID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO cloud_bet_records
    (record_no, member_id, sim_bet, scheme_id, scheme_name, lottery_code, period_no, play_type,
     multiplier, round_label, amount, status, bet_content, third_party_bet_id,
     rule_snapshot, rule_version, rule_snapshot_hash)
VALUES ($1, $2, false, $3, 'formal phase test', $4, $5, 'test',
        '1', '1/1', 1, 'hit', '1\n2\n3', 'accepted-formal-phase', $6::jsonb, 1, 'formal-phase-rule')
RETURNING id`, fmt.Sprintf("formal-phase-record-%d", stamp), memberID, schemeID, lotteryCode, periodNo, snapshot).Scan(&recordID); err != nil {
		t.Fatal(err)
	}
	instance, err := sqlcdb.New(tx).GetSchemeInstanceFull(ctx, schemeID)
	if err != nil {
		t.Fatal(err)
	}
	return &formalStrategyFixture{
		t: t, ctx: ctx, tx: tx, q: sqlcdb.New(tx), processor: &StrategyProcessor{}, instance: instance,
		definitionConfig: definitionConfig, drawnAt: drawnAt,
		row: sqlcdb.PendingFormalStrategyRow{
			RecordID: recordID, SchemeID: schemeID, LotteryCode: lotteryCode, PeriodNo: periodNo,
			BetContent: "1\n2\n3", RuleSnapshot: snapshot, RuleVersion: pgtype.Int4{Int32: 1, Valid: true},
			RuleSnapshotHash: pgtype.Text{String: "formal-phase-rule", Valid: true}, Balls: []string{"1", "2", "3", "4", "5"},
			DrawnAt: drawnAt,
		},
	}
}

func (f *formalStrategyFixture) stateVersion() int64 {
	f.t.Helper()
	var version int64
	if err := f.tx.QueryRow(f.ctx, `SELECT state_version FROM scheme_instances WHERE id = $1`, f.row.SchemeID).Scan(&version); err != nil {
		f.t.Fatal(err)
	}
	return version
}

func (f *formalStrategyFixture) decisionStatus() string {
	f.t.Helper()
	var status string
	if err := f.tx.QueryRow(f.ctx, `SELECT status FROM scheme_period_decisions WHERE scheme_id = $1 AND source_period_no = $2`, f.row.SchemeID, f.row.PeriodNo).Scan(&status); err != nil {
		f.t.Fatal(err)
	}
	return status
}

func (f *formalStrategyFixture) outboxCount() int {
	f.t.Helper()
	var count int
	if err := f.tx.QueryRow(f.ctx, `SELECT COUNT(*)::int FROM scheme_bet_outbox WHERE scheme_id = $1`, f.row.SchemeID).Scan(&count); err != nil {
		f.t.Fatal(err)
	}
	return count
}
