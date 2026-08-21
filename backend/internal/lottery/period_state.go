package lottery

import (
	"strings"
	"sync"
	"time"
)

// PeriodState 来自第三方开奖 WS 的当期/下期快照（drawsync 写入；供投注期号，不参与展示倒计时）。
type PeriodState struct {
	CurrentIssue string
	NextIssue    string
	CloseAt      time.Time // 封盘时刻（仅内部参考；展示倒计时用墙钟取模）
	UpdatedAt    time.Time
	IntervalSec  int
}

var periodState sync.Map // lotteryCode -> PeriodState

// UpdatePeriodState 在开奖 WS 入库时更新彩种期号与封盘时刻。
func UpdatePeriodState(lotteryCode, currentIssue, nextIssue string, drawnAt time.Time, intervalSec int) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" || intervalSec <= 0 {
		return
	}
	if drawnAt.IsZero() {
		drawnAt = time.Now().UTC()
	} else {
		drawnAt = drawnAt.UTC()
	}
	closeAt := drawnAt.Add(time.Duration(intervalSec) * time.Second)
	periodState.Store(lotteryCode, PeriodState{
		CurrentIssue: strings.TrimSpace(currentIssue),
		NextIssue:    strings.TrimSpace(nextIssue),
		CloseAt:      closeAt,
		UpdatedAt:    time.Now().UTC(),
		IntervalSec:  intervalSec,
	})
}

// PeriodStateFor 读取彩种最新期号快照。
func PeriodStateFor(lotteryCode string) (PeriodState, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return PeriodState{}, false
	}
	v, ok := periodState.Load(lotteryCode)
	if !ok {
		return PeriodState{}, false
	}
	st, ok := v.(PeriodState)
	return st, ok
}

// FreshShortPeriodWSBetTarget returns the provider websocket's next_periods
// value for formal short-period betting. The periods REST feed may expose a
// future candidate before the placement endpoint switches periods, so it must
// not override a fresh websocket boundary for these lotteries.
func FreshShortPeriodWSBetTarget(lotteryCode, sourcePeriod string, now time.Time) (PeriodState, bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	switch lotteryCode {
	case "tron_ffc_3s", "tron_ffc_6s", "tron_ffc_15s":
	default:
		return PeriodState{}, false
	}
	state, ok := PeriodStateFor(lotteryCode)
	if !ok || state.IntervalSec <= 0 || state.IntervalSec > 15 {
		return PeriodState{}, false
	}
	now = now.UTC()
	updatedAt := state.UpdatedAt.UTC()
	closeAt := state.CloseAt.UTC()
	maxAge := time.Duration(state.IntervalSec) * time.Second
	if updatedAt.IsZero() || now.Before(updatedAt) || now.Sub(updatedAt) > maxAge || closeAt.IsZero() || !now.Before(closeAt) {
		return PeriodState{}, false
	}
	state.CurrentIssue = strings.TrimSpace(state.CurrentIssue)
	state.NextIssue = strings.TrimSpace(state.NextIssue)
	sourcePeriod = strings.TrimSpace(sourcePeriod)
	if state.NextIssue == "" || state.NextIssue == sourcePeriod {
		return PeriodState{}, false
	}
	if sourcePeriod != "" && state.CurrentIssue != sourcePeriod {
		return PeriodState{}, false
	}
	return state, true
}

// BetCloseSec 距封盘多少秒内允许 Worker 尝试投注。
func BetCloseSec(intervalSec int) int {
	if intervalSec <= 0 {
		return 1
	}
	switch {
	case intervalSec <= 6:
		return 1
	case intervalSec <= 15:
		return 2
	case intervalSec <= 60:
		return 8
	default:
		buf := intervalSec / 6
		if buf > 15 {
			buf = 15
		}
		if buf < 5 {
			buf = 5
		}
		return buf
	}
}

type countdownTick struct {
	sec   int
	issue string
	at    time.Time
}

var countdownSmooth sync.Map // lotteryCode -> countdownTick

// smoothCountdown 保证同一期内展示倒计时单调递减（允许新期重置为更大值）。
func smoothCountdown(lotteryCode string, now time.Time, raw int) int {
	if raw < 0 {
		raw = 0
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return raw
	}
	now = now.UTC()

	issue := periodIssueForSmooth(lotteryCode)

	if v, ok := countdownSmooth.Load(lotteryCode); ok {
		prev := v.(countdownTick)
		if issue != "" && prev.issue != "" && issue != prev.issue {
			countdownSmooth.Store(lotteryCode, countdownTick{sec: raw, issue: issue, at: now})
			return raw
		}
		elapsed := int(now.Sub(prev.at).Round(time.Second).Seconds())
		if elapsed < 0 {
			elapsed = 0
		}
		floor := prev.sec - elapsed
		if floor < 0 {
			floor = 0
		}
		if raw > floor {
			raw = floor
		}
	}

	countdownSmooth.Store(lotteryCode, countdownTick{sec: raw, issue: issue, at: now})
	return raw
}
