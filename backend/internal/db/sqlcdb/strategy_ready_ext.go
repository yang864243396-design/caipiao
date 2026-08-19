package sqlcdb

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type StrategyReadyCandidate struct {
	RecordID     int64
	SchemeID     string
	StateVersion int64
}

func (q *Queries) ListStrategyReadyCandidates(ctx context.Context, lotteryCode, periodNo string, afterRecordID int64, rowLimit int32) ([]StrategyReadyCandidate, error) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	periodNo = strings.TrimSpace(periodNo)
	if lotteryCode == "" || periodNo == "" {
		return nil, nil
	}
	if rowLimit <= 0 || rowLimit > 2000 {
		rowLimit = 1000
	}
	rows, err := q.db.Query(ctx, `
SELECT c.id, c.scheme_id, si.state_version
FROM cloud_bet_records c
JOIN scheme_instances si ON si.id = c.scheme_id
JOIN lottery_draws d ON d.lottery_code = c.lottery_code AND d.issue_no = c.period_no
WHERE c.id > $3
  AND c.sim_bet = FALSE
  AND c.status = 'pending'
  AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
  AND c.rule_snapshot IS NOT NULL
  AND c.strategy_evaluated_at IS NULL
  AND c.lottery_code = $1
  AND c.period_no = $2
  AND NOT EXISTS (
      SELECT 1 FROM cloud_bet_records prior
      WHERE prior.scheme_id = c.scheme_id
        AND prior.sim_bet = FALSE
        AND prior.status = 'pending'
        AND NULLIF(TRIM(prior.third_party_bet_id), '') IS NOT NULL
        AND prior.strategy_evaluated_at IS NULL
        AND (prior.placed_at < c.placed_at OR (prior.placed_at = c.placed_at AND prior.id < c.id))
  )
  AND jsonb_typeof(d.balls) = 'array'
  AND jsonb_array_length(d.balls) > 0
ORDER BY c.id
LIMIT $4`, lotteryCode, periodNo, afterRecordID, rowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]StrategyReadyCandidate, 0, rowLimit)
	for rows.Next() {
		var item StrategyReadyCandidate
		if err := rows.Scan(&item.RecordID, &item.SchemeID, &item.StateVersion); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) PendingFormalStrategyRowForSchemeDraw(ctx context.Context, recordID int64, schemeID, lotteryCode, periodNo string, expectedStateVersion int64) (PendingFormalStrategyRow, bool, error) {
	var item PendingFormalStrategyRow
	var balls []byte
	err := q.db.QueryRow(ctx, `
SELECT c.id, c.scheme_id, COALESCE(c.lottery_code, ''), c.period_no,
       COALESCE(c.bet_content, ''), c.rule_snapshot, c.rule_version,
       c.rule_snapshot_hash, d.balls
FROM cloud_bet_records c
JOIN scheme_instances si ON si.id = c.scheme_id
JOIN lottery_draws d ON d.lottery_code = c.lottery_code AND d.issue_no = c.period_no
WHERE c.scheme_id = $1
  AND c.id = $5
  AND c.lottery_code = $2
  AND c.period_no = $3
  AND si.state_version = $4
  AND c.sim_bet = FALSE
  AND c.status = 'pending'
  AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
  AND c.rule_snapshot IS NOT NULL
  AND c.strategy_evaluated_at IS NULL
  AND jsonb_typeof(d.balls) = 'array'
  AND jsonb_array_length(d.balls) > 0
ORDER BY c.placed_at, c.id
LIMIT 1`, schemeID, lotteryCode, periodNo, expectedStateVersion, recordID).Scan(
		&item.RecordID, &item.SchemeID, &item.LotteryCode, &item.PeriodNo,
		&item.BetContent, &item.RuleSnapshot, &item.RuleVersion, &item.RuleSnapshotHash, &balls,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return item, false, nil
		}
		return item, false, err
	}
	item.Balls = ParseDrawBalls(balls)
	return item, true, nil
}
