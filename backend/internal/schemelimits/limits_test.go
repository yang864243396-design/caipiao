package schemelimits

import "testing"

func TestEvaluate(t *testing.T) {
	cfg := []byte(`{"stopLoss":"100","takeProfit":"200"}`)
	cases := []struct {
		pnl    float64
		reason string
		hit    bool
	}{
		{-99.99, "", false},
		{-100, ReasonStopLoss, true},
		{200, ReasonTakeProfit, true},
	}
	for _, c := range cases {
		reason, hit := Evaluate(c.pnl, cfg)
		if hit != c.hit || reason != c.reason {
			t.Fatalf("pnl=%v got hit=%v reason=%q want hit=%v reason=%q", c.pnl, hit, reason, c.hit, c.reason)
		}
	}
}
