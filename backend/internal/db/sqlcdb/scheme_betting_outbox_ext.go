package sqlcdb

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProviderPeriodSnapshotRow struct {
	ID         int64
	PeriodNo   string
	OpenAt     pgtype.Timestamptz
	CloseAt    time.Time
	ObservedAt time.Time
}

func (q *Queries) ListOpenProviderPeriodSnapshots(ctx context.Context, lotteryCode, sourcePeriod string, now, observedAfter time.Time, rowLimit int32) ([]ProviderPeriodSnapshotRow, error) {
	if rowLimit <= 0 {
		rowLimit = 8
	}
	rows, err := q.db.Query(ctx, `
SELECT DISTINCT ON (period_no) id, period_no, open_at, close_at, observed_at
FROM provider_period_snapshots
WHERE lottery_code = $1
  AND period_no <> $2
  AND (open_at IS NULL OR open_at <= $3)
  AND close_at > $3
  AND observed_at >= $4
ORDER BY period_no, observed_at DESC, id DESC
LIMIT $5`, lotteryCode, sourcePeriod, now, observedAfter, rowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ProviderPeriodSnapshotRow, 0, rowLimit)
	for rows.Next() {
		var row ProviderPeriodSnapshotRow
		if err := rows.Scan(&row.ID, &row.PeriodNo, &row.OpenAt, &row.CloseAt, &row.ObservedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (q *Queries) LockSchemeStateVersion(ctx context.Context, schemeID string) (int64, error) {
	var version int64
	err := q.db.QueryRow(ctx, `SELECT state_version FROM scheme_instances WHERE id = $1 FOR UPDATE`, schemeID).Scan(&version)
	return version, err
}

type InsertSchemePeriodDecisionParams struct {
	SchemeID           string
	LotteryCode        string
	SourcePeriodNo     string
	SourceBetRecordID  int64
	DrawHash           string
	StateVersionBefore int64
	StateVersionAfter  int64
	RuleVersion        pgtype.Int4
	RuleSnapshotHash   pgtype.Text
	LocalHit           bool
	WinningUnits       int
	Status             string
	Diagnostics        []byte
}

func (q *Queries) InsertSchemePeriodDecision(ctx context.Context, arg InsertSchemePeriodDecisionParams) (int64, bool, error) {
	var id int64
	err := q.db.QueryRow(ctx, `
INSERT INTO scheme_period_decisions
    (scheme_id, lottery_code, source_period_no, source_bet_record_id, draw_hash,
     state_version_before, state_version_after, rule_version, rule_snapshot_hash,
     local_hit, winning_units, status, diagnostics)
VALUES ($1, $2, $3, NULLIF($4, 0), NULLIF($5, ''), $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (scheme_id, source_period_no) DO NOTHING
RETURNING id`, arg.SchemeID, arg.LotteryCode, arg.SourcePeriodNo, arg.SourceBetRecordID, arg.DrawHash,
		arg.StateVersionBefore, arg.StateVersionAfter, arg.RuleVersion, arg.RuleSnapshotHash,
		arg.LocalHit, arg.WinningUnits, arg.Status, arg.Diagnostics).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}
	err = q.db.QueryRow(ctx, `SELECT id FROM scheme_period_decisions WHERE scheme_id = $1 AND source_period_no = $2`, arg.SchemeID, arg.SourcePeriodNo).Scan(&id)
	return id, false, err
}

type InsertShadowSchemeBetOutboxParams struct {
	DecisionID         int64
	SchemeID           string
	LotteryCode        string
	SourcePeriodNo     string
	TargetPeriodNo     string
	RequestID          string
	PayloadHash        string
	Payload            []byte
	ProviderSnapshotID int64
	CloseAt            time.Time
	SafeDeadlineAt     time.Time
	ShardNo            int32
}

func (q *Queries) InsertShadowSchemeBetOutbox(ctx context.Context, arg InsertShadowSchemeBetOutboxParams) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO scheme_bet_outbox
    (decision_id, scheme_id, lottery_code, source_period_no, target_period_no,
     mode, state, request_id, payload_hash, payload, provider_snapshot_id,
     close_at, safe_deadline_at, shard_no)
VALUES ($1, $2, $3, $4, $5, 'shadow', 'pending', $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (decision_id) DO NOTHING`, arg.DecisionID, arg.SchemeID, arg.LotteryCode,
		arg.SourcePeriodNo, arg.TargetPeriodNo, arg.RequestID, arg.PayloadHash, arg.Payload,
		arg.ProviderSnapshotID, arg.CloseAt, arg.SafeDeadlineAt, arg.ShardNo)
	return err
}
