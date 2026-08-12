package schemes

import "testing"

// 复式的 betMode 同时存在于时时彩与六合彩；显式 ssc_std 必须按三位直选的位积计数，
// 而不是被当成六合彩复式 C(n,2)。
func TestSchemeMinSingleBetAmount_SSCFushiUsesPositionProduct(t *testing.T) {
	cfg := []byte(`{
		"betUnit":"0.001",
		"schemeCurrency":"USDT",
		"playTemplate":"ssc_std",
		"typeId":"g001",
		"subId":"1",
		"playTypeId":"g001",
		"subPlayId":"1",
		"betMode":"fushi",
		"schemeGroups":["1,0,2,3,4,5,6\n2,3,4\n5,6,7,8,9"],
		"rounds":[{"mult":1,"afterHit":0,"afterMiss":1}]
	}`)
	if got := schemeMinSingleBetAmount(cfg, "custom", numericFromFloat(1)); got != 0.11 {
		t.Fatalf("7×3×5 注 × 0.001 应为 0.11，got %.2f", got)
	}
	if err := validateSchemeMinBetAmount(cfg, "custom", "USDT", numericFromFloat(1)); err != nil {
		t.Fatalf("0.105 应通过 USDT 0.1 最低额校验: %v", err)
	}
}
