package schemes

import "strings"

var lhcPlayTypeLabels = map[string]string{
	"tema":           "特码",
	"erquanzhong":    "二全中",
	"erzhongte":      "二中特",
	"techuan":        "特串",
	"sanzhonger":     "三中二",
	"sanquanzhong":   "三全中",
	"shengxiao":      "生肖",
	"weishu":         "尾数",
	"buzhong_xuanyi": "不中/选一",
	"guoguan":        "过关",
	"tematouwei":     "特码头尾",
	"wuxingjiaye":    "五行家野",
	"bose":           "波色",
	"qima":           "七码",
	"renzhong":       "任中",
}

var lhcSubPlayLabels = map[string]string{
	"tema_a":      "特码A",
	"zheng1_te":   "正1特",
	"zheng2_te":   "正2特",
	"zheng3_te":   "正3特",
	"zheng4_te":   "正4特",
	"zheng5_te":   "正5特",
	"zheng6_te":   "正6特",
	"fushi":       "复式",
	"tuotou":      "拖头",
	"sx_dp":       "生肖对碰",
	"ws_dp":       "尾数对碰",
	"sw_dp":       "生尾对碰",
	"renyi_dp":    "任意对碰",
	"texiao":      "特肖",
	"zongxiao":    "总肖",
	"weishu":      "尾数",
	"weishu_bz":   "尾数不中",
	"guoguan":     "过关",
	"tematouwei":  "特码头尾",
	"wuxing":      "五行",
	"jiaye":       "家野",
	"bose":        "波色",
	"banbo":       "半波",
	"banbanbo":    "半半波",
	"qima":        "七码",
}

func formatLHCPlayLabel(typeID, subID string) string {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	typeLabel := lhcPlayTypeLabels[typeID]
	if typeLabel == "" {
		typeLabel = typeID
	}
	subLabel := lhcSubPlayLabels[subID]
	if subLabel == "" {
		subLabel = subID
	}
	if typeLabel != "" && subLabel != "" {
		return typeLabel + subLabel
	}
	if typeLabel != "" {
		return typeLabel
	}
	return subLabel
}
