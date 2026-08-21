package sqlcdb

import (
	"context"
	"time"
)

type SchemeBettingExecutionState struct {
	Owner      string
	ChainState string
	ChainID    string
	ChainSeq   int64
}

func (q *Queries) GetSchemeBettingExecutionState(ctx context.Context, schemeID string) (SchemeBettingExecutionState, error) {
	var state SchemeBettingExecutionState
	err := q.db.QueryRow(ctx, `
SELECT betting_owner, strict_chain_state, COALESCE(chain_id, ''), chain_seq
FROM scheme_instances
WHERE id = $1`, schemeID).Scan(&state.Owner, &state.ChainState, &state.ChainID, &state.ChainSeq)
	return state, err
}

func (q *Queries) BlockSchemeBettingChain(ctx context.Context, schemeID, reason string, now time.Time) error {
	_, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET strict_chain_state = 'blocked_requires_rearm',
    chain_block_reason = NULLIF($2, ''),
    bet_failed_detail = NULLIF($2, ''),
    updated_at = $3
WHERE id = $1`, schemeID, reason, now)
	return err
}

// BlockSchemeBettingChainIfCurrent blocks only the chain that produced the
// failure. A concurrent rearm changes chain_id and makes this update a no-op.
func (q *Queries) BlockSchemeBettingChainIfCurrent(
	ctx context.Context,
	schemeID, expectedChainID, reason string,
	now time.Time,
) (bool, error) {
	tag, err := q.db.Exec(ctx, `
UPDATE scheme_instances
SET strict_chain_state = 'blocked_requires_rearm',
    bet_failed_detail = NULLIF($3, ''),
    updated_at = $4
WHERE id = $1
  AND betting_owner = 'event'
  AND strict_chain_state = 'active'
  AND NULLIF(TRIM($2), '') IS NOT NULL
  AND chain_id = $2`, schemeID, expectedChainID, reason, now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

type InsertFormalSchemeBetOutboxParams struct {
	DecisionID         int64
	SchemeID           string
	MemberID           int64
	LotteryCode        string
	SourcePeriodNo     string
	TargetPeriodNo     string
	Mode               string
	RequestID          string
	PayloadHash        string
	Payload            []byte
	FrozenRequest      []byte
	FrozenRequestHash  string
	ProviderSnapshotID int64
	CloseAt            time.Time
	SafeDeadlineAt     time.Time
	ShardNo            int32
	SourceStateVersion int64
	InitialBet         bool
	ChainID            string
	ChainSeq           int64
}

func (q *Queries) InsertFormalSchemeBetOutbox(ctx context.Context, arg InsertFormalSchemeBetOutboxParams) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO scheme_bet_outbox
    (decision_id, scheme_id, member_id, lottery_code, source_period_no, target_period_no,
     mode, state, request_id, payload_hash, payload, frozen_request, frozen_request_hash,
     command_frozen_at, provider_snapshot_id, close_at, safe_deadline_at, shard_no,
     source_state_version, initial_bet, chain_id, chain_seq)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9, $10, $11, $12,
        now(), $13, $14, $15, $16, $17, $18, NULLIF($19, ''), $20)
ON CONFLICT (decision_id) DO NOTHING`,
		arg.DecisionID, arg.SchemeID, arg.MemberID, arg.LotteryCode, arg.SourcePeriodNo, arg.TargetPeriodNo,
		arg.Mode, arg.RequestID, arg.PayloadHash, arg.Payload, arg.FrozenRequest, arg.FrozenRequestHash,
		arg.ProviderSnapshotID, arg.CloseAt, arg.SafeDeadlineAt, arg.ShardNo, arg.SourceStateVersion,
		arg.InitialBet, arg.ChainID, arg.ChainSeq)
	return err
}
