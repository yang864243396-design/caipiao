package schemes

import (
	"encoding/json"
	"testing"
)

func TestParseSchemeConfig_preservesFastSSCTemplate(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"playTemplate": "fast_ssc_std",
		"playTypeId":   "g017",
		"subPlayId":    "387",
		"subId":        "387",
		"betMode":      "danshuang",
		"playMethodLabel": "尾数单双",
		"schemeGroups": []string{"单"},
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	if cfg.Play.PlayTemplate != "fast_ssc_std" {
		t.Fatalf("PlayTemplate=%q want fast_ssc_std（勿被 resolveSSCPlayRule 写成 ssc_std）", cfg.Play.PlayTemplate)
	}
	if cfg.Play.PlayTypeID != "g017" || cfg.Play.CatalogSubID != "387" {
		t.Fatalf("play=%+v", cfg.Play)
	}
}

func TestPlayTemplateLookupOrder_sscAliases(t *testing.T) {
	got := playTemplateLookupOrder("ssc_std")
	if len(got) != 2 || got[0] != "ssc_std" || got[1] != "fast_ssc_std" {
		t.Fatalf("ssc_std order=%v", got)
	}
	got = playTemplateLookupOrder("fast_ssc_std")
	if len(got) != 2 || got[0] != "fast_ssc_std" || got[1] != "ssc_std" {
		t.Fatalf("fast_ssc_std order=%v", got)
	}
}
