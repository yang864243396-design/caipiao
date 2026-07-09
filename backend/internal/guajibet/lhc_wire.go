package guajibet

import (
	"fmt"
	"strconv"
	"strings"
)

var lhcZodiacSamples = []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}

// 2026 马年生肖号码（与 schemes/lhc_constants 对齐，供 guajibet wire 采样）。
var lhcWireZodiacNumbers = map[string][]int{
	"马": {1, 13, 25, 37, 49},
	"蛇": {2, 14, 26, 38},
	"龙": {3, 15, 27, 39},
	"兔": {4, 16, 28, 40},
	"虎": {5, 17, 29, 41},
	"牛": {6, 18, 30, 42},
	"鼠": {7, 19, 31, 43},
	"猪": {8, 20, 32, 44},
	"狗": {9, 21, 33, 45},
	"鸡": {10, 22, 34, 46},
	"猴": {11, 23, 35, 47},
	"羊": {12, 24, 36, 48},
}

func inferLHCBetMode(meta RuleMeta) string {
	if meta.PlayTemplate != "lhc_std" {
		return ""
	}
	switch strings.TrimSpace(meta.TypeID) {
	case "g001":
		return "tema"
	case "g002":
		return "zhengte"
	case "g013":
		if strings.Contains(meta.Label, "复式") {
			return "buzhong"
		}
	case "g014":
		if strings.Contains(meta.Label, "复式") {
			return "xuanyi"
		}
	}
	label := strings.TrimSpace(meta.Label)
	text := meta.combinedText()
	switch {
	case label == "复式" || strings.Contains(label, "复式"):
		return "fushi"
	case strings.Contains(label, "拖头"):
		return "tuotou"
	case strings.Contains(label, "生肖对碰"):
		return "sx_dp"
	case strings.Contains(label, "尾数对碰"):
		return "ws_dp"
	case strings.Contains(label, "生尾对碰"):
		return "sw_dp"
	case strings.Contains(label, "任意对碰"):
		return "renyi_dp"
	case strings.Contains(label, "特肖"):
		return "texiao"
	case strings.Contains(label, "总肖"):
		return "zongxiao"
	case strings.Contains(label, "特码头尾"):
		return "tematouwei"
	case strings.Contains(label, "过关"):
		return "guoguan"
	case strings.Contains(label, "七码"):
		return "qima"
	case strings.Contains(label, "任中"):
		return "renzhong"
	case strings.Contains(label, "尾数"):
		if strings.Contains(label, "不中") {
			return "wei_bz"
		}
		return "weishu"
	case strings.Contains(label, "肖"):
		if strings.Contains(label, "不中") {
			return "xiao_bz"
		}
		if strings.Contains(text, "中") && !strings.Contains(label, "不中") {
			return "xiao_z"
		}
		return "xiao"
	case strings.Contains(label, "不中"):
		return "buzhong"
	case strings.Contains(label, "选中一"):
		return "xuanyi"
	case strings.Contains(label, "五行"):
		return "wuxing"
	case strings.Contains(label, "家野"):
		return "jiaye"
	case strings.Contains(label, "半半波"):
		return "banbanbo"
	case strings.Contains(label, "半波"):
		return "banbo"
	case strings.Contains(label, "波色") || label == "波色":
		return "bose"
	}
	return ""
}

func lhcContextText(meta RuleMeta) string {
	return meta.TypeLabel + meta.TeamLabel + meta.Label + meta.FullName + meta.TypeID
}

func lhcPickCountFromLabel(text string) int {
	pairs := []struct {
		key string
		n   int
	}{
		{"十五", 15}, {"十五不中", 15},
		{"十二", 12}, {"十一", 11}, {"十不中", 10}, {"10", 10},
		{"九", 9}, {"八", 8}, {"七", 7}, {"六", 6}, {"五", 5},
		{"四", 4}, {"三", 3}, {"二", 2}, {"一", 1},
	}
	for _, p := range pairs {
		if strings.Contains(text, p.key+"肖") || strings.Contains(text, p.key+"尾") || strings.Contains(text, p.key+"粒") {
			return p.n
		}
	}
	for _, p := range pairs {
		if strings.Contains(text, p.key+"不中") || strings.Contains(text, p.key+"选中一") || strings.Contains(text, p.key+"x1") {
			return p.n
		}
	}
	return 0
}

