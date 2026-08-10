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
	"特码": true, "正特码": true,
	"二全中": true, "连码": true,
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
	case "tema", "zhengte", "g001", "g002":
		return true
	case "erquanzhong":
		// 二全中复式 / 生肖对碰 / 尾数对碰（拖头及其它对碰另论）
		if subPlayID == "tuotou" || strings.Contains(subLabel, "拖头") {
			return false
		}
		if subPlayID == "sx_dp" || subPlayID == "281" || strings.Contains(subLabel, "生肖对碰") {
			return true
		}
		if subPlayID == "ws_dp" || subPlayID == "282" || strings.Contains(subLabel, "尾数对碰") {
			return true
		}
		if subPlayID == "sw_dp" || subPlayID == "283" || strings.Contains(subLabel, "生尾对碰") {
			return true
		}
		if strings.Contains(subLabel, "对碰") {
			return false
		}
		return subPlayID == "fushi" || subPlayID == "279" || subPlayID == "277" || subPlayID == "" || strings.Contains(subLabel, "复式") || subLabel == ""
	case "pc28_20", "pc28_28":
		return advTriggerPC28Subs[subPlayID]
	}
	// 目录数字 id：二全中复式 279（兼容旧误写 277）、生肖 281、尾数 282、生尾 283
	if subPlayID == "279" || subPlayID == "277" || subPlayID == "281" || subPlayID == "sx_dp" ||
		subPlayID == "282" || subPlayID == "ws_dp" ||
		subPlayID == "283" || subPlayID == "sw_dp" ||
		(guajiGroup == "二全中" && (subLabel == "复式" || strings.Contains(subLabel, "复式") || strings.Contains(subLabel, "生肖对碰") || strings.Contains(subLabel, "尾数对碰") || strings.Contains(subLabel, "生尾对碰"))) {
		return true
	}
	if guajiGroup == "连码" && (subLabel == "二全中复式" ||
		(strings.Contains(subLabel, "二全中") && strings.Contains(subLabel, "复式")) ||
		(strings.Contains(subLabel, "二全中") && strings.Contains(subLabel, "生肖对碰")) ||
		(strings.Contains(subLabel, "二全中") && strings.Contains(subLabel, "尾数对碰")) ||
		(strings.Contains(subLabel, "二全中") && strings.Contains(subLabel, "生尾对碰")) ||
		strings.Contains(subLabel, "生肖对碰") ||
		strings.Contains(subLabel, "尾数对碰") ||
		strings.Contains(subLabel, "生尾对碰")) {
		return true
	}

	if advTriggerPlayGroups[guajiGroup] {
		// 连码组下仅二全中复式 / 生肖/尾数/生尾对碰开放（勿整组连码全开）
		if guajiGroup == "连码" {
			return subPlayID == "279" || subPlayID == "277" || subPlayID == "281" || subPlayID == "sx_dp" ||
				subPlayID == "282" || subPlayID == "ws_dp" ||
				subPlayID == "283" || subPlayID == "sw_dp" ||
				(strings.Contains(subLabel, "二全中") && strings.Contains(subLabel, "复式")) ||
				strings.Contains(subLabel, "生肖对碰") ||
				strings.Contains(subLabel, "尾数对碰") ||
				strings.Contains(subLabel, "生尾对碰")
		}
		if guajiGroup == "二全中" {
			if strings.Contains(subLabel, "生肖对碰") || subPlayID == "281" || subPlayID == "sx_dp" {
				return true
			}
			if strings.Contains(subLabel, "尾数对碰") || subPlayID == "282" || subPlayID == "ws_dp" {
				return true
			}
			if strings.Contains(subLabel, "生尾对碰") || subPlayID == "283" || subPlayID == "sw_dp" {
				return true
			}
			return !strings.Contains(subLabel, "拖头") && !strings.Contains(subLabel, "对碰")
		}
		return true
	}
	if advTriggerPC28Groups[guajiGroup] {
		return advTriggerPC28SubLabels[subLabel] || advTriggerPC28Subs[subPlayID]
	}
	return false
}
