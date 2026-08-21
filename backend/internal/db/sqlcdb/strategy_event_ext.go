package sqlcdb

import (
	"context"
	"strings"
)

func (q *Queries) ListPendingFormalStrategyRowsForDraw(ctx context.Context, lotteryCode, periodNo string, rowLimit int32) ([]PendingFormalStrategyRow, error) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	periodNo = strings.TrimSpace(periodNo)
	if lotteryCode == "" || periodNo == "" {
		return nil, nil
	}
	if rowLimit <= 0 {
		rowLimit = 50
	}
	rows, err := q.db.Query(ctx, `
SELECT c.id, c.scheme_id, COALESCE(c.lottery_code, ''), c.period_no,
       COALESCE(c.bet_content, ''), c.rule_snapshot, c.rule_version,
       c.rule_snapshot_hash, d.balls,
       CASE c.status WHEN 'hit' THEN TRUE WHEN 'miss' THEN FALSE ELSE NULL END,
       d.drawn_at
FROM cloud_bet_records c
JOIN lottery_draws d ON d.lottery_code = c.lottery_code AND d.issue_no = c.period_no
WHERE c.sim_bet = FALSE
  AND c.status IN ('pending', 'hit', 'miss')
  AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
  AND c.rule_snapshot IS NOT NULL
  AND c.strategy_evaluated_at IS NULL
  AND c.lottery_code = $1
  AND c.period_no = $2
  AND NOT EXISTS (
      SELECT 1
      FROM cloud_bet_records prior
      WHERE prior.scheme_id = c.scheme_id
        AND prior.sim_bet = FALSE
        AND prior.status IN ('pending', 'hit', 'miss')
        AND NULLIF(TRIM(prior.third_party_bet_id), '') IS NOT NULL
        AND prior.strategy_evaluated_at IS NULL
        AND (prior.placed_at < c.placed_at
             OR (prior.placed_at = c.placed_at AND prior.id < c.id))
  )
  AND jsonb_typeof(d.balls) = 'array'
  AND jsonb_array_length(d.balls) > 0
ORDER BY c.placed_at ASC, c.id ASC
LIMIT $3`, lotteryCode, periodNo, rowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PendingFormalStrategyRow, 0, rowLimit)
	for rows.Next() {
		var item PendingFormalStrategyRow
		var balls []byte
		if err := rows.Scan(&item.RecordID, &item.SchemeID, &item.LotteryCode, &item.PeriodNo, &item.BetContent, &item.RuleSnapshot, &item.RuleVersion, &item.RuleSnapshotHash, &balls, &item.ProviderHit, &item.DrawnAt); err != nil {
			return nil, err
		}
		item.Balls = ParseDrawBalls(balls)
		out = append(out, item)
	}
	return out, rows.Err()
}
