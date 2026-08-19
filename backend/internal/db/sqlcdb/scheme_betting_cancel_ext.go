package sqlcdb

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type CancelSchemeBetOutboxResult struct {
	SchemeID    string
	BeforeState json.RawMessage
}

func (q *Queries) CancelSchemeBetOutbox(ctx context.Context, outboxID int64, now time.Time) (CancelSchemeBetOutboxResult, error) {
	var result CancelSchemeBetOutboxResult
	err := q.db.QueryRow(ctx, `
WITH candidate AS (
    SELECT id, scheme_id,
           jsonb_build_object('state', state, 'dispatchStartedAt', dispatch_started_at, 'leaseOwner', lease_owner) AS before_state
    FROM scheme_bet_outbox
    WHERE id = $1
      AND state IN ('pending', 'leased')
      AND dispatch_started_at IS NULL
    FOR UPDATE
), cancelled AS (
    UPDATE scheme_bet_outbox o
    SET state = 'cancelled', outcome_reason = 'admin_cancelled_before_send',
        lease_until = NULL, terminal_at = $2, updated_at = $2
    FROM candidate c
    WHERE o.id = c.id
    RETURNING c.scheme_id, c.before_state
)
SELECT scheme_id, before_state FROM cancelled`, outboxID, now).Scan(&result.SchemeID, &result.BeforeState)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, errors.New("outbox is missing, terminal, or dispatch already started")
	}
	return result, err
}
