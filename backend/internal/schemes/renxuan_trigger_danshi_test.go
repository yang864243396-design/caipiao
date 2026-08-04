package schemes

import "testing"

func TestTriggerBetUsesPosition_renxuanDanshiPerPosFlag(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g011",
		SubPlayID:    "75",
		CatalogSubID: "75",
		BetMode:      "danshi",
		SegmentLen:   2,
	}
	if !isRenxuanNeedsPositionTriggerPlay(rule) {
		t.Fatal("want renxuan needs-position trigger play")
	}
	if !isRenxuanZhixuanDanshiTriggerPlay(rule) {
		t.Fatal("任选直选单式应按投注选位分列正/反投")
	}
	// 仍走独立开奖/投注选位路径，不进通用 triggerBetUsesPosition 段内分列
	if triggerBetUsesPosition(rule) {
		t.Fatal("任选不应进通用按位分列路径")
	}
}

func TestPickTriggerBetRenxuanNeedsPosition_openAndBet(t *testing.T) {
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
			"openPositionIdx":0,
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
	if !cfg.Trigger.HasOpenPosition || cfg.Trigger.OpenPositionIdx != 0 {
		t.Fatalf("OpenPositionIdx=%v Has=%v want 0", cfg.Trigger.OpenPositionIdx, cfg.Trigger.HasOpenPosition)
	}
	// 开奖选位=万，上期万=1 → 取该行正投「4\n5」展开 → 万,千\n45
	dec := resolveTriggerBetDecision(cfg, []string{"1", "2", "0", "0", "0"}, "")
	if dec.Skip {
		t.Fatal("want pick, got skip")
	}
	if dec.Content != "万,千\n45" {
		t.Fatalf("content=%q want 万,千\\n45", dec.Content)
	}
}

func TestPickTriggerBetRenxuanNeedsPosition_openGeBetWanQian(t *testing.T) {
	t.Parallel()
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g011",
		"subPlayId":"76",
		"betMode":"zuxuan_fs",
		"playMethodLabel":"任二组选复式",
		"playTypeLabel":"任选",
		"guajiGroup":"任选",
		"triggerBet":{
			"mode":"always_pos",
			"openPositionIdx":4,
			"positionIdxs":[0,1,2],
			"rows":[
				{"enabled":true,"open":"1","pos":"0,1,2","neg":"3,4,5"},
				{"enabled":true,"open":"9","pos":"6,7,8","neg":"0,1,2"}
			]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if cfg.Trigger.OpenPositionIdx != 4 {
		t.Fatalf("OpenPositionIdx=%d want 4", cfg.Trigger.OpenPositionIdx)
	}
	if got := cfg.Trigger.PositionIdxs; len(got) != 3 || got[0] != 0 || got[2] != 2 {
		t.Fatalf("PositionIdxs=%v want [0 1 2]", got)
	}
	// 个位=1 → 投 0,1,2，前缀万,千,百
	dec := resolveTriggerBetDecision(cfg, []string{"9", "9", "9", "9", "1"}, "")
	if dec.Skip {
		t.Fatal("want pick, got skip")
	}
	if dec.Content != "万,千,百\n0,1,2" {
		t.Fatalf("content=%q want 万,千,百\\n0,1,2", dec.Content)
	}
}

func TestPickTriggerBetRenxuanDanshi_openGeMultiBetCols(t *testing.T) {
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
			"openPositionIdx":4,
			"positionIdxs":[0,1,2],
			"rows":[
				{"enabled":true,"open":"1","pos":"4\n5","neg":"7\n8"}
			]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	// 启用区固定 2 列（第一/第二位）；个位=1 → 45；投注选位万千百作前缀
	dec := resolveTriggerBetDecision(cfg, []string{"9", "9", "9", "9", "1"}, "")
	if dec.Skip {
		t.Fatal("want pick, got skip")
	}
	if dec.Content != "万,千,百\n45" {
		t.Fatalf("content=%q want 万,千,百\\n45", dec.Content)
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
	if cfg.Trigger.OpenPositionIdx != 0 {
		t.Fatalf("default OpenPositionIdx=%d want 0", cfg.Trigger.OpenPositionIdx)
	}
}
