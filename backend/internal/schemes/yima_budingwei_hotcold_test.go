package schemes

import (
	"strings"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestHotColdYimaBudingweiMaxTwoFromMultiRanks(t *testing.T) {
	// 前三一码：ranks 勾多档时最多出 2 个号，避免第三方「投注数字不可超过两位数」
	raw := `{
		"runTypeId":"hot_cold_warm","playTemplate":"ssc_std",
		"playTypeId":"g009","subPlayId":"113","betMode":"budingwei",
		"playMethodLabel":"前三一码不定位",
		"hotColdWarm":{"totalPeriods":20,"ranks":[[0,1,2,3,4]],"strategy":"keep"}
	}`
	cfg := pickTestConfig(t, raw)
	if !isYimaBudingweiPlayRule(cfg.Play) {
		t.Fatalf("want yima budingwei, rule=%+v", cfg.Play)
	}
	draws := [][]string{
		{"1", "3", "5", "7", "9"},
		{"1", "2", "4", "6", "8"},
		{"0", "1", "3", "5", "7"},
		{"1", "3", "5", "0", "2"},
		{"9", "1", "3", "4", "5"},
	}
	dec := pickHotColdWarmFromDraws(cfg, sqlcdb.SchemeInstance{Kind: "custom"}, draws)
	content := strings.TrimSpace(dec.Content)
	if content == "" {
		t.Fatal("empty yima budingwei pick")
	}
	parts := strings.Split(content, ",")
	if len(parts) > 2 {
		t.Fatalf("一码不定位冷热最多 2 个号, got %q (%d)", content, len(parts))
	}
}

func TestValidateWuxingErmaBudingweiMinFour(t *testing.T) {
	raw := []byte(`{
		"runTypeId":"fixed_rotate","playTemplate":"ssc_std",
		"playTypeId":"g009","subPlayId":"151","betMode":"budingwei",
		"playMethodLabel":"五星二码不定位",
		"schemeGroups":["1,2,3"]
	}`)
	vs := ValidateSchemeBetContent("custom", raw, "1,2,3", 0)
	found := false
	for _, v := range vs {
		if strings.Contains(v.Detail, "至少选择 4") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("want 至少选择 4 个号码, got %+v", vs)
	}
	vs2 := ValidateSchemeBetContent("custom", raw, "1,2,3,4", 0)
	for _, v := range vs2 {
		if strings.Contains(v.Detail, "至少选择 4") {
			t.Fatalf("1,2,3,4 should pass, got %+v", vs2)
		}
	}
}

func TestRandomWuxingErmaBudingweiMinFour(t *testing.T) {
	raw := `{
		"runTypeId":"random_draw","playTemplate":"ssc_std",
		"playTypeId":"g009","subPlayId":"151","betMode":"budingwei",
		"playMethodLabel":"五星二码不定位",
		"randomDraw":{"counts":[2],"strategy":"every"}
	}`
	cfg := pickTestConfig(t, raw)
	for i := 0; i < 20; i++ {
		dec := pickRandomDraw(cfg, sqlcdb.SchemeInstance{Kind: "custom"})
		parts := strings.Split(strings.TrimSpace(dec.Content), ",")
		if len(parts) < 4 {
			t.Fatalf("五星二码随机至少 4 个号, got %q", dec.Content)
		}
	}
}

func TestValidateErmaBudingweiMinTwo(t *testing.T) {
	raw := []byte(`{
		"runTypeId":"fixed_rotate","playTemplate":"ssc_std",
		"playTypeId":"g009","subPlayId":"114","betMode":"budingwei",
		"playMethodLabel":"前三二码不定位",
		"schemeGroups":["1"]
	}`)
	vs := ValidateSchemeBetContent("custom", raw, "1", 0)
	found := false
	for _, v := range vs {
		if strings.Contains(v.Detail, "不能低于两个") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("want 投注数字不能低于两个, got %+v", vs)
	}
	vs2 := ValidateSchemeBetContent("custom", raw, "1,2", 0)
	for _, v := range vs2 {
		if strings.Contains(v.Detail, "不能低于两个") {
			t.Fatalf("1,2 should pass, got %+v", vs2)
		}
	}
}

func TestRandomYimaBudingweiMaxTwo(t *testing.T) {
	raw := `{
		"runTypeId":"random_draw","playTemplate":"ssc_std",
		"playTypeId":"g009","subPlayId":"113","betMode":"budingwei",
		"playMethodLabel":"前三一码不定位",
		"randomDraw":{"counts":[9],"strategy":"every"}
	}`
	cfg := pickTestConfig(t, raw)
	for i := 0; i < 20; i++ {
		dec := pickRandomDraw(cfg, sqlcdb.SchemeInstance{Kind: "custom"})
		content := strings.TrimSpace(dec.Content)
		if content == "" {
			t.Fatal("empty random pick")
		}
		parts := strings.Split(content, ",")
		if len(parts) > 2 {
			t.Fatalf("一码不定位随机最多 2 个号, got %q", content)
		}
	}
}
