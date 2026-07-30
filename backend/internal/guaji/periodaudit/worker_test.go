package periodaudit

import (
	"testing"
	"time"

	"caipiao/backend/internal/lottery"
)

func newTestWorker() *Worker {
	return &Worker{
		lastAlert:  map[string]time.Time{},
		lastStatus: map[string]lottery.PeriodFamilyStatus{},
	}
}

// 映射错了会每轮都命中；不压一下会把日志刷满，反而让人忽略它。
func TestReportAlertCooldown(t *testing.T) {
	w := newTestWorker()
	t0 := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)

	w.report("hash_jisu", lottery.PeriodFamilyMismatch, "位数不同", t0)
	if got := w.lastAlert["hash_jisu"]; !got.Equal(t0) {
		t.Fatalf("首次应告警，lastAlert=%v", got)
	}

	// 冷却期内重复命中不刷新告警时间
	w.report("hash_jisu", lottery.PeriodFamilyMismatch, "位数不同", t0.Add(30*time.Minute))
	if got := w.lastAlert["hash_jisu"]; !got.Equal(t0) {
		t.Fatalf("冷却期内不应重复告警，lastAlert=%v", got)
	}

	next := t0.Add(alertCooldown + time.Minute)
	w.report("hash_jisu", lottery.PeriodFamilyMismatch, "位数不同", next)
	if got := w.lastAlert["hash_jisu"]; !got.Equal(next) {
		t.Fatalf("超出冷却期应再次告警，lastAlert=%v", got)
	}
}

// 修复后要能观测到恢复，并清掉冷却状态——否则下次再坏会被静音一小时。
func TestReportRecoveryClearsCooldown(t *testing.T) {
	w := newTestWorker()
	t0 := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)

	w.report("tron_jisu", lottery.PeriodFamilyMismatch, "位数不同", t0)
	w.report("tron_jisu", lottery.PeriodFamilyOK, "", t0.Add(time.Minute))
	if _, ok := w.lastAlert["tron_jisu"]; ok {
		t.Fatal("恢复后应清除告警冷却")
	}

	w.report("tron_jisu", lottery.PeriodFamilyMismatch, "位数不同", t0.Add(2*time.Minute))
	if got := w.lastAlert["tron_jisu"]; !got.Equal(t0.Add(2 * time.Minute)) {
		t.Fatalf("恢复后再次劣化应立即告警，lastAlert=%v", got)
	}
}

// unknown（缓存未就绪 / 期号非数字）不能当成正常，也不能当成故障。
func TestReportUnknownDoesNotAlert(t *testing.T) {
	w := newTestWorker()
	t0 := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	w.report("bnb_ffc_1m", lottery.PeriodFamilyUnknown, "缺少下注期号或开奖期号", t0)
	if len(w.lastAlert) != 0 {
		t.Fatalf("unknown 不应告警：%v", w.lastAlert)
	}
}
