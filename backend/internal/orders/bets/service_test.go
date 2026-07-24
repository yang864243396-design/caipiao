package bets

import (
	"testing"
	"time"
)

func TestMapLotteryCodeFilter(t *testing.T) {
	if !mapLotteryCodeFilter("tron_ffc_1m").Valid || mapLotteryCodeFilter("tron_ffc_1m").String != "tron_ffc_1m" {
		t.Fatal("valid lottery code")
	}
	if mapLotteryCodeFilter("all").Valid {
		t.Fatal("all should be null")
	}
	if mapLotteryCodeFilter("").Valid {
		t.Fatal("empty should be null")
	}
}

func TestMapCurrencyFilter(t *testing.T) {
	c, err := mapCurrencyFilter("all")
	if err != nil || c.Valid {
		t.Fatal("all should be null")
	}
	c, err = mapCurrencyFilter("usdt")
	if err != nil || !c.Valid || c.String != "USDT" {
		t.Fatal("usdt should normalize")
	}
	if _, err = mapCurrencyFilter("BTC"); err == nil {
		t.Fatal("btc should reject")
	}
}

func TestStatusLabel(t *testing.T) {
	if statusLabel("pending") != "未开奖" {
		t.Fatal("label")
	}
}

func TestCloudStatusLabel(t *testing.T) {
	if cloudStatusLabel("hit") != "已中奖" {
		t.Fatal("hit")
	}
	if cloudStatusLabel("miss") != "未中奖" {
		t.Fatal("miss")
	}
}

func TestExpandRangeStartForOrderNo(t *testing.T) {
	to := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC).Add(24 * time.Hour) // exclusive end of 7/24
	fromToday := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	got := expandRangeStartForOrderNo(fromToday, to, 3)
	want := to.Add(-3 * 24 * time.Hour) // 7/22 00:00
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	// 已覆盖更宽区间时不收缩
	fromWide := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if expandRangeStartForOrderNo(fromWide, to, 3) != fromWide {
		t.Fatal("should keep wider start")
	}
}
