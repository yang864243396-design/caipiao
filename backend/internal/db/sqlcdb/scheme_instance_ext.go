package sqlcdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

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
	ID      string
	RunType string
}

func (q *Queries) ListSchemeDefinitionRunTypesByMember(ctx context.Context, memberID int64) ([]SchemeDefinitionRunTypeRow, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, COALESCE(config->>'runTypeId', '') AS run_type
FROM scheme_definitions
WHERE member_id = $1`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SchemeDefinitionRunTypeRow
	for rows.Next() {
		var row SchemeDefinitionRunTypeRow
		if err := rows.Scan(&row.ID, &row.RunType); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
