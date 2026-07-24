package schemes

import (
	"context"
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

func (s *Service) enrichInstanceForDisplay(ctx context.Context, row sqlcdb.SchemeInstance, now time.Time) Instance {
	// 开启/启停响应路径：不要同步刷新 periods（ForceRefresh 持锁，易被 worker 拖死）。
	// 倒计时字段走本地缓存即可；列表轮询会另行刷新。
	item := enrichInstanceForDisplay(ctx, s.q, row, now)
	if s != nil && s.q != nil {
		if def, err := s.q.GetSchemeDefinitionByID(ctx, row.DefinitionID); err == nil {
			item.SchemeCurrency = schemeCurrencyFromConfig(def.Config)
		} else {
			item.SchemeCurrency = normalizeSchemeCurrency("")
		}
	}
	return item
}

// enrichInstanceListItem 列表路径：不查 definition、不同步 periods，币种由批量 meta 注入。
func enrichInstanceListItem(row sqlcdb.SchemeInstance, now time.Time, schemeCurrency string) Instance {
	item := enrichInstanceForDisplay(context.Background(), nil, row, now)
	item.SchemeCurrency = normalizeSchemeCurrency(schemeCurrency)
	return item
}
