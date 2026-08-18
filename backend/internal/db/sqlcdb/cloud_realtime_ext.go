package sqlcdb

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type CloudRealtimeDefinitionMeta struct {
	ID             string
	RunType        string
	SchemeCurrency string
	MemberID       int64
}

type CloudRealtimeStatsRow struct {
	MemberID                int64
	GeneratedAt             time.Time
	FormalTotalTurnover     pgtype.Numeric
	FormalTotalSessionPnl   pgtype.Numeric
	FormalRunningSessionPnl pgtype.Numeric
	SimTotalTurnover        pgtype.Numeric
	SimTotalSessionPnl      pgtype.Numeric
	SimRunningSessionPnl    pgtype.Numeric
	RunningSimSchemes       int32
	TodaySimSchemeStarts    int32
}

type SchemeRealtimeChange struct {
	MemberID   int64
	InstanceID string
	UpdatedAt  time.Time
}

func (q *Queries) ListSchemeInstancesRealtimeByIDs(ctx context.Context, ids []string) ([]SchemeInstance, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := q.db.Query(ctx, `
SELECT
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, bet_failed_detail, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    start_skip_period, start_skip_close_at,
    running_since, created_at, updated_at
FROM scheme_instances
WHERE id = ANY($1::text[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SchemeInstance, 0, len(ids))
	for rows.Next() {
		var fields instanceDisplayFields
		if err := rows.Scan(
			&fields.ID, &fields.DefinitionID, &fields.MemberID, &fields.Kind,
			&fields.SchemeName, &fields.LotteryCode, &fields.LotteryLabel,
			&fields.Status, &fields.StatusReason, &fields.BetFailedDetail,
			&fields.Turnover, &fields.Pnl, &fields.RunTimeSec, &fields.LookbackPnl,
			&fields.SessionPnl, &fields.Multiplier, &fields.CountdownSec, &fields.SimBet,
			&fields.StartSkipPeriod, &fields.StartSkipCloseAt,
			&fields.RunningSince, &fields.CreatedAt, &fields.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, schemeInstanceFromDisplay(fields))
	}
	return items, rows.Err()
}

func (q *Queries) ListSchemeDefinitionRealtimeMeta(ctx context.Context, ids []string) ([]CloudRealtimeDefinitionMeta, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := q.db.Query(ctx, `
SELECT
    id,
    member_id,
    COALESCE(config->>'runTypeId', '') AS run_type,
    COALESCE(config->>'schemeCurrency', '') AS scheme_currency
FROM scheme_definitions
WHERE id = ANY($1::text[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CloudRealtimeDefinitionMeta, 0, len(ids))
	for rows.Next() {
		var item CloudRealtimeDefinitionMeta
		if err := rows.Scan(&item.ID, &item.MemberID, &item.RunType, &item.SchemeCurrency); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) ListCloudRealtimeStats(ctx context.Context, memberIDs []int64, today time.Time) ([]CloudRealtimeStatsRow, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}
	rows, err := q.db.Query(ctx, `
SELECT
    requested.member_id,
    statement_timestamp() AS generated_at,
    COALESCE(SUM(i.turnover) FILTER (WHERE i.sim_bet = false), 0)::numeric AS formal_total_turnover,
    COALESCE(SUM(i.session_pnl) FILTER (WHERE i.sim_bet = false), 0)::numeric AS formal_total_session_pnl,
    COALESCE(SUM(i.session_pnl) FILTER (WHERE i.sim_bet = false AND i.status = 'running'), 0)::numeric AS formal_running_session_pnl,
    COALESCE(SUM(i.turnover) FILTER (WHERE i.sim_bet = true), 0)::numeric AS sim_total_turnover,
    COALESCE(SUM(i.session_pnl) FILTER (WHERE i.sim_bet = true), 0)::numeric AS sim_total_session_pnl,
    COALESCE(SUM(i.session_pnl) FILTER (WHERE i.sim_bet = true AND i.status = 'running'), 0)::numeric AS sim_running_session_pnl,
    COUNT(i.id) FILTER (WHERE i.sim_bet = true AND i.status = 'running')::int AS running_sim_schemes,
    CASE
      WHEN m.sim_scheme_starts_date IS NOT DISTINCT FROM $2::date
      THEN GREATEST(m.sim_scheme_starts_count, 0)
      ELSE 0
    END::int AS today_sim_scheme_starts
FROM unnest($1::bigint[]) AS requested(member_id)
LEFT JOIN members m ON m.id = requested.member_id
LEFT JOIN scheme_instances i ON i.member_id = requested.member_id
GROUP BY requested.member_id, m.sim_scheme_starts_date, m.sim_scheme_starts_count
ORDER BY requested.member_id`, memberIDs, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CloudRealtimeStatsRow, 0, len(memberIDs))
	for rows.Next() {
		var item CloudRealtimeStatsRow
		if err := rows.Scan(
			&item.MemberID,
			&item.GeneratedAt,
			&item.FormalTotalTurnover, &item.FormalTotalSessionPnl, &item.FormalRunningSessionPnl,
			&item.SimTotalTurnover, &item.SimTotalSessionPnl, &item.SimRunningSessionPnl,
			&item.RunningSimSchemes, &item.TodaySimSchemeStarts,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) ListSchemeRealtimeChanges(ctx context.Context, after time.Time, afterID string, limit int) ([]SchemeRealtimeChange, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := q.db.Query(ctx, `
SELECT member_id, id, updated_at
FROM scheme_instances
WHERE (updated_at, id) > ($1, $2)
ORDER BY updated_at ASC, id ASC
LIMIT $3`, after, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SchemeRealtimeChange, 0, limit)
	for rows.Next() {
		var item SchemeRealtimeChange
		if err := rows.Scan(&item.MemberID, &item.InstanceID, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
