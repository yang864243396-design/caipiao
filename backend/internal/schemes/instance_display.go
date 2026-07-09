package schemes

import (
	"context"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

func displayRunTimeSec(row sqlcdb.SchemeInstance, now time.Time) int {
	base := int(row.RunTimeSec)
	if row.Status != "running" || !row.RunningSince.Valid {
		return base
	}
	elapsed := int(now.Sub(row.RunningSince.Time).Round(time.Second).Seconds())
	if elapsed < 0 {
		elapsed = 0
	}
	return base + elapsed
}

func applyUnifiedPeriodCountdownFields(item *Instance, lotteryCode string, now time.Time) {
	cd := lottery.BuildPeriodsDisplayCountdown(lotteryCode, now)
	item.CountdownSec = cd.Sec
	item.CountdownPeriod = cd.Period
	item.CountdownEndTime = cd.EndTimeRaw
	item.CountdownCloseAt = cd.CloseAtRFC3339
	item.CountdownWindowSec = cd.WindowSec
	item.CountdownLabel = cd.WaitingLabel
}

func enrichInstanceForDisplay(_ context.Context, _ *sqlcdb.Queries, row sqlcdb.SchemeInstance, now time.Time) Instance {
	item := mapInstanceRow(row)
	item.RunTimeSec = displayRunTimeSec(row, now)
	applyUnifiedPeriodCountdownFields(&item, row.LotteryCode, now.UTC())
	return item
}

// ensurePeriodsFreshForDisplay 展示前刷新 periods 缓存，避免单 instance 接口与列表倒计时来源不一致。
func (s *Service) ensurePeriodsFreshForDisplay(ctx context.Context, lotteryCode string) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" || s.periodSync == nil {
		return
	}
	_ = s.periodSync.EnsureFreshIfStale(ctx, lotteryCode)
	now := time.Now()
	if _, ok := lottery.PeriodsDisplayCloseAt(lotteryCode, now); ok {
		return
	}
	if lottery.PeriodsScheduleNeedsRefresh(lotteryCode, now) {
		_ = s.periodSync.ForceRefresh(ctx, lotteryCode)
	}
}

func (s *Service) enrichInstanceForDisplay(ctx context.Context, row sqlcdb.SchemeInstance, now time.Time) Instance {
	s.ensurePeriodsFreshForDisplay(ctx, row.LotteryCode)
	return enrichInstanceForDisplay(ctx, s.q, row, now)
}
