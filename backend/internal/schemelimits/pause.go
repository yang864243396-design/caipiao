package schemelimits

import (
	"context"
	"log/slog"
	"math"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeevents"
	"caipiao/backend/internal/ws"
)

func numericToFloat(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return math.Round(f.Float64*100) / 100
}

// PauseRunningInstanceIfHit 派奖后检查 session_pnl，触达方案止损/止盈则停投。
func PauseRunningInstanceIfHit(
	ctx context.Context,
	q *sqlcdb.Queries,
	_ *ws.Hub,
	inst sqlcdb.SchemeInstance,
	defConfig []byte,
	marker schemeevents.Marker,
) bool {
	if q == nil || inst.Status != "running" {
		return false
	}
	sessionPnl := numericToFloat(inst.SessionPnl)
	reason, hit := Evaluate(sessionPnl, defConfig)
	if !hit {
		return false
	}
	limits := Parse(defConfig)
	detail := Detail(reason, sessionPnl, limits)
	rows, err := q.PauseSchemeInstanceByWorker(ctx, sqlcdb.PauseSchemeInstanceByWorkerParams{
		ID:           inst.ID,
		StatusReason: reason,
		Column3:      "",
	})
	if err != nil {
		slog.Warn("scheme session limit pause failed",
			"instanceId", inst.ID, "reason", reason, "sessionPnl", sessionPnl, "err", err)
		return false
	}
	if rows == 0 {
		return false
	}
	if marker != nil && inst.MemberID > 0 && inst.ID != "" {
		marker.MarkScheme(inst.MemberID, inst.ID)
	}
	slog.Info("scheme auto stopped: session limit",
		"instanceId", inst.ID, "reason", reason, "sessionPnl", sessionPnl, "detail", detail)
	return true
}
