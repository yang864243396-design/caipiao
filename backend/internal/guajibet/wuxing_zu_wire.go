package guajibet

import "strings"

// 五星组选 wire（抓包 hash.iyes.dev / game 77）：
//   - zu60:  重号,单号池     如 1,234      → C(n,3)
//   - zu30:  三号码,二号码   如 123,45     → 6 注
//   - zu20:  重号(2位),单号池 如 12,345    → 2 注
//   - zu10:  0,五码池       如 0,12345    → 5 注
//   - zu5:   0,五码池       如 0,12345    → 5 注

func sampleWuxingZu60Content() string {
	return "1,234"
}

func sampleWuxingZu30Content() string {
	return "123,45"
}

func sampleWuxingZu20Content() string {
	return "12,345"
}

func sampleWuxingZuZeroPoolContent() string {
	return "0,12345"
}

func formatWuxingZuWire(mode, groupContent string) string {
	switch mode {
	case "zu60":
		return formatWuxingZuDoubleSingleWire(groupContent)
	case "zu30":
		return formatWuxingZu30Wire(groupContent)
	case "zu20":
		return formatWuxingZu20Wire(groupContent)
	case "zu10", "zu5":
		return formatWuxingZuZeroPoolWire(groupContent)
	default:
		return formatCommaPickDigits(groupContent)
	}
}

// formatZu12Wire 四星/前后四组选12：双区「二重号池,单号池」，如 12,34。
func formatZu12Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return "12,34"
	}
	parts := splitCommaParts(groupContent)
	if len(parts) == 2 {
		a := normalizePickDigits(parts[0])
		b := normalizePickDigits(parts[1])
		if len(a) >= 2 && len(b) >= 2 {
			return a + "," + b
		}
	}
	digits := splitPickDigits(groupContent)
	if len(digits) >= 4 {
		return strings.Join(digits[:2], "") + "," + strings.Join(digits[2:4], "")
	}
	return "12,34"
}

// formatZu4Wire 四星/前后四组选4：双区「三重号,单号」，如 1,2。
func formatZu4Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return "1,2"
	}
	parts := splitCommaParts(groupContent)
	if len(parts) == 2 {
		a := normalizePickDigits(parts[0])
		b := normalizePickDigits(parts[1])
		if len(a) >= 1 && len(b) >= 1 {
			return a[:1] + "," + b[:1]
		}
	}
	digits := splitPickDigits(groupContent)
	if len(digits) >= 2 {
		return digits[0] + "," + digits[1]
	}
	return "1,2"
}

func formatWuxingZuDoubleSingleWire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZu60Content()
	}
	if _, wire, ok := parseWuxingZuDoubleSingleWire(groupContent); ok {
		return wire
	}
	// 扁选「0,1,2,3,4」→「0,1234」（重号+单号池）；实测 flat 原样会「投注数字不合规」
	if wire, ok := coerceFlatDigitsToDoubleSingle(groupContent, 1, 3); ok {
		return wire
	}
	return sampleWuxingZu60Content()
}

// coerceFlatDigitsToDoubleSingle 把扁选号池压成「前 headLen 码作头段, 其余拼成尾段」。
func coerceFlatDigitsToDoubleSingle(groupContent string, headLen, minTail int) (string, bool) {
	digits := splitPickDigits(groupContent)
	if headLen <= 0 || len(digits) < headLen+minTail {
		return "", false
	}
	// 已是两段且头段长度匹配时勿再压
	if parts := splitCommaParts(groupContent); len(parts) == 2 {
		if len(normalizePickDigits(parts[0])) == headLen && len(normalizePickDigits(parts[1])) >= minTail {
			return "", false
		}
	}
	head := strings.Join(digits[:headLen], "")
	tail := strings.Join(digits[headLen:], "")
	if head == "" || tail == "" {
		return "", false
	}
	return head + "," + tail, true
}

