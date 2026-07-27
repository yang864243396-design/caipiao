package schemes

import "testing"

func TestTriggerBetUsesPosition_hou2Dxds(t *testing.T) {
	rule := resolveSSCPlayRule("g016", "266", "dxds", "后二大小单双")
	if rule.SegmentStart != 3 || rule.SegmentLen != 2 {
		t.Fatalf("seg=%d+%d want 3+2", rule.SegmentStart, rule.SegmentLen)
	}
	if !triggerBetUsesPosition(rule) {
		t.Fatal("后二大小单双应按位开某投某")
	}
}

func TestTriggerBetPerSegment_hou2Dxds(t *testing.T) {
	cfg := pickTestConfig(t, `{
		"runTypeId":"adv_trigger_bet",
		"playTypeId":"g016",
		"subPlayId":"266",
		"betMode":"dxds",
		"playTemplate":"ssc_std",
		"playMethodLabel":"后二大小单双",
		"triggerBet":{
			"mode":"always_pos",
			"rows":[
				{"enabled":true,"open":"6","pos":"大\n小","neg":"小\n大"},
				{"enabled":true,"open":"3","pos":"小\n双","neg":"大\n单"}
			]
		}
	}`)
	if !triggerBetUsesPosition(cfg.Play) {
		t.Fatalf("usesPos=false play=%+v", cfg.Play)
	}
	// 上期 …6,3 → 十位开6→大\\n小；个位开3→小\\n双
	dec := resolveTriggerBetDecision(cfg, []string{"1", "2", "4", "6", "3"}, "")
	if dec.Skip {
		t.Fatal("should not skip")
	}
	if dec.Content != "大\n小" && dec.Content != "大\n双" {
		// 十位命中 open=6 → 大\n小；个位命中 open=3 → 小\n双；按位拼接应为 大\n双
		// pickTriggerBetPerSegment: 每位独立查映射
		t.Logf("content=%q", dec.Content)
	}
	if dec.Content != "大\n双" {
		t.Fatalf("content=%q want 大\\n双 (十←6→大/小 取正投第一段大；个←3→小/双 取正投第二段双…)", dec.Content)
	}
}
