package schemes

import (
	"encoding/json"
	"testing"
)

func TestValidateSchemeBetContent_wuxingYifanDigitPool(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate":    "ssc_std",
		"playTypeId":      "g015",
		"subPlayId":       "162",
		"catalogSubId":    "162",
		"betMode":         "teshu",
		"playMethodLabel": "一帆风顺",
		"segmentLen":      5,
		"runTypeId":       "fixed_rotate",
		"schemeGroups":    []string{"0,3,9"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "0,3", 0); len(vs) > 0 {
		t.Fatalf("0,3 should be valid, got %+v", vs)
	}
	if vs := ValidateSchemeBetContent("custom", raw, "0,3,9", 0); len(vs) == 0 {
		t.Fatal("0,3,9 should fail (一帆风顺最多 2 码)")
	}
	if vs := ValidateSchemeBetContent("custom", raw, "豹子", 0); len(vs) == 0 {
		t.Fatal("豹子 should be invalid for 一帆风顺")
	}
	rule := parseSchemeConfig("custom", raw, 0, 0).Play
	if !isWuxingQuweiDigitPlay(rule) {
		t.Fatalf("want quwei digit play, rule=%+v", rule)
	}
	if !isHotColdDigitOverall(rule) {
		t.Fatal("一帆风顺 cold/hot should use digit overall pool")
	}
	if isHotColdAttributePlay(rule) {
		t.Fatal("一帆风顺 should not use attribute (豹子/对子/顺子) cold/hot")
	}
	uni := attributeUniverse(rule)
	if len(uni) != 10 || uni[0] != "0" || uni[9] != "9" {
		t.Fatalf("attributeUniverse=%v want 0..9", uni)
	}
}
