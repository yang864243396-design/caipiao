package schemes

import "strings"

func resolveK3PlayRule(typeID, subID, betMode string) playRule {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	if betMode == "" {
		betMode = inferK3BetMode(subID)
	}
	return playRule{
		PlayTemplate:  "k3_std",
		PlayTypeID:    typeID,
		SubPlayID:     legacySubMode(subID, betMode),
		BetMode:       betMode,
		CatalogSubID:  subID,
		NumberPoolMin: 1,
		NumberPoolMax: 6,
		SegmentLen:    3,
	}
}

func inferK3BetMode(subID string) string {
	switch strings.TrimSpace(subID) {
	case "k3_hezhi":
		return "hezhi"
	case "ertong_dan":
		return "danshi"
	case "ertong_fu", "biaozhun":
		return "fushi"
	case "santong":
		return "tonghao"
	case "2butong":
		return "butong"
	case "shoudong":
		return "shoudong"
	case "sanlian":
		return "lianhao"
	case "dantiao":
		return "dantiao"
	default:
		return subID
	}
}

func isK3LotteryCode(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	return strings.Contains(code, "_k3") || strings.HasSuffix(code, "k3")
}
