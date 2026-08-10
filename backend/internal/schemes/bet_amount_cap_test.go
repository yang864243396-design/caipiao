package schemes

import (
	"errors"
	"strings"
	"testing"
)

func TestErrMaxBetAmountExceeded_message(t *testing.T) {
	t.Parallel()
	err := errMaxBetAmountExceeded("USDT")
	if got := err.Error(); got != "最高下注限额100000.00USDT" {
		t.Fatalf("msg=%q", got)
	}
	if !errors.Is(err, ErrMaxBetAmountExceeded) {
		t.Fatal("errors.Is ErrMaxBetAmountExceeded")
	}
	if !isBetAmountExceededError(err) {
		t.Fatal("isBetAmountExceededError local")
	}
	if !isBetAmountExceededError(errors.New("guaji api code=40053: 最高下注限额100000.00USDT")) {
		t.Fatal("isBetAmountExceededError third-party")
	}
}

func TestBetAmountExceedsMax(t *testing.T) {
	t.Parallel()
	if betAmountExceedsMax(100000) {
		t.Fatal("100000 should be allowed")
	}
	if !betAmountExceedsMax(100000.01) {
		t.Fatal("100000.01 should exceed")
	}
	if !betAmountExceedsMax(144000) {
		t.Fatal("144000 should exceed")
	}
}

func TestValidateSchemeMaxBetAmount_sixingZuhe(t *testing.T) {
	t.Parallel()
	// 36000 注 × 2 元 × 倍数 2 = 144000 > 100000
	cfg := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"sixing","subPlayId":"zuhe","betMode":"zuhe",
		"schemeCurrency":"USDT","betUnit":"2",
		"schemeGroups":["0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8"],
		"rounds":[{"mult":2,"afterHit":0,"afterMiss":0}]
	}`)
	err := validateSchemeMaxBetAmount(cfg, "custom", "USDT", numericFromFloat(1))
	if err == nil {
		t.Fatal("want max amount error")
	}
	if !errors.Is(err, ErrMaxBetAmountExceeded) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "100000.00USDT") {
		t.Fatalf("err=%v", err)
	}

	// 倍数 1 → 72000，可通过
	cfgOK := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"sixing","subPlayId":"zuhe","betMode":"zuhe",
		"schemeCurrency":"USDT","betUnit":"2",
		"schemeGroups":["0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8,9\n0,1,2,3,4,5,6,7,8"],
		"rounds":[{"mult":1,"afterHit":0,"afterMiss":0}]
	}`)
	if err := validateSchemeMaxBetAmount(cfgOK, "custom", "USDT", numericFromFloat(1)); err != nil {
		t.Fatalf("72000 should pass: %v", err)
	}
}
