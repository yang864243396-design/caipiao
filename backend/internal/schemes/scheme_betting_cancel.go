package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

func (w *Worker) CancelEventBet(ctx context.Context, outboxID int64, actor, reason string) error {
	if w == nil || w.pool == nil || w.q == nil {
		return errors.New("scheme event worker unavailable")
	}
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if outboxID <= 0 || actor == "" || len([]rune(reason)) < 4 {
		return errors.New("outbox, actor, and a reason of at least 4 characters are required")
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := w.q.WithTx(tx)
	now := time.Now().UTC()
	cancelled, err := q.CancelSchemeBetOutbox(ctx, outboxID, now)
	if err != nil {
		return err
	}
	if err := q.BlockSchemeBettingChain(ctx, cancelled.SchemeID, "admin_cancelled_before_send", now); err != nil {
		return err
	}
	afterState, _ := json.Marshal(map[string]any{"state": "cancelled", "chainState": "blocked_requires_rearm"})
	if err := q.InsertSchemeBettingAdminAction(ctx, sqlcdb.InsertSchemeBettingAdminActionParams{
		SchemeID: cancelled.SchemeID, OutboxID: outboxID, Action: "cancel", Actor: actor,
		Reason: reason, BeforeState: cancelled.BeforeState, AfterState: afterState,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
