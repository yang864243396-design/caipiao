package schemes

import "testing"

func TestIsRenxuanNeedsPositionRule_excludesZhixuanFushi(t *testing.T) {
	t.Parallel()
	fs := playRule{
		PlayTypeID:   "g011",
		SubPlayID:    "74",
		CatalogSubID: "74",
		BetMode:      "fushi",
		SegmentLen:   2,
	}
	if isRenxuanNeedsPositionRule(fs) {
		t.Fatal("直选复式不应走选位壳")
	}
	if !isRenxuanZhixuanFushiRule(fs) {
		t.Fatal("want zhixuan fushi")
	}
}

func TestCountRenxuanNeedsPosition_zuxuanPool(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTypeID:   "g011",
		SubPlayID:    "76",
		CatalogSubID: "76",
		BetMode:      "zuxuan_fs",
		SegmentLen:   2,
	}
	// 万千个 三位 × 号池 3 码 → C(3,2)*C(3,2)=3*3=9
	content := "万,千,个\n1,2,3"
	got := countRenxuanNeedsPositionBetUnits(rule, content)
	if got != 9 {
		t.Fatalf("units=%d want 9", got)
	}
}

func TestPickTriggerBetRenxuanPool_wrapsPositions(t *testing.T) {
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
			"positionIdxs":[0,1],
			"rows":[
				{"enabled":true,"open":"1","pos":"0,1,2","neg":"3,4,5"},
				{"enabled":true,"open":"9","pos":"6,7,8","neg":"0,1,2"}
			]
		}
	}`
	cfg := parseSchemeConfig("custom", []byte(raw), 0, 0)
	if !isRenxuanNeedsPositionTriggerPlay(cfg.Play) {
		t.Fatalf("want needs-position trigger, rule=%+v", cfg.Play)
	}
	if isRenxuanPerPosTriggerPlay(cfg.Play) {
		t.Fatal("组选复式不应按位分列")
	}
	// 万=1 千=9 → 所选位任一位命中开出 1 → 投 0,1,2，带选位前缀
	dec := resolveTriggerBetDecision(cfg, []string{"1", "9", "0", "0", "0"}, "")
	if dec.Skip {
		t.Fatal("want pick, got skip")
	}
	if dec.Content != "万,千\n0,1,2" {
		t.Fatalf("content=%q want 万,千\\n0,1,2", dec.Content)
	}
}

func TestWrapRenxuanNeedsPosition_random(t *testing.T) {
	t.Parallel()
	rule := playRule{
		PlayTypeID:   "g011",
		CatalogSubID: "75",
		BetMode:      "danshi",
		SegmentLen:   2,
	}
	got := wrapRenxuanNeedsPositionContent(rule, "12,34", []int{0, 1})
	if got != "万,千\n12,34" {
		t.Fatalf("got %q", got)
	}
	// 已带前缀不重复包
	got2 := wrapRenxuanNeedsPositionContent(rule, "万,千\n12", []int{0, 1})
	if got2 != "万,千\n12" {
		t.Fatalf("got2 %q", got2)
	}
}
