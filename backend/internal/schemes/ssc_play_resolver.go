package schemes

import "strings"

// resolveSSCPlayRule maps ssc_std catalog typeId/subId to settlement playRule.
func resolveSSCPlayRule(typeID, subID, betMode string) playRule {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)

	rule := playRule{
		PlayTemplate: "ssc_std",
		PlayTypeID:   typeID,
		SubPlayID:    legacySubMode(subID, betMode),
		BetMode:      betMode,
		CatalogSubID: subID,
	}
	if rule.SubPlayID == "" && subID != "" {
		// guaji 同步后子玩法为数字 id（如 13）；勿用误存的 betMode（如 "1"）覆盖。
		rule.SubPlayID = subID
	}

	if typeID == "dingwei" {
		rule.SegmentLen = 1
		rule.PositionIdx = dingweiPositionIndex(subID)
		rule.SegmentStart = rule.PositionIdx
		return rule
	}
	if typeID == "budingwei" {
		rule.SegmentStart, rule.SegmentLen = budingweiSegmentRange(subID)
		return rule
	}
	if typeID == "dxds" {
		rule.SegmentStart, rule.SegmentLen = dxdsSegmentRange(subID)
		return rule
	}
	if typeID == "renxuan" {
		rule.SegmentLen = renPickCount(subID)
		return rule
	}

	if typeID == "combo24" {
		if pos := combo24SegmentPositions(subID); len(pos) > 0 {
			rule.SegmentPos = pos
			rule.SegmentLen = len(pos)
			return rule
		}
	}
	if pos := sscSegmentPositions(typeID); len(pos) > 0 {
		rule.SegmentPos = pos
		rule.SegmentLen = len(pos)
		return rule
	}
	start, length := sscSegmentRange(typeID)
	rule.SegmentStart = start
	rule.SegmentLen = length
	return rule
}

func budingweiSegmentRange(subID string) (int, int) {
	s := strings.ToLower(subID)
	switch {
	case strings.HasPrefix(s, "qian3"):
		return 0, 3
	case strings.HasPrefix(s, "zhong3"):
		return 1, 3
	case strings.HasPrefix(s, "hou3"):
		return 2, 3
	case strings.HasPrefix(s, "qian4"):
		return 0, 4
	case strings.HasPrefix(s, "hou4"):
		return 1, 4
	case strings.HasPrefix(s, "wuxing"):
		return 0, 5
	default:
		return 0, 3
	}
}

func dxdsSegmentRange(subID string) (int, int) {
	s := strings.ToLower(subID)
	switch {
	case strings.HasPrefix(s, "qian2"):
		return 0, 2
	case strings.HasPrefix(s, "hou2"):
		return 3, 2
	case strings.HasPrefix(s, "qian3"):
		return 0, 3
	case strings.HasPrefix(s, "hou3"):
		return 2, 3
	case strings.HasPrefix(s, "wuxing"):
		return 0, 5
	default:
		return 0, 2
	}
}

func sscSegmentPositions(typeID string) []int {
	switch typeID {
	case "qianhou3":
		return []int{0, 2, 4}
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

func dingweiPositionIndex(subID string) int {
	switch {
	case strings.HasSuffix(subID, "_wan"):
		return 0
	case strings.HasSuffix(subID, "_qian"):
		return 1
	case strings.HasSuffix(subID, "_bai"):
		return 2
	case strings.HasSuffix(subID, "_shi"):
		return 3
	case strings.HasSuffix(subID, "_ge"):
		return 4
	default:
		return 0
	}
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
