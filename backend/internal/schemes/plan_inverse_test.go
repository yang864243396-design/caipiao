package schemes

import (
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestFormatPlanInverseDigits_SSCSinglePosition(t *testing.T) {
	rule := playRule{PlayTemplate: "ssc_std", SegmentLen: 1, PositionIdx: 0}
	got := formatPlanInverseDigits("1,3,7", rule)
	if got != "1,3,7" {
		t.Fatalf("got %q want 1,3,7", got)
	}
}

func TestFormatPlanInverseDigits_SSCMultiPosition(t *testing.T) {
	rule := playRule{PlayTemplate: "ssc_std", SegmentLen: 5, SegmentStart: 0}
	got := formatPlanInverseDigits("1,3\n4,5\n6\n7\n8", rule)
	want := "1,3\n4,5\n6\n7\n8"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestComputePlanInverseDisplay_FromConfig(t *testing.T) {
	cfg := []byte(`{
		"runTypeId":"fixed_rotate",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"schemeGroups":["1,3,7"]
	}`)
	display := ComputePlanInverseDisplay("seed", "custom", cfg, nil)
	if display.Digits != "0,2,4,5,6,8,9" {
		t.Fatalf("digits = %q", display.Digits)
	}
	if display.BetCount != 7 {
		t.Fatalf("betCount = %d want 7", display.BetCount)
	}
}

func TestComputePlanInverseDisplay_ContraryKind(t *testing.T) {
	cfg := []byte(`{
		"runTypeId":"fixed_rotate",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"schemeGroups":["1,3,7"]
	}`)
	display := ComputePlanInverseDisplay("seed", "contrary", cfg, nil)
	if display.Digits != "0,2,4,5,6,8,9" {
		t.Fatalf("digits = %q want complementary of 1,3,7", display.Digits)
	}
	if display.BetCount != 7 {
		t.Fatalf("betCount = %d want 7", display.BetCount)
	}
}

func TestComputePlanInverseDisplay_ContraryKindWithPlanInverseNumbers(t *testing.T) {
	// 反买落库 / 反集看板：planInverseNumbers 已是反集，不可再取补
	cfg := []byte(`{
		"runTypeId":"run_std",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"playTemplate":"ssc_std",
		"betMode":"dingwei",
		"planInverseNumbers":"0,2,4,5,6,8,9",
		"schemeGroups":["1,3,7"]
	}`)
	display := ComputePlanInverseDisplay("seed", "contrary", cfg, nil)
	if display.Digits != "0,2,4,5,6,8,9" {
		t.Fatalf("digits = %q want stored inverse (no double complement)", display.Digits)
	}
	if display.BetCount != 7 {
		t.Fatalf("betCount = %d want 7", display.BetCount)
	}
}

func TestComputePlanInverseDisplay_ContraryBetConfigOnlyInverse(t *testing.T) {
	// 反买投注落库：仅有 planInverseNumbers，无真实 schemeGroups
	cfg := []byte(`{
		"schemeName":"反买-定位胆",
		"planInverseNumbers":"0,2,4,5,6,8,9",
		"runTypeId":"run_std",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"playTemplate":"ssc_std",
		"betMode":"dingwei"
	}`)
	display := ComputePlanInverseDisplay("snap1", "contrary", cfg, nil)
	if display.Digits != "0,2,4,5,6,8,9" {
		t.Fatalf("digits = %q want planInverseNumbers", display.Digits)
	}
	if display.BetCount != 7 {
		t.Fatalf("betCount = %d want 7", display.BetCount)
	}
}

func TestComputePlanInverseDisplay_CustomBoardUsesStoredInverseFallback(t *testing.T) {
	// master 看板打开仅含 planInverseNumbers 的配置时，正集 demo 取补可能非目标；
	// 若正集盖满号池导致补集为空，应回退到已存反集。
	cfg := []byte(`{
		"planInverseNumbers":"0,2,4,5,6,8,9",
		"runTypeId":"fixed_rotate",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"betMode":"dingwei",
		"schemeGroups":["0,1,2,3,4,5,6,7,8,9"]
	}`)
	display := ComputePlanInverseDisplay("seed", "custom", cfg, nil)
	if display.Digits != "0,2,4,5,6,8,9" {
		t.Fatalf("digits = %q want stored inverse fallback", display.Digits)
	}
}

func TestPreviewGroupBetUnits_SingleDingweiLine(t *testing.T) {
	cfg := parseSchemeConfig("custom", []byte(`{
		"runTypeId":"fixed_rotate",
		"playTypeId":"g006",
		"subPlayId":"13",
		"playTemplate":"ssc_std",
		"betMode":"dingwei",
		"schemeGroups":["3\n\n\n\n"]
	}`), 0, 0)
	got := previewGroupBetUnits(cfg, []byte(`{"schemeGroups":["3\n\n\n\n"]}`), "seed", "custom", nil)
	if got != 1 {
		t.Fatalf("groupBets = %d want 1", got)
	}
}

func TestResolveNextPlanPick_FixedRotate(t *testing.T) {
	cfg := parseSchemeConfig("custom", []byte(`{
		"runTypeId":"fixed_rotate",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"schemeGroups":["2,4,6","8,9"]
	}`), 0, 0)
	pick := resolveNextPlanPick(cfg, nil)
	if pick != "2,4,6" {
		t.Fatalf("pick = %q", pick)
	}
}

func TestResolvePlayTypeLabel_G006NumericSubID(t *testing.T) {
	cfg := map[string]interface{}{
		"playTypeId": "g006",
		"subPlayId":  "13",
		"betMode":    "dingwei",
	}
	if got := resolvePlayTypeLabel(cfg); got != "定位胆" {
		t.Fatalf("got %q want 定位胆", got)
	}
	cfg["playMethod"] = "13"
	if got := resolvePlayTypeLabel(cfg); got != "定位胆" {
		t.Fatalf("playMethod=13 got %q want 定位胆", got)
	}
}

func TestPlayMethodDisplay_BareSubPlayID(t *testing.T) {
	if got := PlayMethodDisplay("13", "g006", "13"); got != "定位胆" {
		t.Fatalf("got %q want 定位胆", got)
	}
	if got := PlayMethodDisplay("定位胆万位", "g006", "13"); got != "定位胆万位" {
		t.Fatalf("got %q want 定位胆万位", got)
	}
}

func TestResolveNextPlanPick_AdvancesAfterDraw(t *testing.T) {
	cfg := parseSchemeConfig("custom", []byte(`{
		"runTypeId":"fixed_rotate",
		"playTypeId":"dingwei",
		"subPlayId":"dingwei_wan",
		"schemeGroups":["1","2","3"]
	}`), 0, 0)
	draws := []sqlcdb.ListLotteryDrawsRow{
		{IssueNo: "100", Balls: []byte(`["1","2","3","4","5"]`)},
	}
	pick := resolveNextPlanPick(cfg, draws)
	if pick != "2" {
		t.Fatalf("pick after win rotate = %q want 2", pick)
	}
}
