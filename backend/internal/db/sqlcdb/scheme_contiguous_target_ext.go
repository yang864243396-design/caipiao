package sqlcdb

import (
	"context"
	"time"
)

type AwaitingContiguousTargetRow struct {
	DecisionID        int64
	SchemeID          string
	MemberID          int64
	LotteryCode       string
	SourcePeriodNo    string
	SourceBetRecordID int64
	TargetDeadlineAt  time.Time
	StateVersionAfter int64
	ChainID           string
	ChainSeq          int64
	ShardNo           int32
	Mode              string
}

type CompleteAwaitingContiguousTargetParams struct {
	DecisionID     int64
	TargetPeriodNo string
	Diagnostics    []byte
}

type MissAwaitingContiguousTargetParams struct {
	DecisionID    int64
	FailureReason string
	Diagnostics   []byte
}

const awaitingContiguousTargetSelect = `
SELECT d.id AS decision_id, d.scheme_id, i.member_id, d.lottery_code, d.source_period_no,
       COALESCE(d.source_bet_record_id, 0), d.target_deadline_at, d.state_version_after,
       COALESCE(i.chain_id, ''), i.chain_seq, d.shard_no,
       CASE WHEN i.sim_bet THEN 'shadow' ELSE 'production' END
FROM scheme_period_decisions d
JOIN scheme_instances i ON i.id = d.scheme_id
WHERE d.status = 'awaiting_target'`

func (q *Queries) GetAwaitingContiguousTargetForUpdate(ctx context.Context, decisionID int64) (AwaitingContiguousTargetRow, bool, error) {
	var row AwaitingContiguousTargetRow
	err := q.db.QueryRow(ctx, awaitingContiguousTargetSelect+`
  AND d.id = $1
FOR UPDATE OF d, i`, decisionID).Scan(
		&row.DecisionID, &row.SchemeID, &row.MemberID, &row.LotteryCode, &row.SourcePeriodNo,
		&row.SourceBetRecordID, &row.TargetDeadlineAt, &row.StateVersionAfter,
		&row.ChainID, &row.ChainSeq, &row.ShardNo, &row.Mode,
	)
	if err != nil {
		if isNoRowsError(err) {
			return AwaitingContiguousTargetRow{}, false, nil
		}
		return AwaitingContiguousTargetRow{}, false, err
	}
	return row, true, nil
}

func (q *Queries) ListAwaitingContiguousTargets(
	ctx context.Context, lotteryCodes []string, shards []int32, cursor int64, limit int32,
) ([]AwaitingContiguousTargetRow, error) {
	if limit <= 0 || limit > 32 {
		limit = 32
	}
	rows, err := q.db.Query(ctx, `
WITH scopes AS (
    SELECT lottery_code, shard_no
    FROM unnest($1::text[]) AS lotteries(lottery_code)
    CROSS JOIN unnest($2::integer[]) AS shards(shard_no)
), candidates AS (
    SELECT scoped.*
    FROM scopes scope
    CROSS JOIN LATERAL (`+awaitingContiguousTargetSelect+`
      AND d.lottery_code = scope.lottery_code
      AND d.shard_no = scope.shard_no
      AND d.id > $3
    ORDER BY d.id
    LIMIT 32
    ) AS scoped
)
SELECT *
FROM candidates
ORDER BY decision_id
LIMIT $4`, lotteryCodes, shards, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AwaitingContiguousTargetRow, 0, limit)
	for rows.Next() {
		var row AwaitingContiguousTargetRow
		if err := rows.Scan(
			&row.DecisionID, &row.SchemeID, &row.MemberID, &row.LotteryCode, &row.SourcePeriodNo,
			&row.SourceBetRecordID, &row.TargetDeadlineAt, &row.StateVersionAfter,
			&row.ChainID, &row.ChainSeq, &row.ShardNo, &row.Mode,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (q *Queries) CompleteAwaitingContiguousTarget(ctx context.Context, arg CompleteAwaitingContiguousTargetParams) (bool, error) {
	tag, err := q.db.Exec(ctx, `
WITH locked_decision AS (
    SELECT d.id, d.scheme_id
    FROM scheme_period_decisions d
    WHERE d.id = $1 AND d.status = 'awaiting_target'
    FOR UPDATE
), locked_instance AS (
    SELECT i.id
    FROM scheme_instances i
    JOIN locked_decision d ON d.scheme_id = i.id
    FOR UPDATE
)
UPDATE scheme_period_decisions d
SET status = 'completed',
    target_period_no = NULLIF($2, ''),
    diagnostics = $3,
    decided_at = now()
FROM locked_decision ld
JOIN locked_instance li ON li.id = ld.scheme_id
WHERE d.id = ld.id
  AND d.status = 'awaiting_target'
  AND clock_timestamp() < d.target_deadline_at`, arg.DecisionID, arg.TargetPeriodNo, arg.Diagnostics)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (q *Queries) MissAwaitingContiguousTarget(ctx context.Context, arg MissAwaitingContiguousTargetParams) (bool, error) {
	tag, err := q.db.Exec(ctx, `
WITH locked_decision AS (
    SELECT d.id, d.scheme_id
    FROM scheme_period_decisions d
    WHERE d.id = $1 AND d.status = 'awaiting_target'
    FOR UPDATE
), locked_instance AS (
    SELECT i.id
    FROM scheme_instances i
    JOIN locked_decision d ON d.scheme_id = i.id
    FOR UPDATE
), missed AS (
    UPDATE scheme_period_decisions d
    SET status = 'missed_contiguous_period',
        failure_reason = NULLIF($2, ''),
        diagnostics = $3,
        decided_at = now()
    FROM locked_decision ld
    JOIN locked_instance li ON li.id = ld.scheme_id
    WHERE d.id = ld.id
      AND d.status = 'awaiting_target'
      AND clock_timestamp() >= d.target_deadline_at
    RETURNING d.scheme_id
)
UPDATE scheme_instances i
SET status = 'paused',
    status_reason = 'bet_failed',
    strict_chain_state = 'blocked_requires_rearm',
    chain_block_reason = 'missed_contiguous_period',
    bet_failed_detail = NULLIF($2, ''),
    updated_at = now()
FROM missed
WHERE i.id = missed.scheme_id`, arg.DecisionID, arg.FailureReason, arg.Diagnostics)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
