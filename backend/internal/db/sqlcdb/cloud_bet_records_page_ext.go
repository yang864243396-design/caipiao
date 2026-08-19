package sqlcdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type CloudBetRecordFilter struct {
	MemberID       int64
	SinceAt        pgtype.Timestamptz
	UntilAt        pgtype.Timestamptz
	SimBet         pgtype.Bool
	LotteryCode    pgtype.Text
	GuajiAccountID pgtype.Int8
}

type CloudBetAggregate struct {
	SchemeName string
	TotalBet   float64
	TotalPrize float64
	PnL        float64
	TotalRows  int64
	HitRows    int64
}

type CloudBetGroupAggregate struct {
	SchemeID string
	CloudBetAggregate
}

func (q *Queries) SummarizeCloudBetRecordGroups(ctx context.Context, filter CloudBetRecordFilter) (CloudBetAggregate, error) {
	var result CloudBetAggregate
	err := q.db.QueryRow(ctx, `
SELECT
    COALESCE(SUM(TRUNC(c.amount, 2)), 0)::float8,
    COALESCE(SUM(CASE WHEN c.status = 'hit' THEN TRUNC(c.amount, 2) + c.pnl ELSE 0 END), 0)::float8,
    COALESCE(SUM(c.pnl), 0)::float8,
    COUNT(*)::bigint,
    COUNT(*) FILTER (WHERE c.status = 'hit')::bigint
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND c.placed_at >= $2
  AND c.placed_at < $3
  AND ($4::boolean IS NULL OR c.sim_bet = $4::boolean)
  AND ($5::text IS NULL OR $5::text = '' OR c.lottery_code = $5::text)
  AND (
    $4::boolean IS NOT DISTINCT FROM true
    OR (
      NOT c.sim_bet
      AND $6::bigint IS NOT NULL
      AND c.guaji_account_id = $6::bigint
      AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
    )
  )`,
		filter.MemberID, filter.SinceAt, filter.UntilAt, filter.SimBet,
		filter.LotteryCode, filter.GuajiAccountID,
	).Scan(&result.TotalBet, &result.TotalPrize, &result.PnL, &result.TotalRows, &result.HitRows)
	return result, err
}

