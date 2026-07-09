package guaji

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseGuajiPeriodTimeUTCWallClock(t *testing.T) {
	tm, err := ParseGuajiPeriodTime("2026-06-10 15:04:05")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 10, 15, 4, 5, 0, time.UTC)
	if !tm.Equal(want) {
		t.Fatalf("got %v want %v", tm, want)
	}
}

func TestParseGuajiPeriodTimeForLottery_tronBeijing(t *testing.T) {
	tm, err := ParseGuajiPeriodTimeForLottery("tron_ffc_1m", "2026-06-21 14:40:00")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 21, 6, 40, 0, 0, time.UTC)
	if !tm.Equal(want) {
		t.Fatalf("got %v want %v", tm, want)
	}
}

func TestParseGuajiPeriodTimeForLottery_hashUtc(t *testing.T) {
	tm, err := ParseGuajiPeriodTimeForLottery("hash_ffc_1m", "2026-06-21 06:40:00")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 21, 6, 40, 0, 0, time.UTC)
	if !tm.Equal(want) {
		t.Fatalf("got %v want %v", tm, want)
	}
}

func TestPickOpenLottPeriod(t *testing.T) {
	now := time.Date(2026, 6, 10, 15, 0, 0, 0, time.UTC)
	periods := []LottPeriod{
		{Period: "p1", StartTime: "2026-06-10 14:58:59", EndTime: "2026-06-10 14:59:59"},
		{Period: "p2", StartTime: "2026-06-10 14:59:59", EndTime: "2026-06-10 15:01:00"},
		{Period: "p3", StartTime: "2026-06-10 15:01:00", EndTime: "2026-06-10 15:02:00"},
	}
	p, closeAt, ok := PickOpenLottPeriod(periods, "hash_ffc_1m", now)
	if !ok || p.Period != "p2" {
		t.Fatalf("pick=%+v ok=%v", p, ok)
	}
	wantClose := time.Date(2026, 6, 10, 15, 1, 0, 0, time.UTC)
	if !closeAt.Equal(wantClose) {
		t.Fatalf("closeAt=%v want %v", closeAt, wantClose)
	}
}

func TestPickOpenLottPeriod_hashUtcWallClock(t *testing.T) {
	now := time.Date(2026, 6, 21, 5, 17, 55, 0, time.UTC)
	periods := []LottPeriod{
		{Period: "1013963900173", StartTime: "2026-06-21 05:17:00", EndTime: "2026-06-21 05:18:00"},
	}
	p, closeAt, ok := PickOpenLottPeriod(periods, "hash_ffc_1m", now)
	if !ok || p.Period != "1013963900173" {
		t.Fatalf("pick=%+v ok=%v", p, ok)
	}
	wantClose := time.Date(2026, 6, 21, 5, 18, 0, 0, time.UTC)
	if !closeAt.Equal(wantClose) {
		t.Fatalf("closeAt=%v want %v", closeAt, wantClose)
	}
}

func TestPickOpenLottPeriod_tronBeijingWallClock(t *testing.T) {
	// 第三方 tron game_id=27 返回北京时间墙钟 14:40，对应 UTC 06:40
	now := time.Date(2026, 6, 21, 6, 39, 50, 0, time.UTC)
	periods := []LottPeriod{
		{Period: "111202606210880", StartTime: "2026-06-21 14:40:00", EndTime: "2026-06-21 14:41:00"},
	}
	p, closeAt, ok := PickOpenLottPeriod(periods, "tron_ffc_1m", now)
	if !ok || p.Period != "111202606210880" {
		t.Fatalf("pick=%+v ok=%v", p, ok)
	}
	wantClose := time.Date(2026, 6, 21, 6, 41, 0, 0, time.UTC)
	if !closeAt.Equal(wantClose) {
		t.Fatalf("closeAt=%v want %v", closeAt, wantClose)
	}
}

// 第三方 periods 列表常不含「当前已开盘」期；12:23:49 应封盘 12:24:00，而非下一期 end 12:25:00。
func TestPickOpenLottPeriod_tron_missingCurrentPeriodUsesNextStart(t *testing.T) {
	now := time.Date(2026, 6, 24, 4, 23, 49, 0, time.UTC) // 北京 12:23:49
	periods := []LottPeriod{
		{Period: "111202606240744", StartTime: "2026-06-24 12:24:00", EndTime: "2026-06-24 12:25:00"},
		{Period: "111202606240745", StartTime: "2026-06-24 12:25:00", EndTime: "2026-06-24 12:26:00"},
	}
	p, closeAt, ok := PickOpenLottPeriod(periods, "tron_ffc_1m", now)
	if !ok || p.Period != "111202606240744" {
		t.Fatalf("pick=%+v ok=%v", p, ok)
	}
	wantClose := time.Date(2026, 6, 24, 4, 24, 0, 0, time.UTC)
	if !closeAt.Equal(wantClose) {
		t.Fatalf("closeAt=%v want %v (next period start)", closeAt, wantClose)
	}
	_, raw, ok := EffectiveBetCloseAt("tron_ffc_1m", p, now)
	if !ok || raw != "2026-06-24 12:24:00" {
		t.Fatalf("raw=%q ok=%v want 12:24:00", raw, ok)
	}
}

