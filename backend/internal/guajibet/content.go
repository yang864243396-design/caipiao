package guajibet

import (
	"strings"
)

const sscPositionCount = 5

// FormatBetContent 将方案内部号码内容转为第三方 web_bets/lott 的 bet_content。
// 时时彩定位胆：五位逗号分隔，如 "13579,,,," / ",,,13579,"（见接口文档 §11）。
func FormatBetContent(template, betMode, playMethod string, positionIdx int, groupContent string) string {
	template = strings.TrimSpace(template)
	betMode = strings.TrimSpace(betMode)
	groupContent = normalizeGroupContentEdges(groupContent)
	if strings.TrimSpace(groupContent) == "" {
		return ""
	}

	mode := strings.ToLower(betMode)
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(playMethod))
	}

	switch template {
	case "", "ssc_std", "fast_ssc_std":
		if isDingweiMode(mode, playMethod) {
			pos := positionIdx
			if pos < 0 {
				pos = positionIndexFromLabel(playMethod)
			}
			return formatSSCDingweiContent(pos, groupContent)
		}
	}
	return groupContent
}

func isDingweiMode(betMode, playMethod string) bool {
	if betMode == "dingwei" {
		return true
	}
	label := strings.TrimSpace(playMethod)
	return strings.Contains(label, "定位胆")
}

func positionIndexFromLabel(label string) int {
	for idx, r := range []rune{'万', '千', '百', '十', '个'} {
		if strings.ContainsRune(label, r) {
			return idx
		}
	}
	return 0
}

func formatSSCDingweiContent(positionIdx int, groupContent string) string {
	return formatDingweiWire("ssc_std", positionIdx, groupContent)
}

func splitPositionLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

// IsSSCDingweiBetContent 判断是否为 SSC 定位胆五位逗号 wire 格式（含 4 个分隔逗号）。
func IsSSCDingweiBetContent(betContent string) bool {
	return IsPositionWireContent(betContent, sscPositionCount)
}

// CountDingweiBetsNums 统计 bet_content 各位非空段号码个数之和（= 第三方 bets_nums）。
func CountDingweiBetsNums(betContent string) int {
	return CountPositionWireBetsNums(betContent, sscPositionCount)
}

// NeedsSoloBet 定位胆 solo 标志（v6hs1 实测：单注定位胆须 solo=false，true 会报「单挑参数错误」）。
func NeedsSoloBet(betContent string) bool {
	return false
}

func normalizePickDigits(groupContent string) string {
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return ""
	}
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString(t)
	}
	return b.String()
}

func splitPickTokens(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	parts := strings.FieldsFunc(content, func(r rune) bool {
		return r == ',' || r == ' ' || r == '|' || r == '，' || r == '\n' || r == '\r' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
