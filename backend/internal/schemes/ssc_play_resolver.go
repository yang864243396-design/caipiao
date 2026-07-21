package schemes

import "strings"

// resolveSSCPlayRule maps ssc_std catalog typeId/subId to settlement playRule.
// playMethod 可选：中文玩法名（如前四一码不定位），用于数字 subId 时解析区位/码数。
func resolveSSCPlayRule(typeID, subID, betMode string, playMethod ...string) playRule {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	method := ""
	if len(playMethod) > 0 {
		method = strings.TrimSpace(playMethod[0])
	}
	labelHint := strings.TrimSpace(method + " " + subID)

	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   typeID,
		SubPlayID:    legacySubMode(subID, betMode),
		BetMode:      betMode,
		CatalogSubID: subID,
	}
	if labelHint != "" && (typeID == "budingwei" || typeID == "g009" || strings.Contains(method, "不定位")) {
		// 数字 guaji rule id 无法表达前四/二码等，合并 playMethod 供结算解析
		rule.CatalogSubID = labelHint
	}
	if labelHint != "" && (typeID == "dxds" || typeID == "g016" || strings.Contains(method, "大小单双") ||
		strings.Contains(method, "和值大小") || strings.Contains(method, "和值单双")) {
		rule.CatalogSubID = labelHint
	}
	if rule.SubPlayID == "" && subID != "" {
		// guaji 同步后子玩法为数字 id（如 13）；勿用误存的 betMode（如 "1"）覆盖。
		rule.SubPlayID = subID
	}

	// g006 = rules/v2「一星/定位胆」；与旧 dingwei 同属定位胆族。
	// SegmentLen 恒为 1（走 evaluateDingwei / 多行五位结算）；
	// 未锁定位的统一定位胆（如 subId=13）用 SegmentPos 表达五位面板，供冷热/随机按位取码。
	if typeID == "dingwei" || typeID == "g006" {
		rule.SegmentLen = 1
		rule.BetMode = "dingwei"
		if isSSCDingweiFivePosition(subID, method) {
			rule.SegmentPos = []int{0, 1, 2, 3, 4}
			rule.PositionIdx = 0
			rule.SegmentStart = 0
		} else {
			rule.PositionIdx = dingweiPositionIndex(subID, method)
			rule.SegmentStart = rule.PositionIdx
		}
		return rule
	}
	if typeID == "budingwei" || typeID == "g009" {
		rule.SegmentStart, rule.SegmentLen = budingweiSegmentRange(labelHint)
		return rule
	}
	// g016=rules 同步后的大小单双（含五星和值大小/单双）；勿落到 sscSegmentRange 默认 0,1
	if typeID == "dxds" || typeID == "g016" || strings.Contains(method, "大小单双") ||
		strings.Contains(method, "和值大小") || strings.Contains(method, "和值单双") {
		rule.SegmentStart, rule.SegmentLen = dxdsSegmentRange(labelHint)
		if rule.BetMode == "" || rule.BetMode == "fushi" || rule.BetMode == "1" {
			if strings.Contains(labelHint, "单双") {
				rule.BetMode = "danshuang"
			} else if strings.Contains(labelHint, "大小") {
				rule.BetMode = "daxiao"
			} else {
				rule.BetMode = "dxds"
			}
		}
		return rule
	}
	if typeID == "renxuan" || typeID == "g011" {
		// 统一语义 id，便于 evaluateSSCByBetMode 走任选专用评估
		rule.PlayTypeID = "renxuan"
		// 数字 subId（如 76）无法表达任二/任三；合并 playMethod
		if labelHint != "" {
			rule.CatalogSubID = labelHint
		}
		rule.SegmentLen = renPickCount(rule.CatalogSubID)
		if rule.SegmentLen <= 0 {
			rule.SegmentLen = renPickCount(subID)
		}
		return rule
	}

	if typeID == "combo24" {
		if pos := combo24SegmentPositions(subID); len(pos) > 0 {
			rule.SegmentPos = pos
			rule.SegmentLen = len(pos)
			return rule
		}
	}
	start, length := sscSegmentRange(typeID)
	rule.SegmentStart = start
	rule.SegmentLen = length
	return rule
}

func budingweiSegmentRange(subID string) (int, int) {
	s := strings.ToLower(subID)
	raw := subID
	switch {
	case strings.Contains(raw, "前四") || strings.HasPrefix(s, "qian4"):
		return 0, 4
	case strings.Contains(raw, "后四") || strings.HasPrefix(s, "hou4"):
		return 1, 4
	case strings.Contains(raw, "五星") || strings.HasPrefix(s, "wuxing"):
		return 0, 5
	case strings.Contains(raw, "前三") || strings.HasPrefix(s, "qian3"):
		return 0, 3
	case strings.Contains(raw, "中三") || strings.HasPrefix(s, "zhong3"):
		return 1, 3
	case strings.Contains(raw, "后三") || strings.HasPrefix(s, "hou3"):
		return 2, 3
	default:
		return 0, 3
	}
}

func dxdsSegmentRange(subID string) (int, int) {
	s := strings.ToLower(subID)
	raw := subID
	switch {
	case strings.Contains(raw, "五星") || strings.Contains(s, "wuxing") ||
		strings.Contains(raw, "和值大小") || strings.Contains(raw, "和值单双"):
		return 0, 5
	case strings.Contains(raw, "前三") || strings.HasPrefix(s, "qian3"):
		return 0, 3
	case strings.Contains(raw, "后三") || strings.HasPrefix(s, "hou3"):
		return 2, 3
	case strings.Contains(raw, "前二") || strings.HasPrefix(s, "qian2"):
		return 0, 2
	case strings.Contains(raw, "后二") || strings.HasPrefix(s, "hou2"):
		return 3, 2
	default:
		return 0, 2
	}
}

