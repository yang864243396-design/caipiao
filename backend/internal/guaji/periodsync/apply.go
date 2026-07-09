package periodsync

import (
	"strings"
	"time"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/lottery"
)

const workerNumPeriods = 5

// applyPeriodsListToCache 将第三方 periods 列表写入平台本地缓存（仅写入当前可投期）。
// 无开盘候选时保留已封盘快照，避免「请等待」间隙丢失 countdownEndTime。
func applyPeriodsListToCache(lotteryCode string, periods []guaji.LottPeriod, now time.Time) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return
	}
	now = now.UTC()
	if len(periods) == 0 {
		lottery.ClearPeriodsSchedule(lotteryCode)
		return
	}

	candidates := guaji.ListOpenLottPeriodCandidates(periods, lotteryCode, now)
	if len(candidates) == 0 {
		// 期间间隙：不清空已封盘缓存，展示层仍可返回 period / countdownEndTime
		return
	}

	// 当前期仍开盘：优先保持，避免多期间来回切换
	if ps, found := lottery.PeriodsScheduleFor(lotteryCode); found {
		cur := strings.TrimSpace(ps.CurrentPeriod)
		if cur != "" && !ps.CloseAt.IsZero() && now.Before(ps.CloseAt.UTC()) {
			if p, ok := matchOpenPeriodInList(candidates, lotteryCode, cur, now); ok {
				if tryApplyOpenPeriod(lotteryCode, periods, p, now) {
					return
				}
			}
		}
	}

	for _, open := range candidates {
		if tryApplyOpenPeriod(lotteryCode, periods, open, now) {
			return
		}
	}
}

func tryApplyOpenPeriod(lotteryCode string, periods []guaji.LottPeriod, open guaji.LottPeriod, now time.Time) bool {
	closeAt, closeEndTimeRaw, ok := guaji.EffectiveBetCloseAt(lotteryCode, open, now)
	if !ok || closeAt.IsZero() || !now.Before(closeAt) {
		return false
	}
	currentPeriod := strings.TrimSpace(open.Period)
	if currentPeriod == "" {
		return false
	}
	periodDurationSec := guaji.LottPeriodDurationSec(periods, lotteryCode, open, closeAt)
	openStartAt, _ := guaji.ParseGuajiPeriodTimeForLottery(lotteryCode, open.StartTime)
	return lottery.TryUpdatePeriodsScheduleFullWithDurationAt(
		lotteryCode,
		currentPeriod,
		currentPeriod,
		closeAt,
		closeAt,
		periodDurationSec,
		closeEndTimeRaw,
		openStartAt,
		now,
	)
}

// pickPeriodForCache 选取写入缓存的可投期：当前期仍有效则保持，避免同步时在多期间来回切换。
func pickPeriodForCache(lotteryCode string, periods []guaji.LottPeriod, now time.Time) (guaji.LottPeriod, bool) {
	candidates := guaji.ListOpenLottPeriodCandidates(periods, lotteryCode, now)
	if len(candidates) == 0 {
		return guaji.LottPeriod{}, false
	}
	if ps, found := lottery.PeriodsScheduleFor(lotteryCode); found {
		cur := strings.TrimSpace(ps.CurrentPeriod)
		if cur != "" && !ps.CloseAt.IsZero() && now.Before(ps.CloseAt.UTC()) {
			if p, ok := matchOpenPeriodInList(candidates, lotteryCode, cur, now); ok {
				return p, true
			}
		}
	}
	return candidates[0], true
}

func matchOpenPeriodInList(periods []guaji.LottPeriod, lotteryCode, period string, now time.Time) (guaji.LottPeriod, bool) {
	period = strings.TrimSpace(period)
	for _, p := range periods {
		if strings.TrimSpace(p.Period) != period {
			continue
		}
		if _, _, ok := guaji.EffectiveBetCloseAt(lotteryCode, p, now); ok {
			return p, true
		}
	}
	return guaji.LottPeriod{}, false
}
