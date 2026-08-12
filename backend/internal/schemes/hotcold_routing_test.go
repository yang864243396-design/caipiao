package schemes

import (
	"encoding/json"
	"strings"
	"testing"
)

// 冷热出号的分流由两套名单共同决定，它们必须说同一件事：
//
//   - universeKindForRule  判断「这个玩法下的是什么形态的内容」
//   - isHotColdAttributePlay + attributeUniverse  决定「冷热计频按什么统计」
//
// 两边对不上时不会报错，只会静默算错：漏掉的玩法掉进按位分支，
// 统计的是原始球号频次，选出来的内容形态也跟着错。
// 和值尾数（weishu）就是这么漏了很久——给「后三和值尾数」选出 "180,280,380,680,780,880"，
// 而合法内容是 0-9 的单个数字，第三方直接拒单，注单永远卡在 pending。

// flatAttributeBetModes 是冷热出号按单一属性池统计的 betMode。
// 前二/后二/前三/后三大小单双虽也使用属性 token，但须逐位统计，不能放入此列表。
var flatAttributeBetModes = []string{
	"daxiao", "danshuang", "zhuangxian",
	"longhu", "longhuhe", "longhubao",
	"hezhi", "kuadu", "weishu",
}

// TestHotColdAttributeListsAgree 属性盘名单两侧必须一致，且都能给出非空候选宇宙。
func TestHotColdAttributeListsAgree(t *testing.T) {
	for _, mode := range flatAttributeBetModes {
		t.Run(mode, func(t *testing.T) {
			// 用后三：多数属性玩法都要一个有长度的区位才算得出和值/跨度
			rule := playRule{
				PlayTemplate: "ssc_std", PlayTypeID: "hou3", BetMode: mode,
				SegmentStart: 2, SegmentLen: 3,
			}
			if got := universeKindForRule(rule); got != UniverseAttribute {
				t.Fatalf("universeKindForRule = %q，本用例前提是属性盘", got)
			}
			if !isHotColdAttributePlay(rule) {
				t.Errorf("universeKindForRule 认定为属性盘，isHotColdAttributePlay 却说不是" +
					"——冷热出号会掉进按位分支统计原始球号")
			}
			if u := attributeUniverse(rule); len(u) == 0 {
				t.Errorf("attributeUniverse 为空，冷热计频无从统计")
			}
		})
	}
}

func TestHotColdPerPositionDxdsKeepsDedicatedBranch(t *testing.T) {
	rule := playRule{
		PlayTemplate: "ssc_std", PlayTypeID: "hou3", BetMode: "dxds",
		SegmentStart: 2, SegmentLen: 3,
	}
	if !isPerPosDxdsRandom(rule) {
		t.Fatalf("后三大小单双应使用按位分支，rule=%+v", rule)
	}
	if isHotColdAttributePlay(rule) {
		t.Fatal("后三大小单双不得落入单一属性池分支")
	}
}

// TestHotColdRoutingConsistent 每个属性玩法的实际分支都要与应走分支一致。
func TestHotColdRoutingConsistent(t *testing.T) {
	for _, mode := range flatAttributeBetModes {
		t.Run(mode, func(t *testing.T) {
			raw, err := json.Marshal(map[string]interface{}{
				"playTemplate": "ssc_std", "playTypeId": "hou3",
				"subPlayId": "hou3_" + mode, "betMode": mode,
			})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			actual, expected := HotColdRouting("guaji", raw)
			if actual != expected {
				t.Fatalf("冷热分流走 %q、应走 %q", actual, expected)
			}
		})
	}
}