func (q *Queries) ListCloudBetRecordGroupsPage(
	ctx context.Context,
	filter CloudBetRecordFilter,
	offset, limit int32,
) ([]CloudBetGroupAggregate, error) {
	rows, err := q.db.Query(ctx, `
SELECT
    c.scheme_id,
    MAX(c.scheme_name) AS scheme_name,
    COALESCE(SUM(TRUNC(c.amount, 2)), 0)::float8 AS total_bet,
    COALESCE(SUM(CASE WHEN c.status = 'hit' THEN TRUNC(c.amount, 2) + c.pnl ELSE 0 END), 0)::float8 AS total_prize,
    COALESCE(SUM(c.pnl), 0)::float8 AS pnl,
    COUNT(*)::bigint AS total_rows,
    COUNT(*) FILTER (WHERE c.status = 'hit')::bigint AS hit_rows
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND c.placed_at >= $2
  AND c.placed_at < $3
  AND ($4::boolean IS NULL OR c.sim_bet = $4::boolean)
  AND ($5::text IS NULL OR $5::text = '' OR c.lottery_code = $5::text)
  AND (
    $4::boolean IS NOT DISTINCT FROM true
    OR (
      NOT c.sim_bet
      AND $6::bigint IS NOT NULL
      AND c.guaji_account_id = $6::bigint
      AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
    )
  )
GROUP BY c.scheme_id
ORDER BY MAX(c.placed_at) DESC, MAX(c.id) DESC
OFFSET $7
LIMIT $8`,
		filter.MemberID, filter.SinceAt, filter.UntilAt, filter.SimBet,
		filter.LotteryCode, filter.GuajiAccountID, offset, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]CloudBetGroupAggregate, 0)
	for rows.Next() {
		var row CloudBetGroupAggregate
		if err := rows.Scan(
			&row.SchemeID, &row.SchemeName, &row.TotalBet, &row.TotalPrize,
			&row.PnL, &row.TotalRows, &row.HitRows,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (q *Queries) SummarizeCloudBetRecordsByScheme(
	ctx context.Context,
	filter CloudBetRecordFilter,
	schemeID string,
) (CloudBetAggregate, error) {
	var result CloudBetAggregate
	err := q.db.QueryRow(ctx, `
SELECT
    COALESCE(MAX(c.scheme_name), ''),
    COALESCE(SUM(TRUNC(c.amount, 2)), 0)::float8,
    COALESCE(SUM(CASE WHEN c.status = 'hit' THEN TRUNC(c.amount, 2) + c.pnl ELSE 0 END), 0)::float8,
    COALESCE(SUM(c.pnl), 0)::float8,
    COUNT(*)::bigint,
    COUNT(*) FILTER (WHERE c.status = 'hit')::bigint
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND c.placed_at >= $2
  AND c.placed_at < $3
  AND ($4::boolean IS NULL OR c.sim_bet = $4::boolean)
  AND ($5::text IS NULL OR $5::text = '' OR c.lottery_code = $5::text)
  AND (
    $4::boolean IS NOT DISTINCT FROM true
    OR (
      NOT c.sim_bet
      AND $6::bigint IS NOT NULL
      AND c.guaji_account_id = $6::bigint
      AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
    )
  )
  AND c.scheme_id = $7`,
		filter.MemberID, filter.SinceAt, filter.UntilAt, filter.SimBet,
		filter.LotteryCode, filter.GuajiAccountID, schemeID,
	).Scan(
		&result.SchemeName, &result.TotalBet, &result.TotalPrize,
		&result.PnL, &result.TotalRows, &result.HitRows,
	)
	return result, err
}

func (q *Queries) ListCloudBetRecordsBySchemePage(
	ctx context.Context,
	filter CloudBetRecordFilter,
	schemeID string,
	offset, limit int32,
) ([]ListCloudBetRecordsFilteredRow, error) {
	rows, err := q.db.Query(ctx, `
WITH page AS (
    SELECT c.*
    FROM cloud_bet_records c
    WHERE c.member_id = $1
      AND c.placed_at >= $2
      AND c.placed_at < $3
      AND ($4::boolean IS NULL OR c.sim_bet = $4::boolean)
      AND ($5::text IS NULL OR $5::text = '' OR c.lottery_code = $5::text)
      AND (
        $4::boolean IS NOT DISTINCT FROM true
        OR (
          NOT c.sim_bet
          AND $6::bigint IS NOT NULL
          AND c.guaji_account_id = $6::bigint
          AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
        )
      )
      AND c.scheme_id = $7
    ORDER BY c.placed_at DESC, c.id DESC
    OFFSET $8
    LIMIT $9
)
SELECT
    c.record_no,
    NULLIF(TRIM(c.third_party_bet_id), '') AS third_party_bet_id,
    c.scheme_id,
    c.scheme_name,
    c.lottery_code,
    c.period_no,
    c.third_party_period,
    c.play_type,
    c.multiplier,
    c.round_label,
    c.amount::float8,
    c.pnl::float8,
    c.status,
    c.bet_content,
    c.placed_at
FROM page c
ORDER BY c.placed_at DESC, c.id DESC`,
		filter.MemberID, filter.SinceAt, filter.UntilAt, filter.SimBet,
		filter.LotteryCode, filter.GuajiAccountID, schemeID, offset, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ListCloudBetRecordsFilteredRow, 0)
	for rows.Next() {
		var row ListCloudBetRecordsFilteredRow
		if err := rows.Scan(
			&row.RecordNo, &row.ThirdPartyBetID, &row.SchemeID, &row.SchemeName,
			&row.LotteryCode, &row.PeriodNo, &row.ThirdPartyPeriod, &row.PlayType,
			&row.Multiplier, &row.RoundLabel, &row.Amount, &row.Pnl, &row.Status,
			&row.BetContent, &row.PlacedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
