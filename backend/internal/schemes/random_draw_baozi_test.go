package schemes

import (
	"strings"
	"testing"
)

func TestSanitizeRandomDrawContent_qianhou3DanshiFiltersBaozi(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g012",
		SubPlayID:    "zhixuan_ds",
		CatalogSubID: "90",
		BetMode:      "danshi",
		SegmentLen:   3,
		SegmentStart: 0,
	}
	if !isSoloBaoziRestrictedRule(rule) {
		t.Fatal("前后三直选单式应受豹子约束")
	}
	// 按位号池展开含 111/222
	got := sanitizeRandomDrawContent(rule, "1,2\n1,2\n1,2")
	if got == "" {
		t.Fatal("expected non-empty after filter")
	}
	for _, tok := range strings.Split(got, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if isBaoziToken(tok) {
			t.Fatalf("unexpected baozi ticket %q in %q", tok, got)
		}
		if len(tok) != 3 {
			t.Fatalf("token %q want 3 digits", tok)
		}
	}
	if sanitizeRandomDrawContent(rule, "5\n5\n5") != "" {
		t.Fatal("solo baozi pool should sanitize to empty")
	}
}

func TestRandomDrawContentUnderMax_qianhou3NoBaozi(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"playTemplate":"ssc_std","playTypeId":"g012","subPlayId":"90","catalogSubId":"90",
		"betMode":"danshi","playMethodLabel":"直选单式","playTypeLabel":"前后三","guajiGroup":"前后三",
		"runTypeId":"random_draw",
		"randomDraw":{"counts":[1,1,1],"strategy":"every"}
	}`)
	cfg := parseSchemeConfig("custom", raw, 0, 0)
	for i := 0; i < 40; i++ {
		got := randomDrawContentUnderMax(cfg)
		if got == "" {
			t.Fatalf("iter %d: empty content", i)
		}
		if !randomDrawContentAcceptable(cfg.Play, got) {
			t.Fatalf("iter %d: unacceptable %q", i, got)
		}
		for _, tok := range strings.Split(got, ",") {
			tok = strings.TrimSpace(tok)
			if tok != "" && isBaoziToken(tok) {
				t.Fatalf("iter %d: baozi %q in %q", i, tok, got)
			}
		}
	}
}

func TestSanitizeRandomDrawContent_fushiSoloBaoziEmpty(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g001",
		SubPlayID:    "zhixuan_fs",
		BetMode:      "fushi",
		SegmentLen:   3,
		SegmentStart: 0,
	}
	if sanitizeRandomDrawContent(rule, "7\n7\n7") != "" {
		t.Fatal("fushi solo baozi should be empty")
	}
	got := sanitizeRandomDrawContent(rule, "1,2\n3\n4")
	if got != "1,2\n3\n4" {
		t.Fatalf("got %q want keep position pool", got)
	}
}
