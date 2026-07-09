package lottery

import (
	"strings"
	"time"
)

const PeriodsDisplayWaitingLabel = "请等待"

// PeriodsDisplayCountdown 系统统一展示倒计时（与方案启停无关，仅认 periods 第三方 end_time）。
type PeriodsDisplayCountdown struct {
	Period         string
	Sec            int
	EndTimeRaw     string
	CloseAtRFC3339 string
	WindowSec      int
	WaitingLabel   string
}

// BuildPeriodsDisplayCountdown 返回当前彩种距下一期封盘的统一展示倒计时。
// 封盘后的「请等待」间隙仍返回缓存中的 period / countdownEndTime，供客户端本地重算与轮询合并。
func BuildPeriodsDisplayCountdown(lotteryCode string, now time.Time) PeriodsDisplayCountdown {
	out := PeriodsDisplayCountdown{}
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		out.WaitingLabel = PeriodsDisplayWaitingLabel
		return out
	}
	now = now.UTC()

	ps, hasPS := PeriodsScheduleFor(lotteryCode)
	if hasPS {
		if period := strings.TrimSpace(ps.CurrentPeriod); period != "" {
			out.Period = period
		}
		if ps.PeriodDurationSec > 0 {
			out.WindowSec = ps.PeriodDurationSec
		}
		if raw := strings.TrimSpace(ps.CloseEndTimeRaw); raw != "" {
			out.EndTimeRaw = raw
		} else if !ps.CloseAt.IsZero() {
			out.EndTimeRaw = FormatUTCWallClock(ps.CloseAt)
		}
		if !ps.CloseAt.IsZero() {
			out.CloseAtRFC3339 = ps.CloseAt.UTC().Format(time.RFC3339)
		}
	}

	ca, open := PeriodsDisplayCloseAt(lotteryCode, now)
	if !open {
		out.Sec = 0
		out.WaitingLabel = PeriodsDisplayWaitingLabel
		return out
	}

	if hasPS {
		if sec, ok := BetCountdownSecFromSchedule(ps, now); ok {
			out.Sec = sec
		} else {
			out.Sec = countdownSecUntilCloseAt(ca, now)
		}
	} else {
		out.Sec = countdownSecUntilCloseAt(ca, now)
	}

	if out.EndTimeRaw == "" {
		if raw, ok := PeriodsDisplayCloseEndTimeRaw(lotteryCode, now); ok {
			out.EndTimeRaw = raw
		} else {
			out.EndTimeRaw = FormatUTCWallClock(ca)
		}
	}
	if out.CloseAtRFC3339 == "" {
		out.CloseAtRFC3339 = ca.UTC().Format(time.RFC3339)
	}

	if out.Sec <= 0 {
		out.WaitingLabel = PeriodsDisplayWaitingLabel
	}
	return out
}

func countdownSecUntilCloseAt(closeAt time.Time, now time.Time) int {
	if closeAt.IsZero() {
		return 0
	}
	rem := int(closeAt.Sub(now.UTC()).Round(time.Second).Seconds())
	if rem < 0 {
		return 0
	}
	return rem
}
