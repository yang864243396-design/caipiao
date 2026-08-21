package sqlcdb

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type PreSendFailureOutbox struct {
	ID           int64
	SchemeID     string
	LotteryCode  string
	FailedPeriod string
	Mode         string
	LastError    string
	ChainID      string
}

func (q *Queries) GetPreSendFailureOutbox(ctx context.Context, outboxID int64) (PreSendFailureOutbox, bool, error) {
	var row PreSendFailureOutbox
	err := q.db.QueryRow(ctx, `
SELECT o.id, o.scheme_id, o.lottery_code, o.target_period_no, o.mode,
       COALESCE(o.last_error, ''), COALESCE(o.chain_id, '')
FROM scheme_bet_outbox o
JOIN scheme_instances i ON i.id = o.scheme_id
WHERE o.id = $1
  AND (
      (o.state = 'rejected' AND o.outcome_reason = 'provider_pre_send_failed')
      OR
      (o.state = 'expired' AND o.outcome_reason IN ('safe_deadline_elapsed', 'dispatcher_lost_before_start_deadline_elapsed'))
  )
  AND i.betting_owner = 'event'
  AND i.strict_chain_state = 'active'
  AND NULLIF(TRIM(i.chain_id), '') IS NOT NULL
  AND o.chain_id = i.chain_id`, outboxID).Scan(
		&row.ID, &row.SchemeID, &row.LotteryCode, &row.FailedPeriod, &row.Mode, &row.LastError, &row.ChainID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PreSendFailureOutbox{}, false, nil
		}
		return PreSendFailureOutbox{}, false, err
	}
	return row, true, nil
}

func (q *Queries) ListPendingPreSendFailureOutboxIDs(ctx context.Context, limit int32) ([]int64, error) {
	if limit <= 0 {
		limit = 32
	}
	rows, err := q.db.Query(ctx, `
SELECT o.id
FROM scheme_bet_outbox o
JOIN scheme_instances i ON i.id = o.scheme_id
WHERE (
      (o.state = 'rejected' AND o.outcome_reason = 'provider_pre_send_failed')
      OR
      (o.state = 'expired' AND o.outcome_reason IN ('safe_deadline_elapsed', 'dispatcher_lost_before_start_deadline_elapsed'))
  )
  AND i.betting_owner = 'event'
  AND i.strict_chain_state = 'active'
  AND NULLIF(TRIM(i.chain_id), '') IS NOT NULL
  AND o.chain_id = i.chain_id
  AND o.reconcile_next_attempt_at <= clock_timestamp()
ORDER BY o.updated_at, o.id
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (q *Queries) DeferPreSendFailureReschedule(ctx context.Context, outboxID int64, detail string) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_bet_outbox
SET reconcile_publish_attempts = reconcile_publish_attempts + 1,
    reconcile_next_attempt_at = clock_timestamp() + LEAST(
        interval '3 seconds',
        interval '250 milliseconds' * power(2::double precision, LEAST(reconcile_publish_attempts, 4))
    ),
    last_error = concat_ws('; ',
        NULLIF(split_part(COALESCE(last_error, ''), '; pre_send_reschedule_deferred=', 1), ''),
        'pre_send_reschedule_deferred=' || NULLIF($2, '')
    ),
    updated_at = clock_timestamp()
WHERE id = $1
  AND (
      (state = 'rejected' AND outcome_reason = 'provider_pre_send_failed')
      OR
      (state = 'expired' AND outcome_reason IN ('safe_deadline_elapsed', 'dispatcher_lost_before_start_deadline_elapsed'))
  )`, outboxID, detail)
	return err
}

func (q *Queries) MarkPreSendFailureRescheduled(ctx context.Context, outboxID, replacementOutboxID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_bet_outbox
SET outcome_reason = 'provider_pre_send_rescheduled',
    last_error = concat_ws('; ', NULLIF(last_error, ''), 'replacement_outbox_id=' || $2::bigint::text),
    updated_at = clock_timestamp()
WHERE id = $1
  AND (
      (state = 'rejected' AND outcome_reason = 'provider_pre_send_failed')
      OR
      (state = 'expired' AND outcome_reason IN ('safe_deadline_elapsed', 'dispatcher_lost_before_start_deadline_elapsed'))
  )`, outboxID, replacementOutboxID)
	return err
}
