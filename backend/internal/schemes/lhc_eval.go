package schemes

import (
	"strconv"
	"strings"

	"caipiao/backend/internal/guajibet"
)

func evaluateLHCByBetMode(rule playRule, balls []string, content string) (betEvaluation, bool) {
	mode := strings.TrimSpace(rule.BetMode)
	if mode == "" {
		return betEvaluation{}, false
	}
	switch mode {
	case "tema":
		return evaluateLHCTema(rule, balls, content), true
	case "zhengte":
		return evaluateLHCZhengte(rule, balls, content), true
	case "fushi":
		return evaluateLHCFushi(rule, balls, content), true
	case "tuotou":
		return evaluateLHCTuotou(rule, balls, content), true
	case "sx_dp", "ws_dp", "sw_dp", "renyi_dp":
		return evaluateLHCDuipeng(rule, balls, content), true
	case "texiao":
		return evaluateLHCTexiao(balls, content), true
	case "zongxiao":
		return evaluateLHCZongxiao(balls, content), true
	case "xiao":
		return evaluateLHCXiao(rule, balls, content, false), true
	case "xiao_z":
		return evaluateLHCXiao(rule, balls, content, false), true
	case "xiao_bz":
		return evaluateLHCXiao(rule, balls, content, true), true
	case "weishu":
		return evaluateLHCWeishu(balls, content, false), true
	case "weishu_bz":
		return evaluateLHCWeishu(balls, content, true), true
	case "wei_z":
		return evaluateLHCWeiMulti(balls, content, rule.CatalogSubID, false), true
	case "wei_bz":
		return evaluateLHCWeiMulti(balls, content, rule.CatalogSubID, true), true
	case "buzhong":
		return evaluateLHCBuzhong(rule, balls, content), true
	case "xuanyi":
		return evaluateLHCXuanyi(rule, balls, content), true
	case "guoguan":
		return evaluateLHCGuguan(balls, content), true
	case "tematouwei":
		return evaluateLHCTematouwei(balls, content), true
	case "wuxing":
		return evaluateLHCWuxing(balls, content), true
	case "jiaye":
		return evaluateLHCJiaye(balls, content), true
	case "bose":
		return evaluateLHCBose(balls, content, "bose"), true
	case "banbo":
		return evaluateLHCBose(balls, content, "banbo"), true
	case "banbanbo":
		return evaluateLHCBose(balls, content, "banbanbo"), true
	case "qima":
		return evaluateLHCQima(balls, content), true
	case "renzhong":
		return evaluateLHCRenzhong(rule, balls, content), true
	}
	return betEvaluation{}, false
}

