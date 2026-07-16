package schemes

import "strings"

func resolveCatalogPlayRule(cfg map[string]interface{}) (playRule, bool) {
	template := strings.TrimSpace(stringVal(cfg, "playTemplate"))
	typeID := strings.TrimSpace(stringVal(cfg, "typeId"))
	if typeID == "" {
		typeID = strings.TrimSpace(stringVal(cfg, "playTypeId"))
	}
	subID := strings.TrimSpace(stringVal(cfg, "subId"))
	if subID == "" {
		subID = strings.TrimSpace(stringVal(cfg, "subPlayId"))
	}
	betMode := playBetModeFromConfig(cfg)

	if (template == "ssc_std" || template == "fast_ssc_std") && typeID != "" && subID != "" {
		playMethod := strings.TrimSpace(stringVal(cfg, "playMethod"))
		return resolveSSCPlayRule(typeID, subID, betMode, playMethod), true
	}
	if template == "syxw_std" && typeID != "" && subID != "" {
		return resolveSYXWPlayRule(typeID, subID, betMode), true
	}
	if template == "pk10_std" && typeID != "" && subID != "" {
		return resolvePK10PlayRule(typeID, subID, betMode), true
	}
	if template == "k3_std" && typeID != "" && subID != "" {
		return resolveK3PlayRule(typeID, subID, betMode), true
	}
	if template == "pc28_std" && typeID != "" && subID != "" {
		return resolvePC28PlayRule(typeID, subID, betMode), true
	}
	if template == "lhc_std" && typeID != "" && subID != "" {
		return resolveLHCPlayRule(typeID, subID, betMode), true
	}
	if isLHCTypeID(typeID) && subID != "" {
		return resolveLHCPlayRule(typeID, subID, betMode), true
	}
	return playRule{}, false
}

func isLHCTypeID(typeID string) bool {
	switch strings.TrimSpace(typeID) {
	case "tema", "erquanzhong", "erzhongte", "techuan", "sanzhonger", "sanquanzhong",
		"shengxiao", "weishu", "buzhong_xuanyi", "guoguan", "tematouwei",
		"wuxingjiaye", "bose", "qima", "renzhong":
		return true
	default:
		return false
	}
}

func stringVal(cfg map[string]interface{}, key string) string {
	if v, ok := cfg[key].(string); ok {
		return v
	}
	return ""
}
