package schemes

import "strings"

func resolveSYXWPlayRule(typeID, subID, betMode string) playRule {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	rule := playRule{
		PlayTemplate:  "syxw_std",
		PlayTypeID:    typeID,
		SubPlayID:     legacySubMode(subID, betMode),
		BetMode:       betMode,
		CatalogSubID:  subID,
		NumberPoolMin: 1,
		NumberPoolMax: 11,
	}
	if typeID == "dingwei" {
		rule.SegmentLen = 1
		rule.PositionIdx = dingweiPositionIndex(subID)
		rule.SegmentStart = rule.PositionIdx
		return rule
	}
	if typeID == "renxuan_fs" || typeID == "renxuan_ds" {
		return rule
	}
	if betMode == "budingwei" {
		rule.SegmentStart, rule.SegmentLen = 0, 3
		return rule
	}
	start, length := syxwSegmentRange(typeID)
	rule.SegmentStart = start
	rule.SegmentLen = length
	return rule
}

func syxwSegmentRange(typeID string) (start, length int) {
	switch strings.TrimSpace(typeID) {
	case "qian3":
		return 0, 3
	case "qian2":
		return 0, 2
	default:
		return 0, 1
	}
}

func isSYXWLotteryCode(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	return strings.Contains(code, "syxw")
}
