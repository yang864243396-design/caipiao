package sqlcdb

import (
	"context"
	"time"
)

type PendingBetReadyEvent struct {
	OutboxID     int64
	RequestID    string
	ShardNo      int32
	SafeDeadline time.Time
}

func (q *Queries) ListUnpublishedBetReady(ctx context.Context, limit int32) ([]PendingBetReadyEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := q.db.Query(ctx, `
SELECT id, request_id, shard_no, safe_deadline_at
FROM scheme_bet_outbox
WHERE mode IN ('gray', 'production')
  AND state = 'pending'
  AND ready_published_at IS NULL
  AND ready_next_attempt_at <= now()
ORDER BY safe_deadline_at, id
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PendingBetReadyEvent, 0, limit)
	for rows.Next() {
		var event PendingBetReadyEvent
		if err := rows.Scan(&event.OutboxID, &event.RequestID, &event.ShardNo, &event.SafeDeadline); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

// ListPendingFormalBetWakeups is the low-frequency database safety net for
// JetStream delivery. It scans the assigned shard set once, rather than doing
// two round trips for every shard on every hot tick.
func (q *Queries) ListPendingFormalBetWakeups(
	ctx context.Context, mode string, lotteryCodes []string, shards []int32, limit int32,
) ([]PendingBetReadyEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := q.db.Query(ctx, `
SELECT id, request_id, shard_no, safe_deadline_at
FROM scheme_bet_outbox
WHERE mode = $1
  AND state = 'pending'
  AND lottery_code = ANY($2::text[])
  AND shard_no = ANY($3::integer[])
  AND safe_deadline_at > clock_timestamp()
ORDER BY safe_deadline_at, id
LIMIT $4`, mode, lotteryCodes, shards, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PendingBetReadyEvent, 0, limit)
	for rows.Next() {
		var event PendingBetReadyEvent
		if err := rows.Scan(&event.OutboxID, &event.RequestID, &event.ShardNo, &event.SafeDeadline); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

func (q *Queries) MarkBetReadyPublished(ctx context.Context, outboxID int64, publishedAt time.Time) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_bet_outbox
SET ready_published_at = COALESCE(ready_published_at, $2),
    ready_publish_attempts = ready_publish_attempts + 1,
    ready_next_attempt_at = $2,
    updated_at = GREATEST(updated_at, $2)
WHERE id = $1`, outboxID, publishedAt)
	return err
}

func (q *Queries) MarkBetReadyPublishFailed(ctx context.Context, outboxID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_bet_outbox
SET ready_publish_attempts = ready_publish_attempts + 1,
    ready_next_attempt_at = now() + LEAST(
        interval '30 seconds',
        interval '250 milliseconds' * power(2::double precision, LEAST(ready_publish_attempts, 7))
    )
WHERE id = $1 AND ready_published_at IS NULL`, outboxID)
	return err
}

type PendingBetReconcileEvent struct {
	OutboxID  int64
	RequestID string
	ShardNo   int32
	State     string
	Reason    string
}

func (q *Queries) ListUnpublishedBetReconcile(ctx context.Context, limit int32) ([]PendingBetReconcileEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := q.db.Query(ctx, `
SELECT id, request_id, shard_no, state, COALESCE(outcome_reason, '')
FROM scheme_bet_outbox
WHERE mode IN ('gray', 'production')
  AND state NOT IN ('pending', 'leased')
  AND reconcile_published_state IS DISTINCT FROM state
  AND reconcile_next_attempt_at <= now()
ORDER BY updated_at, id
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PendingBetReconcileEvent, 0, limit)
	for rows.Next() {
		var event PendingBetReconcileEvent
		if err := rows.Scan(&event.OutboxID, &event.RequestID, &event.ShardNo, &event.State, &event.Reason); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

func (q *Queries) MarkBetReconcilePublished(ctx context.Context, outboxID int64, state string, publishedAt time.Time) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_bet_outbox
SET reconcile_published_at = $3,
    reconcile_published_state = $2,
    reconcile_publish_attempts = reconcile_publish_attempts + 1,
    reconcile_next_attempt_at = $3,
    updated_at = GREATEST(updated_at, $3)
WHERE id = $1 AND state = $2`, outboxID, state, publishedAt)
	return err
}

func (q *Queries) MarkBetReconcilePublishFailed(ctx context.Context, outboxID int64, state string) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_bet_outbox
SET reconcile_publish_attempts = reconcile_publish_attempts + 1,
    reconcile_next_attempt_at = now() + LEAST(
        interval '30 seconds',
        interval '250 milliseconds' * power(2::double precision, LEAST(reconcile_publish_attempts, 7))
    )
WHERE id = $1 AND state = $2 AND reconcile_published_state IS DISTINCT FROM state`, outboxID, state)
	return err
}
