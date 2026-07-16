package historysync

import "testing"

func TestHistoryAPIPathForCode_tronFfc1m(t *testing.T) {
	if got := HistoryAPIPathForCode("tron_ffc_1m"); got != "lottery_logs" {
		t.Fatalf("got %q want lottery_logs", got)
	}
}

func TestHistoryAPIPathForCode_tronSecondLotteries(t *testing.T) {
	if got := HistoryAPIPathForCode("tron_ffc_6s"); got != "lottery_log101s" {
		t.Fatalf("6s got %q want lottery_log101s", got)
	}
	if got := HistoryAPIPathForCode("tron_ffc_15s"); got != "lottery_log125s" {
		t.Fatalf("15s got %q want lottery_log125s", got)
	}
	if got := HistoryAPIPathForCode("tron_ffc_3s"); got != "" {
		t.Fatalf("3s got %q want empty (WS-only)", got)
	}
	if !IsWSOnlyHistory("tron_ffc_3s") {
		t.Fatal("tron_ffc_3s should be WS-only")
	}
}

func TestHistoryAPIPathForCode_hashFfc1m(t *testing.T) {
	if got := HistoryAPIPathForCode("hash_ffc_1m"); got != "lottery_log103s" {
		t.Fatalf("got %q want lottery_log103s", got)
	}
}

func TestHistoryAPIPathForCode_unknown(t *testing.T) {
	if got := HistoryAPIPathForCode("unknown_lottery"); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
