package schemes

import "testing"

func TestOddsZhixuanQian3AlignsV6Prize(t *testing.T) {
	// base=0（未取到第三方赔率）→ 参考基准 970
	if got := oddsZhixuan(3, 0); got != 970 {
		t.Fatalf("oddsZhixuan(3,0)=%v want 970（参考基准）", got)
	}
	// 玩法详情跟单预估：1 元 × 1 倍 × 赔率
	if p := estimateMaxPrize(1, 1, oddsZhixuan(3, 0)); p != 970 {
		t.Fatalf("estimateMaxPrize=%v want 970", p)
	}
	if p := estimateMaxPrize(2, 1, oddsZhixuan(3, 0)); p != 1940 {
		t.Fatalf("estimateMaxPrize 2元=%v want 1940", p)
	}
}

// TestOddsZhixuanScalesWithGuajiOdds 第三方赔率线变化时，各玩法赔率按比例缩放。
func TestOddsZhixuanScalesWithGuajiOdds(t *testing.T) {
	// 模拟第三方 lott_odds=1960（2元三星直选）→ 1元基准 980
	base := oddsBaseForLotteryTestHelper(1960, "tron_ffc_1m")
	if base != 980 {
		t.Fatalf("base=%v want 980", base)
	}
	if got := oddsZhixuan(3, base); got != 980 {
		t.Fatalf("oddsZhixuan(3,980)=%v want 980", got)
	}
	if got := oddsZhixuan(4, base); got != 9800 {
		t.Fatalf("oddsZhixuan(4,980)=%v want 9800", got)
	}
	// 哈希彩用 hash_odds
	hashBase := oddsBaseForLotteryTestHelper2(1950, "hash_ffc_1m")
	if hashBase != 975 {
		t.Fatalf("hashBase=%v want 975", hashBase)
	}
}

func oddsBaseForLotteryTestHelper(lottOdds float64, code string) float64 {
	SetGuajiOdds(lottOdds, 0)
	defer SetGuajiOdds(1940, 1950)
	return oddsBaseForLottery(code)
}

func oddsBaseForLotteryTestHelper2(hashOdds float64, code string) float64 {
	SetGuajiOdds(0, hashOdds)
	defer SetGuajiOdds(1940, 1950)
	return oddsBaseForLottery(code)
}
