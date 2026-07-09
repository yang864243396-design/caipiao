package schemes

import "strings"

func resolvePK10PlayRule(typeID, subID, betMode string) playRule {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	rule := playRule{
		PlayTemplate:  "pk10_std",
		PlayTypeID:    typeID,
		SubPlayID:     legacySubMode(subID, betMode),
		BetMode:       betMode,
		CatalogSubID:  subID,
		NumberPoolMin: 1,
		NumberPoolMax: 10,
	}
	switch typeID {
	case "dingwei":
		rule.SegmentLen = 1
		rule.PositionIdx = dingweiPositionIndex(subID)
		rule.SegmentStart = rule.PositionIdx
	case "qian1":
		rule.SegmentStart, rule.SegmentLen = 0, 1
	case "qian2":
		rule.SegmentStart, rule.SegmentLen = 0, 2
	case "qian3":
		rule.SegmentStart, rule.SegmentLen = 0, 3
	case "qian4":
		rule.SegmentStart, rule.SegmentLen = 0, 4
	case "qian5":
		rule.SegmentStart, rule.SegmentLen = 0, 5
	case "hezhi":
		rule.SegmentLen = pk10HezhiSegmentLen(subID)
	case "dxds_combo":
		rule.SegmentLen = pk10DxdsSegmentLen(subID)
	case "daxiao", "danshuang":
		rule.SegmentLen = 1
	}
	return rule
}

func pk10HezhiSegmentLen(subID string) int {
	switch strings.TrimSpace(subID) {
	case "hz_guanya":
		return 2
	case "hz_shouwei":
		return 2
	case "hz_qian3", "hz_hou3":
		return 3
	default:
		return 2
	}
}

func pk10DxdsSegmentLen(subID string) int {
	switch strings.TrimSpace(subID) {
	case "dxds_guanya":
		return 2
	case "dxds_qian3", "dxds_hou3":
		return 3
	default:
		return 2
	}
}

func isPK10LotteryCode(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	return strings.Contains(code, "pk10") || strings.Contains(code, "feiting")
}
