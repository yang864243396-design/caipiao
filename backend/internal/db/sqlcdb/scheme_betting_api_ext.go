package sqlcdb

import (
	"context"
	"errors"
	"time"

	"caipiao/backend/internal/schemebetting"
	"github.com/jackc/pgx/v5"
)

type InsertAPIFormalBetOutboxParams struct {
	MemberID           int64
	LotteryCode        string
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
	LocalOrderNo       string
}

func (q *Queries) InsertAPIFormalBetOutbox(ctx context.Context, arg InsertAPIFormalBetOutboxParams) (int64, string, error) {
	if _, err := q.db.Exec(ctx, `
INSERT INTO scheme_bet_outbox
    (origin, decision_id, scheme_id, member_id, lottery_code, source_period_no, target_period_no,
     mode, state, request_id, payload_hash, payload, frozen_request, frozen_request_hash,
     command_frozen_at, provider_snapshot_id, close_at, safe_deadline_at, shard_no, local_order_no)
VALUES ('api', NULL, NULL, $1, $2, $3, $3, $4, 'pending', $5, $6, $7, $8, $9,
        now(), $10, $11, $12, $13, $14)
ON CONFLICT (request_id) DO NOTHING`,
		arg.MemberID, arg.LotteryCode, arg.TargetPeriodNo, arg.Mode, arg.RequestID,
		arg.PayloadHash, arg.Payload, arg.FrozenRequest, arg.FrozenRequestHash,
		arg.ProviderSnapshotID, arg.CloseAt, arg.SafeDeadlineAt, arg.ShardNo, arg.LocalOrderNo); err != nil {
		return 0, "", err
	}
	var id int64
	var payloadHash string
	err := q.db.QueryRow(ctx, `
SELECT id, payload_hash
FROM scheme_bet_outbox
WHERE request_id = $1 AND origin = 'api'`, arg.RequestID).Scan(&id, &payloadHash)
	return id, payloadHash, err
}

func (q *Queries) LeaseFormalOutboxByID(ctx context.Context, id int64, owner string, now, leaseUntil time.Time) (schemebetting.LeasedCommand, bool, error) {
	var command schemebetting.LeasedCommand
	err := q.db.QueryRow(ctx, `
UPDATE scheme_bet_outbox
SET state = 'leased',
    lease_owner = $2,
    lease_fencing_token = lease_fencing_token + 1,
    lease_until = $4,
    updated_at = $3
WHERE id = $1
  AND state = 'pending'
  AND safe_deadline_at > $3
RETURNING id, COALESCE(scheme_id, ''), target_period_no, frozen_request, frozen_request_hash,
          close_at, safe_deadline_at, lease_owner, lease_fencing_token, lease_until`,
		id, owner, now, leaseUntil).Scan(
		&command.ID, &command.SchemeID, &command.TargetPeriod, &command.FrozenRequest, &command.FrozenRequestHash,
		&command.CloseAt, &command.SafeDeadline, &command.Lease.Owner, &command.Lease.Token, &command.Lease.Until,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schemebetting.LeasedCommand{}, false, nil
		}
		return schemebetting.LeasedCommand{}, false, err
	}
	return command, true, nil
}

type APIFormalBetOutboxResult struct {
	State              string
	OutcomeReason      string
	LocalOrderNo       string
	TargetPeriodNo     string
	ProviderOrderNo    string
	ProviderAmount     float64
	FinancialFinalized bool
	CreatedAt          time.Time
}

func (q *Queries) GetAPIFormalBetOutboxResult(ctx context.Context, id int64) (APIFormalBetOutboxResult, error) {
	var result APIFormalBetOutboxResult
	err := q.db.QueryRow(ctx, `
SELECT state, COALESCE(outcome_reason, ''), COALESCE(local_order_no, ''), target_period_no,
       COALESCE(provider_order_no, ''), COALESCE(provider_amount, 0),
       financial_finalized_at IS NOT NULL, created_at
FROM scheme_bet_outbox
WHERE id = $1 AND origin = 'api'`, id).Scan(
		&result.State, &result.OutcomeReason, &result.LocalOrderNo, &result.TargetPeriodNo,
		&result.ProviderOrderNo, &result.ProviderAmount, &result.FinancialFinalized, &result.CreatedAt,
	)
	return result, err
}
