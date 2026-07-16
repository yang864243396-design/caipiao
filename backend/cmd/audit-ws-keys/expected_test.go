package main

import "testing"

func TestExpectedWSKey_coversP0P1(t *testing.T) {
	for _, code := range []string{
		"hash_ffc_1m", "hash_ffc_3m", "hash_ffc_5m",
		"tron_jisu", "bnb_ffc_1m",
		"tron_k3_1m", "tron_k3_3m", "tron_k3_5m",
		"tron_ffc_1m", "tron_ffc_3m", "tron_ffc_5m",
		"tron_ffc_3s", "tron_ffc_6s", "tron_ffc_15s",
	} {
		if _, ok := expectedWSKey[code]; !ok {
			t.Errorf("missing expectedWSKey[%q]", code)
		}
	}
}

func TestWsKeyCandidates_restPath(t *testing.T) {
	cands := wsKeyCandidates("lottery_log103s", "lottery_log103s")
	found := false
	for _, c := range cands {
		if c == "lottery_log103" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want lottery_log103 in %v", cands)
	}
}

func TestPeriodAligned_suffix(t *testing.T) {
	if !periodAligned("111202607090911", "111202607090910") {
		t.Fatal("same prefix and near suffix should align")
	}
}
