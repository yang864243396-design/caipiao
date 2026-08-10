package schemes

import "strings"

// 五星趣味（一帆风顺/好事成双/三星报喜/四季发财）：选 0–9 数字池，每码 1 注。
// 与前三「特殊号」（豹子/对子/顺子）同属 betMode=teshu，须按子玩法区分。

const wuxingQuweiFormatDetail = "趣味玩法：输入 0–9，每个数字用逗号分隔（如 0,3,9）"

// 一帆风顺第三方最多 2 个号（超过 →「投注数字不可超过两位」）；好事成双/三星报喜/四季发财可满选 0–9。
const wuxingYifanMaxPicks = 2
const wuxingYifanFormatDetail = "一帆风顺：输入 0–9，每个数字用逗号分隔，最多 2 个（如 0,3）"

// isWuxingQuweiDigitPlay 五星趣味数字池玩法（非豹子/对子/顺子文字特殊号）。
func isWuxingQuweiDigitPlay(rule playRule) bool {
	return wuxingQuweiDigitMinCount(rule) > 0
}

// isWuxingYifanPlay 五星一帆风顺（rule 162）：最多选 2 码。
func isWuxingYifanPlay(rule playRule) bool {
	text := strings.ToLower(strings.TrimSpace(
		rule.CatalogSubID + " " + rule.SubPlayID + " " + rule.PlayTypeID,
	))
	if strings.Contains(text, "yifan") || strings.Contains(text, "一帆风顺") {
		return true
	}
	sid := strings.TrimSpace(rule.CatalogSubID)
	if i := strings.IndexAny(sid, " \t"); i >= 0 {
		sid = strings.TrimSpace(sid[:i])
	}
	if sid == "" {
		sid = strings.TrimSpace(rule.SubPlayID)
	}
	return sid == "162" || sid == "wuxing_yifan"
}

func isWuxingQuweiLabel(text string) bool {
	return strings.Contains(text, "一帆风顺") || strings.Contains(text, "好事成双") ||
		strings.Contains(text, "三星报喜") || strings.Contains(text, "四季发财") ||
		strings.Contains(strings.ToLower(text), "yifan") ||
		strings.Contains(strings.ToLower(text), "haoshi") ||
		strings.Contains(strings.ToLower(text), "sanxing") ||
		strings.Contains(strings.ToLower(text), "siji")
}

func isWuxingQuweiSubID(subID string) bool {
	switch strings.TrimSpace(subID) {
	case "162", "163", "164", "165",
		"wuxing_yifan", "wuxing_haoshi", "wuxing_sanxing", "wuxing_siji":
		return true
	default:
		return false
	}
}

// wuxingQuweiDigitMinCount 所选号码在开奖五码中至少出现的次数；非趣味返回 0。
func wuxingQuweiDigitMinCount(rule playRule) int {
	text := strings.ToLower(strings.TrimSpace(
		rule.CatalogSubID + " " + rule.SubPlayID + " " + rule.PlayTypeID,
	))
	switch {
	case strings.Contains(text, "siji") || strings.Contains(text, "四季发财"):
		return 4
	case strings.Contains(text, "sanxing") || strings.Contains(text, "三星报喜"):
		return 3
	case strings.Contains(text, "haoshi") || strings.Contains(text, "好事成双"):
		return 2
	case strings.Contains(text, "yifan") || strings.Contains(text, "一帆风顺"):
		return 1
	}
	// guaji 数字 rule id（g015 五星趣味）
	sid := strings.TrimSpace(rule.CatalogSubID)
	if i := strings.IndexAny(sid, " \t"); i >= 0 {
		sid = strings.TrimSpace(sid[:i])
	}
	if sid == "" {
		sid = strings.TrimSpace(rule.SubPlayID)
	}
	switch sid {
	case "162", "wuxing_yifan":
		return 1
	case "163", "wuxing_haoshi":
		return 2
	case "164", "wuxing_sanxing":
		return 3
	case "165", "wuxing_siji":
		return 4
	}
	return 0
}

func validateWuxingQuweiDigitContent(rule playRule, content string) []Violation {
	digits := parseQuweiDigitTokens(content)
	if len(digits) == 0 {
		detail := wuxingQuweiFormatDetail
		if isWuxingYifanPlay(rule) {
			detail = wuxingYifanFormatDetail
		}
		return []Violation{{Code: ViolationZeroUnits, Detail: detail}}
	}
	if isWuxingYifanPlay(rule) && len(digits) > wuxingYifanMaxPicks {
		return []Violation{{Code: ViolationUnitsOverLimit, Detail: wuxingYifanFormatDetail}}
	}
	return nil
}

func parseQuweiDigitTokens(content string) []string {
	content = strings.TrimSpace(strings.ReplaceAll(content, "，", ","))
	if content == "" {
		return nil
	}
	parts := strings.FieldsFunc(content, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\r' || r == '\t' || r == '|'
	})
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// 连写「039」→ 0,3,9
		if len(p) > 1 && isAllDigits(p) {
			for _, ch := range p {
				d := string(ch)
				if d < "0" || d > "9" || seen[d] {
					continue
				}
				seen[d] = true
				out = append(out, d)
			}
			continue
		}
		if len(p) != 1 || p < "0" || p > "9" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func digitAppearCount(seg []string, digit string) int {
	n := 0
	for _, d := range seg {
		if d == digit {
			n++
		}
	}
	return n
}
