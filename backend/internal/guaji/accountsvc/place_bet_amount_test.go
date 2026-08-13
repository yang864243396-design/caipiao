package accountsvc

import (
	"encoding/json"
	"testing"
)

func TestRoundLottBetAmount_avoidsFloatDrift(t *testing.T) {
	// 与第三方工具成功样例一致：0.01 × 3 × 15 = 0.45
	got := roundLottBetAmount(0.01, 3, 15)
	if got != 0.45 {
		t.Fatalf("got %v want 0.45", got)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "0.45" {
		t.Fatalf("json=%s want 0.45 (dirty float would be 0.449999…)", raw)
	}
}

func TestRoundLottBetAmount_commonUnits(t *testing.T) {
	cases := []struct {
		unit float64
		bets int
		mult int
		want float64
	}{
		{0.001, 3, 15, 0.045},
		{0.02, 3, 15, 0.9},
		{0.1, 3, 15, 4.5},
		{0.2, 3, 15, 9},
		{2, 1, 1, 2},
	}
	for _, tc := range cases {
		if got := roundLottBetAmount(tc.unit, tc.bets, tc.mult); got != tc.want {
			t.Fatalf("unit=%v bets=%d mult=%d got %v want %v", tc.unit, tc.bets, tc.mult, got, tc.want)
		}
	}
}

func TestRoundLottBetAmount_preservesUnitProductForThirdPartyValidation(t *testing.T) {
	// amount_unit、bets_nums、multiple、bet_amount 须严格相乘一致。
	// 1 厘 × 五星组选120满选 252 注 = 0.252；若先截成 0.25，Guaji 会以 40000 拒单。
	if got := roundLottBetAmount(0.001, 252, 1); got != 0.252 {
		t.Fatalf("got %v want 0.252", got)
	}
	if got := roundLottBetAmount(0.001, 176, 1); got != 0.176 {
		t.Fatalf("got %v want 0.176", got)
	}
}
