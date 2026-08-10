package schemes

import (
	"strings"
)

// 第三方单组注数上限（对齐 client/src/utils/betPayload.ts）。
const (
	hezhiKuaduMaxBetUnits = 900
	weishuMaxBetUnitsCap  = 9
)

// 任二直选复式第三方上限 900（勿按 SegmentLen=2 套前二公式 → 90）。
const ren2ZhixuanFushiMaxBetUnits = 900

// 任二直选和值第三方上限 900（对齐任二直选复式/单式，勿套前二 90）。
const ren2ZhixuanHezhiMaxBetUnits = 900

// maxBetUnitsForPlay 该玩法单组最大注数；0 表示本端尚未定义上限（不拦截）。
// 按 BetMode / 区位独立取值，供随机出号重抽、真下单与审计共用。
func maxBetUnitsForPlay(rule playRule) int {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	z := playZoneMultiplier(rule)
	if z < 1 {
		z = 1
	}
	// 任二直选和值：第三方上限 900（勿套前二 SegmentLen 公式 → 90）
	if isRen2ZhixuanHezhiRule(rule) {
		return ren2ZhixuanHezhiMaxBetUnits
	}
	// 任选直选复式 / 任选选位类（单式·号池·和值）：上限不能套前二 SegmentLen=2 → 90（任二第三方 900）
	if isRenxuanPlayType(rule.PlayTypeID) && (isRenxuanZhixuanFushiRule(rule) || isRenxuanNeedsPositionRule(rule)) {
		return renxuanZhixuanFushiMaxBetUnits(rule)
	}
	switch bm {
	case "fushi", "zhixuan_fs":
		base := zhixuanFushiMaxBetUnits(rule)
		if base <= 0 {
			return 0
		}
		return base * z
	case "danshi", "zhixuan_ds":
		// 直选单式与复式同第三方上限（前二=90、前三=900…）；组选单式不走此分支
		if !isZhixuanDanshiRule(rule) {
			return 0
		}
		base := zhixuanSegmentMaxBetUnits(rule)
		if base <= 0 {
			return 0
		}
		return base * z
	case "hezhi", "kuadu":
		// 任选和值勿落到前二 90（任二直选和值=900；其它选位类=任选上限）
		if isRen2ZhixuanHezhiRule(rule) {
			return ren2ZhixuanHezhiMaxBetUnits
		}
		if isRenxuanNeedsPositionRule(rule) {
			return renxuanZhixuanFushiMaxBetUnits(rule)
		}
		// 与直选复式同第三方上限：前二/后二=90、前三=900；再×区位倍乘
		if base := zhixuanSegmentMaxBetUnits(rule); base > 0 {
			return base * z
		}
		return hezhiKuaduMaxBetUnits
	case "weishu":
		// 单区最多 9 个尾数；前中后三等再×区位（9×3=27）
		return weishuMaxBetUnitsCap * z
	case "zuhe":
		// 直选组合 = 直选复式上限 × 段长 × 区位（三星 900×3=2700；四星 9000×4=36000）
		base := zhixuanSegmentMaxBetUnits(rule)
		if base <= 0 {
			return 0
		}
		seg := rule.SegmentLen
		if seg <= 0 {
			seg = 3
		}
		return base * seg * z
	default:
		return 0
	}
}

