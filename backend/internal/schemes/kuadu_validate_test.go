package schemes

import (
	"strings"
	"testing"
)

func TestValidateSchemeConfig_kuaduRejectsAboveNine(t *testing.T) {
	t.Parallel()
	base := `"playTemplate":"ssc_std","playTypeId":"g002","typeId":"g002","subPlayId":"17","subId":"17","betMode":"kuadu"`
	cases := []struct {
		name string
		raw  string
	}{
		{"定码轮换", `{"runTypeId":"fixed_rotate",` + base + `,"schemeGroups":["3,11"]}`},
		{"固定出号", `{"runTypeId":"fixed_number",` + base + `,"schemeGroups":["15"]}`},
		{"高级定码", `{"runTypeId":"adv_fixed_rotate",` + base + `,"jushuList":[{"ju":1,"content":"11","afterHit":1,"afterMiss":1}],"schemeGroups":["0"]}`},
		{"开某投某", `{"runTypeId":"adv_trigger_bet",` + base + `,"triggerBet":{"mode":"always_pos","rows":[{"enabled":true,"open":"0","pos":"11","neg":"1"}]},"schemeGroups":["0"]}`},
		{"冷热出号", `{"runTypeId":"hot_cold_warm",` + base + `,"hotColdWarm":{"totalPeriods":20,"pool":["11,3"],"ranks":[[0]]},"schemeGroups":["0"]}`},
		{"随机出号预览", `{"runTypeId":"random_draw",` + base + `,"randomDraw":{"counts":[2]},"schemeGroups":["11,12"]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vs := ValidateSchemeConfig("custom", []byte(tc.raw))
			if len(vs) == 0 {
				t.Fatal("跨度 >9 应被拒绝")
			}
			joined := ""
			for _, v := range vs {
				joined += v.Detail
			}
			if !strings.Contains(joined, "号池") && !strings.Contains(joined, "11") && !strings.Contains(joined, "15") {
				t.Fatalf("期望号池越界提示, got %v", vs)
			}
		})
	}
}

func TestValidateSchemeConfig_kuaduAllowsZeroToNine(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"runTypeId":"fixed_rotate","playTemplate":"ssc_std",
		"playTypeId":"g002","subPlayId":"17","betMode":"kuadu",
		"schemeGroups":["0,3,9"]
	}`)
	if vs := ValidateSchemeConfig("custom", raw); len(vs) != 0 {
		t.Fatalf("0–9 应放行, got %v", vs)
	}
}