func lhcBuzhongMinPick(meta RuleMeta) int {
	if id, err := strconv.Atoi(strings.TrimSpace(meta.RuleID)); err == nil {
		// 拖头 outbound 为奇数 rule_id，与复式 sub_id 成对（347↔346）。
		if strings.Contains(meta.Label, "拖头") && id%2 == 1 {
			id--
		}
		switch strings.TrimSpace(meta.TypeID) {
		case "g013":
			switch id {
			case 346:
				return 5
			case 348:
				return 6
			case 350:
				return 7
			case 352:
				return 8
			case 354:
				return 9
			case 356:
				return 10
			case 358:
				return 11
			case 360:
				return 12
			case 362:
				return 15
			}
		case "g014":
			switch id {
			case 364:
				return 5
			case 366:
				return 6
			case 368:
				return 7
			case 370:
				return 8
			case 372:
				return 9
			case 374:
				return 10
			}
		}
	}
	if n := lhcPickCountFromLabel(lhcContextText(meta)); n > 0 {
		return n
	}
	return 5
}

func lhcMinPickCount(meta RuleMeta, betMode string) int {
	sub := strings.ToLower(strings.TrimSpace(meta.SubID))
	text := lhcContextText(meta)
	switch betMode {
	case "fushi":
		if k := lhcFushiComboSize(meta); k > 0 {
			switch strings.TrimSpace(meta.Group) {
			case "生肖连", "尾数连", "特平中", "连码":
				return k
			}
		}
		if id, err := strconv.Atoi(strings.TrimSpace(meta.RuleID)); err == nil {
			switch id {
			case 295, 297, 376:
				return 3
			case 277, 283, 289, 377:
				return 2
			}
		}
		if strings.Contains(text, "三全中") {
			return 3
		}
		if strings.Contains(text, "三") {
			return 3
		}
		return 2
	case "buzhong":
		return lhcBuzhongMinPick(meta)
	case "xuanyi":
		if n := lhcBuzhongMinPick(meta); n > 0 && strings.TrimSpace(meta.TypeID) == "g014" {
			return n
		}
		if m := matchLeadingInt(sub, "x1"); m > 0 {
			return m
		}
		if n := lhcPickCountFromLabel(text); n > 0 {
			return n
		}
		return 5
	case "renzhong":
		if m := matchLeadingInt(sub, "l_rz"); m > 0 {
			return m
		}
		return 1
	case "xiao", "xiao_z", "xiao_bz", "wei_z", "wei_bz":
		if n := lhcPickCountFromLabel(meta.Label); n > 0 {
			return n
		}
		if m := matchLeadingInt(sub, "xiao"); m > 0 {
			return m
		}
		if m := matchLeadingInt(sub, "wei"); m > 0 {
			return m
		}
		return 1
	default:
		return 1
	}
}

func matchLeadingInt(s, suffix string) int {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, suffix) {
		return 0
	}
	prefix := strings.TrimSuffix(s, suffix)
	n := 0
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func sampleLHCPickNumbers(n int) string {
	if n < 1 {
		n = 1
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("%02d", i+1)
	}
	return strings.Join(parts, ",")
}

func sampleLHCZodiacPicks(n int) string {
	if n <= 0 {
		n = 1
	}
	if n > len(lhcZodiacSamples) {
		n = len(lhcZodiacSamples)
	}
	return strings.Join(lhcZodiacSamples[:n], ",")
}

func sampleLHCGroupContent(meta RuleMeta) string {
	mode := inferLHCBetMode(meta)
	switch mode {
	case "fushi":
		if strings.TrimSpace(meta.TypeID) == "g013" || strings.TrimSpace(meta.TypeID) == "g014" {
			return sampleLHCPickNumbers(lhcBuzhongMinPick(meta))
		}
		return sampleLHCFushiContent(meta)
	case "buzhong", "xuanyi":
		n := lhcMinPickCount(meta, mode)
		if n < 1 {
			n = 2
		}
		return sampleLHCPickNumbers(n)
	case "tuotou":
		return sampleLHCTuotouContent(meta)
	case "sx_dp", "renyi_dp", "ws_dp":
		return "01|02"
	case "sw_dp":
		return sampleLHCSwDuipengContent()
	case "tema", "zhengte":
		return "07"
	case "renzhong":
		return "01"
	case "texiao":
		return "鼠"
	case "xiao", "xiao_z", "xiao_bz":
		return sampleLHCZodiacPicks(lhcMinPickCount(meta, mode))
	case "weishu", "wei_z", "wei_bz":
		return "0尾"
	case "zongxiao":
		return "二肖"
	case "tematouwei":
		return "0|1"
	case "wuxing":
		return "金"
	case "jiaye":
		return "家禽"
	case "bose":
		return "红波"
	case "banbo":
		return "红大"
	case "banbanbo":
		return "红大单"
	case "guoguan":
		return "大,小"
	case "qima":
		return "双1"
	}
	if strings.Contains(meta.combinedText(), "肖") {
		return "鼠"
	}
	return "01"
}

