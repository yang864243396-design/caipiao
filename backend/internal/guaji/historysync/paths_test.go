package historysync

import "testing"

func TestHistoryAPIPathForCode_tronFfc1m(t *testing.T) {
	if got := HistoryAPIPathForCode("tron_ffc_1m"); got != "lottery_logs" {
		t.Fatalf("got %q want lottery_logs", got)
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
