package schemes

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestLotteryCategoryForCode(t *testing.T) {
	if lotteryCategoryForCode("tron_ffc_1m") != "ssc" {
		t.Fatal("expected ssc for ffc")
	}
	if lotteryCategoryForCode("tron_pk10_jisu") != "pk10" {
		t.Fatal("expected pk10")
	}
	if lotteryCategoryForCode("tron_syxw") != "x5" {
		t.Fatal("expected x5")
	}
}

func TestResolveSSCPlayRule_g006NumericSubKeepsCatalogID(t *testing.T) {
	rule := resolveSSCPlayRule("g006", "13", "1")
	if rule.SubPlayID != "13" {
		t.Fatalf("SubPlayID=%q want 13", rule.SubPlayID)
	}
	if rule.CatalogSubID != "13" {
		t.Fatalf("CatalogSubID=%q want 13", rule.CatalogSubID)
	}
}

func TestIsNumericBetModeArtifact(t *testing.T) {
	if !isNumericBetModeArtifact("1") {
		t.Fatal("1 should be artifact")
	}
	if isNumericBetModeArtifact("dingwei") {
		t.Fatal("dingwei should not be artifact")
	}
}

func TestLookupSubPlayFromRows_g006CatalogSub13(t *testing.T) {
	rows := []sqlcdb.GetSubPlayRow{
		guajiSubPlayRow("g006", "13", "定位胆", 1),
	}
	got, err := lookupSubPlayFromRows("ssc_std", rows, "g006", "13", "1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.SubID != "13" {
		t.Fatalf("sub_id=%q want 13", got.SubID)
	}
}