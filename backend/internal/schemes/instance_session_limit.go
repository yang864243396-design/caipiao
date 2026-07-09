package schemes

import (
	"context"
	"log/slog"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemelimits"
)

func (w *Worker) pauseRunningForSessionLimit(ctx context.Context, inst sqlcdb.SchemeInstance, defConfig []byte) bool {
	if w == nil {
		return false
	}
	sessionPnl := numericToFloat(inst.SessionPnl)
	reason, hit := schemelimits.Evaluate(sessionPnl, defConfig)
	if !hit {
		return false
	}
	limits := schemelimits.Parse(defConfig)
	detail := schemelimits.Detail(reason, sessionPnl, limits)
	if !w.pauseRunningInstance(ctx, inst, reason, detail) {
		slog.Warn("scheme worker session limit pause failed",
			"instanceId", inst.ID, "reason", reason, "sessionPnl", sessionPnl, "detail", detail)
		return false
	}
	slog.Info("scheme worker auto stopped: session limit",
		"instanceId", inst.ID, "reason", reason, "sessionPnl", sessionPnl, "detail", detail)
	return true
}