// TestWeishuAttributeUniverse 和值尾数的候选宇宙恒为 0-9。
//
// 尾数取的是和值的个位，与号池上下界和区位长度都无关；
// 之前 attributeUniverse 里没有这一支，返回 nil，于是整条属性分支被跳过。
func TestWeishuAttributeUniverse(t *testing.T) {
	for _, tc := range []struct {
		name string
		rule playRule
	}{
		{name: "后三", rule: playRule{PlayTemplate: "ssc_std", BetMode: "weishu", SegmentStart: 2, SegmentLen: 3}},
		{name: "前二", rule: playRule{PlayTemplate: "ssc_std", BetMode: "weishu", SegmentStart: 0, SegmentLen: 2}},
		{name: "号池非 0-9", rule: playRule{
			PlayTemplate: "pk10_std", BetMode: "weishu",
			SegmentStart: 0, SegmentLen: 2, NumberPoolMin: 1, NumberPoolMax: 10,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := attributeUniverse(tc.rule)
			want := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
			if strings.Join(u, ",") != strings.Join(want, ",") {
				t.Fatalf("候选宇宙 = %v，期望 0-9", u)
			}
		})
	}
}

// TestWeishuHotColdPickIsSingleDigit 冷热出号给和值尾数选出的内容必须是 0-9 的单个数字。
//
// 这是缺陷的直接表现面：修之前这里会选出 "180,280,380,680,780,880"。
func TestWeishuHotColdPickIsSingleDigit(t *testing.T) {
	rule := playRule{
		PlayTemplate: "ssc_std", PlayTypeID: "hou3", BetMode: "weishu",
		SegmentStart: 2, SegmentLen: 3,
	}
	// 造一批开奖，让后三和值尾数明显偏向某几个值
	draws := [][]string{
		{"0", "0", "1", "2", "3"}, // 后三 1+2+3=6 → 尾 6
		{"0", "0", "2", "2", "2"}, // 6 → 尾 6
		{"0", "0", "1", "1", "4"}, // 6 → 尾 6
		{"0", "0", "9", "9", "9"}, // 27 → 尾 7
		{"0", "0", "5", "5", "5"}, // 15 → 尾 5
	}
	res := HotColdWarmAttributeTiers(rule, draws)
	if res.Mode != "attribute" {
		t.Fatalf("计频模式 = %q，期望 attribute", res.Mode)
	}
	if len(res.Universe) != 10 {
		t.Fatalf("候选宇宙 %v，期望 10 个", res.Universe)
	}
	if res.Counts["6"] != 3 {
		t.Errorf("尾数 6 出现 %d 次，按开奖应为 3 次", res.Counts["6"])
	}
	if res.Counts["7"] != 1 || res.Counts["5"] != 1 {
		t.Errorf("尾数 7/5 各应 1 次，实际 %d/%d", res.Counts["7"], res.Counts["5"])
	}
	// 最热的必须是 6
	if len(res.Hot) == 0 || res.Hot[0] != "6" {
		t.Errorf("最热尾数 = %v，期望 6", res.Hot)
	}
	for _, tok := range append(append([]string{}, res.Hot...), res.Cold...) {
		if len(tok) != 1 || tok < "0" || tok > "9" {
			t.Errorf("选出的 %q 不是 0-9 的单个数字——这正是修复前产出 \"180,280,380\" 的形态", tok)
		}
	}
}

// TestWeishuPickContentValidates 冷热出号选出的内容要能通过合法投注空间校验。
// 把出号与校验两端接起来，避免只有一端被改。
func TestWeishuPickContentValidates(t *testing.T) {
	cfg := map[string]interface{}{
		"playTemplate": "ssc_std", "playTypeId": "hou3",
		"subPlayId": "hou3_weishu", "betMode": "weishu",
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if vs := ValidateSchemeBetContent("guaji", raw, "3,6,9", 0); len(vs) > 0 {
		t.Fatalf("合法尾数内容被判违规：%s", vs[0].Detail)
	}
	// 修复前实际产出的内容，必须被判违规
	if vs := ValidateSchemeBetContent("guaji", raw, "180,280,380,680,780,880", 0); len(vs) == 0 {
		t.Fatal("多位内容越出 0-9，应被判违规")
	}
}

// TestNormalizeZhixuanDanshiContent_skipsAttributePlays 属性内容不得被按位 reshape/展开。
func TestNormalizeZhixuanDanshiContent_skipsAttributePlays(t *testing.T) {
	rule := resolveSSCPlayRule("hou3", "hou3_weishu", "weishu")
	got := normalizeZhixuanDanshiContent(rule, "3,6,9")
	if got != "3,6,9" {
		t.Fatalf("weishu content must stay as tokens, got %q", got)
	}
}