func TestPickOpenLottPeriod_prefersEarliestIssueNumber(t *testing.T) {
	now := time.Date(2026, 6, 20, 11, 14, 10, 0, time.UTC)
	periods := []LottPeriod{
		{Period: "2606201116", StartTime: "2026-06-20 11:15:00", EndTime: "2026-06-20 11:16:00"},
		{Period: "2606201115", StartTime: "2026-06-20 11:14:00", EndTime: "2026-06-20 11:15:00"},
	}
	p, closeAt, ok := PickOpenLottPeriod(periods, "hash_ffc_1m", now)
	if !ok || p.Period != "2606201115" {
		t.Fatalf("pick=%+v ok=%v", p, ok)
	}
	wantClose := time.Date(2026, 6, 20, 11, 15, 0, 0, time.UTC)
	if !closeAt.Equal(wantClose) {
		t.Fatalf("closeAt=%v want %v", closeAt, wantClose)
	}
	rem := int(closeAt.Sub(now).Seconds())
	if rem < 48 || rem > 50 {
		t.Fatalf("rem=%d want ~50", rem)
	}
}

func TestComparePeriodNumber(t *testing.T) {
	if comparePeriodNumber("2606201115", "2606201116") >= 0 {
		t.Fatal("expected 1115 < 1116")
	}
	if comparePeriodNumber("115001", "115002") >= 0 {
		t.Fatal("expected 115001 < 115002")
	}
}

func TestPickOpenLottPeriod_skipsNotStarted(t *testing.T) {
	now := time.Date(2026, 6, 21, 5, 30, 10, 0, time.UTC)
	periods := []LottPeriod{
		{Period: "174", StartTime: "2026-06-21 05:30:39", EndTime: "2026-06-21 05:31:39"},
	}
	p, closeAt, ok := PickOpenLottPeriod(periods, "hash_ffc_1m", now)
	if !ok || p.Period != "174" {
		t.Fatalf("expected fallback pick 174 before start, got %+v ok=%v", p, ok)
	}
	wantClose := time.Date(2026, 6, 21, 5, 30, 39, 0, time.UTC)
	if !closeAt.Equal(wantClose) {
		t.Fatalf("closeAt=%v want start %v before period open", closeAt, wantClose)
	}
	now = time.Date(2026, 6, 21, 5, 30, 39, 0, time.UTC)
	p, closeAt, ok = PickOpenLottPeriod(periods, "hash_ffc_1m", now)
	if !ok || p.Period != "174" {
		t.Fatalf("pick=%+v ok=%v want 174", p, ok)
	}
	wantClose = time.Date(2026, 6, 21, 5, 31, 39, 0, time.UTC)
	if !closeAt.Equal(wantClose) {
		t.Fatalf("closeAt=%v want %v", closeAt, wantClose)
	}
}

func TestPickStartSkipLottPeriod_usesOpenPeriod(t *testing.T) {
	now := time.Date(2026, 6, 10, 15, 0, 0, 0, time.UTC)
	periods := []LottPeriod{
		{Period: "p_closed", StartTime: "2026-06-10 14:58:59", EndTime: "2026-06-10 14:59:59"},
		{Period: "p_open", StartTime: "2026-06-10 14:59:59", EndTime: "2026-06-10 15:01:00"},
	}
	p, _, ok := PickStartSkipLottPeriod(periods, "hash_ffc_1m", now)
	if !ok || p.Period != "p_open" {
		t.Fatalf("pick=%+v ok=%v", p, ok)
	}
}

func TestDecodeLottPeriodsDataArray(t *testing.T) {
	raw := []byte(`{"data":[{"period":"115001","start_time":"2026-06-10 15:00:00","end_time":"2026-06-10 15:01:00"}],"code":"201"}`)
	items, err := decodeLottPeriods(json.RawMessage(`[{"period":"115001","start_time":"2026-06-10 15:00:00","end_time":"2026-06-10 15:01:00"}]`), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Period != "115001" {
		t.Fatalf("items=%+v", items)
	}
}

func TestLottPeriodDurationSec_fromStartEnd(t *testing.T) {
	now := time.Date(2026, 6, 10, 15, 0, 0, 0, time.UTC)
	periods := []LottPeriod{
		{Period: "p1", StartTime: "2026-06-10 15:00:00", EndTime: "2026-06-10 15:05:00"},
		{Period: "p2", StartTime: "2026-06-10 15:05:00", EndTime: "2026-06-10 15:10:00"},
	}
	p, closeAt, ok := PickOpenLottPeriod(periods, "hash_ffc_1m", now)
	if !ok {
		t.Fatal("no open period")
	}
	if d := LottPeriodDurationSec(periods, "hash_ffc_1m", p, closeAt); d != 300 {
		t.Fatalf("duration=%d want 300", d)
	}
}

func TestLottPeriodDurationSec_fromAdjacentEndTimes(t *testing.T) {
	periods := []LottPeriod{
		{Period: "p1", EndTime: "2026-06-10 15:01:00"},
		{Period: "p2", EndTime: "2026-06-10 15:02:00"},
	}
	if d := LottPeriodDurationSec(periods, "hash_ffc_1m", periods[0], time.Time{}); d != 60 {
		t.Fatalf("duration=%d want 60", d)
	}
}
