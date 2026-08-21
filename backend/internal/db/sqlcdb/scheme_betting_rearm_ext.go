package sqlcdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (q *Queries) CheckSchemeBettingCapacity(ctx context.Context, lotteryCode string, shardNo int32) error {
	var maxDue, maxActive int
	err := q.db.QueryRow(ctx, `
SELECT max_due_outbox, max_active_schemes
FROM scheme_betting_capacity_limits
WHERE lottery_code = $1 AND enabled`, lotteryCode).Scan(&maxDue, &maxActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("capacity_not_configured")
	}
	if err != nil {
		return err
	}
	var due, active int
	if err := q.db.QueryRow(ctx, `
SELECT
    (SELECT count(*) FROM scheme_bet_outbox WHERE lottery_code = $1 AND state IN ('pending', 'leased')),
    (SELECT count(*) FROM scheme_instances WHERE lottery_code = $1 AND betting_owner = 'event' AND status = 'running' AND strict_chain_state = 'active')`,
		lotteryCode).Scan(&due, &active); err != nil {
		return err
	}
	if due >= maxDue {
		return fmt.Errorf("capacity_due_outbox:%d/%d", due, maxDue)
	}
	if active >= maxActive {
		return fmt.Errorf("capacity_active_schemes:%d/%d", active, maxActive)
	}
	var strategyReady, dispatcherReady bool
	if err := q.db.QueryRow(ctx, `
SELECT
    EXISTS (
        SELECT 1 FROM scheme_betting_shard_leases
        WHERE lease_kind = 'strategy' AND shard_no = $1 AND lease_until > now()
    ),
    EXISTS (
        SELECT 1 FROM scheme_betting_shard_leases
        WHERE lease_kind = 'dispatcher' AND shard_no = $1 AND lease_until > now()
    )`, shardNo).Scan(&strategyReady, &dispatcherReady); err != nil {
		return err
	}
	if !strategyReady {
		return fmt.Errorf("capacity_strategy_worker_unavailable:shard_%d", shardNo)
	}
	if !dispatcherReady {
		return fmt.Errorf("capacity_dispatch_worker_unavailable:shard_%d", shardNo)
	}
	return nil
}

func (q *Queries) EnsureNoUnresolvedSchemeBet(ctx context.Context, schemeID string) error {
	var unresolved bool
	if err := q.db.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM scheme_bet_outbox
    WHERE scheme_id = $1
      AND state IN ('sent_unknown', 'external_acceptance_unknown')
)`, schemeID).Scan(&unresolved); err != nil {
		return err
	}
	if unresolved {
		return errors.New("scheme_has_unresolved_external_acceptance")
	}
	return nil
}

func (q *Queries) ActivateSchemeBettingChain(ctx context.Context, schemeID, chainID string, allowLegacy bool) error {
	tag, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET betting_owner = 'event', strict_chain_state = 'active', chain_id = $2, chain_seq = 0,
    bet_failed_detail = NULL, chain_block_reason = NULL, updated_at = now()
WHERE id = $1 AND (betting_owner = 'event' OR $3)`, schemeID, chainID, allowLegacy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("scheme betting owner transition rejected")
	}
	return nil
}

type InsertSchemeBettingAdminActionParams struct {
	SchemeID    string
	OutboxID    int64
	Action      string
	Actor       string
	Reason      string
	BeforeState []byte
	AfterState  []byte
}

func (q *Queries) InsertSchemeBettingAdminAction(ctx context.Context, arg InsertSchemeBettingAdminActionParams) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO scheme_betting_admin_actions
    (scheme_id, outbox_id, action, actor_account, reason, before_state, after_state)
VALUES (NULLIF($1, ''), NULLIF($2, 0), $3, $4, $5, $6, $7)`,
		arg.SchemeID, arg.OutboxID, arg.Action, arg.Actor, arg.Reason, arg.BeforeState, arg.AfterState)
	return err
}

func (q *Queries) SchemeBetOutboxIDByDecision(ctx context.Context, decisionID int64) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, `SELECT id FROM scheme_bet_outbox WHERE decision_id = $1`, decisionID).Scan(&id)
	return id, err
}
