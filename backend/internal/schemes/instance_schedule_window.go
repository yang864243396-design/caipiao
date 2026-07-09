package schemes

import (
	"context"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

type schemeScheduleGate int

const (
	schemeScheduleOK schemeScheduleGate = iota
	schemeScheduleBeforeStart
	schemeSchedulePastEnd
)

// evaluateSchemeScheduleGate 判断当前时刻是否在方案配置的运行时段内。
func evaluateSchemeScheduleGate(cfgBytes []byte, now time.Time) schemeScheduleGate {
	if schemeConfigEndTimeReached(cfgBytes, now) {
		return schemeSchedulePastEnd
	}
	if schemeConfigStartTimeNotReached(cfgBytes, now) {
		return schemeScheduleBeforeStart
	}
	return schemeScheduleOK
}

// gateScheduleBeforeBet 下注前校验开启/结束时间：未到开始则跳过；已过结束则停投。
func (w *Worker) gateScheduleBeforeBet(ctx context.Context, inst sqlcdb.SchemeInstance, cfgBytes []byte) schemeScheduleGate {
	if w == nil {
		return schemeScheduleOK
	}
	gate := evaluateSchemeScheduleGate(cfgBytes, time.Now())
	if gate == schemeSchedulePastEnd {
		w.pauseRunningInstance(ctx, inst, StatusReasonEndTime, "")
	}
	return gate
}
