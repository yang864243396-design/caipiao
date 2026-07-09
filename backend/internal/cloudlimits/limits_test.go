package cloudlimits

import "testing"

func TestEvaluateTotalLimits(t *testing.T) {
	limits := Limits{StopLossYuan: 100, TakeProfitYuan: 200}

	if reason, hit := Evaluate(-100, limits); !hit || reason != ReasonTotalStopLoss {
		t.Fatalf("stop loss got reason=%q hit=%v", reason, hit)
	}
	if reason, hit := Evaluate(200, limits); !hit || reason != ReasonTotalTakeProfit {
		t.Fatalf("take profit got reason=%q hit=%v", reason, hit)
	}
	if _, hit := Evaluate(-50, limits); hit {
		t.Fatal("within range should not hit")
	}
}

func TestEvaluateZeroMeansUnlimited(t *testing.T) {
	if _, hit := Evaluate(-999, Limits{}); hit {
		t.Fatal("zero limits should not hit")
	}
}
