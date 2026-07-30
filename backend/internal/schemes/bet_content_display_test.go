package schemes

import (
	"reflect"
	"testing"
)

func TestPlayRuleDisplayPositions(t *testing.T) {
	cases := []struct {
		name string
		rule playRule
		want []string
	}{
		{"前三", playRule{PlayTemplate: "ssc_std", PlayTypeID: "qian3", SegmentStart: 0, SegmentLen: 3}, []string{"万位", "千位", "百位"}},
		{"中三", playRule{PlayTemplate: "ssc_std", PlayTypeID: "zhong3", SegmentStart: 1, SegmentLen: 3}, []string{"千位", "百位", "十位"}},
		{"后三", playRule{PlayTemplate: "ssc_std", PlayTypeID: "hou3", SegmentStart: 2, SegmentLen: 3}, []string{"百位", "十位", "个位"}},
		{"后二", playRule{PlayTemplate: "ssc_std", PlayTypeID: "hou2", SegmentStart: 3, SegmentLen: 2}, []string{"十位", "个位"}},
		{"五星", playRule{PlayTemplate: "ssc_std", PlayTypeID: "wuxing", SegmentStart: 0, SegmentLen: 5}, []string{"万位", "千位", "百位", "十位", "个位"}},
		{"四星", playRule{PlayTemplate: "ssc_std", PlayTypeID: "g013", SegmentStart: 1, SegmentLen: 4}, []string{"千位", "百位", "十位", "个位"}},
		{"前后三", playRule{PlayTemplate: "ssc_std", PlayTypeID: "qianhou3", SegmentStart: 0, SegmentLen: 3}, []string{"万位", "百位", "个位"}},
		{"前中后三", playRule{PlayTemplate: "ssc_std", PlayTypeID: "qianzhonghou3", SegmentStart: 0, SegmentLen: 3}, []string{"万位", "千位", "百位"}},
		{"前后四_combo24", playRule{PlayTemplate: "ssc_std", PlayTypeID: "combo24", SegmentPos: []int{0, 1, 3, 4}, SegmentLen: 4}, []string{"万位", "千位", "十位", "个位"}},
		{"定位胆五位面板", playRule{PlayTemplate: "ssc_std", PlayTypeID: "dingwei", BetMode: "dingwei", SegmentPos: []int{0, 1, 2, 3, 4}, SegmentLen: 1}, []string{"万位", "千位", "百位", "十位", "个位"}},
		{"PK10前三", playRule{PlayTemplate: "pk10_std", PlayTypeID: "qian3", SegmentStart: 0, SegmentLen: 3}, []string{"冠军", "亚军", "季军"}},
		{"十一选五前三", playRule{PlayTemplate: "syxw_std", PlayTypeID: "g001", SegmentStart: 0, SegmentLen: 3}, []string{"一位", "二位", "三位"}},
		{"和值无位名", playRule{PlayTemplate: "ssc_std", PlayTypeID: "hou3", BetMode: "hezhi", SegmentStart: 2, SegmentLen: 3}, nil},
		{"任选无位名", playRule{PlayTemplate: "syxw_std", PlayTypeID: "g011", BetMode: "zuxuan_fs", SegmentLen: 3}, nil},
		{"快三无位名", playRule{PlayTemplate: "k3_std", PlayTypeID: "hezhi", SegmentLen: 3}, nil},
		{"单位定位胆不标", playRule{PlayTemplate: "ssc_std", PlayTypeID: "dingwei", BetMode: "dingwei", PositionIdx: 4, SegmentLen: 1}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := playRuleDisplayPositions(tc.rule)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("playRuleDisplayPositions = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFormatBetContentLines(t *testing.T) {
	// 中三码直选复式：三行分别对应千/百/十，绝不能标成万/千/百。
	cfg := []byte(`{"playTemplate":"ssc_std","playTypeId":"zhong3","playMethod":"中三码","betMode":"fushi"}`)
	got := FormatBetContentLines("guaji", cfg, "1,3,5\n2\n0,9")
	want := []string{"千位 1 3 5", "百位 2", "十位 0 9"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("中三码 = %v, want %v", got, want)
	}

	// 定位胆只投个位：空行不占位，也不误标。
	got = FormatBetContentLines("guaji", []byte(`{"playTemplate":"ssc_std","playTypeId":"dingwei","subPlayId":"sub_ge","playMethod":"定位胆"}`), "\n\n\n\n7")
	if len(got) != 1 || got[0] != "7" {
		t.Fatalf("单位定位胆 = %v, want [7]", got)
	}

	// 无方案定义（历史单/已删方案）：原样分行，不标位名。
	got = FormatBetContentLines("", nil, "1,2\n3,4")
	if !reflect.DeepEqual(got, []string{"1 2", "3 4"}) {
		t.Fatalf("无 config = %v", got)
	}

	if got := FormatBetContentLines("guaji", cfg, "   "); got != nil {
		t.Fatalf("空内容 = %v, want nil", got)
	}

	// 行数与位段对不上时整体放弃标注，宁可不标也不标错。
	if got := FormatBetContentLines("guaji", cfg, "1\n2\n3\n4"); !reflect.DeepEqual(got, []string{"1", "2", "3", "4"}) {
		t.Fatalf("行数不匹配 = %v", got)
	}
}
