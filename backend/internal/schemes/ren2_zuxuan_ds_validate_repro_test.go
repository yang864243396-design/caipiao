package schemes

import "testing"

func TestValidateSchemeBetContent_ren2ZuxuanDanshiWanGe(t *testing.T) {
	raw := `{
		"runTypeId":"fixed_rotate",
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"78",
		"betMode":"zuxuan_ds",
		"playMethodLabel":"任二组选单式",
		"playTypeLabel":"任选",
		"guajiGroup":"任选",
		"schemeGroups":["万,个\n12,34"]
	}`
	cfg := []byte(raw)
	for _, content := range []string{
		"万,个\n12,34",
		"万,千\n12,24",
		"万,千\n12,23",
		"千,个\n12,34",
		"万,个\n12",
	} {
		vs := ValidateSchemeBetContent("custom", cfg, content, 0)
		if len(vs) > 0 {
			t.Fatalf("content=%q violations=%+v", content, vs)
		}
	}
}

func TestNormalizeZhixuanDanshiContent_ren2ZuxuanDsKeepsWanGe(t *testing.T) {
	rule := parseSchemeConfig("custom", []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g011","subPlayId":"78","betMode":"zuxuan_ds",
		"playMethodLabel":"任二组选单式","guajiGroup":"任选"
	}`), 0, 0).Play
	got := normalizeZhixuanDanshiContent(rule, "万,个\n12,34")
	if got != "万个|12,34" && got != "万,个|12,34" {
		// Format 位名无逗号：万个|12,34
		if !(len(got) > 0 && stripPositionLabelPrefix(rule, got) == "12,34") {
			t.Fatalf("normalize=%q want keep picks 12,34 with 万/个 prefix", got)
		}
	}
	stripped := stripPositionLabelPrefix(rule, got)
	if stripped != "12,34" {
		t.Fatalf("stripped=%q want 12,34 (from %q)", stripped, got)
	}
}
