package schemes

import (
	"reflect"
	"testing"
)

func TestLuckyZhuangXianSupportsTieAsSinglePick(t *testing.T) {
	rule := playRule{
		PlayTemplate: "fast_ssc_std",
		PlayTypeID:   "g017",
		CatalogSubID: "388",
		BetMode:      "zhuangxian",
		SegmentLen:   1,
	}
	if got := attributeUniverse(rule); !reflect.DeepEqual(got, []string{"庄", "和", "闲"}) {
		t.Fatalf("attributeUniverse=%v, want 庄/和/闲", got)
	}
	if got := randomDrawCountMax(rule); got != 1 {
		t.Fatalf("random draw max=%d, want 1", got)
	}
	if got := countPlayWireBetUnits(rule, "和"); got != 1 {
		t.Fatalf("和 bet units=%d, want 1", got)
	}
	if err := validateGroupContent(rule, "庄,和"); err == nil {
		t.Fatal("multiple lucky zhuangxian picks must be rejected")
	}
}
