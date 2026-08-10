package guajibet

import (
	"fmt"
	"regexp"
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
	// 方案 betMode 优先（与 InferBetMode 一致）：定码生肖对碰 ForcedBetMode=sx_dp
	// 时即便 label 残缺也须走号码展开，否则 wire 仍是「马|兔」→ 第三方「投注数字不合规」。
	if mode := normalizeForcedBetMode(meta.ForcedBetMode); mode != "" {
		return mode
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
	// 目录 rule id：二全中/二中特/特串·生肖/尾数/生尾对碰（防文案漂移漏判）
	if id, err := strconv.Atoi(strings.TrimSpace(meta.RuleID)); err == nil {
		switch id {
		case 281, 287, 293:
			return "sx_dp"
		case 282, 288, 294:
			return "ws_dp"
		case 283, 289, 295:
			return "sw_dp"
		}
	}
	if id, err := strconv.Atoi(strings.TrimSpace(meta.SubID)); err == nil {
		switch id {
		case 281, 287, 293:
			return "sx_dp"
		case 282, 288, 294:
			return "ws_dp"
		case 283, 289, 295:
			return "sw_dp"
		}
	}
	label := strings.TrimSpace(meta.Label)
	text := meta.combinedText()
	// 玩法名是「组名+子玩法名」，而组名本身可能含别的子玩法关键词：
	// 「五行家野家野」里「五行」先命中、「一肖尾数一肖」里「尾数」先命中，
	// 直接匹配整串会取到前缀里的词。先用去掉组名前缀的子玩法名判定，取不到再退回整串。
	if sub := lhcSubPlayName(meta); sub != "" {
		if m := lhcModeFromLabel(sub, text); m != "" {
			return m
		}
	}
	return lhcModeFromLabel(label, text)
}

// lhcSubPlayName 去掉组名/类型名前缀后的子玩法名，如「五行家野家野」→「家野」。
// 取不到（玩法名不以组名开头）时返回空串。
func lhcSubPlayName(meta RuleMeta) string {
	label := strings.TrimSpace(meta.Label)
	for _, prefix := range []string{strings.TrimSpace(meta.Group), strings.TrimSpace(meta.TypeLabel)} {
		if prefix == "" || prefix == label {
			continue
		}
		if rest := strings.TrimSpace(strings.TrimPrefix(label, prefix)); rest != "" && rest != label {
			return rest
		}
	}
	return ""
}

func lhcModeFromLabel(label, text string) string {
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
		{"十二", 12}, {"十一", 11}, {"十不中", 10}, {"10", 10}, {"十", 10},
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

// lhcBuzhongMinPick 全不中（g013）/ 多选中一（g014）的最少选号数。
//
// 一律取玩法名里的数字：2026-07-28 真实下单时第三方回传的约束与玩法名完全一致——
// rule 362「全不中12不中」→「最少投注12个号码」、364「全不中15不中」→「最少投注15个号码」、
// 376「十选中一」→「最少投注10个号码」、379「特平中二粒」→「最少投注2个号码」。
//
// 原先按 rule_id 硬编码，整张表错位一位（348「全不中5不中」被映射成 6，
// 350「6不中」映射成 7 …），既被第三方拒单，也让 CountBetNums 的注数与金额偏大。
func lhcBuzhongMinPick(meta RuleMeta) int {
	// 取完整上下文：拖头玩法的 label 常只是「拖头」，数字在 team/fullName 里。
	if n := lhcArabicPickCount(lhcContextText(meta)); n > 0 {
		return n
	}
	if n := lhcPickCountFromLabel(lhcContextText(meta)); n > 0 {
		return n
	}
	return 5
}

// lhcArabicPickCount 取玩法名中阿拉伯数字写法的选号数，如「全不中12不中」→ 12、「10选中一」→ 10。
// 「全不中5不中」里「不中」出现两次，只有带数字的那处会被匹配到。
func lhcArabicPickCount(label string) int {
	m := lhcArabicPickRe.FindStringSubmatch(label)
	if len(m) < 2 {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

var lhcArabicPickRe = regexp.MustCompile(`(\d+)\s*(?:不中|选中一|粒)`)

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
			case 279, 285, 291, 377: // 二全中/二中特/特串复式（旧误写 277）
				return 2
			case 277: // 兼容旧配置
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
		// 生产库 sub_id 是数字（如 379），matchLeadingInt 取不到，须回退玩法名。
		// 2026-07-28 实测：rule 379/381/383/385「特平中二/三/四/五粒任中」
		// 第三方要求「最少投注 2/3/4/5 个号码」，与玩法名一致；原先默认 1 个必被拒单。
		if n := lhcPickCountFromLabel(text); n > 0 {
			return n
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
	case "sx_dp":
		// 采样直接给第三方可用的号码 wire（方案侧仍可存生肖）
		return formatLHCSxDuipengWire("马|蛇")
	case "ws_dp":
		// 采样直接给第三方可用的号码 wire（方案侧仍可存尾数 0|1）
		return formatLHCWsDuipengWire("0|1")
	case "renyi_dp":
		return "01|02"
	case "sw_dp":
		return sampleLHCSwDuipengContent()
	case "tema", "zhengte":
		return "07"
	case "renzhong":
		return sampleLHCPickNumbers(lhcMinPickCount(meta, mode))
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
		case 279, 285, 291: // 二全中/二中特/特串复式
			return 2
		case 277: // 兼容旧配置
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

// sampleLHCSwDuipengContent 生尾对碰：生肖侧/尾数侧均展开为完整号码列表（目录 283/289/295）。
func sampleLHCSwDuipengContent() string {
	return formatLHCSwDuipengWire("鼠|0")
}

// ParseLHCSwDuipengSides 解析生尾对碰：恰好 1 生肖 + 1 尾（肖|尾 / 尾|肖 / 扁选）。
func ParseLHCSwDuipengSides(groupContent string) (zodiac string, tail int, ok bool) {
	s := strings.TrimSpace(groupContent)
	if s == "" {
		return "", 0, false
	}
	var tokens []string
	if strings.Contains(s, "|") || strings.Contains(s, "#") {
		sep := "|"
		if strings.Contains(s, "#") && !strings.Contains(s, "|") {
			sep = "#"
		}
		parts := strings.SplitN(s, sep, 2)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				tokens = append(tokens, p)
			}
		}
	} else {
		tokens = splitPickTokens(s)
	}
	var z string
	var t int
	hasZ, hasT := false, false
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if _, isZ := lhcWireZodiacNumbers[tok]; isZ {
			if hasZ {
				return "", 0, false
			}
			z = tok
			hasZ = true
			continue
		}
		tails := splitLHCTailTokens(tok)
		if len(tails) == 1 {
			if hasT {
				return "", 0, false
			}
			t = tails[0]
			hasT = true
			continue
		}
		return "", 0, false
	}
	if !hasZ || !hasT {
		return "", 0, false
	}
	return z, t, true
}

// countLHCSwDuipengBetNums 生尾对碰注数：两侧展开号码笛卡尔积，减去同号（左右皆含）对数。
// 实测 狗|5 展开含 45|…45 时 bets=20 拒「投注注数不正确」，bets=19 过。
func countLHCSwDuipengBetNums(wireContent string) int {
	wire := formatLHCSwDuipengWire(wireContent)
	if !strings.Contains(wire, "|") {
		return 0
	}
	parts := strings.SplitN(wire, "|", 2)
	left := parseLHCWireNumberSide(parts[0])
	right := parseLHCWireNumberSide(parts[1])
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	rightSet := make(map[int]struct{}, len(right))
	for _, n := range right {
		rightSet[n] = struct{}{}
	}
	overlap := 0
	for _, n := range left {
		if _, ok := rightSet[n]; ok {
			overlap++
		}
	}
	return len(left)*len(right) - overlap
}

func parseLHCWireNumberSide(side string) []int {
	out := make([]int, 0, 8)
	seen := make(map[int]struct{}, 8)
	for _, tok := range splitLHCBarSide(side) {
		n, err := strconv.Atoi(strings.TrimSpace(tok))
		if err != nil || n < 1 || n > 49 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// formatLHCSwDuipengWire 生尾对碰：方案可存「马|0」，第三方下单须展开为号码列表，
// 如「01,13,25,37,49|10,20,30,40」。已是号码 wire 时两侧按号码规范化。
func formatLHCSwDuipengWire(groupContent string) string {
	if z, t, ok := ParseLHCSwDuipengSides(groupContent); ok {
		left := lhcWireZodiacNumbers[z]
		right := lhcWireTailNumbers(t)
		if len(left) == 0 || len(right) == 0 {
			return ""
		}
		return formatLHCWireNumbers(left) + "|" + formatLHCWireNumbers(right)
	}
	s := strings.TrimSpace(groupContent)
	if s == "" || (!strings.Contains(s, "|") && !strings.Contains(s, "#")) {
		return ""
	}
	sep := "|"
	if strings.Contains(s, "#") && !strings.Contains(s, "|") {
		sep = "#"
	}
	parts := strings.SplitN(s, sep, 2)
	left := expandLHCSxDuipengSide(parts[0]) // 生肖或号码
	if len(left) == 0 {
		left = expandLHCWsDuipengSide(parts[0])
	}
	rightRaw := ""
	if len(parts) > 1 {
		rightRaw = parts[1]
	}
	right := expandLHCWsDuipengSide(rightRaw) // 尾或号码
	if len(right) == 0 {
		right = expandLHCSxDuipengSide(rightRaw)
	}
	if len(left) == 0 || len(right) == 0 {
		return ""
	}
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

// 特码 wire：号码|属性|波色（第三方示例含全选三段）。
var lhcTemaAttrOrder = []string{
	"尾双", "尾单", "尾小", "尾大", "总分大", "总分小", "合小", "合大",
	"大", "小", "单", "双", "合双", "合单", "总分单", "总分双",
}

var lhcTemaWaveOrder = []string{"红波", "蓝波", "绿波"}

func lhcTemaAliasToken(t string) string {
	t = strings.TrimSpace(t)
	t = strings.TrimSuffix(t, "||")
	t = strings.TrimSpace(t)
	switch t {
	case "洪波":
		return "红波"
	case "绿播":
		return "绿波"
	default:
		return t
	}
}

func isLHCTemaAttrToken(t string) bool {
	for _, a := range lhcTemaAttrOrder {
		if t == a {
			return true
		}
	}
	return false
}

func isLHCTemaWaveToken(t string) bool {
	for _, w := range lhcTemaWaveOrder {
		if t == w {
			return true
		}
	}
	return false
}

// ParseLHCTemaParts 解析特码内容为 号码 / 属性 / 波色（兼容旧逗号混选与 07||,13||）。
func ParseLHCTemaParts(groupContent string) (nums, attrs, waves []string) {
	return parseLHCTemaParts(groupContent)
}

func parseLHCTemaParts(groupContent string) (nums, attrs, waves []string) {
	groupContent = strings.TrimSpace(groupContent)
	if groupContent == "" {
		return nil, nil, nil
	}
	numSet := map[string]struct{}{}
	attrSet := map[string]struct{}{}
	waveSet := map[string]struct{}{}
	take := func(raw string) {
		t := lhcTemaAliasToken(raw)
		if t == "" {
			return
		}
		if isLHCTemaWaveToken(t) {
			waveSet[t] = struct{}{}
			return
		}
		if isLHCTemaAttrToken(t) {
			attrSet[t] = struct{}{}
			return
		}
		n, err := strconv.Atoi(t)
		// 特码号池仅 1–49；00 会被第三方拒「投注数字不合规」
		if err != nil || n < 1 || n > 49 {
			return
		}
		numSet[fmt.Sprintf("%02d", n)] = struct{}{}
	}

	if strings.Contains(groupContent, "|") {
		// 旧多注：07||,13||
		if strings.Contains(groupContent, "||,") || strings.Contains(groupContent, ",||") {
			for _, part := range splitPickTokens(groupContent) {
				take(part)
			}
		} else {
			sections := strings.Split(groupContent, "|")
			for _, sec := range sections {
				for _, part := range splitPickTokens(sec) {
					take(part)
				}
			}
		}
	} else {
		for _, part := range splitPickTokens(groupContent) {
			take(part)
		}
	}

	for i := 1; i <= 49; i++ {
		k := fmt.Sprintf("%02d", i)
		if _, ok := numSet[k]; ok {
			nums = append(nums, k)
		}
	}
	for _, a := range lhcTemaAttrOrder {
		if _, ok := attrSet[a]; ok {
			attrs = append(attrs, a)
		}
	}
	for _, w := range lhcTemaWaveOrder {
		if _, ok := waveSet[w]; ok {
			waves = append(waves, w)
		}
	}
	return nums, attrs, waves
}

// expandLHCSxDuipengSide 生肖对碰单侧：生肖→号码列表；已是号码则规范化去重。
func expandLHCSxDuipengSide(side string) []int {
	side = strings.TrimSpace(side)
	if side == "" {
		return nil
	}
	if zs := splitLHCZodiacTokens(side); len(zs) > 0 {
		out := make([]int, 0, 8)
		seen := map[int]struct{}{}
		for _, z := range zs {
			for _, n := range lhcWireZodiacNumbers[z] {
				if _, ok := seen[n]; ok {
					continue
				}
				seen[n] = struct{}{}
				out = append(out, n)
			}
		}
		return out
	}
	out := make([]int, 0, 8)
	seen := map[int]struct{}{}
	for _, t := range splitLHCBarSide(side) {
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil || n < 1 || n > 49 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// formatLHCSxDuipengWire 生肖对碰：方案可存「马|蛇」，第三方下单须展开为号码列表
// （与生尾对碰一致），如「01,13,25,37,49|02,14,26,38」。
func formatLHCSxDuipengWire(groupContent string) string {
	s := strings.TrimSpace(groupContent)
	if s == "" {
		return ""
	}
	var leftRaw, rightRaw string
	if strings.Contains(s, "|") || strings.Contains(s, "#") {
		sep := "|"
		if strings.Contains(s, "#") && !strings.Contains(s, "|") {
			sep = "#"
		}
		parts := strings.SplitN(s, sep, 2)
		leftRaw = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			rightRaw = strings.TrimSpace(parts[1])
		}
	} else {
		zs := splitLHCZodiacTokens(s)
		if len(zs) < 2 {
			return ""
		}
		leftRaw, rightRaw = zs[0], zs[1]
	}
	if leftRaw == "" || rightRaw == "" {
		return ""
	}
	left := expandLHCSxDuipengSide(leftRaw)
	right := expandLHCSxDuipengSide(rightRaw)
	if len(left) == 0 || len(right) == 0 {
		return ""
	}
	return formatLHCWireNumbers(left) + "|" + formatLHCWireNumbers(right)
}

// expandLHCWsDuipengSide 尾数对碰单侧：尾数 0–9 → 号码列表；已是号码则规范化去重。
// 0 尾：10,20,30,40；1–9 尾：各 5 个（如 1→01,11,21,31,41）。
func expandLHCWsDuipengSide(side string) []int {
	side = strings.TrimSpace(side)
	if side == "" {
		return nil
	}
	if tails := splitLHCTailTokens(side); len(tails) > 0 {
		out := make([]int, 0, 8)
		seen := map[int]struct{}{}
		for _, t := range tails {
			for _, n := range lhcWireTailNumbers(t) {
				if _, ok := seen[n]; ok {
					continue
				}
				seen[n] = struct{}{}
				out = append(out, n)
			}
		}
		return out
	}
	out := make([]int, 0, 8)
	seen := map[int]struct{}{}
	for _, tok := range splitLHCBarSide(side) {
		n, err := strconv.Atoi(strings.TrimSpace(tok))
		if err != nil || n < 1 || n > 49 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// formatLHCWsDuipengWire 尾数对碰：方案可存「0|1」，第三方下单须展开为号码列表
// （与生肖对碰一致），如「10,20,30,40|01,11,21,31,41」。
func formatLHCWsDuipengWire(groupContent string) string {
	s := strings.TrimSpace(groupContent)
	if s == "" {
		return ""
	}
	var leftRaw, rightRaw string
	if strings.Contains(s, "|") || strings.Contains(s, "#") {
		sep := "|"
		if strings.Contains(s, "#") && !strings.Contains(s, "|") {
			sep = "#"
		}
		parts := strings.SplitN(s, sep, 2)
		leftRaw = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			rightRaw = strings.TrimSpace(parts[1])
		}
	} else {
		tails := splitLHCTailTokens(s)
		if len(tails) < 2 {
			return ""
		}
		leftRaw, rightRaw = strconv.Itoa(tails[0]), strconv.Itoa(tails[1])
	}
	if leftRaw == "" || rightRaw == "" {
		return ""
	}
	left := expandLHCWsDuipengSide(leftRaw)
	right := expandLHCWsDuipengSide(rightRaw)
	if len(left) == 0 || len(right) == 0 {
		return ""
	}
	return formatLHCWireNumbers(left) + "|" + formatLHCWireNumbers(right)
}

func splitLHCTailTokens(raw string) []int {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\n' || r == '\r' || r == '\t' || r == '|' || r == '#'
	})
	seen := map[int]struct{}{}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimSuffix(p, "尾")
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 9 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func splitLHCZodiacTokens(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\n' || r == '\r' || r == '\t' || r == '|' || r == '#'
	})
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := lhcWireZodiacNumbers[p]; !ok {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// formatLHCTemaWire 特码/正特：第三方 wire 为 号码|属性|波色。
func formatLHCTemaWire(groupContent string) string {
	nums, attrs, waves := parseLHCTemaParts(groupContent)
	if len(nums) == 0 && len(attrs) == 0 && len(waves) == 0 {
		return ""
	}
	return strings.Join(nums, ",") + "|" + strings.Join(attrs, ",") + "|" + strings.Join(waves, ",")
}

// formatLHCTuotouSide 拖头单侧：数字补零，生肖/尾数等原文保留；逗号分隔。
func formatLHCTuotouSide(part string) string {
	tokens := splitLHCBarSide(part)
	if len(tokens) == 0 {
		return ""
	}
	out := make([]string, 0, len(tokens))
	seen := map[string]struct{}{}
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if n, err := strconv.Atoi(t); err == nil && n >= 1 && n <= 49 {
			t = fmt.Sprintf("%02d", n)
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return strings.Join(out, ",")
}

// formatLHCTuotouWire 拖头：第三方 wire 为 胆|拖。
// 方案扁选「01,13,25」→「01|13,25」（首号胆，其余拖）；已是 胆|拖 / 胆#拖 则规范化补零。
func formatLHCTuotouWire(groupContent string) string {
	s := strings.TrimSpace(groupContent)
	if s == "" {
		return ""
	}
	sep := ""
	if strings.Contains(s, "|") {
		sep = "|"
	} else if strings.Contains(s, "#") {
		sep = "#"
	}
	if sep != "" {
		parts := strings.SplitN(s, sep, 2)
		dan := formatLHCTuotouSide(parts[0])
		tuo := ""
		if len(parts) > 1 {
			tuo = formatLHCTuotouSide(parts[1])
		}
		if dan == "" && tuo == "" {
			return ""
		}
		return dan + "|" + tuo
	}
	// 扁选号码：仅当全部为 1–49 时合成胆|拖；否则原样返回（避免误伤生肖等）
	rawParts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})
	nums := make([]string, 0, len(rawParts))
	seen := map[string]struct{}{}
	for _, p := range rawParts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 49 {
			return s
		}
		t := fmt.Sprintf("%02d", n)
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		nums = append(nums, t)
	}
	if len(nums) < 2 {
		if len(nums) == 1 {
			return nums[0]
		}
		return ""
	}
	return nums[0] + "|" + strings.Join(nums[1:], ",")
}

func countLHCTemaBetNums(wireContent string) int {
	nums, attrs, waves := parseLHCTemaParts(wireContent)
	n := len(nums) + len(attrs) + len(waves)
	if n == 0 && strings.TrimSpace(wireContent) != "" {
		return 1
	}
	return n
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
		needTuo := lhcTuotouMinTuoCount(meta)
		dan, tuo := []string(nil), []string(nil)
		if strings.Contains(wireContent, "|") {
			parts := strings.SplitN(wireContent, "|", 2)
			dan = splitLHCBarSide(parts[0])
			tuo = splitLHCBarSide(parts[1])
		} else if strings.Contains(wireContent, "#") {
			parts := strings.SplitN(wireContent, "#", 2)
			dan = splitLHCBarSide(parts[0])
			tuo = splitLHCBarSide(parts[1])
		} else if len(tokens) >= 2 {
			// 扁选：首号胆，其余拖（与 formatLHCTuotouWire / 前端一致）
			dan = []string{tokens[0]}
			tuo = tokens[1:]
		}
		if len(dan) == 0 || len(tuo) < needTuo {
			return 0
		}
		if lhcTuotouBetsAlwaysOne(meta) {
			return 1
		}
		return len(dan) * comboCount(len(tuo), needTuo)
	case "sx_dp":
		// 注数=两侧展开号码数之积（马5×其它4=20；两肖均非马=16）；wire 已是号码列表
		wire := formatLHCSxDuipengWire(wireContent)
		if !strings.Contains(wire, "|") {
			return 0
		}
		parts := strings.SplitN(wire, "|", 2)
		left := len(splitLHCBarSide(parts[0]))
		right := len(splitLHCBarSide(parts[1]))
		if left <= 0 || right <= 0 {
			return 0
		}
		return left * right
	case "ws_dp":
		// 注数=两侧展开号码数之积（0尾4×1尾5=20；1尾×2尾=25）
		wire := formatLHCWsDuipengWire(wireContent)
		if !strings.Contains(wire, "|") {
			return 0
		}
		parts := strings.SplitN(wire, "|", 2)
		left := len(splitLHCBarSide(parts[0]))
		right := len(splitLHCBarSide(parts[1]))
		if left <= 0 || right <= 0 {
			return 0
		}
		return left * right
	case "renyi_dp":
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
		// 注数=左×右 − 两侧共有号码数（第三方：狗|5 → 4×5−1=19；无交集如鼠|0=16）
		return countLHCSwDuipengBetNums(wireContent)
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