// multiZoneSegmentStarts 返回同一组选号覆盖的各区位起点（段长见 rule.SegmentLen）。
// 前中后三：前三(0)/中三(1)/后三(2)；前后三：前三(0)/后三(2)；前后二：前二(0)/后二(3)。
func multiZoneSegmentStarts(rule playRule) []int {
	switch strings.TrimSpace(rule.PlayTypeID) {
	case "qianzhonghou3", "g007":
		return []int{0, 1, 2}
	case "qianhou3", "g012":
		return []int{0, 2}
	case "g008": // rules/v2 前后二
		return []int{0, 3}
	default:
		return nil
	}
}

func combo24SegmentPositions(subID string) []int {
	s := strings.ToLower(subID)
	switch {
	case strings.HasPrefix(s, "qh4"):
		return []int{0, 1, 3, 4}
	case strings.HasPrefix(s, "qh2"):
		return []int{0, 4}
	default:
		return []int{0, 4}
	}
}

func dingweiPositionIndex(subID string, playMethod ...string) int {
	method := ""
	if len(playMethod) > 0 {
		method = strings.TrimSpace(playMethod[0])
	}
	hint := method + " " + subID
	switch {
	case strings.HasSuffix(subID, "_wan"), strings.Contains(hint, "万"):
		return 0
	case strings.HasSuffix(subID, "_qian"), strings.Contains(hint, "千"):
		return 1
	case strings.HasSuffix(subID, "_bai"), strings.Contains(hint, "百"):
		return 2
	case strings.HasSuffix(subID, "_shi"), strings.Contains(hint, "十"):
		return 3
	case strings.HasSuffix(subID, "_ge"), strings.Contains(hint, "个"):
		return 4
	default:
		// 一星定位胆（sub_id=13 等）默认按万位触发；与第三方 wire「号,,,,」一致。
		return 0
	}
}

// isSSCDingweiFivePosition 是否为前端五位定位胆面板（未锁定万/千/百/十/个）。
// 与 client playConfig.isDingweiFivePositionScheme 对齐：g006 + 数字 subId（如 13）→ 五位。
func isSSCDingweiFivePosition(subID, playMethod string) bool {
	s := strings.ToLower(strings.TrimSpace(subID))
	if strings.HasPrefix(s, "dingwei_") {
		return false
	}
	for _, suf := range []string{"_wan", "_qian", "_bai", "_shi", "_ge"} {
		if strings.HasSuffix(s, suf) {
			return false
		}
	}
	hint := strings.TrimSpace(playMethod) + " " + strings.TrimSpace(subID)
	for _, label := range []string{"万位", "千位", "百位", "十位", "个位"} {
		if strings.Contains(hint, label) {
			return false
		}
	}
	// 显式「· 万」等短标签
	for _, label := range []string{"· 万", "· 千", "· 百", "· 十", "· 个"} {
		if strings.Contains(hint, label) {
			return false
		}
	}
	return true
}

func sscSegmentRange(typeID string) (start, length int) {
	switch typeID {
	case "g001", "qian3", "qianzhonghou3", "qianhou3":
		return 0, 3
	case "g002", "zhong3":
		return 1, 3
	case "g003", "hou3":
		return 2, 3
	case "g004", "qian2":
		return 0, 2
	case "g005", "hou2":
		return 3, 2
	case "g007", "g012":
		return 0, 3
	case "g008":
		return 0, 2
	case "g013", "g014":
		return 0, 4
	case "sixing":
		return 1, 4
	case "g015", "wuxing":
		return 0, 5
	case "combo24":
		return 0, 2
	default:
		return 0, 1
	}
}

func legacySubMode(subID, betMode string) string {
	s := strings.ToLower(subID)
	switch {
	case strings.Contains(s, "zhixuan_ds"), betMode == "danshi":
		return "zhixuan_ds"
	case strings.Contains(s, "zhixuan_fs"), betMode == "fushi":
		return "zhixuan_fs"
	case betMode == "zu24", betMode == "zu12", betMode == "zu60", betMode == "zu30", betMode == "zu120":
		return betMode
	case strings.Contains(s, "zu3"), strings.Contains(s, "zu6"),
		strings.Contains(s, "zuxuan"),
		betMode == "zu3", betMode == "zu6":
		return "zuxuan_fs"
	case strings.Contains(s, "_zu3"), strings.Contains(s, "_zu6"):
		return "zuxuan_fs"
	case betMode == "dingwei", strings.HasPrefix(s, "dingwei_"):
		return "dingwei"
	case betMode == "longhu", betMode == "longhuhe", betMode == "hezhi", betMode == "kuadu",
		betMode == "budingwei", betMode == "dxds", betMode == "daxiao", betMode == "danshuang",
		betMode == "zuhe", betMode == "baodan", betMode == "hunhe", betMode == "weishu", betMode == "teshu",
		betMode == "zu24", betMode == "zu12", betMode == "zu60", betMode == "zu30", betMode == "zu120":
		return betMode
	default:
		if betMode == "fushi" {
			return "zhixuan_fs"
		}
		if betMode == "danshi" {
			return "zhixuan_ds"
		}
		return ""
	}
}
