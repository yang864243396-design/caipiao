package schemes

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"caipiao/backend/internal/guajibet"
)

func rangeTokens(lo, hi int) []string {
	out := make([]string, 0, hi-lo+1)
	for v := lo; v <= hi; v++ {
		out = append(out, strconv.Itoa(v))
	}
	return out
}

// 可达值域必须按「同期号码能否重复」推导，而不是 单号池上下界 × 位数。
func TestReachableAttributeUniverse(t *testing.T) {
	cases := []struct {
		name string
		rule playRule
		want []string
	}{
		{
			"SSC三星直选和值：各位独立可重复 → 0..27",
			playRule{PlayTemplate: "ssc_std", BetMode: "hezhi", SegmentLen: 3},
			rangeTokens(0, 27),
		},
		{
			"SSC三星组选和值：豹子不可下 → 1..26",
			playRule{PlayTemplate: "ssc_std", BetMode: "hezhi", SegmentLen: 3, HezhiZuxuan: true},
			rangeTokens(1, 26),
		},
		{
			"SSC二星组选和值：0/18 仅豹子可组成 → 1..17",
			playRule{PlayTemplate: "ssc_std", BetMode: "hezhi", SegmentLen: 2, HezhiZuxuan: true},
			rangeTokens(1, 17),
		},
		{
			"PK10冠亚和：两名次不重复 → 3..19",
			playRule{PlayTemplate: "pk10_std", BetMode: "hezhi", SegmentLen: 2},
			rangeTokens(3, 19),
		},
		{
			"快三和值：三骰可重复 → 3..18",
			playRule{PlayTemplate: "k3_std", BetMode: "hezhi", SegmentLen: 3},
			rangeTokens(3, 18),
		},
		{
			"SSC三星跨度 → 0..9",
			playRule{PlayTemplate: "ssc_std", BetMode: "kuadu", SegmentLen: 3},
			rangeTokens(0, 9),
		},
		{
			"PK10冠亚跨度：不重复故最小为 1 → 1..9",
			playRule{PlayTemplate: "pk10_std", BetMode: "kuadu", SegmentLen: 2},
			rangeTokens(1, 9),
		},
		{
			"和值尾数 → 0..9",
			playRule{PlayTemplate: "ssc_std", BetMode: "weishu", SegmentLen: 3},
			rangeTokens(0, 9),
		},
		{
			"龙虎和：固定文字选项",
			playRule{PlayTemplate: "ssc_std", BetMode: "longhuhe", SegmentLen: 2},
			[]string{"龙", "虎", "和"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, exact := reachableAttributeUniverse(tc.rule)
			if !exact {
				t.Fatalf("穷举未完成，退回了上下界推导")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v\nwant %v", got, tc.want)
			}
		})
	}
}

