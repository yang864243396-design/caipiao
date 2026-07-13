package guajibet

import (
	"encoding/json"
	"strings"
)

type RuleMeta struct {
	PlayTemplate string
	TypeID       string
	SubID        string
	Label        string
	TypeLabel    string
	TeamLabel    string
	FullName     string
	RuleID       string
	Group        string
}

func ParseRuleMeta(template, typeID, subID, label, typeLabel string, segmentRule []byte, outboundCode string) RuleMeta {
	meta := RuleMeta{
		PlayTemplate: strings.TrimSpace(template),
		TypeID:       strings.TrimSpace(typeID),
		SubID:        strings.TrimSpace(subID),
		Label:        strings.TrimSpace(label),
		TypeLabel:    strings.TrimSpace(typeLabel),
		RuleID:       strings.TrimSpace(outboundCode),
	}
	if len(segmentRule) == 0 {
		return meta
	}
	var seg struct {
		GuajiGroup    string `json:"guajiGroup"`
		GuajiTeam     string `json:"guajiTeam"`
		GuajiFullName string `json:"guajiFullName"`
		GuajiRuleID   string `json:"guajiRuleId"`
	}
	if json.Unmarshal(segmentRule, &seg) == nil {
		meta.Group = strings.TrimSpace(seg.GuajiGroup)
		meta.TeamLabel = strings.TrimSpace(seg.GuajiTeam)
		meta.FullName = strings.TrimSpace(seg.GuajiFullName)
		if meta.RuleID == "" {
			meta.RuleID = strings.TrimSpace(seg.GuajiRuleID)
		}
	}
	if meta.FullName == "" {
		meta.FullName = joinNonEmpty(meta.TypeLabel, meta.TeamLabel, meta.Label, " · ")
	}
	return meta
}

