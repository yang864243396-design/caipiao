package schemes

import "testing"

func TestTriggerBetUsesPosition_zhong3ZhixuanDanshi(t *testing.T) {
	t.Parallel()
	raw := `{
		"runTypeId":"adv_trigger_bet",
		"playTemplate":"ssc_std",
		"playTypeId":"g002",
		"subPlayId":"2",
		"typeId":"g002",
		"subId":"2",
		"betMode":"danshi",
		"playMethodLabel":"直选单式",
		"triggerBet":{"mode":"always_pos","rows":[
			{"enabled":true,"open":"1","pos":"x","neg":""},
			{"enabled":true,"open":"2","pos":"x","neg":""},
			{"enabled":true,"open":"3","pos":"x","neg":""}
		]}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if cfg.Play.SegmentLen != 3 {
		t.Fatalf("SegmentLen=%d want 3; play=%+v", cfg.Play.SegmentLen, cfg.Play)
	}
	if cfg.Play.SegmentStart != 1 {
		t.Fatalf("SegmentStart=%d want 1 (中三)", cfg.Play.SegmentStart)
	}
	if !triggerBetUsesPosition(cfg.Play) {
		t.Fatalf("中三直选单式应按位开某投某; play=%+v", cfg.Play)
	}
	if !isZhixuanDanshiTriggerPlay(cfg.Play) {
		t.Fatalf("isZhixuanDanshiTriggerPlay=false; play=%+v", cfg.Play)
	}

	perPos := "4" + "\n" + "5" + "\n" + "6"
	cfg.Trigger = &triggerBetCfg{
		Mode: "always_pos",
		Rows: []triggerRow{
			{Enabled: true, Open: "1", Pos: perPos},
			{Enabled: true, Open: "2", Pos: perPos},
			{Enabled: true, Open: "3", Pos: perPos},
		},
	}

	// 上期千/百/十 = 1/2/3 → 各位取 open=1/2/3 行的千/百/十列 → 4/5/6
	dec := resolveTriggerBetDecision(cfg, []string{"9", "1", "2", "3", "0"}, "")
	if dec.Skip || dec.Content != perPos {
		t.Fatalf("per-seg pick skip=%v content=%q want %q", dec.Skip, dec.Content, perPos)
	}
	expanded := normalizeZhixuanDanshiContent(cfg.Play, dec.Content)
	if expanded != "456" {
		t.Fatalf("expanded=%q want 456", expanded)
	}
}

func TestTriggerBetUsesPosition_renxuanDanshiNotFixedSegment(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   "g011",
		SubPlayID:    "75",
		BetMode:      "danshi",
		SegmentLen:   2,
	}
	// 任选走专用路径，不进前三式 isZhixuanDanshiTriggerPlay 段内连续位
	if isZhixuanDanshiTriggerPlay(rule) {
		t.Fatal("任选不应被当成固定区位直选单式分列")
	}
	if !isRenxuanZhixuanDanshiTriggerPlay(rule) {
		t.Fatal("任选应走 renxuan trigger 选位路径")
	}
}
