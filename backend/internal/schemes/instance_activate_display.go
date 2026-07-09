package schemes

import (
	"context"
	"log/slog"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

// maybeActivateAfterStartPeriod 跳过期结束后，在列表/详情读取时切换为云端挂机。
// 仅读 DB 快照与本地 periods 缓存，禁止在列表接口里兜底拉第三方 periods。
func (s *Service) maybeActivateAfterStartPeriod(ctx context.Context, row sqlcdb.SchemeInstance, now time.Time) sqlcdb.SchemeInstance {
	if s == nil || s.q == nil {
		return row
	}
	if row.Status != "running" || row.StatusReason != StatusReasonAwaitNextBet {
		return row
	}
	def, err := s.q.GetSchemeDefinitionByID(ctx, row.DefinitionID)
	if err != nil {
		return row
	}
	if !schemeStartPeriodEnded(row, def.Config, now) {
		return row
	}
	n, err := s.q.ActivateSchemeInstanceCloud(ctx, row.ID)
	if err != nil {
		slog.Debug("scheme list activate after start period failed", "id", row.ID, "err", err)
		return row
	}
	if n == 0 {
		return row
	}
	row.StatusReason = StatusReasonCloudActive
	slog.Info("scheme activated on list read after start period ended", "id", row.ID, "skippedPeriod", startSkipPeriod(row))
	return row
}
