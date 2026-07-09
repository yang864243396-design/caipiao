package cloudlimits

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/ws"
)

// PauseAllRunningIfHit 汇总会员正式盘 session_pnl，触达总止损/止盈则停投全部 running 方案。
func PauseAllRunningIfHit(
	ctx context.Context,
	q *sqlcdb.Queries,
	hub *ws.Hub,
	memberID int64,
) bool {
	if q == nil || memberID <= 0 {
		return false
	}
	settings, err := q.GetMemberCloudSettings(ctx, memberID)
	if err != nil {
		if isNoRows(err) {
			return false
		}
		slog.Warn("cloud limits load settings failed", "memberId", memberID, "err", err)
		return false
	}
	limits := LimitsFromSettings(settings.TotalStopLoss, settings.TotalTakeProfit)
	if limits.StopLossYuan <= 0 && limits.TakeProfitYuan <= 0 {
		return false
	}

	sum, err := q.SumMemberFormalSessionPnl(ctx, memberID)
	if err != nil {
		slog.Warn("cloud limits sum session pnl failed", "memberId", memberID, "err", err)
		return false
	}
	totalPnl := numericToFloat(sum)
	reason, hit := Evaluate(totalPnl, limits)
	if !hit {
		return false
	}

	rows, err := q.PauseAllRunningInstancesByMember(ctx, sqlcdb.PauseAllRunningInstancesByMemberParams{
		MemberID:     memberID,
		StatusReason: reason,
	})
	if err != nil {
		slog.Warn("cloud limits pause all running failed", "memberId", memberID, "reason", reason, "err", err)
		return false
	}
	if len(rows) == 0 {
		return false
	}

	detail := Detail(reason, totalPnl, limits)
	account := ""
	if hub != nil {
		if acct, aerr := q.GetMemberAccountByID(ctx, memberID); aerr == nil {
			account = acct
		}
	}
	for _, inst := range rows {
		if hub != nil && account != "" {
		runMode := "real"
		if inst.SimBet {
			runMode = "sim"
		}
		ws.PublishSchemeInstance(hub, account, ws.SchemeInstancePayload{
			InstanceID: inst.ID,
			RunMode:    runMode,
			SimBet:     inst.SimBet,
				Status:     "pending",
				Reason:     reason,
				Hint:       "refresh_running_list",
			})
		}
	}
	slog.Info("cloud auto stopped all running",
		"memberId", memberID, "reason", reason, "totalPnl", totalPnl, "count", len(rows), "detail", detail)
	return true
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
