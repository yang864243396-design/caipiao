package schemes

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
)

func (w *Worker) checkAutoPause(ctx context.Context, inst sqlcdb.SchemeInstance, def sqlcdb.GetSchemeDefinitionByIDRow) (reason string, shouldPause bool) {
	if maint, err := w.q.GetMaintenanceAdmin(ctx); err == nil && maint.Enabled {
		return StatusReasonMaintenance, true
	}
	if cat, err := w.q.GetLotteryCatalogByCode(ctx, inst.LotteryCode); err == nil && cat.SaleStatus != "on_sale" {
		return StatusReasonMaintenance, true
	}
	return "", false
}

// refuseRandomDrawMaxUnitsPause 随机出号遇注数上限：禁止 bet_failed 停方案（应重抽/跳过本期）。
func refuseRandomDrawMaxUnitsPause(ctx context.Context, q *sqlcdb.Queries, instanceID, reason, detail string) bool {
	if q == nil || reason != StatusReasonBetFailed {
		return false
	}
	if !strings.Contains(detail, errBetUnitsExceeded.Error()) {
		return false
	}
	inst, err := q.GetSchemeInstanceFull(ctx, instanceID)
	if err != nil {
		return false
	}
	def, err := q.GetSchemeDefinitionByID(ctx, inst.DefinitionID)
	if err != nil {
		return false
	}
	cfg := parseSchemeConfig(inst.Kind, def.Config, 0, 0)
	return cfg.RunTypeID == RunTypeRandomDraw
}

func (w *Worker) pauseRunningInstance(ctx context.Context, inst sqlcdb.SchemeInstance, reason, detail string) bool {
	detail = normalizeBetFailedDetail(detail)
	if refuseRandomDrawMaxUnitsPause(ctx, w.q, inst.ID, reason, detail) {
		slog.Info("scheme worker refuse pause: random_draw over max",
			"instanceId", inst.ID, "reason", reason, "detail", detail)
		return false
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		slog.Warn("scheme worker pause tx failed", "id", inst.ID, "err", err)
		return false
	}
	defer tx.Rollback(ctx)

	qtx := w.q.WithTx(tx)
	rows, err := qtx.PauseSchemeInstanceByWorker(ctx, sqlcdb.PauseSchemeInstanceByWorkerParams{
		ID:           inst.ID,
		StatusReason: reason,
		Column3:      detail,
	})
	if err != nil {
		slog.Warn("scheme worker pause failed", "id", inst.ID, "reason", reason, "err", err)
		return false
	}
	if rows == 0 {
		slog.Warn("scheme worker pause skipped: instance not running", "id", inst.ID, "reason", reason)
		return false
	}
	if err := tx.Commit(ctx); err != nil {
		slog.Warn("scheme worker pause commit failed", "id", inst.ID, "err", err)
		return false
	}
	w.notifySchemeInstance(ctx, inst.MemberID, inst.ID, runModeFromSimBet(inst.SimBet), "pending", reason)
	slog.Info("scheme worker auto paused", "instanceId", inst.ID, "reason", reason)
	return true
}

func (w *Worker) loadDefinitionForInstance(ctx context.Context, inst sqlcdb.SchemeInstance) (sqlcdb.GetSchemeDefinitionByIDRow, error) {
	return w.q.GetSchemeDefinitionByID(ctx, inst.DefinitionID)
}

func isDefinitionNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