func formatWuxingZu30Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZu30Content()
	}
	if wire, ok := normalizeWuxingZu30Wire(groupContent); ok {
		return wire
	}
	// 扁选 5 码 →「abc,de」
	if wire, ok := coerceFlatDigitsToDoubleSingle(groupContent, 3, 2); ok {
		if nwire, ok2 := normalizeWuxingZu30Wire(wire); ok2 {
			return nwire
		}
	}
	return sampleWuxingZu30Content()
}

func formatWuxingZu20Wire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZu20Content()
	}
	if wire, ok := normalizeWuxingZu20Wire(groupContent); ok {
		return wire
	}
	if wire, ok := coerceFlatDigitsToDoubleSingle(groupContent, 2, 3); ok {
		if nwire, ok2 := normalizeWuxingZu20Wire(wire); ok2 {
			return nwire
		}
	}
	return sampleWuxingZu20Content()
}

func formatWuxingZuZeroPoolWire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return sampleWuxingZuZeroPoolContent()
	}
	if wire, ok := normalizeWuxingZuZeroPoolWire(groupContent); ok {
		return wire
	}
	// 扁选 5 码 →「0,abcde」
	digits := splitPickDigits(groupContent)
	if len(digits) >= 5 {
		tail := strings.Join(digits[:5], "")
		if wire, ok := normalizeWuxingZuZeroPoolWire("0," + tail); ok {
			return wire
		}
	}
	return sampleWuxingZuZeroPoolContent()
}

func parseWuxingZuDoubleSingleWire(wire string) (double string, wireOut string, ok bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", "", false
	}
	double = normalizePickDigits(parts[0])
	singles := normalizePickDigits(parts[1])
	if len(double) != 1 || len(singles) < 3 {
		return "", "", false
	}
	return double, parts[0] + "," + parts[1], true
}

func normalizeWuxingZu30Wire(wire string) (string, bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", false
	}
	a := normalizePickDigits(parts[0])
	b := normalizePickDigits(parts[1])
	if len(a) != 3 || len(b) != 2 {
		return "", false
	}
	return a + "," + b, true
}

func normalizeWuxingZu20Wire(wire string) (string, bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", false
	}
	a := normalizePickDigits(parts[0])
	b := normalizePickDigits(parts[1])
	if len(a) != 2 || len(b) < 3 {
		return "", false
	}
	return a + "," + b, true
}

func normalizeWuxingZuZeroPoolWire(wire string) (string, bool) {
	parts := splitCommaParts(strings.TrimSpace(wire))
	if len(parts) != 2 {
		return "", false
	}
	head := normalizePickDigits(parts[0])
	tail := normalizePickDigits(parts[1])
	if head != "0" || len(tail) != 5 {
		return "", false
	}
	return "0," + tail, true
}

func countWuxingZu60BetNums(wireContent string) int {
	if _, _, ok := parseWuxingZuDoubleSingleWire(wireContent); !ok {
		if n := len(splitPickDigits(wireContent)); n >= 5 {
			return countZuGroupBetNums("zu60", n)
		}
		return 0
	}
	parts := splitCommaParts(wireContent)
	n := len(normalizePickDigits(parts[1]))
	if n < 3 {
		return 0
	}
	return combin(n, 3)
}

func countWuxingZu30BetNums(wireContent string) int {
	if _, ok := normalizeWuxingZu30Wire(wireContent); !ok {
		return 0
	}
	return 6
}

func countWuxingZu20BetNums(wireContent string) int {
	if _, ok := normalizeWuxingZu20Wire(wireContent); !ok {
		return 0
	}
	return 2
}

func countWuxingZuZeroPoolBetNums(wireContent string) int {
	if _, ok := normalizeWuxingZuZeroPoolWire(wireContent); !ok {
		return 0
	}
	return 5
}

func countWuxingZuBetNums(mode, wireContent string) int {
	switch mode {
	case "zu60":
		return countWuxingZu60BetNums(wireContent)
	case "zu30":
		return countWuxingZu30BetNums(wireContent)
	case "zu20":
		return countWuxingZu20BetNums(wireContent)
	case "zu10", "zu5":
		return countWuxingZuZeroPoolBetNums(wireContent)
	default:
		return 0
	}
}