func joinNonEmpty(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

func (m RuleMeta) combinedText() string {
	return strings.Join([]string{m.FullName, m.Group, m.TeamLabel, m.TypeLabel, m.Label}, " ")
}

// InferBetMode 按 label / guajiGroup 推断 bet_mode（对齐 client runTypeMatrix + seeds bet_mode）。
func InferBetMode(meta RuleMeta) string {
	label := strings.TrimSpace(meta.Label)
	if label == "" {
		label = strings.TrimSpace(meta.FullName)
	}
	text := meta.combinedText()
	group := strings.TrimSpace(meta.Group)

	switch {
	// 仅看子玩法 label / guajiGroup，避免 TypeLabel 默认「定位胆」污染（见 resolvePlayTypeLabel）。
	case strings.Contains(label, "定位胆") || group == "一星":
		return "dingwei"
	case meta.TypeID == "g010":
		if meta.PlayTemplate == "pk10_std" {
			if strings.Contains(label, "和值") {
				return "hezhi"
			}
			if strings.Contains(label, "大小") || strings.Contains(label, "单双") {
				return "dxds"
			}
			return "hezhi"
		}
		if strings.Contains(label, "和") {
			return "longhuhe"
		}
		return "longhu"
	case strings.Contains(meta.TypeLabel, "龙虎") || strings.Contains(text, "龙虎斗"):
		if strings.Contains(label, "和") {
			return "longhuhe"
		}
		return "longhu"
	case strings.Contains(label, "龙虎"):
		if strings.Contains(label, "和") {
			return "longhuhe"
		}
		return "longhu"
	case strings.Contains(label, "组选复式"):
		return "zuxuan_fs"
	case strings.Contains(label, "组选单式"):
		return "zuxuan_ds"
	case strings.Contains(label, "直选复式"), strings.Contains(label, "复式") && strings.Contains(label, "直选"):
		return "fushi"
	case strings.Contains(label, "直选单式"), strings.Contains(label, "单式") && strings.Contains(label, "直选"):
		return "danshi"
	case strings.Contains(label, "直选和值"), label == "和值" && !strings.Contains(label, "尾数"):
		return "hezhi"
	case strings.Contains(label, "组选和值"):
		return "hezhi"
	case strings.Contains(label, "和值") && !strings.Contains(label, "单双") && !strings.Contains(label, "大小") && !strings.Contains(label, "尾数"):
		return "hezhi"
	case strings.Contains(label, "跨度"):
		return "kuadu"
	case strings.Contains(label, "混合"):
		return "hunhe"
	case label == "组合" || strings.Contains(label, "组合"):
		return "zuhe"
	case strings.Contains(label, "组三") && strings.Contains(label, "单式"):
		return "zuxuan_ds"
	case strings.Contains(label, "组六") && strings.Contains(label, "单式"):
		return "zuxuan_ds"
	case strings.Contains(label, "组三"):
		return "zu3"
	case strings.Contains(label, "组六"):
		return "zu6"
	case strings.Contains(label, "包胆"):
		return "baodan"
	case strings.Contains(label, "和值单双"), strings.Contains(label, "尾数单双"):
		return "danshuang"
	case strings.Contains(label, "和值大小"), strings.Contains(label, "尾数大小"):
		return "daxiao"
	case strings.Contains(label, "庄闲"):
		return "zhuangxian"
	case strings.Contains(label, "和值尾数"):
		return "weishu"
	case strings.Contains(label, "尾数"):
		return "weishu"
	case strings.Contains(label, "特殊号"), strings.Contains(label, "特殊"):
		return "teshu"
	case strings.Contains(label, "一帆风顺"), strings.Contains(label, "好事成双"),
		strings.Contains(label, "三星报喜"), strings.Contains(label, "四季发财"):
		return "teshu"
	case strings.Contains(label, "大小") || strings.Contains(label, "单双"):
		return "dxds"
	case strings.Contains(label, "不定位"):
		return "budingwei"
	case strings.Contains(label, "组选24"), strings.Contains(text, "zu24"):
		return "zu24"
	case strings.Contains(label, "组选120"), strings.Contains(text, "zu120"):
		return "zu120"
	case strings.Contains(label, "组选60"), strings.Contains(text, "zu60"):
		return "zu60"
	case strings.Contains(label, "组选30"), strings.Contains(text, "zu30"):
		return "zu30"
	case strings.Contains(label, "组选20"), strings.Contains(text, "zu20"):
		return "zu20"
	case strings.Contains(label, "组选12"), strings.Contains(text, "zu12"):
		return "zu12"
	case strings.Contains(label, "组选10"), strings.Contains(text, "zu10"):
		return "zu10"
	case strings.Contains(label, "组选6"), strings.Contains(text, "zu6"):
		return "zu6"
	case strings.Contains(label, "组选5"), strings.Contains(text, "zu5"):
		return "zu5"
	case strings.Contains(label, "组选4"), strings.Contains(text, "zu4"):
		return "zu4"
	}
	if meta.PlayTemplate == "pk10_std" {
		switch strings.TrimSpace(meta.TypeID) {
		case "g008":
			return "daxiao"
		case "g009":
			return "danshuang"
		}
	}
	if meta.PlayTemplate == "lhc_std" {
		if mode := inferLHCBetMode(meta); mode != "" {
			return mode
		}
	}
	if meta.PlayTemplate == "syxw_std" {
		switch strings.TrimSpace(meta.TypeID) {
		case "g006", "renxuan_ds":
			return "danshi"
		case "g005", "renxuan_fs":
			return "fushi"
		}
	}
	if meta.PlayTemplate == "k3_std" {
		if strings.Contains(label, "复选") || strings.Contains(label, "标准选号") {
			return "fushi"
		}
		if strings.Contains(label, "手动输入") || strings.Contains(label, "三连号") {
			return "danshi"
		}
	}
	// 兼容：仅当子玩法未给出明确信号时，才用 TypeLabel 判断定位胆
	if group == "" && label == "" && strings.Contains(meta.TypeLabel, "定位胆") {
		return "dingwei"
	}
	return ""
}
