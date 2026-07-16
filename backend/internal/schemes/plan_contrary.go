package schemes

import (
	"strconv"
	"strings"
)

// SupportsPlanContrary 是否适用「计划反集」：固定有限号码/选项池，补集仍为同玩法合法注单。
// 和值/跨度/形态/两面属性等不适用；龙虎对立项适用；定位/任选/直选复式号码池适用。
func SupportsPlanContrary(rule playRule) bool {
	return supportsPlanContrary(rule)
}

// PlanContrarySupportedFromConfig 由方案 config 解析玩法后判断是否展示「计划反集」。
func PlanContrarySupportedFromConfig(kind string, configJSON []byte) bool {
	if kind == "" {
		kind = "custom"
	}
	cfg := parseSchemeConfig(kind, configJSON, 0, 0)
	return supportsPlanContrary(cfg.Play)
}

func supportsPlanContrary(rule playRule) bool {
	if isLonghuPlay(rule) {
		return true
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	switch bm {
	case "hezhi", "kuadu", "teshu", "zu3", "zu6", "zuhe", "baodan", "hunhe",
		"zu24", "zu12", "zu60", "zu30", "zu120", "longhubao",
		"daxiao", "danshuang", "dxds",
		"danshi", "zhixuan_ds", "zuxuan_ds":
		return false
	}
	ptid := strings.ToLower(strings.TrimSpace(rule.PlayTypeID))
	switch ptid {
	case "hezhi", "kuadu", "dxds", "dxds_combo", "daxiao", "danshuang", "longhubao":
		return false
	}
	return true
}

// complementPlanContent 由正集计划内容生成反集投注串（同玩法号码/选项池补集）。
func complementPlanContent(rule playRule, plan string) string {
	plan = strings.TrimSpace(plan)
	if plan == "" || !supportsPlanContrary(rule) {
		return ""
	}
	if isLonghuPlay(rule) {
		return complementLonghuContent(plan)
	}

	if strings.Contains(plan, "\n") {
		lines := splitDingweiPositionLines(plan)
		posCount := playPositionCount(rule)
		if rule.SegmentLen <= 1 {
			// 定位胆多行（万千百十个）：逐非空行取补，空行保持空
			parts := make([]string, len(lines))
			hasAny := false
			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					parts[i] = ""
					continue
				}
				inv := complementNumberLine(rule, line)
				parts[i] = inv
				if inv != "" {
					hasAny = true
				}
			}
			if !hasAny {
				return ""
			}
			return strings.Join(parts, "\n")
		}
		if posCount <= 1 {
			posCount = len(splitGroupLines(plan))
		}
		if posCount <= 0 {
			posCount = rule.SegmentLen
		}
		parts := make([]string, 0, posCount)
		for i := 0; i < posCount; i++ {
			line := ""
			if i < len(lines) {
				line = strings.TrimSpace(lines[i])
			}
			parts = append(parts, complementNumberLine(rule, line))
		}
		return strings.Join(parts, "\n")
	}

	return complementNumberLine(rule, plan)
}

func complementNumberLine(rule playRule, line string) string {
	line = strings.TrimSpace(line)
	forbidden := parsePickTokensForRule(rule, line)
	if len(forbidden) == 0 && line != "" {
		forbidden = parseDigitTokens(line)
	}
	inv := poolExcept(rule, forbidden)
	if len(inv) == 0 {
		return ""
	}
	return strings.Join(inv, ",")
}

// complementLonghuContent 龙虎对立：龙↔虎；单独「和」则投龙+虎。
func complementLonghuContent(plan string) string {
	picks := parseTextTokens(plan)
	if len(picks) == 0 {
		return ""
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 2)
	add := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, p := range picks {
		switch normalizeLonghuPick(p) {
		case "龙":
			add("虎")
		case "虎":
			add("龙")
		case "和":
			add("龙")
			add("虎")
		}
	}
	return strings.Join(out, ",")
}

// poolExcept 号码池补集（按数值匹配，兼容 "07"/"7"）。
func poolExcept(rule playRule, forbidden []string) []string {
	pool := playNumberPool(rule)
	block := map[int]struct{}{}
	for _, f := range forbidden {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if n, err := strconv.Atoi(f); err == nil {
			block[n] = struct{}{}
		}
	}
	out := make([]string, 0, len(pool))
	for _, p := range pool {
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		if _, ok := block[n]; ok {
			continue
		}
		out = append(out, p)
	}
	return out
}
