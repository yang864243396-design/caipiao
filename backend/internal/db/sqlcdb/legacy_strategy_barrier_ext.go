package sqlcdb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const legacyStrategyBarrierCandidateLimit = 32

type LegacyStrategyBarrier struct {
	RecordID          int64
	LotteryCode       string
	PeriodNo          string
	StrategyEvaluated bool
	HasDraw           bool
}

func (q *Queries) GetLegacyStrategyBarrier(ctx context.Context, schemeID string) (LegacyStrategyBarrier, bool, error) {
	schemeID = strings.TrimSpace(schemeID)
	for offset := 0; offset < legacyStrategyBarrierCandidateLimit; offset++ {
		var recordID int64
		var placedAt time.Time
		err := q.db.QueryRow(ctx, `
SELECT id, placed_at
FROM cloud_bet_record_identity
WHERE scheme_id = $1
ORDER BY placed_at DESC, id DESC
OFFSET $2
LIMIT 1`, schemeID, offset).Scan(&recordID, &placedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return LegacyStrategyBarrier{}, false, nil
		}
		if err != nil {
			return LegacyStrategyBarrier{}, false, err
		}

		var row LegacyStrategyBarrier
		err = q.db.QueryRow(ctx, `
SELECT c.id,
       COALESCE(c.lottery_code, ''),
       c.period_no,
       c.rule_snapshot IS NULL OR c.strategy_evaluated_at IS NOT NULL,
       EXISTS (
           SELECT 1
           FROM lottery_draws d
           WHERE d.lottery_code = c.lottery_code
             AND d.issue_no = c.period_no
             AND jsonb_typeof(d.balls) = 'array'
             AND jsonb_array_length(d.balls) > 0
       )
FROM cloud_bet_records c
WHERE c.scheme_id = $1
  AND c.placed_at = $2
  AND c.id = $3
  AND c.sim_bet = FALSE
  AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
LIMIT 1`, pgx.QueryExecModeExec, schemeID, placedAt, recordID).Scan(
			&row.RecordID,
			&row.LotteryCode,
			&row.PeriodNo,
			&row.StrategyEvaluated,
			&row.HasDraw,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return LegacyStrategyBarrier{}, false, err
		}
		return row, true, nil
	}
	return LegacyStrategyBarrier{}, false, fmt.Errorf(
		"no accepted formal bet found in latest %d records for scheme %s",
		legacyStrategyBarrierCandidateLimit,
		schemeID,
	)
}

func (q *Queries) GetSchemeStateVersion(ctx context.Context, schemeID string) (int64, error) {
	var version int64
	err := q.db.QueryRow(ctx, `
SELECT state_version
FROM scheme_instances
WHERE id = $1`, strings.TrimSpace(schemeID)).Scan(&version)
	return version, err
}