// isRenxuanZhixuanFushiRule 任选·直选复式（五位号池，按 C(5,n) 计注）。
func isRenxuanZhixuanFushiRule(rule playRule) bool {
	if !isRenxuanPlayType(rule.PlayTypeID) {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if strings.Contains(sub, "单式") || strings.Contains(sub, "组选") || strings.Contains(sub, "和值") ||
		strings.Contains(sub, "组三") || strings.Contains(sub, "组六") || strings.Contains(sub, "混合") ||
		strings.Contains(sub, "zuxuan") || strings.Contains(sub, "danshi") || strings.Contains(sub, "hezhi") ||
		strings.Contains(sub, "hunhe") || strings.Contains(sub, "zu3") || strings.Contains(sub, "zu6") {
		return false
	}
	return bm == "fushi" || bm == "zhixuan_fs" || strings.Contains(sub, "直选复式") || strings.Contains(sub, "zhixuan_fs")
}

// isRenxuanNeedsPositionRule 任选非直选复式：均需万千百十个选位（对齐任二直选单式）。
func isRenxuanNeedsPositionRule(rule playRule) bool {
	if !isRenxuanPlayType(rule.PlayTypeID) {
		return false
	}
	if isRenxuanZhixuanFushiRule(rule) {
		return false
	}
	k := rule.SegmentLen
	if k <= 0 {
		k = renPickCount(rule.CatalogSubID)
	}
	if k <= 0 {
		k = renPickCount(rule.SubPlayID)
	}
	return k >= 2 && k <= 5
}

// isRenxuanPositionDanshiRule 任选选位 + 单式票面（直选/组选/混合单式）。
func isRenxuanPositionDanshiRule(rule playRule) bool {
	if !isRenxuanNeedsPositionRule(rule) {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if bm == "danshi" || bm == "zhixuan_ds" || bm == "zuxuan_ds" || bm == "hunhe" {
		return true
	}
	if strings.Contains(sub, "直选单式") || strings.Contains(sub, "组选单式") ||
		strings.Contains(sub, "混合组选") || strings.Contains(sub, "组三单式") ||
		strings.Contains(sub, "组六单式") || strings.Contains(sub, "zhixuan_ds") ||
		strings.Contains(sub, "zuxuan_ds") || strings.Contains(sub, "hunhe") {
		return true
	}
	return isZhixuanDanshiRule(rule)
}

// isRenxuanZhixuanDanshiRule 任选·直选单式（兼容旧名；含组选/混合单式票面）。
func isRenxuanZhixuanDanshiRule(rule playRule) bool {
	return isRenxuanPositionDanshiRule(rule)
}

// isRen2ZhixuanHezhiRule 任选·任二直选和值（不含组选和值；catalog 常为 76）。
func isRen2ZhixuanHezhiRule(rule playRule) bool {
	if !isRenxuanNeedsPositionRule(rule) {
		return false
	}
	if rule.HezhiZuxuan {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if strings.Contains(sub, "组选") {
		return false
	}
	if bm != "hezhi" && !strings.Contains(sub, "hezhi") && !strings.Contains(sub, "和值") {
		return false
	}
	k := rule.SegmentLen
	if k <= 0 {
		k = renPickCount(rule.CatalogSubID)
	}
	if k <= 0 {
		k = renPickCount(rule.SubPlayID)
	}
	return k == 2
}

// renxuanZhixuanFushiMaxBetUnits 任选直选复式单组上限。
// 对齐第三方：同星直选上限 × C(5,k)（任二 90×10=900，任三 900×10=9000，任四 9000×5=45000）。
func renxuanZhixuanFushiMaxBetUnits(rule playRule) int {
	n := rule.SegmentLen
	if n <= 0 {
		n = renPickCount(rule.CatalogSubID)
	}
	if n <= 0 {
		n = renPickCount(rule.SubPlayID)
	}
	if n <= 0 || n > 5 {
		n = 2
	}
	tmp := rule
	tmp.SegmentLen = n
	base := zhixuanSegmentMaxBetUnits(tmp)
	if base <= 0 {
		if n <= 2 {
			return ren2ZhixuanFushiMaxBetUnits
		}
		return 0
	}
	mul := combinInt(5, n)
	if mul <= 0 {
		return base
	}
	return base * mul
}

// playZoneMultiplier 多区位玩法注数倍乘（前中后三×3、前后三/二/四×2）。
func playZoneMultiplier(rule playRule) int {
	if n := len(rule.SegmentPos); n > 1 {
		return n
	}
	switch strings.ToLower(strings.TrimSpace(rule.PlayTypeID)) {
	case "qianzhonghou3", "g007":
		return 3
	case "qianhou3", "g012", "qianhou2", "g008", "qianhou4", "g014":
		return 2
	}
	return 1
}

// contentExceedsBetUnitsMax 内容组合注数是否超过该玩法上限。
// 上限为 0（未定义）时视为不超限。
func contentExceedsBetUnitsMax(rule playRule, content string) bool {
	max := maxBetUnitsForPlay(rule)
	if max <= 0 {
		return false
	}
	n := countPlayWireBetUnits(rule, content)
	return n > max
}

// countLHCPlayWireBetUnits 六合注数（与 evaluateLHCByBetMode 对齐）。
func countLHCPlayWireBetUnits(rule playRule, content string) int {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0
	}
	if ev, ok := evaluateLHCByBetMode(rule, nil, content); ok && ev.BetUnits > 0 {
		return ev.BetUnits
	}
	return 0
}

// countPlayWireBetUnits 按第三方 bets_nums 口径统计注数（用已解析的 SegmentLen，
// 不依赖 guajibet 对 typeId 文案的区位推断）。
func countPlayWireBetUnits(rule playRule, content string) int {
	content = strings.TrimSpace(normalizeZhixuanDanshiContent(rule, content))
	if content == "" {
		return 0
	}
	// 六合：勿走时时彩直选复式位积（SegmentLen=1 时恒为 0）
	if rule.PlayTemplate == "lhc_std" || isLHCPlayRule(rule) {
		if n := countLHCPlayWireBetUnits(rule, content); n > 0 {
			return n
		}
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	seg := rule.SegmentLen
	if seg <= 0 {
		seg = 1
	}
	z := playZoneMultiplier(rule)
	if z < 1 {
		z = 1
	}
	var base int
	// 任选直选复式：C(5,n) 位组合计注（勿按五位乘积 10×10×9…）
	if isRenxuanPlayType(rule.PlayTypeID) && isRenxuanZhixuanFushiRule(rule) {
		return countRenxuanZhixuanFushiBetUnits(rule, content)
	}
	// 任选选位类：C(选位数,k)×内层注数（单式/号池/和值）
	if isRenxuanNeedsPositionRule(rule) {
		return countRenxuanNeedsPositionBetUnits(rule, content)
	}
	switch bm {
	case "fushi", "zhixuan_fs":
		base = countZhixuanFushiBetUnits(content, seg)
	case "kuadu":
		base = countKuaduCombinatorialUnits(content, seg)
	case "hezhi":
		base = countHezhiCombinatorialUnits(content, seg, rule.HezhiZuxuan)
	case "weishu":
		base = len(parseIntTokens(content))
	case "zuhe":
		// 直选组合：位积 × 段长（三星×3 / 四星×4），再×区位
		base = countZhixuanFushiBetUnits(content, seg) * seg
		if base <= 0 {
			return 0
		}
	case "danshi", "zhixuan_ds", "zuxuan_ds", "hunhe":
		base = len(parseSegmentTokensForRule(rule, content, seg))
		if base <= 0 {
			// 整注逗号串
			base = len(splitContentTokens(content))
		}
	case "zu12":
		// 双区「二重,单号」：C(m,1)×C(n,2)
		base = countZu12DualZoneBetUnits(content)
	case "zu4":
		// 双区「三重,单号」：对每个三重 t 计 |单号\{t}|
		base = countZu4DualZoneBetUnits(content)
	case "zu60":
		// 双区「二重,单号」：对每个二重 d 计 C(|单号\{d}|, 3)
		base = countZu60DualZoneBetUnits(content)
	case "zu30":
		// 双区「二重≥3,单号≥1」：对每个二重对计 |单号\{d1,d2}|
		base = countZu30DualZoneBetUnits(content)
	case "zu20":
		// 双区「三重,单号」个数相同且各≥2：对每个三重 t 计 C(|单号\{t}|, 2)
		base = countZu20DualZoneBetUnits(content)
	case "zu10", "zu5":
		// 双区各≥1：对每个头区码计 |尾区\{头}|
		base = countZuPairDualZoneBetUnits(content)
	case "zuxuan_fs", "zu3", "zu6", "zu24", "zu120":
		// 与 evaluateZuxuanFushi / guajibet C(n,k) 对齐，供上限与金额同步
		pool := uniqueStringTokens(parseDigitTokens(content))
		base = zuxuanPoolUnitsForRule(rule, pool)
	default:
		return 0
	}
	if base <= 0 {
		return 0
	}
	return base * z
}

// countRenxuanNeedsPositionBetUnits 任选选位注数：C(选位数,k)×剥位后内层注数。
func countRenxuanNeedsPositionBetUnits(rule playRule, content string) int {
	k := rule.SegmentLen
	if k <= 0 {
		k = renPickCount(rule.CatalogSubID)
	}
	if k <= 0 {
		k = renPickCount(rule.SubPlayID)
	}
	if k <= 0 || k > 5 {
		k = 2
	}
	posLabel, picks, ok := parseRenxuanPosPicksContent(content, k)
	if !ok {
		if isRenxuanPositionDanshiRule(rule) {
			tickets := 0
			switch {
			case isZu3DanshiPlayRule(rule):
				tickets = len(filterZu3DanshiTokens(content, k))
			case isZu6DanshiPlayRule(rule):
				tickets = len(filterZu6DanshiTokens(content, k))
			case isHunhePlayRule(rule):
				tickets = countHunhePickUnits(content, k)
			case isZuxuanDanshiPlayRule(rule):
				tickets = countZuxuanDanshiPickUnits(content, k)
			default:
				tickets = len(renxuanDanshiTokens(content, k))
			}
			if tickets <= 0 {
				return 0
			}
			return tickets
		}
		picks = content
		return countRenxuanInnerBetUnits(rule, picks)
	}
	positions := renxuanPositionsFromLabel(posLabel, k)
	nPos := len(positions)
	if nPos < k {
		return 0
	}
	mul := combinInt(nPos, k)
	if mul <= 0 {
		return 0
	}
	inner := countRenxuanInnerBetUnits(rule, picks)
	if inner <= 0 {
		return 0
	}
	return mul * inner
}

// countRenxuanInnerBetUnits 剥位后的内层注数（单式票数或号池/和值组合数）。
func countRenxuanInnerBetUnits(rule playRule, picks string) int {
	picks = strings.TrimSpace(picks)
	if picks == "" {
		return 0
	}
	k := rule.SegmentLen
	if k <= 0 {
		k = renPickCount(rule.CatalogSubID)
	}
	if k <= 0 {
		k = renPickCount(rule.SubPlayID)
	}
	if k <= 0 || k > 5 {
		k = 2
	}
	if isRenxuanPositionDanshiRule(rule) {
		switch {
		case isZu3DanshiPlayRule(rule):
			return len(filterZu3DanshiTokens(picks, k))
		case isZu6DanshiPlayRule(rule):
			return len(filterZu6DanshiTokens(picks, k))
		case isHunhePlayRule(rule):
			return countHunhePickUnits(picks, k)
		case isZuxuanDanshiPlayRule(rule):
			return countZuxuanDanshiPickUnits(picks, k)
		default:
			return len(renxuanDanshiTokens(picks, k))
		}
	}
	// 临时去掉任选标记，避免 countPlayWireBetUnits 再走选位分支
	tmp := rule
	tmp.PlayTypeID = "ssc_inner"
	return countPlayWireBetUnits(tmp, picks)
}

// isZuxuanDanshiPlayRule 组选单式（含任选组选单式）。
func isZuxuanDanshiPlayRule(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "zuxuan_ds" {
		return true
	}
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	return strings.Contains(sub, "zuxuan_ds") || strings.Contains(sub, "组选单式")
}

// isZu3DanshiPlayRule 组三单式（含任三组三单式 rule 84；不含组选30）。
func isZu3DanshiPlayRule(rule playRule) bool {
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID + " " + rule.CatalogSubID))
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if strings.Contains(sub, "zu30") || strings.Contains(sub, "组选30") {
		return false
	}
	sid := strings.TrimSpace(rule.SubPlayID)
	if sid == "" {
		sid = strings.TrimSpace(rule.CatalogSubID)
	}
	if sid == "84" || strings.Contains(sub, "zu3_ds") || strings.Contains(sub, "ren3_zu3_ds") ||
		strings.Contains(sub, "组三单式") {
		return true
	}
	hasZu3 := bm == "zu3" || strings.Contains(sub, "zu3") || strings.Contains(sub, "组三")
	hasDanshi := bm == "danshi" || bm == "zuxuan_ds" || strings.Contains(sub, "单式") || strings.Contains(sub, "_ds")
	return hasZu3 && hasDanshi
}

// isZu6DanshiPlayRule 组六单式（含任三组六单式 rule 86；不含组选6/60/120）。
func isZu6DanshiPlayRule(rule playRule) bool {
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID + " " + rule.CatalogSubID))
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if strings.Contains(sub, "组选6") || strings.Contains(sub, "组选60") || strings.Contains(sub, "组选120") ||
		strings.Contains(sub, "zu60") || strings.Contains(sub, "zu120") {
		return false
	}
	sid := strings.TrimSpace(rule.SubPlayID)
	if sid == "" {
		sid = strings.TrimSpace(rule.CatalogSubID)
	}
	if sid == "86" || strings.Contains(sub, "zu6_ds") || strings.Contains(sub, "ren3_zu6_ds") ||
		strings.Contains(sub, "组六单式") {
		return true
	}
	hasZu6 := bm == "zu6" || strings.Contains(sub, "zu6") || strings.Contains(sub, "组六")
	hasDanshi := bm == "danshi" || bm == "zuxuan_ds" || strings.Contains(sub, "单式") || strings.Contains(sub, "_ds")
	return hasZu6 && hasDanshi
}

