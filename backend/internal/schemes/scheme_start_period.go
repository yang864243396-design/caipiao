package schemes

import (
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

func startSkipPeriod(inst sqlcdb.SchemeInstance) string {
	if inst.StartSkipPeriod.Valid {
		if p := strings.TrimSpace(inst.StartSkipPeriod.String); p != "" {
			return p
		}
	}
	if inst.LastSettledIssue.Valid {
		return strings.TrimSpace(inst.LastSettledIssue.String)
	}
	return ""
}

func skipPeriodCloseAt(inst sqlcdb.SchemeInstance, skip string) (time.Time, bool) {
	if inst.StartSkipCloseAt.Valid && !inst.StartSkipCloseAt.Time.IsZero() {
		return inst.StartSkipCloseAt.Time.UTC(), true
	}
	if skip == "" {
		return time.Time{}, false
	}
	ps, ok := lottery.PeriodsScheduleFor(inst.LotteryCode)
	if !ok {
		return time.Time{}, false
	}
	if strings.TrimSpace(ps.StartSkipPeriod) == skip && !ps.StartSkipCloseAt.IsZero() {
		return ps.StartSkipCloseAt.UTC(), true
	}
	if strings.TrimSpace(ps.CurrentPeriod) == skip && !ps.CloseAt.IsZero() {
		return ps.CloseAt.UTC(), true
	}
	return time.Time{}, false
}

// schemeStartPeriodEnded 开启时跳过的最近一期是否已封盘（可切换为云端挂机并首投）。
// 优先以跳过期封盘时刻为准；若封盘时刻不可解析，但当前开放期已严格晚于跳过期，
// 视为跳过期已结束（避免永久卡在 await_next_bet 导致「能运行但不下注」）。
// 若从未写入跳过期（如 periods 缓存不可用），且当前已在方案运行时段内，视为可激活。
func schemeStartPeriodEnded(inst sqlcdb.SchemeInstance, cfgBytes []byte, now time.Time) bool {
	now = now.UTC()
	skipped := startSkipPeriod(inst)
	if skipped == "" {
		return evaluateSchemeScheduleGate(cfgBytes, now) == schemeScheduleOK
	}
	closeAt, ok := skipPeriodCloseAt(inst, skipped)
	if ok && !closeAt.IsZero() {
		return !now.Before(closeAt)
	}
	if cur, ok := lottery.StrictOpenIssueForGuajiBet(inst.LotteryCode); ok && issueAfter(cur, skipped) {
		return evaluateSchemeScheduleGate(cfgBytes, now) == schemeScheduleOK
	}
	return false
}
