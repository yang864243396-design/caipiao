package sqlcdb

import (
	"context"
	"time"

	"caipiao/backend/internal/schemebetting"
)

type LeaseFormalOutboxParams struct {
	Mode          string
	LeaseOwner    string
	LotteryCodes  []string
	ShardNo       int32
	Limit         int32
	LeaseDuration time.Duration
}

func (q *Queries) LeaseFormalSchemeBetOutbox(ctx context.Context, arg LeaseFormalOutboxParams) ([]schemebetting.LeasedCommand, error) {
	if arg.Limit <= 0 {
		arg.Limit = 32
	}
	if arg.LeaseDuration <= 0 {
		return nil, nil
	}
	rows, err := q.db.Query(ctx, `
WITH db_now AS MATERIALIZED (
    SELECT clock_timestamp() AS value
), candidates AS (
    SELECT id
    FROM scheme_bet_outbox
    WHERE mode = $1
      AND state = 'pending'
      AND shard_no = $2
      AND safe_deadline_at > (SELECT value FROM db_now)
      AND lottery_code = ANY($3::text[])
      AND frozen_request IS NOT NULL
      AND frozen_request_hash IS NOT NULL
      AND command_frozen_at IS NOT NULL
    ORDER BY safe_deadline_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT $4
), leased AS (
    UPDATE scheme_bet_outbox o
    SET state = 'leased',
        lease_owner = $5,
        lease_fencing_token = o.lease_fencing_token + 1,
        lease_until = db_now.value + ($6::bigint * interval '1 microsecond'),
        updated_at = db_now.value
    FROM candidates c, db_now
    WHERE o.id = c.id
	    RETURNING o.id, COALESCE(o.scheme_id, '') AS scheme_id, o.target_period_no, o.frozen_request, o.frozen_request_hash,
	              o.close_at, o.safe_deadline_at, o.lease_owner, o.lease_fencing_token, o.lease_until
)
SELECT id, scheme_id, target_period_no, frozen_request, frozen_request_hash, close_at, safe_deadline_at,
       lease_owner, lease_fencing_token, lease_until
FROM leased
ORDER BY safe_deadline_at, id`, arg.Mode, arg.ShardNo, arg.LotteryCodes, arg.Limit, arg.LeaseOwner, arg.LeaseDuration.Microseconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	commands := make([]schemebetting.LeasedCommand, 0, arg.Limit)
	for rows.Next() {
		var command schemebetting.LeasedCommand
		if err := rows.Scan(&command.ID, &command.SchemeID, &command.TargetPeriod, &command.FrozenRequest, &command.FrozenRequestHash,
			&command.CloseAt, &command.SafeDeadline, &command.Lease.Owner, &command.Lease.Token, &command.Lease.Until); err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}
	return commands, rows.Err()
}

func (q *Queries) StartAttempt(ctx context.Context, command schemebetting.LeasedCommand, leaseDuration time.Duration) (schemebetting.AttemptStart, error) {
	if leaseDuration <= 0 {
		return schemebetting.AttemptStart{}, nil
	}
	var started bool
	var safeWindowSeconds float64
	err := q.db.QueryRow(ctx, `
WITH db_now AS MATERIALIZED (
    SELECT clock_timestamp() AS value
), updated AS (
    UPDATE scheme_bet_outbox
    SET dispatch_started_at = db_now.value,
        attempt_count = attempt_count + 1,
        lease_until = GREATEST(
            db_now.value + ($4::bigint * interval '1 microsecond'),
            safe_deadline_at + ($4::bigint * interval '1 microsecond')
        ),
        updated_at = db_now.value
    FROM db_now
    WHERE id = $1
      AND state = 'leased'
      AND lease_owner = $2
      AND lease_fencing_token = $3
      AND lease_until > db_now.value
      AND safe_deadline_at > db_now.value
      AND dispatch_started_at IS NULL
    RETURNING id, attempt_count, request_id, lease_fencing_token, safe_deadline_at, db_now.value AS db_now
), inserted AS (
    INSERT INTO scheme_bet_attempts (outbox_id, attempt_no, request_id, fencing_token, started_at, outcome)
    SELECT id, attempt_count, request_id, lease_fencing_token, db_now, 'started'
    FROM updated
    RETURNING outbox_id
)
SELECT EXISTS(SELECT 1 FROM inserted),
       COALESCE((SELECT EXTRACT(EPOCH FROM (safe_deadline_at - db_now)) FROM updated LIMIT 1), 0)`,
		command.ID, command.Lease.Owner, command.Lease.Token, leaseDuration.Microseconds()).Scan(&started, &safeWindowSeconds)
	if err != nil {
		return schemebetting.AttemptStart{}, err
	}
	return schemebetting.AttemptStart{Started: started, SafeWindow: time.Duration(safeWindowSeconds * float64(time.Second))}, nil
}

func (q *Queries) ReleaseLease(ctx context.Context, command schemebetting.LeasedCommand, reason string, releasedAt time.Time) (bool, error) {
	var released bool
	err := q.db.QueryRow(ctx, `
WITH updated AS (
    UPDATE scheme_bet_outbox
    SET state = 'pending',
        lease_owner = NULL,
        lease_until = NULL,
        outcome_reason = NULLIF($4, ''),
        updated_at = $5
    WHERE id = $1
      AND COALESCE(scheme_id, '') = $2
      AND state = 'leased'
      AND lease_owner = $3
      AND lease_fencing_token = $6
      AND dispatch_started_at IS NULL
    RETURNING id
)
SELECT EXISTS(SELECT 1 FROM updated)`,
		command.ID, command.SchemeID, command.Lease.Owner, reason, releasedAt, command.Lease.Token).Scan(&released)
	return released, err
}

func (q *Queries) RenewLease(ctx context.Context, command schemebetting.LeasedCommand, leaseDuration time.Duration) (bool, error) {
	if leaseDuration <= 0 {
		return false, nil
	}
	var renewed bool
	err := q.db.QueryRow(ctx, `
WITH updated AS (
    UPDATE scheme_bet_outbox
    SET lease_until = GREATEST(
            lease_until,
            clock_timestamp() + ($4::bigint * interval '1 microsecond')
        ),
        updated_at = clock_timestamp()
    WHERE id = $1
      AND COALESCE(scheme_id, '') = $2
      AND state = 'leased'
      AND lease_owner = $3
      AND lease_fencing_token = $5
      AND dispatch_started_at IS NOT NULL
      AND lease_until > clock_timestamp()
    RETURNING id
)
SELECT EXISTS(SELECT 1 FROM updated)`,
		command.ID, command.SchemeID, command.Lease.Owner, leaseDuration.Microseconds(), command.Lease.Token).Scan(&renewed)
	return renewed, err
}

func (q *Queries) FinishAttempt(ctx context.Context, finish schemebetting.FinishDispatch) (bool, error) {
	var finished bool
	err := q.db.QueryRow(ctx, `
WITH updated_outbox AS (
    UPDATE scheme_bet_outbox
    SET state = $4,
        outcome_reason = NULLIF($5, ''),
        provider_order_no = NULLIF($6, ''),
        accepted_period_no = NULLIF($7, ''),
        terminal_at = CASE WHEN $4 = 'sent_unknown' THEN NULL ELSE $8 END,
        provider_account_id = NULLIF($11, 0),
        provider_currency = NULLIF($12, ''),
        provider_amount = NULLIF($13, 0),
        last_error = NULLIF($14, ''),
        lease_until = NULL,
        updated_at = $8
    WHERE id = $1
      AND COALESCE(scheme_id, '') = $2
      AND state = 'leased'
      AND lease_owner = $3
      AND lease_fencing_token = $9
    RETURNING scheme_id, attempt_count
), updated_attempt AS (
    UPDATE scheme_bet_attempts a
    SET outcome = $4,
        finished_at = $8,
        provider_order_no = NULLIF($6, ''),
        accepted_period_no = NULLIF($7, ''),
        error_message = COALESCE(NULLIF($14, ''), NULLIF($5, ''))
    FROM updated_outbox u
    WHERE a.outbox_id = $1 AND a.attempt_no = u.attempt_count
    RETURNING a.id
), blocked AS (
    UPDATE scheme_instances i
    SET strict_chain_state = 'blocked_requires_rearm', updated_at = $8
    FROM updated_outbox u
    WHERE i.id = u.scheme_id AND $10
    RETURNING i.id
)
SELECT EXISTS(SELECT 1 FROM updated_outbox)`, finish.CommandID, finish.SchemeID, finish.LeaseOwner,
		string(finish.State), finish.Reason, finish.ProviderOrderID, finish.AcceptedPeriod,
		finish.FinishedAt, finish.FencingToken, finish.BlocksChain, finish.ProviderAccountID, finish.ProviderCurrency, finish.ProviderAmount,
		finish.ErrorDetail).Scan(&finished)
	return finished, err
}

// RecordFinishAttemptFailure preserves the database error that prevented the
// dispatcher from committing a terminal outcome. It deliberately leaves the
// outbox leased so the normal fenced recovery state machine remains authoritative.
func (q *Queries) RecordFinishAttemptFailure(ctx context.Context, command schemebetting.LeasedCommand, detail string) (bool, error) {
	var recorded bool
	err := q.db.QueryRow(ctx, `
WITH updated_outbox AS (
    UPDATE scheme_bet_outbox
    SET last_error = CASE
            WHEN NULLIF(last_error, '') IS NULL THEN $5
            ELSE left(last_error || '; ' || $5, 4000)
        END,
        updated_at = clock_timestamp()
    WHERE id = $1
      AND COALESCE(scheme_id, '') = $2
      AND state = 'leased'
      AND lease_owner = $3
      AND lease_fencing_token = $4
      AND dispatch_started_at IS NOT NULL
    RETURNING id, attempt_count
), updated_attempt AS (
    UPDATE scheme_bet_attempts a
    SET error_message = CASE
            WHEN NULLIF(a.error_message, '') IS NULL THEN $5
            ELSE left(a.error_message || '; ' || $5, 4000)
        END
    FROM updated_outbox u
    WHERE a.outbox_id = u.id AND a.attempt_no = u.attempt_count
    RETURNING a.id
)
SELECT EXISTS(SELECT 1 FROM updated_outbox)`,
		command.ID, command.SchemeID, command.Lease.Owner, command.Lease.Token, detail).Scan(&recorded)
	return recorded, err
}

func (q *Queries) MarkAbandonedStartedDispatchUnknown(ctx context.Context, rowLimit int32) (int64, error) {
	if rowLimit <= 0 {
		rowLimit = 100
	}
	var count int64
	err := q.db.QueryRow(ctx, `
WITH db_now AS MATERIALIZED (
    SELECT clock_timestamp() AS value
), candidates AS (
    SELECT id, safe_deadline_at
    FROM scheme_bet_outbox
    WHERE state = 'leased'
      AND dispatch_started_at IS NOT NULL
      AND lease_until <= (SELECT value FROM db_now)
    ORDER BY lease_until, id
    FOR UPDATE SKIP LOCKED
    LIMIT $1
), updated AS (
    UPDATE scheme_bet_outbox o
    SET state = CASE WHEN o.safe_deadline_at <= db_now.value THEN 'external_acceptance_unknown' ELSE 'sent_unknown' END,
        outcome_reason = CASE
            WHEN o.safe_deadline_at <= db_now.value THEN 'dispatcher_lost_after_send_started_deadline_elapsed'
            ELSE 'dispatcher_lost_after_send_started'
        END,
        terminal_at = CASE WHEN o.safe_deadline_at <= db_now.value THEN db_now.value ELSE NULL END,
        last_error = COALESCE(NULLIF(o.last_error, ''), CASE
            WHEN o.safe_deadline_at <= db_now.value THEN 'dispatcher_lost_after_send_started_deadline_elapsed'
            ELSE 'dispatcher_lost_after_send_started'
        END),
        lease_until = NULL,
        updated_at = db_now.value
    FROM candidates c, db_now
    WHERE o.id = c.id
    RETURNING o.id, o.scheme_id, o.attempt_count, o.safe_deadline_at
), attempts AS (
    UPDATE scheme_bet_attempts a
    SET outcome = CASE WHEN u.safe_deadline_at <= db_now.value THEN 'external_acceptance_unknown' ELSE 'sent_unknown' END,
        finished_at = db_now.value,
        error_message = COALESCE(NULLIF(a.error_message, ''), CASE
            WHEN u.safe_deadline_at <= db_now.value THEN 'dispatcher_lost_after_send_started_deadline_elapsed'
            ELSE 'dispatcher_lost_after_send_started'
        END)
    FROM updated u, db_now
    WHERE a.outbox_id = u.id AND a.attempt_no = u.attempt_count
    RETURNING a.id
), blocked AS (
    UPDATE scheme_instances i
    SET strict_chain_state = 'blocked_requires_rearm', updated_at = db_now.value
    FROM updated u, db_now
    WHERE i.id = u.scheme_id
    RETURNING i.id
)
SELECT count(*) FROM updated`, rowLimit).Scan(&count)
	return count, err
}

// RecoverExpiredUnstartedFormalOutbox handles a dispatcher crash or pre-send
// stall after it has leased a command but before StartAttempt was committed.
// Such a row has never started an outbound request: retry it while the safety
// window remains open, otherwise expire it and block the strict chain.
func (q *Queries) RecoverExpiredUnstartedFormalOutbox(ctx context.Context, rowLimit int32) (int64, error) {
	if rowLimit <= 0 {
		rowLimit = 100
	}
	var count int64
	err := q.db.QueryRow(ctx, `
WITH db_now AS MATERIALIZED (
    SELECT clock_timestamp() AS value
), candidates AS (
    SELECT id, safe_deadline_at
    FROM scheme_bet_outbox
    WHERE mode IN ('gray', 'production')
      AND state = 'leased'
      AND dispatch_started_at IS NULL
      AND lease_until <= (SELECT value FROM db_now)
    ORDER BY lease_until, id
    FOR UPDATE SKIP LOCKED
    LIMIT $1
), updated AS (
    UPDATE scheme_bet_outbox o
    SET state = CASE WHEN o.safe_deadline_at <= db_now.value THEN 'expired' ELSE 'pending' END,
        outcome_reason = CASE
            WHEN o.safe_deadline_at <= db_now.value THEN 'dispatcher_lost_before_start_deadline_elapsed'
            ELSE 'dispatcher_lost_before_start'
        END,
        lease_owner = NULL,
        lease_until = NULL,
        terminal_at = CASE WHEN o.safe_deadline_at <= db_now.value THEN db_now.value ELSE NULL END,
        updated_at = db_now.value
    FROM candidates c, db_now
    WHERE o.id = c.id
    RETURNING o.id, o.scheme_id, o.state
), blocked AS (
    UPDATE scheme_instances i
    SET strict_chain_state = 'blocked_requires_rearm', updated_at = db_now.value
    FROM updated u, db_now
    WHERE i.id = u.scheme_id AND u.state = 'expired'
    RETURNING i.id
)
SELECT count(*) FROM updated`, rowLimit).Scan(&count)
	return count, err
}

func (q *Queries) ExpireDueFormalOutbox(ctx context.Context, rowLimit int32) (int64, error) {
	if rowLimit <= 0 {
		rowLimit = 500
	}
	var count int64
	err := q.db.QueryRow(ctx, `
WITH db_now AS MATERIALIZED (
    SELECT clock_timestamp() AS value
), candidates AS (
    SELECT id
    FROM scheme_bet_outbox
    WHERE mode IN ('gray', 'production')
      AND state IN ('pending', 'sent_unknown')
      AND safe_deadline_at <= (SELECT value FROM db_now)
    ORDER BY safe_deadline_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT $1
), updated AS (
    UPDATE scheme_bet_outbox o
    SET state = CASE WHEN o.state = 'sent_unknown' THEN 'external_acceptance_unknown' ELSE 'expired' END,
        outcome_reason = CASE
            WHEN o.state = 'sent_unknown' THEN 'reconciliation_deadline_elapsed'
            ELSE 'safe_deadline_elapsed'
        END,
        terminal_at = db_now.value,
        updated_at = db_now.value
    FROM candidates c, db_now
    WHERE o.id = c.id
    RETURNING o.id, o.scheme_id, o.attempt_count, o.state
), attempts AS (
    UPDATE scheme_bet_attempts a
    SET outcome = 'external_acceptance_unknown',
        error_message = COALESCE(NULLIF(a.error_message, ''), 'reconciliation_deadline_elapsed')
    FROM updated u
    WHERE u.state = 'external_acceptance_unknown'
      AND a.outbox_id = u.id
      AND a.attempt_no = u.attempt_count
    RETURNING a.id
), blocked AS (
    UPDATE scheme_instances i
    SET strict_chain_state = 'blocked_requires_rearm', updated_at = db_now.value
    FROM updated u, db_now
    WHERE i.id = u.scheme_id
    RETURNING i.id
)
SELECT count(*) FROM updated`, rowLimit).Scan(&count)
	return count, err
}
