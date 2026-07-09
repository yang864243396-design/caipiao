package schemes

import "strings"

func resolvePC28PlayRule(typeID, subID, betMode string) playRule {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	if betMode == "" {
		betMode = subID
	}
	return playRule{
		PlayTemplate:  "pc28_std",
		PlayTypeID:    typeID,
		SubPlayID:     betMode,
		BetMode:       betMode,
		CatalogSubID:  subID,
		NumberPoolMin: 0,
		NumberPoolMax: 9,
		SegmentLen:    3,
	}
}

func isPC28LotteryCode(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	return strings.Contains(code, "pc28")
}
