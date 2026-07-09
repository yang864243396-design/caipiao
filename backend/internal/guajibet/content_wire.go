package guajibet

import "strings"

const pk10PositionCount = 10

func positionCountForTemplate(template string) int {
	switch strings.TrimSpace(template) {
	case "pk10_std":
		return pk10PositionCount
	default:
		return sscPositionCount
	}
}

func digitPadWidth(template string) int {
	switch strings.TrimSpace(template) {
	case "syxw_std", "pk10_std":
		return 2
	default:
		return 1
	}
}

func usesPaddedDigits(template string) bool {
	return digitPadWidth(template) > 1
}

// IsPositionWireContent 判断 bet_content 是否为 N 段逗号分隔的 position wire。
func IsPositionWireContent(betContent string, positions int) bool {
	betContent = strings.TrimSpace(betContent)
	if betContent == "" || positions <= 0 {
		return false
	}
	return len(strings.Split(betContent, ",")) == positions
}

// CountPositionWireBetsNums 统计各位非空段号码个数之和。
func CountPositionWireBetsNums(betContent string, positions int) int {
	if !IsPositionWireContent(betContent, positions) {
		return 0
	}
	total := 0
	for _, seg := range strings.Split(betContent, ",") {
		seg = strings.TrimSpace(seg)
		if seg != "" {
			total += len([]rune(seg))
		}
	}
	return total
}

// countPaddedDingweiPicks 11选5/PK10 定位胆：每段按 2 位号码计 1 注（如 "07"→1 注）。
func countPaddedDingweiPicks(betContent string, positions int) int {
	if !IsPositionWireContent(betContent, positions) {
		return 0
	}
	width := 2
	total := 0
	for _, seg := range strings.Split(betContent, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		n := len(seg) / width
		if n <= 0 {
			n = 1
		}
		total += n
	}
	return total
}

func countPositionProduct(wireContent string, positions int) int {
	if !IsPositionWireContent(wireContent, positions) {
		return 0
	}
	product := 1
	hasAny := false
	for _, seg := range strings.Split(wireContent, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		hasAny = true
		product *= len([]rune(seg))
	}
	if !hasAny {
		return 0
	}
	return product
}

func countPositionProductForTemplate(template, wireContent string, positions int) int {
	width := digitPadWidth(template)
	if width <= 1 {
		return countPositionProduct(wireContent, positions)
	}
	if !IsPositionWireContent(wireContent, positions) {
		return 0
	}
	product := 1
	hasAny := false
	for _, seg := range strings.Split(wireContent, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		hasAny = true
		n := len(seg) / width
		if n <= 0 {
			n = 1
		}
		product *= n
	}
	if !hasAny {
		return 0
	}
	return product
}

func padNumericToken(token string, width int) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if width <= 1 {
		return token
	}
	for len(token) < width {
		token = "0" + token
	}
	return token
}

func normalizeSegmentDigits(template, line string) string {
	tokens := splitPickTokens(line)
	if len(tokens) == 0 {
		return ""
	}
	width := digitPadWidth(template)
	if width <= 1 {
		return normalizePickDigits(line)
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		parts = append(parts, padNumericToken(t, width))
	}
	return strings.Join(parts, "")
}

func formatDingweiWire(template string, positionIdx int, groupContent string) string {
	positions := positionCountForTemplate(template)
	if strings.Contains(groupContent, "\n") {
		return formatDingweiMultiline(template, positions, groupContent)
	}
	digits := normalizeSegmentDigits(template, groupContent)
	if digits == "" {
		return groupContent
	}
	if positionIdx < 0 {
		positionIdx = 0
	}
	if positionIdx >= positions {
		positionIdx = positions - 1
	}
	segments := make([]string, positions)
	segments[positionIdx] = digits
	return strings.Join(segments, ",")
}

func formatDingweiMultiline(template string, positions int, groupContent string) string {
	lines := splitPositionLines(groupContent)
	segments := make([]string, positions)
	for i := 0; i < positions; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		segments[i] = normalizeSegmentDigits(template, line)
	}
	return strings.Join(segments, ",")
}

func formatPositionWire(template string, start, length int, groupContent string) string {
	positions := positionCountForTemplate(template)
	if length <= 0 {
		length = 1
	}
	if start < 0 {
		start = 0
	}
	lines := splitPositionLines(groupContent)
	segments := make([]string, positions)
	for i := 0; i < length; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		} else if len(lines) == 1 {
			line = lines[0]
		}
		pos := start + i
		if pos >= 0 && pos < positions {
			segments[pos] = normalizeSegmentDigits(template, line)
		}
	}
	return strings.Join(segments, ",")
}

func formatPaddedPickDigits(template, groupContent string) string {
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return strings.TrimSpace(groupContent)
	}
	width := digitPadWidth(template)
	if width <= 1 {
		return strings.Join(tokens, ",")
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		parts = append(parts, padNumericToken(t, width))
	}
	return strings.Join(parts, ",")
}

func formatPaddedDanshiDigits(template string, segLen int, groupContent string) string {
	if segLen <= 0 {
		segLen = 1
	}
	width := digitPadWidth(template)
	if width <= 1 {
		return normalizePickDigits(groupContent)
	}
	raw := normalizePickDigits(groupContent)
	if len(raw) >= segLen {
		var b strings.Builder
		for i := 0; i < segLen; i++ {
			b.WriteString(padNumericToken(string(raw[i]), width))
		}
		return b.String()
	}
	tokens := splitPickTokens(groupContent)
	if len(tokens) >= segLen {
		var b strings.Builder
		for i := 0; i < segLen; i++ {
			b.WriteString(padNumericToken(tokens[i], width))
		}
		return b.String()
	}
	var b strings.Builder
	for i := 0; i < segLen; i++ {
		b.WriteString(padNumericToken(string(rune('1'+i)), width))
	}
	return b.String()
}