// filterZu3DanshiTokens 仅保留组三形态整注，按形态去重保序。
func filterZu3DanshiTokens(picks string, n int) []string {
	if n <= 0 {
		n = 3
	}
	tokens := renxuanDanshiTokens(picks, n)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if !isZu3DigitString(tok) {
			continue
		}
		key := sortDigitString(tok)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tok)
	}
	return out
}

// countHunhePickUnits 混合组选内层注数：排除豹子，按组选形态去重。
func countHunhePickUnits(picks string, n int) int {
	if n <= 0 {
		n = 3
	}
	filtered := filterHunheBetTickets(picks, n)
	if strings.TrimSpace(filtered) == "" {
		return 0
	}
	return len(strings.Split(filtered, ","))
}

// filterZu6DanshiTokens 仅保留组六形态整注（三位互异），按形态去重保序。
func filterZu6DanshiTokens(picks string, n int) []string {
	if n <= 0 {
		n = 3
	}
	tokens := renxuanDanshiTokens(picks, n)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if !isZu6DigitString(tok) {
			continue
		}
		key := sortDigitString(tok)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tok)
	}
	return out
}

// countZuxuanDanshiPickUnits 组选单式内层注数：整注形态去重，或单码号池 C(n,segLen)。
func countZuxuanDanshiPickUnits(picks string, segLen int) int {
	if segLen <= 0 {
		segLen = 2
	}
	raw := strings.NewReplacer("，", ",", "\n", ",", " ", ",").Replace(strings.TrimSpace(picks))
	parts := strings.Split(raw, ",")
	singles := make([]string, 0, len(parts))
	seenSingle := map[string]struct{}{}
	tickets := make([]string, 0, len(parts))
	seenTicket := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		digits := make([]rune, 0, len(p))
		for _, r := range p {
			if r >= '0' && r <= '9' {
				digits = append(digits, r)
			}
		}
		d := string(digits)
		if len(d) == 1 {
			if _, ok := seenSingle[d]; ok {
				continue
			}
			seenSingle[d] = struct{}{}
			singles = append(singles, d)
			continue
		}
		if len(d) != segLen {
			continue
		}
		baozi := true
		for i := 1; i < len(d); i++ {
			if d[i] != d[0] {
				baozi = false
				break
			}
		}
		if baozi {
			continue
		}
		key := sortDigitString(d)
		if _, ok := seenTicket[key]; ok {
			continue
		}
		seenTicket[key] = struct{}{}
		tickets = append(tickets, d)
	}
	if len(tickets) > 0 {
		return len(tickets)
	}
	if len(singles) < segLen {
		return 0
	}
	return combinInt(len(singles), segLen)
}

