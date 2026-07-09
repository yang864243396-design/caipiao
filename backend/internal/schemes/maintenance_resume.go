package schemes

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/cloudlimits"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemelimits"
)

const maintenanceResumeBatch = 50

// ErrMaintenanceResumeBlocked 维护未解除或 §6.4 不满足，无法续投恢复。
var ErrMaintenanceResumeBlocked = errors.New("当前无法恢复运行，请确认维护已结束且未触达止损止盈")

func memberBreakPeriodStop(ctx context.Context, q *sqlcdb.Queries, memberID int64) bool {
	if q == nil || memberID <= 0 {
		return false
	}
	row, err := q.GetMemberCloudSettings(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		return false
	}
	return row.BreakPeriodStop
}

func instanceUnderMaintenance(ctx context.Context, q *sqlcdb.Queries, lotteryCode string) bool {
	if q == nil {
		return true
	}
	if maint, err := q.GetMaintenanceAdmin(ctx); err == nil && maint.Enabled {
		return true
	}
	cat, err := q.GetLotteryCatalogByCode(ctx, lotteryCode)
	if err != nil {
		return true
	}
	return sqlcdb.SaleStatusString(cat.SaleStatus) != "on_sale"
}

func canResumeAfterMaintenance(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	defConfig []byte,
	now time.Time,
) bool {
	if q == nil {
		return false
	}
	if inst.Status != "pending" || inst.StatusReason != StatusReasonMaintenance {
		return false
	}
	if instanceUnderMaintenance(ctx, q, inst.LotteryCode) {
		return false
	}
	if evaluateSchemeScheduleGate(defConfig, now) != schemeScheduleOK {
		return false
	}
	sessionPnl := numericToFloat(inst.SessionPnl)
	if reason, hit := schemelimits.Evaluate(sessionPnl, defConfig); hit {
		_ = reason
		return false
	}
	if !inst.SimBet {
		settings, err := q.GetMemberCloudSettings(ctx, inst.MemberID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		limits := cloudlimits.LimitsFromSettings(settings.TotalStopLoss, settings.TotalTakeProfit)
		if limits.StopLossYuan > 0 || limits.TakeProfitYuan > 0 {
			sum, err := q.SumMemberFormalSessionPnl(ctx, inst.MemberID)
			if err != nil {
				return false
			}
			if _, hit := cloudlimits.Evaluate(numericToFloat(sum), limits); hit {
				return false
			}
		}
	}
	return true
}
