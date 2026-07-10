package schemes

import (
	"errors"
	"testing"
)

func TestValidateSchemeMinBetAmount(t *testing.T) {
	// 1厘 × 1 × 1 × 1注 = 0.001 < 0.1（USDT）
	cfgLow := []byte(`{"betUnit":"0.001","rounds":[{"mult":1,"afterHit":0,"afterMiss":0}]}`)
	if err := validateSchemeMinBetAmount(cfgLow, "custom", "USDT", numericFromFloat(1)); !errors.Is(err, ErrMinBetAmountTooLow) {
		t.Fatalf("low unit want ErrMinBetAmountTooLow got %v", err)
	}

	// 1角 × 1 × 1 × 1注 = 0.1 OK for USDT
	cfgOk := []byte(`{"betUnit":"0.1","rounds":[{"mult":1,"afterHit":0,"afterMiss":0}]}`)
	if err := validateSchemeMinBetAmount(cfgOk, "custom", "USDT", numericFromFloat(1)); err != nil {
		t.Fatalf("0.1 unit should pass USDT: %v", err)
	}

	// 同金额对 CNY 不足 1
	if err := validateSchemeMinBetAmount(cfgOk, "custom", "CNY", numericFromFloat(1)); !errors.Is(err, ErrMinBetAmountTooLow) {
		t.Fatalf("0.1 for CNY want fail got %v", err)
	}

	// 1元对 CNY OK
	cfgYuan := []byte(`{"betUnit":"1","rounds":[{"mult":1,"afterHit":0,"afterMiss":0}]}`)
	if err := validateSchemeMinBetAmount(cfgYuan, "custom", "CNY", numericFromFloat(1)); err != nil {
		t.Fatalf("1 yuan should pass CNY: %v", err)
	}

	// 1厘 × 100 × 1 × 1注 = 0.1 OK for USDT
	if err := validateSchemeMinBetAmount(cfgLow, "custom", "USDT", numericFromFloat(100)); err != nil {
		t.Fatalf("coef 100 should pass USDT: %v", err)
	}
}

func TestValidateSchemeMinBetAmountIncludesBetUnits(t *testing.T) {
	// 1分 × 4 × 1 × 3注 = 0.12 >= 0.1 USDT OK；CNY 仍不足 1
	cfg := []byte(`{
		"betUnit":"0.01",
		"betMode":"dingwei",
		"playTemplate":"ssc_std",
		"typeId":"g006",
		"subId":"13",
		"schemeGroups":["0,1,5\n\n\n\n"],
		"rounds":[{"mult":1,"afterHit":0,"afterMiss":1},{"mult":2,"afterHit":0,"afterMiss":2},{"mult":4,"afterHit":0,"afterMiss":0}]
	}`)
	if err := validateSchemeMinBetAmount(cfg, "custom", "USDT", numericFromFloat(4)); err != nil {
		t.Fatalf("0.01*4*1*3=0.12 should pass USDT: %v", err)
	}
	if err := validateSchemeMinBetAmount(cfg, "custom", "TRX", numericFromFloat(4)); !errors.Is(err, ErrMinBetAmountTooLow) {
		t.Fatalf("0.12 for TRX want fail got %v", err)
	}

	// 同配置但倍数系数=1：0.01*1*1*3=0.03 < 0.1 fail
	if err := validateSchemeMinBetAmount(cfg, "custom", "USDT", numericFromFloat(1)); !errors.Is(err, ErrMinBetAmountTooLow) {
		t.Fatalf("0.01*1*1*3=0.03 want fail got %v", err)
	}
}

func TestSchemeMinModeMultiplierFromSimpleBetMultiplier(t *testing.T) {
	cfg := map[string]interface{}{
		"betMultiplier": map[string]interface{}{
			"kind": "2",
			"simple": map[string]interface{}{
				"multiples": "2,4,8",
			},
		},
	}
	if got := schemeMinModeMultiplier(cfg); got != 2 {
		t.Fatalf("min mult=%v want 2", got)
	}
}

func TestMinSingleBetAmountForCurrency(t *testing.T) {
	if got := minSingleBetAmountForCurrency("USDT"); got != 0.1 {
		t.Fatalf("USDT=%v", got)
	}
	if got := minSingleBetAmountForCurrency("CNY"); got != 1 {
		t.Fatalf("CNY=%v", got)
	}
	if got := minSingleBetAmountForCurrency(""); got != 1 {
		t.Fatalf("empty default=%v", got)
	}
}
