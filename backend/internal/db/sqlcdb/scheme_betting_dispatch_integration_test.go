package sqlcdb_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemebetting"
)

func TestDispatchQueriesAgainstMigratedSchema(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	q := sqlcdb.New(pool)
	commands, err := q.LeaseFormalSchemeBetOutbox(context.Background(), sqlcdb.LeaseFormalOutboxParams{
		Mode: "gray", LeaseOwner: "schema-probe", LotteryCodes: []string{"__schema_probe__"}, ShardNo: -1, Limit: 1,
		LeaseDuration: time.Second,
	})
	if err != nil {
		t.Fatalf("lease formal outbox query: %v", err)
	}
	if len(commands) != 0 {
		t.Fatalf("schema probe unexpectedly leased %d rows", len(commands))
	}
	if _, acquired, err := q.LeaseFormalOutboxByID(context.Background(), -1, "schema-probe", time.Second); err != nil || acquired {
		t.Fatalf("lease formal outbox by id acquired=%v err=%v", acquired, err)
	}
	if _, acquired, err := q.LeaseFormalEventOutboxByID(context.Background(), sqlcdb.LeaseFormalEventOutboxParams{
		ID: -1, Mode: "gray", Owner: "schema-probe", LotteryCodes: []string{"__schema_probe__"}, ShardNo: -1, LeaseDuration: time.Second,
	}); err != nil || acquired {
		t.Fatalf("lease formal event outbox by id acquired=%v err=%v", acquired, err)
	}
	if _, err := q.RecoverExpiredUnstartedFormalOutbox(context.Background(), 1); err != nil {
		t.Fatalf("recover expired unstarted outbox: %v", err)
	}
	if _, err := q.MarkAbandonedStartedDispatchUnknown(context.Background(), 1); err != nil {
		t.Fatalf("mark abandoned started outbox: %v", err)
	}
	if _, err := q.ExpireDueFormalOutbox(context.Background(), 1); err != nil {
		t.Fatalf("expire due outbox: %v", err)
	}
	if _, err := q.ListPendingPreSendFailureOutboxIDs(context.Background(), 1); err != nil {
		t.Fatalf("list pre-send replacement candidates: %v", err)
	}
	if _, err := q.ListPendingFormalBetWakeups(context.Background(), "gray", []string{"__schema_probe__"}, []int32{-1}, 1); err != nil {
		t.Fatalf("list pending formal bet wakeups: %v", err)
	}
	if _, found, err := q.GetPreSendFailureOutbox(context.Background(), -1); err != nil || found {
		t.Fatalf("get pre-send replacement candidate found=%v err=%v", found, err)
	}
	if err := q.MarkPreSendFailureRescheduled(context.Background(), -1, -1); err != nil {
		t.Fatalf("mark pre-send replacement: %v", err)
	}
	if err := q.DeferPreSendFailureReschedule(context.Background(), -1, "no fresh target"); err != nil {
		t.Fatalf("defer pre-send replacement: %v", err)
	}
	for _, table := range []string{"scheme_betting_shard_leases", "scheme_betting_admin_actions", "scheme_betting_capacity_limits"} {
		var exists bool
		if err := pool.QueryRow(context.Background(), `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil || !exists {
			t.Fatalf("table %s exists=%v err=%v", table, exists, err)
		}
	}
}

func TestFinishAttemptSQLParsesAgainstMigratedSchema(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	finished, err := sqlcdb.New(pool).FinishAttempt(context.Background(), schemebetting.FinishDispatch{
		CommandID:    -1,
		SchemeID:     "schema-probe",
		LeaseOwner:   "schema-probe",
		FencingToken: -1,
		State:        schemebetting.OutboxRejected,
		Reason:       "schema_probe",
		FinishedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("finish attempt query must parse against migrated schema: %v", err)
	}
	if finished {
		t.Fatal("schema probe unexpectedly finished an outbox row")
	}
}

func TestFinishAttemptPersistsFloatProviderAmount(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	var memberID, snapshotID int64
	var lotteryCode string
	if err := tx.QueryRow(ctx, `SELECT id FROM members ORDER BY id LIMIT 1`).Scan(&memberID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			t.Skip("database has no member fixture")
		}
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT id, lottery_code FROM provider_period_snapshots ORDER BY id DESC LIMIT 1`).Scan(&snapshotID, &lotteryCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			t.Skip("database has no provider period snapshot fixture")
		}
		t.Fatal(err)
	}

	unique := fmt.Sprintf("finish-provider-amount-%d", time.Now().UnixNano())
	const shardNo int32 = 0
	var outboxID int64
	err = tx.QueryRow(ctx, `
INSERT INTO scheme_bet_outbox
    (origin, local_order_no, member_id, lottery_code, source_period_no, target_period_no,
     mode, state, request_id, payload_hash, payload, frozen_request, frozen_request_hash,
     command_frozen_at, provider_snapshot_id, close_at, safe_deadline_at, shard_no)
VALUES
    ('api', $1, $2, $3, $4, $4,
     'gray', 'pending', $1, $5, '{}'::jsonb, '{}'::jsonb, $5,
     clock_timestamp(), $6, clock_timestamp() + interval '10 seconds',
     clock_timestamp() + interval '5 seconds', $7)
RETURNING id`, unique, memberID, lotteryCode, unique+"-period", unique+"-hash", snapshotID, shardNo).Scan(&outboxID)
	if err != nil {
		t.Fatal(err)
	}

	q := sqlcdb.New(tx)
	command, acquired, err := q.LeaseFormalEventOutboxByID(ctx, sqlcdb.LeaseFormalEventOutboxParams{
		ID: outboxID, RequestID: unique, Mode: "gray", Owner: unique + "-owner",
		LotteryCodes: []string{lotteryCode}, ShardNo: shardNo, LeaseDuration: 2 * time.Second,
	})
	if err != nil || !acquired {
		t.Fatalf("lease acquired=%v err=%v", acquired, err)
	}
	if start, err := q.StartAttempt(ctx, command, 2*time.Second); err != nil || !start.Started {
		t.Fatalf("attempt start=%+v err=%v", start, err)
	}
	finishedAt := time.Now().UTC()
	finished, err := q.FinishAttempt(ctx, schemebetting.FinishDispatch{
		CommandID: command.ID, SchemeID: command.SchemeID, LeaseOwner: command.Lease.Owner,
		FencingToken: command.Lease.Token, State: schemebetting.OutboxAccepted,
		Reason: "accepted", ProviderOrderID: unique + "-provider", AcceptedPeriod: command.TargetPeriod,
		ProviderAmount: 0.2, ProviderCurrency: "USDT", FinishedAt: finishedAt,
	})
	if err != nil || !finished {
		t.Fatalf("finished=%v err=%v", finished, err)
	}
	var storedAmount string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(provider_amount::text, '') FROM scheme_bet_outbox WHERE id=$1`, outboxID).Scan(&storedAmount); err != nil {
		t.Fatal(err)
	}
	if storedAmount != "0.200" {
		t.Fatalf("provider_amount=%q, want exact numeric text 0.200", storedAmount)
	}
}

func TestFormalOutboxLeaseAndAttemptUseDatabaseClock(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	var memberID, snapshotID int64
	var lotteryCode string
	if err := tx.QueryRow(ctx, `SELECT id FROM members ORDER BY id LIMIT 1`).Scan(&memberID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			t.Skip("database has no member fixture")
		}
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT id, lottery_code FROM provider_period_snapshots ORDER BY id DESC LIMIT 1`).Scan(&snapshotID, &lotteryCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			t.Skip("database has no provider period snapshot fixture")
		}
		t.Fatal(err)
	}

	unique := fmt.Sprintf("db-clock-%d", time.Now().UnixNano())
	const shardNo int32 = 63
	var outboxID int64
	err = tx.QueryRow(ctx, `
INSERT INTO scheme_bet_outbox
    (origin, decision_id, scheme_id, member_id, lottery_code, source_period_no, target_period_no,
     mode, state, request_id, payload_hash, payload, frozen_request, frozen_request_hash,
     command_frozen_at, provider_snapshot_id, close_at, safe_deadline_at, shard_no, local_order_no)
VALUES
    ('api', NULL, NULL, $1, $2, $3, $3,
     'gray', 'pending', $4, $5, '{}'::jsonb, '{}'::jsonb, $5,
     clock_timestamp(), $6, clock_timestamp() + interval '10 seconds',
     clock_timestamp() + interval '5 seconds', $7, $4)
RETURNING id`, memberID, lotteryCode, unique+"-period", unique, unique+"-hash", snapshotID, shardNo).Scan(&outboxID)
	if err != nil {
		t.Fatal(err)
	}

	q := sqlcdb.New(tx)
	command, acquired, err := q.LeaseFormalEventOutboxByID(ctx, sqlcdb.LeaseFormalEventOutboxParams{
		ID: outboxID, RequestID: unique, Mode: "gray", Owner: "db-clock-owner", LotteryCodes: []string{lotteryCode},
		ShardNo: shardNo, LeaseDuration: 1500 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !acquired || command.ID != outboxID {
		t.Fatalf("leased command=%+v acquired=%v want outbox=%d", command, acquired, outboxID)
	}
	var leaseRemainingSeconds float64
	if err := tx.QueryRow(ctx, `SELECT EXTRACT(EPOCH FROM (lease_until - clock_timestamp())) FROM scheme_bet_outbox WHERE id = $1`, outboxID).Scan(&leaseRemainingSeconds); err != nil {
		t.Fatal(err)
	}
	if leaseRemainingSeconds <= 0 || leaseRemainingSeconds > 1.5 {
		t.Fatalf("lease remaining=%f seconds", leaseRemainingSeconds)
	}

	start, err := q.StartAttempt(ctx, command, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !start.Started || start.SafeWindow <= 0 || start.SafeWindow > 5*time.Second {
		t.Fatalf("attempt start=%+v", start)
	}
	var protectedLeaseSeconds float64
	if err := tx.QueryRow(ctx, `
SELECT EXTRACT(EPOCH FROM (lease_until - safe_deadline_at))
FROM scheme_bet_outbox
WHERE id = $1`, outboxID).Scan(&protectedLeaseSeconds); err != nil {
		t.Fatal(err)
	}
	if protectedLeaseSeconds < 1.9 {
		t.Fatalf("started attempt lease only protects %.3f seconds past safe deadline, want at least 1.9", protectedLeaseSeconds)
	}
	if renewed, err := q.RenewLease(ctx, command, 2*time.Second); err != nil || !renewed {
		t.Fatalf("renewed=%v err=%v", renewed, err)
	}
	if err := tx.QueryRow(ctx, `
SELECT EXTRACT(EPOCH FROM (lease_until - safe_deadline_at))
FROM scheme_bet_outbox
WHERE id = $1`, outboxID).Scan(&protectedLeaseSeconds); err != nil {
		t.Fatal(err)
	}
	if protectedLeaseSeconds < 1.9 {
		t.Fatalf("heartbeat shortened protected attempt lease to %.3f seconds past safe deadline", protectedLeaseSeconds)
	}
	stale := command
	stale.Lease.Token++
	if renewed, err := q.RenewLease(ctx, stale, 2*time.Second); err != nil || renewed {
		t.Fatalf("stale fencing token renewed=%v err=%v", renewed, err)
	}
	const finishFailureEvidence = "finish_attempt_failed: deadlock detected while updating scheme terminal state"
	if recorded, err := q.RecordFinishAttemptFailure(ctx, command, finishFailureEvidence); err != nil || !recorded {
		t.Fatalf("recorded finish failure=%v err=%v", recorded, err)
	}
	var failureState, outboxFailure, attemptFailure string
	if err := tx.QueryRow(ctx, `
SELECT o.state, COALESCE(o.last_error, ''), COALESCE(a.error_message, '')
FROM scheme_bet_outbox o
JOIN scheme_bet_attempts a ON a.outbox_id = o.id AND a.attempt_no = o.attempt_count
WHERE o.id = $1`, outboxID).Scan(&failureState, &outboxFailure, &attemptFailure); err != nil {
		t.Fatal(err)
	}
	if failureState != "leased" || outboxFailure != finishFailureEvidence || attemptFailure != finishFailureEvidence {
		t.Fatalf("state=%s outbox_error=%q attempt_error=%q", failureState, outboxFailure, attemptFailure)
	}
	if recorded, err := q.RecordFinishAttemptFailure(ctx, stale, "stale owner must not overwrite evidence"); err != nil || recorded {
		t.Fatalf("stale finish failure recorded=%v err=%v", recorded, err)
	}
	const originalEvidence = "provider placement failed phase=tls request_written=true"
	if _, err := tx.Exec(ctx, `
UPDATE scheme_bet_outbox
SET lease_until = clock_timestamp() - interval '100 years',
    safe_deadline_at = clock_timestamp() + interval '4 seconds',
    close_at = clock_timestamp() + interval '8 seconds',
    last_error = $2
WHERE id = $1`, outboxID, originalEvidence); err != nil {
		t.Fatal(err)
	}
	if swept, err := q.MarkAbandonedStartedDispatchUnknown(ctx, 1); err != nil || swept != 1 {
		t.Fatalf("swept=%d err=%v", swept, err)
	}
	var state, persistedEvidence string
	if err := tx.QueryRow(ctx, `SELECT state, COALESCE(last_error, '') FROM scheme_bet_outbox WHERE id = $1`, outboxID).Scan(&state, &persistedEvidence); err != nil {
		t.Fatal(err)
	}
	if state != "sent_unknown" || persistedEvidence != originalEvidence {
		t.Fatalf("state=%s last_error=%q", state, persistedEvidence)
	}
}