// countRenxuanZhixuanDanshiBetUnits 兼容旧名 → 选位计注。
func countRenxuanZhixuanDanshiBetUnits(rule playRule, content string) int {
	return countRenxuanNeedsPositionBetUnits(rule, content)
}

// countRenxuanZhixuanFushiBetUnits 任选直选复式注数（对齐 evaluateRenxuanZhixuan）。
func countRenxuanZhixuanFushiBetUnits(rule playRule, content string) int {
	k := rule.SegmentLen
	if k <= 0 {
		k = renPickCount(rule.CatalogSubID)
	}
	if k <= 0 {
		k = renPickCount(rule.SubPlayID)
	}
	if k <= 0 || k > 5 {
		k = 2
	}
	pools := renxuanZhixuanFushiPools(content)
	units := 0
	for _, combo := range combinations(5, k) {
		u := 1
		ok := true
		for _, pos := range combo {
			n := len(pools[pos])
			if n == 0 {
				ok = false
				break
			}
			u *= n
		}
		if ok {
			units += u
		}
	}
	return units
}

// renxuanZhixuanFushiPools 解析五位号池（换行按位，或五段逗号 wire）。
// 勿用 parseDigitTokens：其空池会回落 ["0"]，把空位/粘连误成单码 0。
func renxuanZhixuanFushiPools(content string) [][]string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	pools := make([][]string, 5)
	// 无换行的五段 wire：如 1234567890,1234567890,123456789,123456789,123456789
	if !strings.Contains(content, "\n") {
		if parts := strings.Split(content, ","); len(parts) == 5 {
			for i := 0; i < 5; i++ {
				pools[i] = expandGluedDigitPool(parts[i])
			}
			return pools
		}
	}
	lines := strings.Split(content, "\n")
	for len(lines) < 5 {
		lines = append(lines, "")
	}
	for i := 0; i < 5; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// "1,2,3" 与粘连 "123" 均按出现过的 0–9 去重计数
		pools[i] = expandGluedDigitPool(line)
	}
	return pools
}

