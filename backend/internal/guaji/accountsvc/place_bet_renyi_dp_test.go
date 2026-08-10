package accountsvc

import (
	"testing"

	"caipiao/backend/internal/guajibet"
)

func TestLottBetContentForRequest_RenyiDuipengProgressionSendsSingleBetAmount(t *testing.T) {
	meta := guajibet.ParseRuleMeta("lhc_std", "g003", "284", "二全中任意对碰", "连码", nil, "284")
	meta.ForcedBetMode = "renyi_dp"

	got := lottBetContentForRequest(meta, "01,02,03,04,05,06|07,08,09", 2, 18, 2)
	if got.BetsNums != 18 || got.AmountUnit != 2 || got.Multiple != 2 || got.BetAmount != 72 {
		t.Fatalf("payload=%+v, want 18 bets × 2 unit × 2 multiple = 72", got)
	}
	if got.SingleBetAmount == nil || *got.SingleBetAmount != 4 {
		t.Fatalf("singleBetAmount=%v, want 4", got.SingleBetAmount)
	}
}
