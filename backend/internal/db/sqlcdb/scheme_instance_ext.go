package sqlcdb

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// EscapeILIKEPattern 转义 ILIKE 通配符，避免用户输入 %/_ 扩大匹配范围。
func EscapeILIKEPattern(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

type instanceDisplayFields struct {
	ID, DefinitionID, Kind, SchemeName, LotteryCode, LotteryLabel string
	MemberID                                                      int64
	Status, StatusReason                                          string
	BetFailedDetail                                               pgtype.Text
	Turnover, Pnl, LookbackPnl, SessionPnl, Multiplier            pgtype.Numeric
	RunTimeSec, CountdownSec                                      int32
	SimBet                                                        bool
	StartSkipPeriod                                               pgtype.Text
	StartSkipCloseAt                                              pgtype.Timestamptz
	RunningSince, CreatedAt, UpdatedAt                            pgtype.Timestamptz
}

func schemeInstanceFromDisplay(f instanceDisplayFields) SchemeInstance {
	return SchemeInstance{
		ID:               f.ID,
		DefinitionID:     f.DefinitionID,
		MemberID:         f.MemberID,
		Kind:             f.Kind,
		SchemeName:       f.SchemeName,
		LotteryCode:      f.LotteryCode,
		LotteryLabel:     f.LotteryLabel,
		Status:           f.Status,
		StatusReason:     f.StatusReason,
		BetFailedDetail:  f.BetFailedDetail,
		Turnover:         f.Turnover,
		Pnl:              f.Pnl,
		RunTimeSec:       f.RunTimeSec,
		LookbackPnl:      f.LookbackPnl,
		SessionPnl:       f.SessionPnl,
		Multiplier:       f.Multiplier,
		CountdownSec:     f.CountdownSec,
		SimBet:           f.SimBet,
		StartSkipPeriod:  f.StartSkipPeriod,
		StartSkipCloseAt: f.StartSkipCloseAt,
		RunningSince:     f.RunningSince,
		CreatedAt:        f.CreatedAt,
		UpdatedAt:        f.UpdatedAt,
	}
}

func schemeInstanceFromStatusUpdate(
	id, definitionID, kind, schemeName, lotteryCode, lotteryLabel string,
	memberID int64,
	status, statusReason string,
	turnover, pnl, lookbackPnl, sessionPnl, multiplier pgtype.Numeric,
	runTimeSec, countdownSec int32,
	simBet bool,
	runningSince, createdAt, updatedAt pgtype.Timestamptz,
) SchemeInstance {
	return schemeInstanceFromDisplay(instanceDisplayFields{
		ID: id, DefinitionID: definitionID, MemberID: memberID, Kind: kind,
		SchemeName: schemeName, LotteryCode: lotteryCode, LotteryLabel: lotteryLabel,
		Status: status, StatusReason: statusReason,
		Turnover: turnover, Pnl: pnl, LookbackPnl: lookbackPnl, SessionPnl: sessionPnl,
		Multiplier: multiplier, RunTimeSec: runTimeSec, CountdownSec: countdownSec, SimBet: simBet,
		RunningSince: runningSince, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func SchemeInstanceFromListRow(r ListSchemeInstancesByMemberRow) SchemeInstance {
	return schemeInstanceFromDisplay(instanceDisplayFields{
		ID: r.ID, DefinitionID: r.DefinitionID, MemberID: r.MemberID, Kind: r.Kind,
		SchemeName: r.SchemeName, LotteryCode: r.LotteryCode, LotteryLabel: r.LotteryLabel,
		Status: r.Status, StatusReason: r.StatusReason, BetFailedDetail: r.BetFailedDetail,
		Turnover: r.Turnover, Pnl: r.Pnl, LookbackPnl: r.LookbackPnl, SessionPnl: r.SessionPnl,
		Multiplier: r.Multiplier, RunTimeSec: r.RunTimeSec, CountdownSec: r.CountdownSec, SimBet: r.SimBet,
		StartSkipPeriod: r.StartSkipPeriod, StartSkipCloseAt: r.StartSkipCloseAt,
		RunningSince: r.RunningSince, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	})
}

func SchemeInstanceFromListRows(rows []ListSchemeInstancesByMemberRow) []SchemeInstance {
	out := make([]SchemeInstance, len(rows))
	for i, r := range rows {
		out[i] = SchemeInstanceFromListRow(r)
	}
	return out
}

func SchemeInstanceFromListIDsRow(r ListSchemeInstancesByMemberIDsRow) SchemeInstance {
	return schemeInstanceFromDisplay(instanceDisplayFields{
		ID: r.ID, DefinitionID: r.DefinitionID, MemberID: r.MemberID, Kind: r.Kind,
		SchemeName: r.SchemeName, LotteryCode: r.LotteryCode, LotteryLabel: r.LotteryLabel,
		Status: r.Status, StatusReason: r.StatusReason, BetFailedDetail: r.BetFailedDetail,
		Turnover: r.Turnover, Pnl: r.Pnl, LookbackPnl: r.LookbackPnl, SessionPnl: r.SessionPnl,
		Multiplier: r.Multiplier, RunTimeSec: r.RunTimeSec, CountdownSec: r.CountdownSec, SimBet: r.SimBet,
		RunningSince: r.RunningSince, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	})
}

// ListSchemeInstancesByMemberIDsEx 含 start_skip_*，供列表激活判定与倒计时使用。
func (q *Queries) ListSchemeInstancesByMemberIDsEx(ctx context.Context, memberID int64, ids []string) ([]SchemeInstance, error) {
	const sql = `
SELECT
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, bet_failed_detail, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    start_skip_period, start_skip_close_at,
    running_since, created_at, updated_at
FROM scheme_instances
WHERE member_id = $1
  AND id = ANY($2::text[])`
	rows, err := q.db.Query(ctx, sql, memberID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SchemeInstance, 0, len(ids))
	for rows.Next() {
		var f instanceDisplayFields
		if err := rows.Scan(
			&f.ID, &f.DefinitionID, &f.MemberID, &f.Kind, &f.SchemeName, &f.LotteryCode, &f.LotteryLabel,
			&f.Status, &f.StatusReason, &f.BetFailedDetail, &f.Turnover, &f.Pnl, &f.RunTimeSec,
			&f.LookbackPnl, &f.SessionPnl, &f.Multiplier, &f.CountdownSec, &f.SimBet,
			&f.StartSkipPeriod, &f.StartSkipCloseAt,
			&f.RunningSince, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, schemeInstanceFromDisplay(f))
	}
	return items, rows.Err()
}

func SchemeInstanceFromListIDsRows(rows []ListSchemeInstancesByMemberIDsRow) []SchemeInstance {
	out := make([]SchemeInstance, len(rows))
	for i, r := range rows {
		out[i] = SchemeInstanceFromListIDsRow(r)
	}
	return out
}

func SchemeInstanceFromListPaginatedRow(r ListSchemeInstancesByMemberPaginatedRow) SchemeInstance {
	return schemeInstanceFromDisplay(instanceDisplayFields{
		ID: r.ID, DefinitionID: r.DefinitionID, MemberID: r.MemberID, Kind: r.Kind,
		SchemeName: r.SchemeName, LotteryCode: r.LotteryCode, LotteryLabel: r.LotteryLabel,
		Status: r.Status, StatusReason: r.StatusReason, BetFailedDetail: r.BetFailedDetail,
		Turnover: r.Turnover, Pnl: r.Pnl, LookbackPnl: r.LookbackPnl, SessionPnl: r.SessionPnl,
		Multiplier: r.Multiplier, RunTimeSec: r.RunTimeSec, CountdownSec: r.CountdownSec, SimBet: r.SimBet,
		RunningSince: r.RunningSince, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	})
}

func SchemeInstanceFromListPaginatedRows(rows []ListSchemeInstancesByMemberPaginatedRow) []SchemeInstance {
	out := make([]SchemeInstance, len(rows))
	for i, r := range rows {
		out[i] = SchemeInstanceFromListPaginatedRow(r)
	}
	return out
}

func SchemeInstanceFromMemberRow(r GetSchemeInstanceByIDAndMemberRow) SchemeInstance {
	return schemeInstanceFromDisplay(instanceDisplayFields{
		ID: r.ID, DefinitionID: r.DefinitionID, MemberID: r.MemberID, Kind: r.Kind,
		SchemeName: r.SchemeName, LotteryCode: r.LotteryCode, LotteryLabel: r.LotteryLabel,
		Status: r.Status, StatusReason: r.StatusReason, BetFailedDetail: r.BetFailedDetail,
		Turnover: r.Turnover, Pnl: r.Pnl, LookbackPnl: r.LookbackPnl, SessionPnl: r.SessionPnl,
		Multiplier: r.Multiplier, RunTimeSec: r.RunTimeSec, CountdownSec: r.CountdownSec, SimBet: r.SimBet,
		RunningSince: r.RunningSince, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	})
}

func SchemeInstanceFromInsertRow(r InsertSchemeInstanceRow) SchemeInstance {
	return schemeInstanceFromStatusUpdate(
		r.ID, r.DefinitionID, r.Kind, r.SchemeName, r.LotteryCode, r.LotteryLabel, r.MemberID,
		r.Status, r.StatusReason,
		r.Turnover, r.Pnl, r.LookbackPnl, r.SessionPnl, r.Multiplier,
		r.RunTimeSec, r.CountdownSec, r.SimBet,
		pgtype.Timestamptz{}, r.CreatedAt, r.UpdatedAt,
	)
}

func SchemeInstanceFromSimBetRow(r UpdateSchemeInstanceSimBetRow) SchemeInstance {
	return schemeInstanceFromStatusUpdate(
		r.ID, r.DefinitionID, r.Kind, r.SchemeName, r.LotteryCode, r.LotteryLabel, r.MemberID,
		r.Status, r.StatusReason,
		r.Turnover, r.Pnl, r.LookbackPnl, r.SessionPnl, r.Multiplier,
		r.RunTimeSec, r.CountdownSec, r.SimBet,
		r.RunningSince, r.CreatedAt, r.UpdatedAt,
	)
}

func SchemeInstanceFromMultiplierRow(r UpdateSchemeInstanceMultiplierRow) SchemeInstance {
	return schemeInstanceFromStatusUpdate(
		r.ID, r.DefinitionID, r.Kind, r.SchemeName, r.LotteryCode, r.LotteryLabel, r.MemberID,
		r.Status, r.StatusReason,
		r.Turnover, r.Pnl, r.LookbackPnl, r.SessionPnl, r.Multiplier,
		r.RunTimeSec, r.CountdownSec, r.SimBet,
		r.RunningSince, r.CreatedAt, r.UpdatedAt,
	)
}

func SchemeInstanceFromPendingToRunningRow(r UpdateSchemeInstanceStatusFromPendingToRunningRow) SchemeInstance {
	return schemeInstanceFromStatusUpdate(
		r.ID, r.DefinitionID, r.Kind, r.SchemeName, r.LotteryCode, r.LotteryLabel, r.MemberID,
		r.Status, r.StatusReason,
		r.Turnover, r.Pnl, r.LookbackPnl, r.SessionPnl, r.Multiplier,
		r.RunTimeSec, r.CountdownSec, r.SimBet,
		r.RunningSince, r.CreatedAt, r.UpdatedAt,
	)
}

func SchemeInstanceFromMaintenanceResumeRow(r ResumeSchemeInstanceAfterMaintenanceRow) SchemeInstance {
	return schemeInstanceFromStatusUpdate(
		r.ID, r.DefinitionID, r.Kind, r.SchemeName, r.LotteryCode, r.LotteryLabel, r.MemberID,
		r.Status, r.StatusReason,
		r.Turnover, r.Pnl, r.LookbackPnl, r.SessionPnl, r.Multiplier,
		r.RunTimeSec, r.CountdownSec, r.SimBet,
		r.RunningSince, r.CreatedAt, r.UpdatedAt,
	)
}

func SchemeInstanceFromMaintenanceStoppedRow(r ListMaintenanceStoppedInstancesRow) SchemeInstance {
	return SchemeInstance{
		ID:               r.ID,
		DefinitionID:     r.DefinitionID,
		MemberID:         r.MemberID,
		Kind:             r.Kind,
		SchemeName:       r.SchemeName,
		LotteryCode:      r.LotteryCode,
		LotteryLabel:     r.LotteryLabel,
		Status:           r.Status,
		StatusReason:     r.StatusReason,
		Turnover:         r.Turnover,
		Pnl:              r.Pnl,
		LookbackPnl:      r.LookbackPnl,
		SessionPnl:         r.SessionPnl,
		Multiplier:       r.Multiplier,
		RunTimeSec:       r.RunTimeSec,
		CountdownSec:     r.CountdownSec,
		SimBet:           r.SimBet,
		RoundIndex:       r.RoundIndex,
		LastSettledIssue: r.LastSettledIssue,
		PickIndex:        r.PickIndex,
		CurrentPick:      r.CurrentPick,
		LastDirection:    r.LastDirection,
		StartSkipPeriod:  r.StartSkipPeriod,
		StartSkipCloseAt: r.StartSkipCloseAt,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func SchemeInstanceFromRunningToPendingRow(r UpdateSchemeInstanceStatusFromRunningToPendingRow) SchemeInstance {
	return schemeInstanceFromStatusUpdate(
		r.ID, r.DefinitionID, r.Kind, r.SchemeName, r.LotteryCode, r.LotteryLabel, r.MemberID,
		r.Status, r.StatusReason,
		r.Turnover, r.Pnl, r.LookbackPnl, r.SessionPnl, r.Multiplier,
		r.RunTimeSec, r.CountdownSec, r.SimBet,
		r.RunningSince, r.CreatedAt, r.UpdatedAt,
	)
}

func SchemeInstanceFromPausedRow(r UpdateSchemeInstanceStatusToPausedRow) SchemeInstance {
	return schemeInstanceFromStatusUpdate(
		r.ID, r.DefinitionID, r.Kind, r.SchemeName, r.LotteryCode, r.LotteryLabel, r.MemberID,
		r.Status, r.StatusReason,
		r.Turnover, r.Pnl, r.LookbackPnl, r.SessionPnl, r.Multiplier,
		r.RunTimeSec, r.CountdownSec, r.SimBet,
		r.RunningSince, r.CreatedAt, r.UpdatedAt,
	)
}

func SchemeInstanceFromRunningRow(r ListRunningSchemeInstancesRow) SchemeInstance {
	return SchemeInstance{
		ID:               r.ID,
		DefinitionID:     r.DefinitionID,
		MemberID:         r.MemberID,
		Kind:             r.Kind,
		SchemeName:       r.SchemeName,
		LotteryCode:      r.LotteryCode,
		LotteryLabel:     r.LotteryLabel,
		Status:           r.Status,
		StatusReason:     r.StatusReason,
		Turnover:         r.Turnover,
		Pnl:              r.Pnl,
		RunTimeSec:       r.RunTimeSec,
		LookbackPnl:      r.LookbackPnl,
		SessionPnl:       r.SessionPnl,
		Multiplier:       r.Multiplier,
		CountdownSec:     r.CountdownSec,
		SimBet:           r.SimBet,
		RoundIndex:       r.RoundIndex,
		LastSettledIssue: r.LastSettledIssue,
		PickIndex:        r.PickIndex,
		CurrentPick:      r.CurrentPick,
		LastDirection:    r.LastDirection,
		StartSkipPeriod:  r.StartSkipPeriod,
		StartSkipCloseAt: r.StartSkipCloseAt,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func SchemeInstanceFromAdminStatusRow(r UpdateSchemeInstanceStatusByAdminRow) SchemeInstance {
	return schemeInstanceFromStatusUpdate(
		r.ID, r.DefinitionID, r.Kind, r.SchemeName, r.LotteryCode, r.LotteryLabel, r.MemberID,
		r.Status, r.StatusReason,
		r.Turnover, r.Pnl, r.LookbackPnl, r.SessionPnl, r.Multiplier,
		r.RunTimeSec, r.CountdownSec, r.SimBet,
		pgtype.Timestamptz{}, r.CreatedAt, r.UpdatedAt,
	)
}

type SchemeDefinitionRunTypeRow struct {
	ID             string
	RunType        string
	SchemeCurrency string
}

func (q *Queries) ListSchemeDefinitionRunTypesByMember(ctx context.Context, memberID int64) ([]SchemeDefinitionRunTypeRow, error) {
	rows, err := q.db.Query(ctx, `
SELECT id,
  COALESCE(config->>'runTypeId', '') AS run_type,
  COALESCE(config->>'schemeCurrency', '') AS scheme_currency
FROM scheme_definitions
WHERE member_id = $1`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SchemeDefinitionRunTypeRow
	for rows.Next() {
		var row SchemeDefinitionRunTypeRow
		if err := rows.Scan(&row.ID, &row.RunType, &row.SchemeCurrency); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListSchemeDefinitionRunTypesByIDs 仅查当前页 definition，避免会员全量定义扫表。
func (q *Queries) ListSchemeDefinitionRunTypesByIDs(ctx context.Context, memberID int64, ids []string) ([]SchemeDefinitionRunTypeRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := q.db.Query(ctx, `
SELECT id,
  COALESCE(config->>'runTypeId', '') AS run_type,
  COALESCE(config->>'schemeCurrency', '') AS scheme_currency
FROM scheme_definitions
WHERE member_id = $1
  AND id = ANY($2::text[])`, memberID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SchemeDefinitionRunTypeRow, 0, len(ids))
	for rows.Next() {
		var row SchemeDefinitionRunTypeRow
		if err := rows.Scan(&row.ID, &row.RunType, &row.SchemeCurrency); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type CountSchemeInstancesByMemberSearchParams struct {
	MemberID int64
	RunMode  string
	Search   string
}

func (q *Queries) CountSchemeInstancesByMemberSearch(ctx context.Context, arg CountSchemeInstancesByMemberSearchParams) (int64, error) {
	const sql = `
SELECT COUNT(*)::bigint
FROM scheme_instances
WHERE member_id = $1
  AND ($2::text IS NULL OR $2::text = '' OR ($2::text = 'real' AND sim_bet = false) OR ($2::text = 'sim' AND sim_bet = true))
  AND (
    $3::text IS NULL OR $3::text = ''
    OR scheme_name ILIKE '%' || $3 || '%' ESCAPE '\'
    OR lottery_label ILIKE '%' || $3 || '%' ESCAPE '\'
    OR definition_id ILIKE '%' || $3 || '%' ESCAPE '\'
    OR id ILIKE '%' || $3 || '%' ESCAPE '\'
  )`
	var total int64
	err := q.db.QueryRow(ctx, sql, arg.MemberID, arg.RunMode, arg.Search).Scan(&total)
	return total, err
}

type ListSchemeInstancesByMemberPaginatedSearchParams struct {
	MemberID int64
	RunMode  string
	CursorAt pgtype.Timestamptz
	CursorID string
	Search   string
	Limit    int32
}

func (q *Queries) ListSchemeInstancesByMemberPaginatedSearch(
	ctx context.Context,
	arg ListSchemeInstancesByMemberPaginatedSearchParams,
) ([]SchemeInstance, error) {
	const sql = `
SELECT
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, bet_failed_detail, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    start_skip_period, start_skip_close_at,
    running_since, created_at, updated_at
FROM scheme_instances
WHERE member_id = $1
  AND ($2::text IS NULL OR $2::text = '' OR ($2::text = 'real' AND sim_bet = false) OR ($2::text = 'sim' AND sim_bet = true))
  AND (
    $3::timestamptz IS NULL
    OR updated_at < $3
    OR (updated_at = $3 AND id < $4::text)
  )
  AND (
    $5::text IS NULL OR $5::text = ''
    OR scheme_name ILIKE '%' || $5 || '%' ESCAPE '\'
    OR lottery_label ILIKE '%' || $5 || '%' ESCAPE '\'
    OR definition_id ILIKE '%' || $5 || '%' ESCAPE '\'
    OR id ILIKE '%' || $5 || '%' ESCAPE '\'
  )
ORDER BY updated_at DESC, id DESC
LIMIT $6`
	rows, err := q.db.Query(ctx, sql, arg.MemberID, arg.RunMode, arg.CursorAt, arg.CursorID, arg.Search, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]SchemeInstance, 0, arg.Limit)
	for rows.Next() {
		var f instanceDisplayFields
		if err := rows.Scan(
			&f.ID, &f.DefinitionID, &f.MemberID, &f.Kind, &f.SchemeName, &f.LotteryCode, &f.LotteryLabel,
			&f.Status, &f.StatusReason, &f.BetFailedDetail, &f.Turnover, &f.Pnl, &f.RunTimeSec,
			&f.LookbackPnl, &f.SessionPnl, &f.Multiplier, &f.CountdownSec, &f.SimBet,
			&f.StartSkipPeriod, &f.StartSkipCloseAt,
			&f.RunningSince, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, schemeInstanceFromDisplay(f))
	}
	return items, rows.Err()
}
