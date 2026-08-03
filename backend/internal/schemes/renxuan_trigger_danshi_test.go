package schemes

import "testing"

func TestTriggerBetUsesPosition_renxuanDanshiIncluded(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g011",
		SubPlayID:    "75",
		CatalogSubID: "75",
		BetMode:      "danshi",
		SegmentLen:   2,
	}
	if !isRenxuanZhixuanDanshiTriggerPlay(rule) {
		t.Fatal("want renxuan zhixuan danshi trigger play")
	}
	if !triggerBetUsesPosition(rule) {
		t.Fatal("任选直选单式应按所选位开某投某")
	}
}

func TestPickTriggerBetRenxuanDanshi_wanQian(t *testing.T) {
	t.Parallel()
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"75",
		"betMode":"danshi",
		"playMethodLabel":"任二直选单式",
		"playTypeLabel":"任选",
		"guajiGroup":"任选",
		"triggerBet":{
			"mode":"always_pos",
			"positionIdxs":[0,1],
			"rows":[
				{"enabled":true,"open":"1","pos":"4\n5","neg":"6\n7"},
				{"enabled":true,"open":"2","pos":"4\n5","neg":"6\n7"},
				{"enabled":true,"open":"3","pos":"8\n9","neg":"0\n1"}
			]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if got := cfg.Trigger.PositionIdxs; len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("PositionIdxs=%v want [0 1]", got)
	}
	// 上期万=1 千=2 → 各位取 open=1/2 行的相对列 → 4 / 5 → 万,千\n45
	dec := resolveTriggerBetDecision(cfg, []string{"1", "2", "0", "0", "0"}, "")
	if dec.Skip {
		t.Fatal("want pick, got skip")
	}
	if dec.Content != "万,千\n45" {
		t.Fatalf("content=%q want 万,千\\n45", dec.Content)
	}
}

func TestPickTriggerBetRenxuanDanshi_defaultWanQian(t *testing.T) {
	t.Parallel()
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"75",
		"betMode":"danshi",
		"playMethodLabel":"任二直选单式",
		"guajiGroup":"任选",
		"triggerBet":{
			"mode":"always_pos",
			"rows":[
				{"enabled":true,"open":"9","pos":"1\n2","neg":"3\n4"}
			]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if got := cfg.Trigger.PositionIdxs; len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("default PositionIdxs=%v want [0 1] 万千", got)
	}
}
