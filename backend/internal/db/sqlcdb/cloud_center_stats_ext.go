package sqlcdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type MemberCloudCenterStatsRow struct {
	SimBet            bool           `json:"sim_bet"`
	TotalTurnover     pgtype.Numeric `json:"total_turnover"`
	TotalSessionPnl   pgtype.Numeric `json:"total_session_pnl"`
	RunningSessionPnl pgtype.Numeric `json:"running_session_pnl"`
}

func (q *Queries) ListMemberCloudCenterStatsBySimBet(ctx context.Context, memberID int64) ([]MemberCloudCenterStatsRow, error) {
	rows, err := q.db.Query(ctx, `
SELECT
    sim_bet,
    COALESCE(SUM(turnover), 0)::numeric AS total_turnover,
    COALESCE(SUM(session_pnl), 0)::numeric AS total_session_pnl,
    COALESCE(SUM(session_pnl) FILTER (WHERE status = 'running'), 0)::numeric AS running_session_pnl
FROM scheme_instances
WHERE member_id = $1
GROUP BY sim_bet`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MemberCloudCenterStatsRow
	for rows.Next() {
		var row MemberCloudCenterStatsRow
		if err := rows.Scan(&row.SimBet, &row.TotalTurnover, &row.TotalSessionPnl, &row.RunningSessionPnl); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
