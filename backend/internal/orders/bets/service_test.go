package bets

import "testing"

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