// 当前 worker 使用的 attributeUniverse 含不可达选项时必须被揪出来。
// 这些选项频次恒为 0，取冷号会稳定押中它们。
func TestUnreachableHotColdOptions(t *testing.T) {
	cases := []struct {
		name string
		rule playRule
		want []string
	}{
		{
			"PK10冠亚和：2 与 20 永远开不出",
			playRule{PlayTemplate: "pk10_std", PlayTypeID: "g010", BetMode: "hezhi", SegmentLen: 2},
			[]string{"2", "20"},
		},
		{
			"SSC三星直选和值：全部可达",
			playRule{PlayTemplate: "ssc_std", PlayTypeID: "qian3", BetMode: "hezhi", SegmentLen: 3},
			nil,
		},
		{
			"11选5前二和值：不重复故 2 与 22 开不出",
			playRule{PlayTemplate: "syxw_std", PlayTypeID: "g001", BetMode: "hezhi", SegmentLen: 2},
			[]string{"2", "22"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := unreachableOptionsForRule(tc.rule)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// 从方案配置走一遍，确认 UnreachableHotColdOptions 在真实解析链路上也能报出问题。
func TestUnreachableHotColdOptions_fromConfig(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{
		"playTemplate": "pk10_std", "playTypeId": "g010",
		"playMethod": "冠亚和值", "betMode": "hezhi",
	})
	if got := UnreachableHotColdOptions("guaji", raw); len(got) == 0 {
		t.Fatal("PK10 冠亚和的候选宇宙应含不可达选项")
	} else {
		t.Logf("不可达选项：%v", got)
	}
}

// 和值尾数曾被 isHotColdAttributePlay 漏掉，掉进按位分支统计原始球号；
// 已在 worker_pick.go 补上，这里断言分流一致。名单同步的通用不变量见
// hotcold_routing_test.go 的 TestHotColdAttributeListsAgree。
func TestHotColdRouting_weishu(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{
		"playTemplate": "ssc_std", "playTypeId": "qian3",
		"playMethod": "前三和值尾数", "subPlayId": "qian3_hz_weishu", "betMode": "weishu",
	})
	actual, expected := HotColdRouting("guaji", raw)
	if expected != UniverseAttribute {
		t.Fatalf("expected=%q want attribute", expected)
	}
	if actual != expected {
		t.Fatalf("冷热分流走 %q、应走 %q", actual, expected)
	}
}

func TestUniverseForRule_maxUnits(t *testing.T) {
	u, ok := universeForRule(playRule{PlayTemplate: "ssc_std", PlayTypeID: "wuxing", SegmentStart: 0, SegmentLen: 5})
	if !ok {
		t.Fatal("universeForRule failed")
	}
	if u.Kind != UniversePerPosition {
		t.Fatalf("kind=%q", u.Kind)
	}
	if u.MaxUnits != 100000 {
		t.Fatalf("五星全选 MaxUnits=%d want 100000", u.MaxUnits)
	}
}

func TestValidateSchemeBetContent(t *testing.T) {
	ssc5, _ := json.Marshal(map[string]interface{}{
		"playTemplate": "ssc_std", "playTypeId": "g015", "subPlayId": "zhixuan_fs",
		"playMethod": "五星直选复式", "betMode": "fushi",
	})
	pk10, _ := json.Marshal(map[string]interface{}{
		"playTemplate": "pk10_std", "playTypeId": "qian3",
		"playMethod": "前三直选复式", "betMode": "fushi",
	})

	allDigits := "0,1,2,3,4,5,6,7,8,9"
	full := strings.Join([]string{allDigits, allDigits, allDigits, allDigits, allDigits}, "\n")

	t.Run("全选五星直选复式超上限", func(t *testing.T) {
		vs := ValidateSchemeBetContent("guaji", ssc5, full, 5000)
		if !hasViolation(vs, ViolationUnitsOverLimit) {
			t.Fatalf("want units_over_limit, got %+v", vs)
		}
	})

	// 不传上限时应取该玩法自身的上限：五星直选复式 90000，全选 100000 仍越界。
	t.Run("全选五星直选复式撞玩法自身上限", func(t *testing.T) {
		if max := MaxBetUnitsForScheme("guaji", ssc5); max != 90000 {
			t.Fatalf("MaxBetUnitsForScheme=%d want 90000", max)
		}
		vs := ValidateSchemeBetContent("guaji", ssc5, full, 0)
		if !hasViolation(vs, ViolationUnitsOverLimit) {
			t.Fatalf("want units_over_limit, got %+v", vs)
		}
	})

	t.Run("正常小注数放行", func(t *testing.T) {
		vs := ValidateSchemeBetContent("guaji", ssc5, "1,2\n3\n4\n5\n6", 5000)
		if len(vs) != 0 {
			t.Fatalf("want no violation, got %+v", vs)
		}
	})

	t.Run("单押豹子注数为0", func(t *testing.T) {
		vs := ValidateSchemeBetContent("guaji", ssc5, "7\n7\n7\n7\n7", 5000)
		if !hasViolation(vs, ViolationZeroUnits) {
			t.Fatalf("want zero_units, got %+v", vs)
		}
	})

	t.Run("PK10号池不含0", func(t *testing.T) {
		vs := ValidateSchemeBetContent("guaji", pk10, "0,1\n2\n3", 0)
		if !hasViolation(vs, ViolationTokenOutOfPool) {
			t.Fatalf("want token_out_of_pool, got %+v", vs)
		}
	})

	t.Run("空内容", func(t *testing.T) {
		vs := ValidateSchemeBetContent("guaji", ssc5, "   ", 0)
		if !hasViolation(vs, ViolationEmptyContent) {
			t.Fatalf("want empty_content, got %+v", vs)
		}
	})
}

// 定位胆各位独立成注，不是笛卡尔乘积。
func TestCountBetUnits_dingweiSumsNotProduct(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{
		"playTemplate": "ssc_std", "playTypeId": "dingwei", "playMethod": "定位胆",
	})
	n, ok := CountBetUnitsForScheme("guaji", raw, "1,2\n3\n\n\n4,5,6")
	if !ok {
		t.Fatal("算不出注数")
	}
	if n != 6 {
		t.Fatalf("units=%d want 6（2+1+3）", n)
	}
}

// 防分叉：豹子判定必须与下注链路的 guajibet.IsFushiBaoziZeroBet 结论一致。
// 两处各判一遍正是位名错位那次的病根，这里用交叉断言钉死。
func TestFushiBaoziMatchesGuajibet(t *testing.T) {
	meta := guajibet.ParseRuleMeta("ssc_std", "g001", "1", "前三直选复式", "前三码", nil, "1")
	if guajibet.InferBetMode(meta) != "fushi" {
		t.Fatalf("前置条件不成立：InferBetMode=%q", guajibet.InferBetMode(meta))
	}
	rule := playRule{PlayTemplate: "ssc_std", PlayTypeID: "qian3", BetMode: "fushi", SegmentLen: 3}

	for _, content := range []string{
		"7\n7\n7",   // 豹子
		"1\n2\n3",   // 正常单注
		"1,2\n2\n2", // 首位多选
		"5\n5",      // 两位豹子
		"5",         // 单行
	} {
		wire := strings.ReplaceAll(content, "\n", ",")
		want := guajibet.IsFushiBaoziZeroBet(meta, wire)
		got := isFushiBaoziContent(rule, content)
		if got != want {
			t.Fatalf("content=%q: isFushiBaoziContent=%v guajibet=%v", content, got, want)
		}
	}
}

func hasViolation(vs []Violation, code string) bool {
	for _, v := range vs {
		if v.Code == code {
			return true
		}
	}
	return false
}
