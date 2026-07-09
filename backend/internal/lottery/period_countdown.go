package lottery

import (
	"context"
	"strconv"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

// ParseDrawIntervalSec 解析 lottery_catalog.draw_interval（1m/3m/5m/3s/jisu 等）为秒数。
func ParseDrawIntervalSec(raw string) int {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return 0
	}
	if raw == "jisu" {
		return 60
	}
	if strings.HasSuffix(raw, "s") {
		n, err := strconv.Atoi(strings.TrimSuffix(raw, "s"))
		if err == nil && n > 0 {
			return n
		}
	}
	if strings.HasSuffix(raw, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(raw, "m"))
		if err == nil && n > 0 {
			return n * 60
		}
	}
	return 0
}

// PeriodCountdownSec 计算距下一开奖（投注截止）剩余秒数。
func PeriodCountdownSec(now time.Time, lastDrawnAt time.Time, intervalSec int) int {
	if intervalSec <= 0 {
		return 0
	}
	if lastDrawnAt.IsZero() {
		unix := now.Unix()
		if unix < 0 {
			return 0
		}
		mod := int(unix) % intervalSec
		if mod == 0 {
			return 0
		}
		return intervalSec - mod
	}
	next := lastDrawnAt.Add(time.Duration(intervalSec) * time.Second)
	rem := int(next.Sub(now).Round(time.Second).Seconds())
	if rem >= 0 {
		return rem
	}
	overdue := int(now.Sub(lastDrawnAt).Seconds())
	if overdue <= 0 {
		return 0
	}
	mod := overdue % intervalSec
	if mod == 0 {
		return 0
	}
	return intervalSec - mod
}

// DrawIntervalSecForLottery 读取彩种开奖间隔秒数。
func DrawIntervalSecForLottery(ctx context.Context, q *sqlcdb.Queries, lotteryCode string) int {
	if q == nil {
		return 0
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return 0
	}
	cat, err := q.GetLotteryCatalogByCode(ctx, lotteryCode)
	if err != nil || !cat.DrawInterval.Valid {
		return 0
	}
	return ParseDrawIntervalSec(cat.DrawInterval.String)
}

// PeriodCountdownBundleForLottery 返回第三方 periods 单期时长与距封盘剩余秒数（仅认 periods end_time，禁止墙钟回退）。
func PeriodCountdownBundleForLottery(ctx context.Context, q *sqlcdb.Queries, lotteryCode string, now time.Time) (intervalSec, countdownSec int, err error) {
	_ = ctx
	_ = q
	now = now.UTC()
	ps, ok := PeriodsScheduleFor(lotteryCode)
	if !ok || !PeriodsScheduleFresh(lotteryCode, periodsScheduleMaxAge, now) || ps.CloseAt.IsZero() {
		return 0, 0, nil
	}
	intervalSec = ps.PeriodDurationSec
	if rem, ok := periodsCountdownFromSchedule(ps, now); ok {
		return intervalSec, rem, nil
	}
	return intervalSec, 0, nil
}

func periodsCountdownFromSchedule(ps PeriodsSchedule, now time.Time) (int, bool) {
	return BetCountdownSecFromSchedule(ps, now)
}

// PeriodCountdownSecForLottery 距第三方 periods 封盘（end_time）剩余秒数；无缓存时返回 0。
func PeriodCountdownSecForLottery(ctx context.Context, q *sqlcdb.Queries, lotteryCode string, now time.Time) (int, error) {
	_, countdown, err := PeriodCountdownBundleForLottery(ctx, q, lotteryCode, now)
	return countdown, err
}