func lhcFushiComboSize(meta RuleMeta) int {
	group := strings.TrimSpace(meta.Group)
	if group == "生肖连" || group == "尾数连" || group == "特平中" {
		if n := lhcTeamMinPick(meta); n > 0 {
			return n
		}
	}
	if id, err := strconv.Atoi(strings.TrimSpace(meta.RuleID)); err == nil {
		switch id {
		case 295, 297:
			return 3
		case 376:
			return 1
		case 377:
			return 2
		case 277, 283, 289:
			return 2
		}
	}
	text := lhcContextText(meta)
	switch {
	case strings.Contains(text, "三全中"):
		return 3
	case strings.Contains(text, "三中二"), strings.Contains(text, "二全中"), strings.Contains(text, "二中特"), strings.Contains(text, "特串"):
		return 2
	default:
		if n := lhcPickCountFromLabel(text); n > 0 {
			return n
		}
		return 2
	}
}

func lhcFushiSamplePickCount(meta RuleMeta) int {
	k := lhcFushiComboSize(meta)
	if id, err := strconv.Atoi(strings.TrimSpace(meta.RuleID)); err == nil && id == 376 {
		return 3
	}
	return k
}

func sampleLHCTailPicks(n int) string {
	if n <= 0 {
		n = 1
	}
	picks := make([]string, n)
	for i := range picks {
		picks[i] = fmt.Sprintf("%d尾", i)
	}
	return strings.Join(picks, ",")
}

func lhcWireTailNumbers(tail int) []int {
	var out []int
	for n := 1; n <= 49; n++ {
		if n%10 == tail {
			out = append(out, n)
		}
	}
	return out
}

func formatLHCWireNumbers(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = fmt.Sprintf("%02d", n)
	}
	return strings.Join(parts, ",")
}

// sampleLHCSwDuipengContent 生尾对碰：生肖侧/尾数侧均展开为完整号码列表（bet-probe 281/287/293）。
func sampleLHCSwDuipengContent() string {
	zodiac := "鼠"
	tail := 0
	left := lhcWireZodiacNumbers[zodiac]
	right := lhcWireTailNumbers(tail)
	return formatLHCWireNumbers(left) + "|" + formatLHCWireNumbers(right)
}

// formatLHCTematouweiWire 特码头尾：第三方 wire 为 headIndex|tailIndex（bet-probe 307）。
// 平台侧选号仍用 头0/尾0；仅头或仅尾时分别为 N| 或 |N，注数 1；双侧为 N|M，注数 2。
func formatLHCTematouweiWire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return ""
	}
	if strings.Contains(groupContent, "|") && !strings.Contains(groupContent, "头") && !strings.Contains(groupContent, "尾") {
		return groupContent
	}
	tokens := splitPickTokens(groupContent)
	var head, tail string
	hasHead, hasTail := false, false
	for _, t := range tokens {
		switch {
		case strings.HasPrefix(t, "头"):
			head = strings.TrimPrefix(t, "头")
			hasHead = true
		case strings.HasPrefix(t, "尾"):
			tail = strings.TrimPrefix(t, "尾")
			hasTail = true
		}
	}
	switch {
	case hasHead && hasTail:
		return head + "|" + tail
	case hasHead:
		return head + "|"
	case hasTail:
		return "|" + tail
	default:
		return groupContent
	}
}

func countLHCTematouweiBetNums(wireContent string) int {
	wireContent = strings.TrimSpace(wireContent)
	if wireContent == "" {
		return 0
	}
	if !strings.Contains(wireContent, "|") {
		return 1
	}
	parts := strings.SplitN(wireContent, "|", 2)
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	if left != "" && right != "" {
		return 2
	}
	return 1
}

var lhcZongxiaoCN = []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九", "十", "十一", "十二"}

// formatLHCTemaWire 特码/正特：第三方 wire 为 NN||（浏览器抓包 rule 385/271）。
func formatLHCTemaWire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return ""
	}
	if strings.Contains(groupContent, "||") {
		return groupContent
	}
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return groupContent
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if strings.HasSuffix(t, "||") {
			parts = append(parts, t)
			continue
		}
		if n, err := strconv.Atoi(t); err == nil && n >= 1 && n <= 49 {
			parts = append(parts, fmt.Sprintf("%02d||", n))
			continue
		}
		parts = append(parts, t+"||")
	}
	return strings.Join(parts, ",")
}

