package schemes

import (
	"errors"
	"testing"
	"time"
)

func TestShanghaiTodayDate(t *testing.T) {
	// 2026-07-21 23:30 UTC = 2026-07-22 07:30 CST → calendar day 22
	utc := time.Date(2026, 7, 21, 23, 30, 0, 0, time.UTC)
	got := shanghaiTodayDate(utc)
	if got.Format("2006-01-02") != "2026-07-22" {
		t.Fatalf("got %s want 2026-07-22", got.Format("2006-01-02"))
	}
	// 2026-07-21 15:59 UTC = 2026-07-21 23:59 CST → still 21
	utc2 := time.Date(2026, 7, 21, 15, 59, 0, 0, time.UTC)
	got2 := shanghaiTodayDate(utc2)
	if got2.Format("2006-01-02") != "2026-07-21" {
		t.Fatalf("got %s want 2026-07-21", got2.Format("2006-01-02"))
	}
}

func TestSimQuotaErrorMessages(t *testing.T) {
	if !errors.Is(ErrSimSchemeConcurrentLimit, ErrSimSchemeConcurrentLimit) {
		t.Fatal("concurrent sentinel")
	}
	if ErrSimSchemeConcurrentLimit.Error() != "最多可同时开启5个模拟测试方案，如需开启新方案，请先关闭一个已开启的方案" {
		t.Fatalf("concurrent msg=%q", ErrSimSchemeConcurrentLimit.Error())
	}
	if maxSimSchemeConcurrent != 5 || maxSimSchemeDailyStarts != 5 {
		t.Fatalf("limits %d/%d", maxSimSchemeConcurrent, maxSimSchemeDailyStarts)
	}
}