func evaluateLHCTema(_ playRule, balls []string, content string) betEvaluation {
	tema := lhcTema(balls)
	nums, attrs, waves := guajibet.ParseLHCTemaParts(content)
	units := len(nums) + len(attrs) + len(waves)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range nums {
		if n, err := strconv.Atoi(p); err == nil && n == tema {
			hit = true
			break
		}
	}
	if !hit {
		for _, p := range attrs {
			if lhcTemaSideHit(balls, tema, p) {
				hit = true
				break
			}
		}
	}
	if !hit {
		for _, p := range waves {
			if lhcTemaSideHit(balls, tema, p) {
				hit = true
				break
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCTema}
}

func evaluateLHCZhengte(rule playRule, balls []string, content string) betEvaluation {
	idx := 0
	sub := strings.ToLower(rule.CatalogSubID)
	if strings.HasPrefix(sub, "zheng") {
		if n, err := strconv.Atoi(sub[5:6]); err == nil && n >= 1 && n <= 6 {
			idx = n - 1
		}
	}
	target := 0
	if idx >= 0 && idx < 6 && len(balls) > idx {
		target = atoiBall(balls[idx])
	}
	nums, attrs, waves := guajibet.ParseLHCTemaParts(content)
	units := len(nums) + len(attrs) + len(waves)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range nums {
		if n, err := strconv.Atoi(p); err == nil && n == target {
			hit = true
			break
		}
	}
	if !hit {
		for _, p := range attrs {
			if lhcTemaSideHit(balls, target, p) {
				hit = true
				break
			}
		}
	}
	if !hit {
		for _, p := range waves {
			if lhcTemaSideHit(balls, target, p) {
				hit = true
				break
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCTema}
}

// lhcTemaSideHit 特码/正特属性与波色命中（target 为特码或正码位号）。
// 开 49 时大小/单双/合数/尾数类按行业口径视为和局（不计中）。
func lhcTemaSideHit(balls []string, target int, pick string) bool {
	pick = strings.TrimSpace(pick)
	if pick == "" || target < 1 || target > 49 {
		return false
	}
	switch pick {
	case "红波", "红":
		return lhcColorOf(target) == "红"
	case "蓝波", "蓝":
		return lhcColorOf(target) == "蓝"
	case "绿波", "绿":
		return lhcColorOf(target) == "绿"
	case "总分大", "总分小", "总分单", "总分双":
		sum := 0
		for _, b := range balls {
			sum += atoiBall(b)
		}
		switch pick {
		case "总分大":
			return sum >= 175
		case "总分小":
			return sum <= 174
		case "总分单":
			return sum%2 == 1
		case "总分双":
			return sum%2 == 0
		}
	}
	if target == 49 {
		return false
	}
	he := target/10 + target%10
	wei := target % 10
	switch pick {
	case "大":
		return target >= 25
	case "小":
		return target <= 24
	case "单":
		return target%2 == 1
	case "双":
		return target%2 == 0
	case "合大":
		return he >= 7
	case "合小":
		return he <= 6
	case "合单":
		return he%2 == 1
	case "合双":
		return he%2 == 0
	case "尾大":
		return wei >= 5
	case "尾小":
		return wei <= 4
	case "尾单":
		return wei%2 == 1
	case "尾双":
		return wei%2 == 0
	}
	return false
}

func evaluateLHCFushi(rule playRule, balls []string, content string) betEvaluation {
	zheng := lhcZhengma(balls)
	tema := lhcTema(balls)
	picks := parseLHCNumbers(content)
	need, hitFn := lhcFushiRule(lhcSettlePlayType(rule))
	units := lhcComboUnits(len(picks), need)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, combo := range lhcCombinations(picks, need) {
		if hitFn(combo, zheng, tema) {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCCombo}
}

func lhcFushiRule(typeID string) (int, func([]int, []int, int) bool) {
	switch typeID {
	case "erquanzhong":
		return 2, func(combo []int, zheng []int, _ int) bool {
			return lhcCountInSet(zheng, combo) == 2
		}
	case "erzhongte":
		return 2, func(combo []int, zheng []int, tema int) bool {
			inZ := lhcCountInSet(zheng, combo)
			inT := false
			for _, c := range combo {
				if c == tema {
					inT = true
				}
			}
			return inZ == 1 && inT
		}
	case "techuan":
		return 2, func(combo []int, zheng []int, tema int) bool {
			if len(combo) != 2 {
				return false
			}
			return lhcCountInSet(zheng, []int{combo[0]}) == 1 && combo[1] == tema
		}
	case "sanzhonger":
		return 3, func(combo []int, zheng []int, _ int) bool {
			return lhcCountInSet(zheng, combo) >= 2
		}
	case "sanquanzhong":
		return 3, func(combo []int, zheng []int, _ int) bool {
			return lhcCountInSet(zheng, combo) == 3
		}
	default:
		return 2, func(combo []int, zheng []int, _ int) bool {
			return lhcCountInSet(zheng, combo) >= len(combo)
		}
	}
}

func lhcComboUnits(n, k int) int {
	if n < k || k <= 0 {
		return n
	}
	units := 1
	for i := 0; i < k; i++ {
		units = units * (n - i) / (i + 1)
	}
	return units
}

func evaluateLHCTuotou(rule playRule, balls []string, content string) betEvaluation {
	zheng := lhcZhengma(balls)
	tema := lhcTema(balls)
	parts := strings.Split(content, "|")
	if len(parts) < 2 {
		parts = strings.Split(content, "#")
	}
	var dan, tuo []int
	if len(parts) >= 2 {
		dan = parseLHCNumbers(parts[0])
		tuo = parseLHCNumbers(parts[1])
	} else {
		// 扁选落库：首号胆，其余拖（勿按复式整池组合）
		nums := parseLHCNumbers(content)
		if len(nums) >= 2 {
			dan = nums[:1]
			tuo = nums[1:]
		}
	}
	need, hitFn := lhcFushiRule(lhcSettlePlayType(rule))
	units := len(dan) * lhcComboUnits(len(tuo), need-1)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, d := range dan {
		for _, combo := range lhcCombinations(tuo, need-1) {
			pick := append([]int{d}, combo...)
			if hitFn(pick, zheng, tema) {
				hit = true
				break
			}
		}
		if hit {
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCCombo}
}

func evaluateLHCDuipeng(rule playRule, balls []string, content string) betEvaluation {
	drawn := lhcAllNumbers(balls)
	parts := strings.Split(content, "|")
	if len(parts) < 2 {
		parts = strings.Split(content, "#")
	}
	mode := strings.TrimSpace(rule.BetMode)
	// 生肖对碰扁选：恰好两个生肖 → 肖A|肖B
	if len(parts) < 2 && mode == "sx_dp" {
		zs := parseLHCZodiacs(content)
		if len(zs) >= 2 {
			parts = []string{zs[0], zs[1]}
		}
	}
	// 尾数对碰扁选：恰好两个尾数 → 尾A|尾B
	if len(parts) < 2 && mode == "ws_dp" {
		tails := parseLHCTailTokens(content)
		if len(tails) >= 2 {
			parts = []string{tails[0], tails[1]}
		}
	}
	// 生尾对碰扁选：1 肖 + 1 尾 → 肖|尾
	if len(parts) < 2 && mode == "sw_dp" {
		if z, t, ok := guajibet.ParseLHCSwDuipengSides(content); ok {
			parts = []string{z, strconv.Itoa(t)}
		}
	}
	if len(parts) < 2 {
		return betEvaluation{BetUnits: 1, Odds: oddsLHCCombo}
	}
	groupA := lhcDuipengGroup(mode, parts[0])
	groupB := lhcDuipengGroup(mode, parts[1])
	// 生肖/尾数对碰：注数=两侧展开号码数之积；生尾另扣两侧共有号码（同号无法成对）
	units := len(groupA) * len(groupB)
	if mode == "sw_dp" {
		bSet := make(map[int]struct{}, len(groupB))
		for _, b := range groupB {
			bSet[b] = struct{}{}
		}
		overlap := 0
		for _, a := range groupA {
			if _, ok := bSet[a]; ok {
				overlap++
			}
		}
		units -= overlap
	}
	if units <= 0 {
		units = 1
	}
	drawnSet := lhcNumberSet(drawn)
	hit := false
	for _, a := range groupA {
		for _, b := range groupB {
			if a == b {
				continue // 生尾两侧共号：开奖球不重复，不成对
			}
			if drawnSet[a] && drawnSet[b] {
				hit = true
				break
			}
		}
		if hit {
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCCombo}
}

func lhcDuipengGroup(mode, raw string) []int {
	switch mode {
	case "sx_dp":
		return lhcZodiacNums(parseLHCZodiacs(raw))
	case "ws_dp":
		tails := parseIntTokens(raw)
		var out []int
		for n := 1; n <= 49; n++ {
			for _, t := range tails {
				if lhcTailOf(n) == t {
					out = append(out, n)
					break
				}
			}
		}
		return out
	case "sw_dp":
		z := lhcZodiacNums(parseLHCZodiacs(raw))
		if len(z) > 0 {
			return z
		}
		tails := parseIntTokens(raw)
		var out []int
		for n := 1; n <= 49; n++ {
			for _, t := range tails {
				if lhcTailOf(n) == t {
					out = append(out, n)
				}
			}
		}
		return out
	default:
		return parseLHCNumbers(raw)
	}
}

func evaluateLHCTexiao(balls []string, content string) betEvaluation {
	tema := lhcTema(balls)
	z := lhcZodiacOf(tema)
	picks := parseLHCZodiacs(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if p == z {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCAttr}
}

func evaluateLHCZongxiao(balls []string, content string) betEvaluation {
	drawn := lhcDistinctZodiacs(lhcAllNumbers(balls))
	count := len(drawn)
	picks := parseLHCZongxiaoPicks(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	bestOdds := oddsLHCAttr
	for _, p := range picks {
		if p != count {
			continue
		}
		hit = true
		if o := lhcZongxiaoOdds(p); o > bestOdds {
			bestOdds = o
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: bestOdds}
}

var lhcZongxiaoOptions = []struct {
	label string
	count int
	odds  float64
}{
	{"二肖", 2, 14.841},
	{"三肖", 3, 14.841},
	{"四肖", 4, 14.841},
	{"五肖", 5, 3.007},
	{"六肖", 6, 1.92},
	{"七肖", 7, 5.335},
}

func lhcZongxiaoOdds(count int) float64 {
	for _, o := range lhcZongxiaoOptions {
		if o.count == count {
			return o.odds
		}
	}
	return oddsLHCAttr
}

func parseLHCZongxiaoPicks(raw string) []int {
	parts := parseTextTokens(raw)
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if n, ok := parseLHCZongxiaoCount(p); ok {
			out = append(out, n)
		}
	}
	return out
}

func parseLHCZongxiaoCount(token string) (int, bool) {
	token = strings.TrimSpace(token)
	for _, o := range lhcZongxiaoOptions {
		if token == o.label {
			return o.count, true
		}
	}
	if n, err := strconv.Atoi(token); err == nil && n >= 2 && n <= 7 {
		return n, true
	}
	if strings.HasSuffix(token, "肖") {
		label := strings.TrimSuffix(token, "肖")
		for _, o := range lhcZongxiaoOptions {
			if strings.TrimSuffix(o.label, "肖") == label {
				return o.count, true
			}
		}
	}
	return 0, false
}

func evaluateLHCXiao(rule playRule, balls []string, content string, invert bool) betEvaluation {
	drawn := lhcDistinctZodiacs(lhcAllNumbers(balls))
	picks := parseLHCZodiacs(content)
	need := lhcXiaoCount(rule.CatalogSubID)
	units := lhcComboUnits(len(picks), need)
	if units <= 0 {
		units = 1
	}
	matched := 0
	for _, p := range picks {
		if drawn[p] {
			matched++
		}
	}
	hit := matched >= need
	if invert {
		hit = matched == 0
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCAttr}
}

func evaluateLHCWeishu(balls []string, content string, invert bool) betEvaluation {
	tema := lhcTema(balls)
	tail := lhcTailOf(tema)
	picks := parseIntTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if invert {
			if p != tail {
				hit = true
				break
			}
		} else if p == tail {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCAttr}
}

func evaluateLHCWeiMulti(balls []string, content string, subID string, invert bool) betEvaluation {
	drawn := lhcAllNumbers(balls)
	tails := map[int]bool{}
	for _, n := range drawn {
		tails[lhcTailOf(n)] = true
	}
	picks := parseIntTokens(content)
	need := lhcWeiCount(subID)
	units := lhcComboUnits(len(picks), need)
	if units <= 0 {
		units = 1
	}
	matched := 0
	for _, p := range picks {
		if tails[p] {
			matched++
		}
	}
	hit := matched >= need
	if invert {
		hit = matched == 0
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCAttr}
}

func evaluateLHCBuzhong(rule playRule, balls []string, content string) betEvaluation {
	drawn := lhcAllNumbers(balls)
	picks := parseLHCNumbers(content)
	need := lhcBuzhongCount(rule.CatalogSubID)
	units := lhcComboUnits(len(picks), need)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, combo := range lhcCombinations(picks, need) {
		if lhcCountInSet(drawn, combo) == 0 {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCCombo}
}

func evaluateLHCXuanyi(rule playRule, balls []string, content string) betEvaluation {
	tema := lhcTema(balls)
	picks := parseLHCNumbers(content)
	need := lhcXuanyiCount(rule.CatalogSubID)
	units := lhcComboUnits(len(picks), need)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, combo := range lhcCombinations(picks, need) {
		c := 0
		for _, p := range combo {
			if p == tema {
				c++
			}
		}
		if c == 1 {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCCombo}
}

func evaluateLHCGuguan(balls []string, content string) betEvaluation {
	positions, ok := parseLHCGuoguanPositions(content)
	if !ok {
		return betEvaluation{BetUnits: 0, Odds: oddsLHCAttr}
	}
	for index, pick := range positions {
		if pick == "" {
			continue
		}
		if index >= len(balls) || !lhcTemaSideHit(balls, atoiBall(balls[index]), pick) {
			return betEvaluation{Hit: false, BetUnits: 1, Odds: oddsLHCAttr}
		}
	}
	return betEvaluation{Hit: true, BetUnits: 1, Odds: oddsLHCAttr}
}

func evaluateLHCTematouwei(balls []string, content string) betEvaluation {
	tema := lhcTema(balls)
	head := lhcHeadOf(tema)
	tail := lhcTailOf(tema)
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, pick := range picks {
		if strings.HasPrefix(pick, "头") {
			if n, err := strconv.Atoi(strings.TrimPrefix(pick, "头")); err == nil && n == head {
				hit = true
				break
			}
		}
		if strings.HasPrefix(pick, "尾") {
			if n, err := strconv.Atoi(strings.TrimPrefix(pick, "尾")); err == nil && n == tail {
				hit = true
				break
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCAttr}
}

func evaluateLHCWuxing(balls []string, content string) betEvaluation {
	tema := lhcTema(balls)
	elem := lhcElementOf(tema)
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		if p == elem {
			hit = true
			break
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCAttr}
}

func evaluateLHCJiaye(balls []string, content string) betEvaluation {
	tema := lhcTema(balls)
	z := lhcZodiacOf(tema)
	isDomestic := lhcDomesticZodiacs[z]
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, p := range picks {
		switch p {
		case "家", "家禽":
			if isDomestic {
				hit = true
			}
		case "野", "野兽":
			if !isDomestic && z != "" {
				hit = true
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCAttr}
}

func evaluateLHCBose(balls []string, content string, mode string) betEvaluation {
	tema := lhcTema(balls)
	color := lhcColorOf(tema)
	big := tema >= 25
	odd := tema%2 == 1
	picks := parseTextTokens(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	hit := false
	for _, pick := range picks {
		switch mode {
		case "bose":
			if pick == color || pick == color+"波" {
				hit = true
			}
		case "banbo":
			if lhcBanboHit(pick, color, big, odd) {
				hit = true
			}
		case "banbanbo":
			if lhcBanbanboHit(pick, color, big, odd) {
				hit = true
			}
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCAttr}
}

func lhcBanboHit(pick, color string, big, odd bool) bool {
	size := "小"
	if big {
		size = "大"
	}
	parity := "双"
	if odd {
		parity = "单"
	}
	return pick == color+size || pick == color+parity
}

func lhcBanbanboHit(pick, color string, big, odd bool) bool {
	size := "小"
	if big {
		size = "大"
	}
	parity := "双"
	if odd {
		parity = "单"
	}
	return pick == color+size+parity
}

func evaluateLHCQima(balls []string, content string) betEvaluation {
	drawn := lhcAllNumbers(balls)
	picks := parseLHCQimaPicks(content)
	units := len(picks)
	if units <= 0 {
		units = 1
	}
	counts := map[string]int{
		"单": lhcQimaCategoryCount(drawn, "单"),
		"双": lhcQimaCategoryCount(drawn, "双"),
		"大": lhcQimaCategoryCount(drawn, "大"),
		"小": lhcQimaCategoryCount(drawn, "小"),
	}
	hit := false
	bestOdds := oddsLHCAttr
	for _, pick := range picks {
		if counts[pick.kind] != pick.count {
			continue
		}
		hit = true
		if o := lhcQimaOdds(pick.kind, pick.count); o > bestOdds {
			bestOdds = o
		}
	}
	return betEvaluation{Hit: hit, BetUnits: units, Odds: bestOdds}
}

func evaluateLHCRenzhong(rule playRule, balls []string, content string) betEvaluation {
	drawn := lhcAllNumbers(balls)
	picks := parseLHCNumbers(content)
	need := lhcRenzhongCount(rule.CatalogSubID)
	units := lhcComboUnits(len(picks), need)
	if units <= 0 {
		units = 1
	}
	hit := lhcCountInSet(drawn, picks) >= 1
	return betEvaluation{Hit: hit, BetUnits: units, Odds: oddsLHCCombo}
}
