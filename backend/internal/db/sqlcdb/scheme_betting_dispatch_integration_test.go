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
	commands, err := q.LeaseFormalSchemeBetOutbox(ctx, sqlcdb.LeaseFormalOutboxParams{
		Mode: "gray", LeaseOwner: "db-clock-owner", LotteryCodes: []string{lotteryCode},
		ShardNo: shardNo, Limit: 1, LeaseDuration: 1500 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].ID != outboxID {
		t.Fatalf("leased commands=%+v want outbox=%d", commands, outboxID)
	}
	var leaseRemainingSeconds float64
	if err := tx.QueryRow(ctx, `SELECT EXTRACT(EPOCH FROM (lease_until - clock_timestamp())) FROM scheme_bet_outbox WHERE id = $1`, outboxID).Scan(&leaseRemainingSeconds); err != nil {
		t.Fatal(err)
	}
	if leaseRemainingSeconds <= 0 || leaseRemainingSeconds > 1.5 {
		t.Fatalf("lease remaining=%f seconds", leaseRemainingSeconds)
	}

	start, err := q.StartAttempt(ctx, commands[0], 2*time.Second)
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
	if renewed, err := q.RenewLease(ctx, commands[0], 2*time.Second); err != nil || !renewed {
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
	stale := commands[0]
	stale.Lease.Token++
	if renewed, err := q.RenewLease(ctx, stale, 2*time.Second); err != nil || renewed {
		t.Fatalf("stale fencing token renewed=%v err=%v", renewed, err)
	}
	const finishFailureEvidence = "finish_attempt_failed: deadlock detected while updating scheme terminal state"
	if recorded, err := q.RecordFinishAttemptFailure(ctx, commands[0], finishFailureEvidence); err != nil || !recorded {
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