func countLHCTemaBetNums(wireContent string) int {
	wireContent = strings.TrimSpace(wireContent)
	if wireContent == "" {
		return 0
	}
	parts := splitPickTokens(wireContent)
	if len(parts) == 0 {
		return 1
	}
	return len(parts)
}

func lhcZongxiaoWireLabel(count int) (string, bool) {
	if count < 2 || count > 7 {
		return "", false
	}
	return lhcZongxiaoCN[count] + "肖", true
}

func isLHCZongxiaoWireLabel(token string) bool {
	token = strings.TrimSpace(token)
	for n := 2; n <= 7; n++ {
		if label, ok := lhcZongxiaoWireLabel(n); ok && label == token {
			return true
		}
	}
	return false
}

// formatLHCZongxiaoWire 总肖：第三方 wire 为「二肖」–「七肖」（rule 301，共 6 项）。
func formatLHCZongxiaoWire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return ""
	}
	tokens := splitPickTokens(groupContent)
	if len(tokens) == 0 {
		return groupContent
	}
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if isLHCZongxiaoWireLabel(t) {
			out = append(out, t)
			continue
		}
		if n, err := strconv.Atoi(t); err == nil {
			if label, ok := lhcZongxiaoWireLabel(n); ok {
				out = append(out, label)
			}
			continue
		}
		if strings.HasSuffix(t, "肖") {
			if isLHCZongxiaoWireLabel(t) {
				out = append(out, t)
			}
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, ",")
}

var lhcQimaKinds = []string{"单", "双", "大", "小"}

func lhcQimaOptions() []string {
	out := make([]string, 0, 32)
	for _, kind := range lhcQimaKinds {
		for n := 0; n <= 7; n++ {
			out = append(out, fmt.Sprintf("%s%d", kind, n))
		}
	}
	return out
}

func isLHCQimaOption(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	for _, kind := range lhcQimaKinds {
		if !strings.HasPrefix(token, kind) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(token, kind))
		if err == nil && n >= 0 && n <= 7 {
			return true
		}
	}
	return false
}

// formatLHCQimaWire 七码：第三方 wire 为「双1」等 32 项选项文案（rule 313）。
func formatLHCQimaWire(groupContent string) string {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return "双1"
	}
	tokens := splitPickTokens(groupContent)
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if isLHCQimaOption(t) {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return formatTextTokens(groupContent)
	}
	return strings.Join(out, ",")
}

func sampleLHCFushiContent(meta RuleMeta) string {
	n := lhcFushiSamplePickCount(meta)
	if n < 1 {
		n = 2
	}
	switch strings.TrimSpace(meta.Group) {
	case "生肖连":
		return sampleLHCZodiacPicks(n)
	case "尾数连":
		return sampleLHCTailPicks(n)
	default:
		return sampleLHCPickNumbers(n)
	}
}

func countLHCBetNums(meta RuleMeta, wireContent string) int {
	mode := inferLHCBetMode(meta)
	wireContent = strings.TrimSpace(wireContent)
	if wireContent == "" {
		return 0
	}
	tokens := splitPickTokens(wireContent)
	switch mode {
	case "fushi", "buzhong", "xuanyi":
		if len(tokens) == 0 {
			return 0
		}
		if mode == "fushi" {
			min := lhcMinPickCount(meta, mode)
			if len(tokens) < min {
				return 0
			}
			k := lhcFushiComboSize(meta)
			if n := comboCount(len(tokens), k); n > 0 {
				return n
			}
		}
		min := lhcMinPickCount(meta, mode)
		if len(tokens) < min {
			return 0
		}
		if n := comboCount(len(tokens), min); n > 0 {
			return n
		}
		return len(tokens)
	case "tuotou":
		if strings.Contains(wireContent, "|") {
			parts := strings.SplitN(wireContent, "|", 2)
			dan := splitLHCBarSide(parts[0])
			tuo := splitLHCBarSide(parts[1])
			needTuo := lhcTuotouMinTuoCount(meta)
			if len(dan) == 0 || len(tuo) < needTuo {
				return 0
			}
			if lhcTuotouBetsAlwaysOne(meta) {
				return 1
			}
			return len(dan) * comboCount(len(tuo), needTuo)
		}
		if len(tokens) == 0 {
			return 0
		}
		return len(tokens)
	case "sx_dp", "ws_dp", "renyi_dp":
		if strings.Contains(wireContent, "|") {
			parts := strings.SplitN(wireContent, "|", 2)
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			if left == "" || right == "" {
				return 0
			}
			return 1
		}
		return 0
	case "sw_dp":
		if strings.Contains(wireContent, "|") {
			parts := strings.SplitN(wireContent, "|", 2)
			left := splitPickTokens(parts[0])
			right := splitPickTokens(parts[1])
			if len(left) == 0 || len(right) == 0 {
				return 0
			}
			return len(left) * len(right)
		}
		return 0
	case "tematouwei":
		return countLHCTematouweiBetNums(wireContent)
	case "tema", "zhengte":
		return countLHCTemaBetNums(wireContent)
	case "zongxiao":
		if len(tokens) == 0 {
			if wireContent != "" {
				return 1
			}
			return 0
		}
		return len(tokens)
	case "qima":
		if len(tokens) == 0 {
			return 0
		}
		return 1
	default:
		if len(tokens) == 0 {
			if wireContent != "" {
				return 1
			}
			return 0
		}
		return 1
	}
}

