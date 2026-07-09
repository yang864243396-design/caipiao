package schemes

import (
	"encoding/json"
	"strings"
)

// rules/v2 同步后玩法类型为 groups[].name（如「一星」「龙虎」），子玩法 label 为 rule.name。

var advTriggerPC28SubLabels = map[string]bool{
	"和值": true, "大小单双": true, "龙虎豹": true,
	// 旧 sub_id 兼容
	"hezhi": true, "dxds": true, "longhubao": true,
}

var advTriggerPlayGroups = map[string]bool{
	"一星": true, "龙虎": true,
}

var advTriggerPC28Groups = map[string]bool{
	"2.0模式": true, "2.8模式": true,
}

func guajiGroupFromSegment(seg json.RawMessage) string {
	if len(seg) == 0 || string(seg) == "null" {
		return ""
	}
	var m struct {
		GuajiGroup string `json:"guajiGroup"`
	}
	if err := json.Unmarshal(seg, &m); err != nil {
		return ""
	}
	return strings.TrimSpace(m.GuajiGroup)
}

func isLonghuPlayGroup(guajiGroup, playTypeID string) bool {
	if guajiGroup == "龙虎" || playTypeID == playTypeLonghu {
		return true
	}
	return false
}

// SupportsAdvTriggerBet 高级开某投某玩法矩阵（兼容旧 type_id 与 rules/v2 guajiGroup）。
func SupportsAdvTriggerBet(playTypeID, subPlayID, guajiGroup, subLabel string) bool {
	playTypeID = strings.TrimSpace(playTypeID)
	subPlayID = strings.TrimSpace(subPlayID)
	guajiGroup = strings.TrimSpace(guajiGroup)
	subLabel = strings.TrimSpace(subLabel)

	switch playTypeID {
	case playTypeDingwei, playTypeLonghu:
		return true
	case "pc28_20", "pc28_28":
		return advTriggerPC28Subs[subPlayID]
	}

	if advTriggerPlayGroups[guajiGroup] {
		return true
	}
	if advTriggerPC28Groups[guajiGroup] {
		return advTriggerPC28SubLabels[subLabel] || advTriggerPC28Subs[subPlayID]
	}
	return false
}
