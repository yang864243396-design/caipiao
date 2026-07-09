package schemes

import (
	"context"
	"log/slog"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

func (w *Worker) periodCountdownForInst(inst sqlcdb.SchemeInstance, now time.Time) int32 {
	if sec, ok := lottery.PeriodsDisplayCountdownSec(inst.LotteryCode, now); ok {
		return int32(sec)
	}
	return w.countdownReset
}

func (w *Worker) syncRunningCountdown(ctx context.Context, inst sqlcdb.SchemeInstance) {
	if w == nil || w.q == nil {
		return
	}
	sec, ok := lottery.PeriodsDisplayCountdownSec(inst.LotteryCode, time.Now())
	if !ok {
		return
	}
	if err := w.q.SetSchemeInstanceCountdownSec(ctx, inst.ID, int32(sec)); err != nil {
		slog.Debug("scheme worker sync countdown failed", "id", inst.ID, "err", err)
	}
}