func sampleDistinctDigitString(segLen int) string {
	if segLen <= 0 {
		segLen = 1
	}
	digits := make([]byte, segLen)
	for i := range digits {
		digits[i] = byte('1' + i)
	}
	return string(digits)
}

func lhcTeamMinPick(meta RuleMeta) int {
	text := lhcContextText(meta)
	if n := lhcPickCountFromLabel(text); n > 0 {
		return n
	}
	switch {
	case strings.Contains(text, "三全中"), strings.Contains(text, "三中二"), strings.Contains(text, "三肖"):
		return 3
	default:
		return 2
	}
}

// lhcTuotouMinTuoCount 拖尾最少个数（胆固定 1 个）。
func lhcTuotouMinTuoCount(meta RuleMeta) int {
	switch strings.TrimSpace(meta.TypeID) {
	case "g013", "g014":
		n := lhcBuzhongMinPick(meta)
		if n > 1 {
			return n - 1
		}
		return 1
	default:
		n := lhcTeamMinPick(meta)
		if n > 1 {
			return n - 1
		}
		return 1
	}
}

func lhcTuotouSampleTuoCount(meta RuleMeta) int {
	min := lhcTuotouMinTuoCount(meta)
	group := strings.TrimSpace(meta.Group)
	typeID := strings.TrimSpace(meta.TypeID)
	if group == "连码" || typeID == "g003" {
		teamMin := lhcTeamMinPick(meta)
		if teamMin <= 2 {
			if min < 2 {
				return 2
			}
			return min
		}
		return min + 1
	}
	return min
}

func lhcTuotouBetsAlwaysOne(meta RuleMeta) bool {
	switch strings.TrimSpace(meta.Group) {
	case "生肖连", "尾数连", "特平中":
		return true
	default:
		return false
	}
}

func sampleLHCTuotouZodiac(tuoCount int) string {
	if tuoCount < 1 {
		tuoCount = 1
	}
	dan := lhcZodiacSamples[0]
	tuo := lhcZodiacSamples[1 : 1+tuoCount]
	if len(tuo) < tuoCount {
		tuo = lhcZodiacSamples[1:]
	}
	return dan + "|" + strings.Join(tuo, ",")
}

func sampleLHCTuotouTail(tuoCount int) string {
	if tuoCount < 1 {
		tuoCount = 1
	}
	tuo := make([]string, tuoCount)
	for i := range tuo {
		tuo[i] = fmt.Sprintf("%d尾", i+1)
	}
	return "0尾|" + strings.Join(tuo, ",")
}

func sampleLHCTuotouNumbers(tuoCount int) string {
	if tuoCount < 1 {
		tuoCount = 1
	}
	tuo := make([]string, tuoCount)
	for i := range tuo {
		tuo[i] = fmt.Sprintf("%02d", i+2)
	}
	return "01|" + strings.Join(tuo, ",")
}

func sampleLHCTuotouContent(meta RuleMeta) string {
	n := lhcTuotouSampleTuoCount(meta)
	switch strings.TrimSpace(meta.Group) {
	case "生肖连":
		return sampleLHCTuotouZodiac(n)
	case "尾数连":
		return sampleLHCTuotouTail(n)
	default:
		return sampleLHCTuotouNumbers(n)
	}
}

func splitLHCBarSide(part string) []string {
	part = strings.TrimSpace(part)
	if part == "" {
		return nil
	}
	parts := strings.Split(part, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