func expandGluedDigitPool(raw string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if r < '0' || r > '9' {
			continue
		}
		s := string(r)
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func countKuaduCombinatorialUnits(content string, segLen int) int {
	if segLen <= 0 {
		segLen = 3
	}
	picks := parseIntTokens(content)
	if len(picks) == 0 {
		return 0
	}
	total := 0
	for _, span := range picks {
		total += countOrderedSpanCombinationsLocal(span, segLen)
	}
	return total
}

func countHezhiCombinatorialUnits(content string, segLen int, zuxuan bool) int {
	if segLen <= 0 {
		segLen = 3
	}
	picks := parseIntTokens(content)
	if len(picks) == 0 {
		return 0
	}
	total := 0
	for _, sum := range picks {
		if zuxuan {
			total += countZuxuanSumCombinationsLocal(sum, segLen)
		} else {
			total += countOrderedSumCombinationsLocal(sum, segLen)
		}
	}
	return total
}

func countOrderedSpanCombinationsLocal(span, positions int) int {
	if positions <= 0 || span < 0 {
		return 0
	}
	count := 0
	var dfs func(idx, min, max int)
	dfs = func(idx, min, max int) {
		if idx == positions {
			if max-min == span {
				count++
			}
			return
		}
		for d := 0; d <= 9; d++ {
			nmin, nmax := min, max
			if idx == 0 {
				nmin, nmax = d, d
			} else {
				if d < nmin {
					nmin = d
				}
				if d > nmax {
					nmax = d
				}
			}
			dfs(idx+1, nmin, nmax)
		}
	}
	dfs(0, 0, 0)
	return count
}

func countOrderedSumCombinationsLocal(targetSum, positions int) int {
	if positions <= 0 || targetSum < 0 {
		return 0
	}
	count := 0
	var dfs func(idx, sum int)
	dfs = func(idx, sum int) {
		if idx == positions {
			if sum == targetSum {
				count++
			}
			return
		}
		for d := 0; d <= 9; d++ {
			dfs(idx+1, sum+d)
		}
	}
	dfs(0, 0)
	return count
}

// countZuxuanSumCombinationsLocal 组选和值组合数（三星：无序、排除豹子；与 guajibet 常用口径对齐的简化版）。
// countZuxuanSumCombinationsLocal 对齐 guajibet.countZuxuanSumCombinations：
// 非豹子多重集个数（每个和值形态 1 注），勿按有序排列把 sum=6 计成 27。
func countZuxuanSumCombinationsLocal(targetSum, segLen int) int {
	if segLen == 4 {
		return countZuxuanSumMultisetLocal(targetSum, 3) * 4
	}
	return countZuxuanSumMultisetLocal(targetSum, segLen)
}

func countZuxuanSumMultisetLocal(targetSum, segLen int) int {
	if segLen <= 0 || targetSum < 0 {
		return 0
	}
	count := 0
	digits := make([]int, segLen)
	var dfs func(pos, minVal, sum int)
	dfs = func(pos, minVal, sum int) {
		if pos == segLen {
			if sum != targetSum {
				return
			}
			for i := 1; i < segLen; i++ {
				if digits[i] != digits[0] {
					count++
					return
				}
			}
			return // 豹子不计
		}
		for d := minVal; d <= 9; d++ {
			if sum+d > targetSum {
				break
			}
			digits[pos] = d
			dfs(pos+1, d, sum+d)
		}
	}
	dfs(0, 0, 0)
	return count
}
