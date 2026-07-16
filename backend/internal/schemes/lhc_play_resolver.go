package schemes

import "strings"

// resolveLHCPlayRule maps lhc_std catalog typeId/subId to settlement playRule.
func resolveLHCPlayRule(typeID, subID, betMode string) playRule {
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	betMode = strings.TrimSpace(betMode)
	if betMode == "" {
		betMode = inferLHCBetMode(typeID, subID)
	}
	return playRule{
		PlayTemplate:  "lhc_std",
		PlayTypeID:    typeID,
		SubPlayID:     betMode,
		BetMode:       betMode,
		CatalogSubID:  subID,
		SegmentLen:    7,
		NumberPoolMin: 1,
		NumberPoolMax: 49,
	}
}

func inferLHCBetMode(typeID, subID string) string {
	s := strings.ToLower(strings.TrimSpace(subID))
	typeID = strings.TrimSpace(typeID)
	if strings.HasPrefix(s, "zheng") && strings.HasSuffix(s, "_te") {
		return "zhengte"
	}
	if s == "tema_a" || typeID == "tema" && !strings.HasPrefix(s, "zheng") {
		return "tema"
	}
	if typeID == "buzhong_xuanyi" && (strings.HasSuffix(s, "bz") || strings.HasSuffix(s, "x1")) {
		if strings.HasSuffix(s, "x1") {
			return "xuanyi"
		}
		return "buzhong"
	}
	if strings.HasSuffix(s, "_bz") && strings.Contains(s, "xiao") {
		return "xiao_bz"
	}
	if strings.HasSuffix(s, "_z") && strings.Contains(s, "xiao") {
		return "xiao_z"
	}
	if strings.Contains(s, "xiao") {
		return "xiao"
	}
	if strings.HasSuffix(s, "_bz") && strings.Contains(s, "wei") {
		return "wei_bz"
	}
	if strings.HasSuffix(s, "_z") && strings.Contains(s, "wei") {
		return "wei_z"
	}
	if strings.HasSuffix(s, "_rz") {
		return "renzhong"
	}
	return s
}

func isLHCLotteryCode(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	return strings.HasPrefix(code, "tron_lhc")
}
