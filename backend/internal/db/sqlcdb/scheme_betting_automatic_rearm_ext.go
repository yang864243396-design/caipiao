package sqlcdb

import "context"

type AutomaticRearmCandidate struct {
	OutboxID         int64
	RequestID        string
	SchemeID         string
	LotteryCode      string
	ShardNo          int32
	State            string
	Reason           string
	ChainBlockReason string
}

const automaticRearmCandidateSelect = `
SELECT o.id, o.request_id, i.id, i.lottery_code, o.shard_no, o.state, COALESCE(o.outcome_reason, ''),
       COALESCE(i.chain_block_reason, '')
FROM scheme_instances i
JOIN LATERAL (
    SELECT id, request_id, lottery_code, shard_no, state, outcome_reason, created_at
    FROM scheme_bet_outbox
    WHERE scheme_id = i.id
    ORDER BY created_at DESC, id DESC
    LIMIT 1
) o ON true
WHERE i.betting_owner = 'event'
  AND i.status = 'running'
  AND i.strict_chain_state = 'blocked_requires_rearm'
  AND (
      (o.state = 'rejected' AND o.outcome_reason = 'provider_pre_send_failed')
      OR
      (o.state = 'expired' AND o.outcome_reason IN (
          'safe_deadline_elapsed', 'dispatcher_lost_before_start_deadline_elapsed'
      ))
  )
  AND NOT EXISTS (
      SELECT 1
      FROM scheme_bet_outbox unresolved
      WHERE unresolved.scheme_id = i.id
        AND unresolved.state IN ('sent_unknown', 'external_acceptance_unknown')
  )`

func (q *Queries) GetAutomaticRearmCandidate(ctx context.Context, outboxID int64) (AutomaticRearmCandidate, bool, error) {
	var row AutomaticRearmCandidate
	err := q.db.QueryRow(ctx, automaticRearmCandidateSelect+`
  AND o.id = $1`, outboxID).Scan(
		&row.OutboxID, &row.RequestID, &row.SchemeID, &row.LotteryCode,
		&row.ShardNo, &row.State, &row.Reason, &row.ChainBlockReason,
	)
	if err != nil {
		if isNoRowsError(err) {
			return AutomaticRearmCandidate{}, false, nil
		}
		return AutomaticRearmCandidate{}, false, err
	}
	return row, true, nil
}

func (q *Queries) ListAutomaticRearmCandidates(
	ctx context.Context, lotteryCodes []string, shards []int32, limit int32,
) ([]AutomaticRearmCandidate, error) {
	if limit <= 0 {
		limit = 32
	}
	rows, err := q.db.Query(ctx, automaticRearmCandidateSelect+`
  AND i.lottery_code = ANY($1::text[])
  AND o.shard_no = ANY($2::integer[])
ORDER BY i.updated_at, i.id
LIMIT $3`, lotteryCodes, shards, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AutomaticRearmCandidate, 0, limit)
	for rows.Next() {
		var row AutomaticRearmCandidate
		if err := rows.Scan(
			&row.OutboxID, &row.RequestID, &row.SchemeID, &row.LotteryCode,
			&row.ShardNo, &row.State, &row.Reason, &row.ChainBlockReason,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
