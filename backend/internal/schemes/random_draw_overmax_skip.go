package schemes

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

// 随机出号持续抽不出不超过注数上限的内容时，连续跳过的上限；达此值停方案。
const randomOverMaxSkipLimit = 10

const randomOverMaxSkipMarkerPrefix = "__rd_overmax_skip:"

func isRandomOverMaxSkipMarker(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), randomOverMaxSkipMarkerPrefix)
}

func randomOverMaxSkipStreak(currentPick string) int {
	s := strings.TrimSpace(currentPick)
	if !strings.HasPrefix(s, randomOverMaxSkipMarkerPrefix) {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(s, randomOverMaxSkipMarkerPrefix)))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func formatRandomOverMaxSkipMarker(n int) string {
	if n < 1 {
		n = 1
	}
	return randomOverMaxSkipMarkerPrefix + strconv.Itoa(n)
}

func randomOverMaxUnsolvableDetail() string {
	return fmt.Sprintf("随机出号连续%d期超过注数上限", randomOverMaxSkipLimit)
}

// skipRandomDrawUnsolvable 随机出号本期无解：累计连续跳过；达上限则停方案，否则跳过本期并记 streak。
func (w *Worker) skipRandomDrawUnsolvable(ctx context.Context, inst sqlcdb.SchemeInstance, period string) error {
	return w.skipRandomDrawUnsolvableWithQ(ctx, w.q, inst, period)
}

func (w *Worker) skipRandomDrawUnsolvableWithQ(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	period string,
) error {
	if w == nil {
		return nil
	}
	streak := randomOverMaxSkipStreak(inst.CurrentPick) + 1
	if streak >= randomOverMaxSkipLimit {
		detail := randomOverMaxUnsolvableDetail()
		slog.Warn("scheme worker stopped: random draw unsolvable",
			"instanceId", inst.ID, "period", period, "streak", streak, "limit", randomOverMaxSkipLimit)
		// 第 10 期也推进游标并清空计数标记，便于手动重启后重新累计
		_ = w.skipPeriodPickWithCurrentPick(ctx, q, inst, period, RunTypeRandomDraw, "")
		w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, detail)
		return errSchemeBetStopped
	}
	slog.Info("scheme worker skip: random draw over max",
		"id", inst.ID, "period", period, "streak", streak, "limit", randomOverMaxSkipLimit)
	return w.skipPeriodPickWithCurrentPick(ctx, q, inst, period, RunTypeRandomDraw, formatRandomOverMaxSkipMarker(streak))
}

func (w *Worker) skipPeriodPickWithCurrentPick(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	period, runType, currentPick string,
) error {
	if q == nil {
		return nil
	}
	skipPeriod := strings.TrimSpace(period)
	if skipPeriod == "" {
		if p, ok := thirdPartyOpenPeriod(inst.LotteryCode); ok {
			skipPeriod = p
		}
	}
	if _, err := q.ApplySchemeInstanceBet(ctx, sqlcdb.ApplySchemeInstanceBetParams{
		ID:               inst.ID,
		CountdownSec:     w.periodCountdownForInst(inst, time.Now()),
		Turnover:         numericFromFloat(0),
		Pnl:              numericFromFloat(0),
		Multiplier:       inst.Multiplier,
		RoundIndex:       inst.RoundIndex,
		LastSettledIssue: pgtype.Text{String: skipPeriod, Valid: skipPeriod != ""},
		LookbackPnl:      numericFromFloat(0),
		PickIndex:        inst.PickIndex,
		CurrentPick:      currentPick,
		LastDirection:    inst.LastDirection,
	}); err != nil {
		return err
	}
	_ = appendPickSkipAudit(ctx, q, inst, period)
	_ = runType
	return nil
}
