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
