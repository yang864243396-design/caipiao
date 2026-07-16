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

// ensurePeriodsFreshForDisplay 展示前尽量刷新 periods；锁占用或第三方慢时立即放弃，只用本地缓存。
func (s *Service) ensurePeriodsFreshForDisplay(ctx context.Context, lotteryCode string) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" || s.periodSync == nil {
		return
	}
	refreshCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	_ = s.periodSync.EnsureFreshIfStale(refreshCtx, lotteryCode)
}

func (s *Service) enrichInstanceForDisplay(ctx context.Context, row sqlcdb.SchemeInstance, now time.Time) Instance {
	// 开启/启停响应路径：不要同步刷新 periods（ForceRefresh 持锁，易被 worker 拖死）。
	// 倒计时字段走本地缓存即可；列表轮询会另行刷新。
	_ = ctx
	return enrichInstanceForDisplay(ctx, s.q, row, now)
}
