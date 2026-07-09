package schemes

import (
	"context"
	"log/slog"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

func (w *Worker) tickMaintenanceResume(ctx context.Context) {
	if w == nil || w.q == nil {
		return
	}
	rows, err := w.q.ListMaintenanceStoppedInstances(ctx, int32(maintenanceResumeBatch))
	if err != nil {
		slog.Warn("scheme worker list maintenance stopped failed", "err", err)
		return
	}
	if len(rows) == 0 {
		return
	}

	now := time.Now()
	breakStopCache := make(map[int64]bool, 8)
	for _, row := range rows {
		inst := sqlcdb.SchemeInstanceFromMaintenanceStoppedRow(row)
		if instanceUnderMaintenance(ctx, w.q, inst.LotteryCode) {
			continue
		}
		stop, ok := breakStopCache[inst.MemberID]
		if !ok {
			stop = memberBreakPeriodStop(ctx, w.q, inst.MemberID)
			breakStopCache[inst.MemberID] = stop
		}
		if stop {
			continue
		}
		def, err := w.loadDefinitionForInstance(ctx, inst)
		if err != nil {
			if !isDefinitionNotFound(err) {
				slog.Warn("scheme worker maintenance resume load def failed", "id", inst.ID, "err", err)
			}
			continue
		}
		if !canResumeAfterMaintenance(ctx, w.q, inst, def.Config, now) {
			continue
		}
		w.tryResumeAfterMaintenance(ctx, inst)
	}
}

func (w *Worker) tryResumeAfterMaintenance(ctx context.Context, inst sqlcdb.SchemeInstance) bool {
	row, err := w.q.ResumeSchemeInstanceAfterMaintenance(ctx, sqlcdb.ResumeSchemeInstanceAfterMaintenanceParams{
		ID:       inst.ID,
		MemberID: inst.MemberID,
	})
	if err != nil {
		slog.Warn("scheme worker maintenance resume failed", "id", inst.ID, "err", err)
		return false
	}
	w.notifySchemeInstance(ctx, inst.MemberID, inst.ID, runModeFromSimBet(inst.SimBet), "running", StatusReasonAwaitNextBet)
	slog.Info("scheme worker maintenance auto resumed",
		"instanceId", inst.ID,
		"memberId", inst.MemberID,
		"lottery", row.LotteryCode,
		"sessionPnl", numericToFloat(row.SessionPnl),
	)
	return true
}

// TickMaintenanceResume 供维护/彩种状态变更后主动触发一次恢复扫描。
func (w *Worker) TickMaintenanceResume(ctx context.Context) {
	w.tickMaintenanceResume(ctx)
}
