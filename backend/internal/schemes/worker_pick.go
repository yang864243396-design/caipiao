package schemes

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
)

// pickDecision 出号策略结果（v8 §4：出号体系与倍投体系独立）。
type pickDecision struct {
	Content   string // 本期下注内容（与方案内容同格式）
	Skip      bool   // 本期跳过不下注
	Direction string // 开某投某本局投向：pos / neg
}

// resolvePick 按运行类型决定本期下注内容（仅 kind=custom 分发；
// 反买/跟单与未识别类型走传统组轮换内容）。
func (w *Worker) resolvePick(
	ctx context.Context,
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	draw sqlcdb.LotteryDraw,
) pickDecision {
	if cfg.Contrary {
		if inv := strings.TrimSpace(cfg.ContraryPlan); inv != "" {
			return pickDecision{Content: inv}
		}
		return pickDecision{Content: cfg.GroupContent}
	}
	if cfg.Kind != "custom" || cfg.RunTypeID == "" {
		return pickDecision{Content: cfg.GroupContent}
	}
	switch cfg.RunTypeID {
	case RunTypeFixedRotate:
		return pickFixedRotate(cfg, inst)
	case RunTypeAdvFixedRotate:
		return pickJushuList(cfg, inst)
	case RunTypeAdvTriggerBet:
		return w.pickTriggerBet(ctx, cfg, inst, draw)
	case RunTypeHotColdWarm:
		return w.pickHotColdWarm(ctx, cfg, inst, draw)
	case RunTypeRandomDraw:
		return pickRandomDraw(cfg, inst)
	case RunTypeFixedNumber:
		return pickFixedNumber(cfg)
	case RunTypeBuiltinPlan:
		// 未物化的内置计画（尚未选择收藏方案）：跳过不下注
		return pickDecision{Skip: true}
	default:
		return pickDecision{Content: cfg.GroupContent}
	}
}

// AdvancePickAfterFormalSettlement 正式盘/模拟盘派奖后补推进出号游标。
//
// 下单时尚未开奖（无中/未中结果），出号游标与投向被冻结（见 worker.go deferSettle
// 分支），因此定码轮换/高级定码轮换等运行类型会一直停在下单时的组，
// 表现为“每期都用第一组号码下注”。派奖拿到结果后在此按 advancePickState 相同语义
// 补推进，使各运行类型逐期切换下注内容。
//
// betContent 为该期实际下注内容（供冷热中奖轮换 / 随机出号保持等策略使用）。
func AdvancePickAfterFormalSettlement(
	kind string,
	definitionConfig []byte,
	inst sqlcdb.SchemeInstance,
	betContent string,
	hit bool,
) (pickIndex int32, currentPick string, lastDirection string) {
	groupIndex := 0
	if inst.RoundIndex > 0 {
		groupIndex = int(inst.RoundIndex)
	}
	cfg := parseSchemeConfig(kind, definitionConfig, int(inst.RoundIndex), groupIndex)
	cfg.Play = attachOddsBase(cfg.Play, inst.LotteryCode)
	dec := pickDecision{Content: betContent}
	if cfg.RunTypeID == RunTypeAdvTriggerBet && cfg.Trigger != nil {
		// 下单时投向未持久化（applyLastDirection=inst.LastDirection），此处按同一起点
		// 重算本期投向，再交由 advancePickState 写回状态机。
		dec.Direction = nextTriggerDirection(cfg.Trigger.Mode, inst.LastDirection)
	}
	return advancePickState(cfg, inst, dec, hit)
}

// advancePickState 结算后推进出号游标（写回 pick_index / current_pick / last_direction）。
func advancePickState(
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	dec pickDecision,
	hit bool,
) (pickIndex int32, currentPick string, lastDirection string) {
	pickIndex = inst.PickIndex
	currentPick = inst.CurrentPick
	lastDirection = inst.LastDirection
	if cfg.Contrary || cfg.Kind != "custom" || cfg.RunTypeID == "" {
		return pickIndex, currentPick, lastDirection
	}
	switch cfg.RunTypeID {
	case RunTypeFixedRotate:
		n := cfg.GroupCount
		if n <= 0 {
			n = 1
		}
		pickIndex = (inst.PickIndex + 1) % int32(n)
	case RunTypeAdvFixedRotate:
		// 优先按本期实际下注内容定位局号，避免游标被并发/回头复位打乱后跳错
		row := jushuRowForContent(cfg.Jushu, dec.Content, int(inst.PickIndex))
		next := row.AfterMiss
		if hit {
			next = row.AfterHit
		}
		if !jushuExists(cfg.Jushu, next) {
			next = firstJu(cfg.Jushu)
		}
		pickIndex = int32(next)
	case RunTypeAdvTriggerBet:
		if dec.Direction != "" {
			lastDirection = dec.Direction
		}
	case RunTypeHotColdWarm:
		// 换号：清空 current_pick，下期按统计期数+出号类型重新取码（非池内 +1）
		strategy := hotColdStrategy(cfg.HotCold)
		switch strategy {
		case "every":
			currentPick = ""
		case "after_hit":
			if hit {
				currentPick = ""
			} else {
				currentPick = dec.Content
			}
		case "after_miss":
			if hit {
				currentPick = dec.Content
			} else {
				currentPick = ""
			}
		default: // keep 不换号：锁定本期内容
			currentPick = dec.Content
		}
	case RunTypeRandomDraw:
		strategy := "every"
		if cfg.Random != nil && cfg.Random.Strategy != "" {
			strategy = cfg.Random.Strategy
		}
		switch strategy {
		case "keep":
			currentPick = dec.Content
		case "after_hit": // 中后换：命中则下期重新随机
			if hit {
				currentPick = ""
			} else {
				currentPick = dec.Content
			}
		case "after_miss": // 挂后换：未中则下期重新随机
			if hit {
				currentPick = dec.Content
			} else {
				currentPick = ""
			}
		default: // every 每期换
			currentPick = ""
		}
	}
	return pickIndex, currentPick, lastDirection
}

// ---------- 定码轮换 ----------

func pickFixedRotate(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance) pickDecision {
	if len(cfg.Groups) == 0 {
		return pickDecision{Content: cfg.GroupContent}
	}
	idx := int(inst.PickIndex) % len(cfg.Groups)
	if idx < 0 {
		idx = 0
	}
	return pickDecision{Content: cfg.Groups[idx]}
}

// ---------- 高级定码轮换（局数列表） ----------

func currentJushuRow(rows []jushuRow, cur int) jushuRow {
	if len(rows) == 0 {
		return jushuRow{Ju: 1, AfterHit: 1, AfterMiss: 1}
	}
	if cur <= 0 {
		cur = rows[0].Ju
	}
	for _, r := range rows {
		if r.Ju == cur {
			return r
		}
	}
	return rows[0]
}

// jushuRowForContent 按本期实际内容匹配局；匹配失败再回退 pick_index。
func jushuRowForContent(rows []jushuRow, content string, pickIndex int) jushuRow {
	want := normalizeSchemeGroupContent(content)
	if strings.TrimSpace(want) != "" {
		for _, r := range rows {
			if normalizeSchemeGroupContent(r.Content) == want {
				return r
			}
		}
	}
	return currentJushuRow(rows, pickIndex)
}

func jushuExists(rows []jushuRow, ju int) bool {
	for _, r := range rows {
		if r.Ju == ju {
			return true
		}
	}
	return false
}

func firstJu(rows []jushuRow) int {
	if len(rows) == 0 {
		return 1
	}
	return rows[0].Ju
}

func pickJushuList(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance) pickDecision {
	row := currentJushuRow(cfg.Jushu, int(inst.PickIndex))
	if strings.TrimSpace(row.Content) == "" {
		return pickDecision{Content: cfg.GroupContent}
	}
	return pickDecision{Content: row.Content}
}

// betRoundLabel 投注记录「轮次」：始终为倍投轮次（1-based），与出号局数/组号无关。
func betRoundLabel(_ parsedSchemeConfig, roundIdx int, _ int) string {
	return strconv.Itoa(roundIdx + 1)
}

// ---------- 固定取码 ----------

// pickFixedNumber 固定取码：每期复投 schemeGroups[0] 指定号码。
func pickFixedNumber(cfg parsedSchemeConfig) pickDecision {
	if len(cfg.Groups) > 0 && strings.TrimSpace(cfg.Groups[0]) != "" {
		return pickDecision{Content: cfg.Groups[0]}
	}
	return pickDecision{Content: cfg.GroupContent}
}

// ---------- 随机出号 ----------

func pickRandomDraw(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance) pickDecision {
	if cur := strings.TrimSpace(inst.CurrentPick); cur != "" && !isRandomOverMaxSkipMarker(cur) {
		// 缓存号若已超该玩法注数上限，丢弃并重抽（真下单前兜底）。
		cur = applyRenxuanRunPositionWrap(cfg, cur)
		if !contentExceedsBetUnitsMax(cfg.Play, cur) {
			return pickDecision{Content: cur}
		}
	}
	return pickDecision{Content: randomDrawContentUnderMax(cfg)}
}

func renxuanRunPositionIdxs(cfg parsedSchemeConfig) []int {
	if cfg.Random != nil && len(cfg.Random.PositionIdxs) > 0 {
		return cfg.Random.PositionIdxs
	}
	if cfg.HotCold != nil && len(cfg.HotCold.PositionIdxs) > 0 {
		return cfg.HotCold.PositionIdxs
	}
	return nil
}

func applyRenxuanRunPositionWrap(cfg parsedSchemeConfig, picks string) string {
	picks = strings.TrimSpace(picks)
	if picks == "" || !isRenxuanNeedsPositionRule(cfg.Play) {
		return picks
	}
	k := cfg.Play.SegmentLen
	if k <= 0 {
		k = renPickCount(cfg.Play.CatalogSubID)
	}
	if k <= 0 {
		k = 2
	}
	// 单式类：冷热/随机若产出按位号池（4\n5），先展开再加选位前缀
	if isRenxuanPositionDanshiRule(cfg.Play) {
		if _, _, ok := parseRenxuanPosPicksContent(picks, k); !ok {
			if exp, ok := expandZhixuanPositionPoolToDanshi(picks, k); ok && strings.TrimSpace(exp) != "" {
				picks = exp
			}
		}
	}
	return wrapRenxuanNeedsPositionContent(cfg.Play, picks, renxuanRunPositionIdxs(cfg))
}

// randomDrawContentUnderMax 生成随机内容；有玩法上限时重抽，并逐步收缩选号规模直至可下单。
func randomDrawContentUnderMax(cfg parsedSchemeConfig) string {
	max := maxBetUnitsForPlay(cfg.Play)
	const (
		maxScale    = 12
		maxAttempts = 16
	)
	var last string
	for scale := 0; scale <= maxScale; scale++ {
		for attempt := 0; attempt < maxAttempts; attempt++ {
			last = applyRenxuanRunPositionWrap(cfg, generateRandomDrawContent(cfg, scale))
			if max <= 0 {
				return last
			}
			n := countPlayWireBetUnits(cfg.Play, last)
			if n > 0 && n <= max {
				return last
			}
		}
	}
	// 属性和值/跨度：贪心取组合数较小的选项，避免无解卡死
	bm := strings.ToLower(strings.TrimSpace(cfg.Play.BetMode))
	if bm == "kuadu" || bm == "hezhi" {
		k := randomDrawCountAt(cfg, 0, 0)
		if fallback := greedyAttributeContentUnderMax(cfg.Play, k, max); fallback != "" {
			return applyRenxuanRunPositionWrap(cfg, fallback)
		}
		// 再降到 1 个选项，保证可下单
		if fallback := greedyAttributeContentUnderMax(cfg.Play, 1, max); fallback != "" {
			return applyRenxuanRunPositionWrap(cfg, fallback)
		}
	}
	// 绝不返回超限内容（避免 worker 回落到 schemeGroups 满选）
	if max > 0 && countPlayWireBetUnits(cfg.Play, last) > max {
		return ""
	}
	return last
}

// generateRandomDrawContent scale>0 时按档收缩选号个数（每位/每档至少 1）。
func generateRandomDrawContent(cfg parsedSchemeConfig, scale int) string {
	if scale < 0 {
		scale = 0
	}
	// 单式/组选单式：整注随机
	if isWholeTicketRandom(cfg.Play) {
		n := randomDrawCountAt(cfg, 0, 1)
		n = shrinkCount(n, scale, 1)
		return randomWholeTickets(cfg.Play, n)
	}
	// 组选号池
	if isZuxuanPoolRandom(cfg.Play) {
		k := randomDrawCountAt(cfg, 0, 0)
		k = shrinkCount(k, scale, 1)
		return randomZuxuanPool(cfg.Play, k)
	}
	// 属性/聚合
	if isAttributeRandom(cfg.Play) {
		k := randomDrawCountAt(cfg, 0, 0)
		k = shrinkCount(k, scale, 1)
		return randomAttributeContent(cfg.Play, k)
	}
	// 按位大小单双
	if isPerPosDxdsRandom(cfg.Play) {
		positions := playPositionCount(cfg.Play)
		lines := make([]string, 0, positions)
		for i := 0; i < positions; i++ {
			count := shrinkCount(randomDrawCountAt(cfg, i, 1), scale, 1)
			lines = append(lines, randomAttributeContent(cfg.Play, count))
		}
		return strings.Join(lines, "\n")
	}
	// 按位号池（直选复式/单式展开等）
	positions := playPositionCount(cfg.Play)
	lines := make([]string, 0, positions)
	for i := 0; i < positions; i++ {
		count := shrinkCount(randomDrawCountAt(cfg, i, 1), scale, 1)
		lines = append(lines, randomDigits(cfg.Play, count))
	}
	return normalizeZhixuanDanshiContent(cfg.Play, strings.Join(lines, "\n"))
}

func randomDrawCountAt(cfg parsedSchemeConfig, idx, defaultN int) int {
	if cfg.Random != nil && idx < len(cfg.Random.Counts) && cfg.Random.Counts[idx] > 0 {
		return cfg.Random.Counts[idx]
	}
	return defaultN
}

func shrinkCount(n, scale, min int) int {
	if min < 1 {
		min = 1
	}
	if n < min {
		n = min
	}
	n -= scale
	if n < min {
		return min
	}
	return n
}

// attributeCountFeasibleUnderMax 选 k 个属性选项时，最小可能组合注数是否 ≤ max。
func attributeCountFeasibleUnderMax(rule playRule, k, max int) bool {
	if k <= 0 || max <= 0 {
		return true
	}
	universe := attributeUniverse(rule)
	if len(universe) == 0 || k > len(universe) {
		return false
	}
	units := make([]int, 0, len(universe))
	for _, tok := range universe {
		n := countPlayWireBetUnits(rule, tok)
		if n > 0 {
			units = append(units, n)
		}
	}
	if len(units) < k {
		return false
	}
	sort.Ints(units)
	sum := 0
	for i := 0; i < k; i++ {
		sum += units[i]
	}
	return sum <= max
}

// isWholeTicketRandom 判定为"整注型"玩法——随机产号需抽完整组合，而非按位号池。
// 组选单式走整注；前/中/后三直选单式、混合组选（段长≥2）走按位随机再展开。
func isWholeTicketRandom(rule playRule) bool {
	// 混合组选：与直选复式同按位（千/百/十），下注前再展成整注并排除豹子
	if isHunhePlayRule(rule) {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID))
	if bm == "zuxuan_ds" || sub == "zuxuan_ds" || strings.Contains(sub, "zuxuan_ds") {
		return true
	}
	// 前三/中三/后三/前二/后二直选单式：按位（万千百 / 千百十 / …）
	if isZhixuanDanshiTriggerPlay(rule) {
		return false
	}
	switch bm {
	case "danshi", "zhixuan_ds":
		return true
	}
	switch sub {
	case "zhixuan_ds":
		return true
	}
	return false
}

// isAttributeRandom 判定为"属性/聚合型"玩法（大小单双/龙虎/特殊号/庄闲/和值/跨度/尾数/不定位/包胆）——
// 随机产号为"从选项宇宙抽 K 个"，非按位号池、非整注单式。
// 前二/后二/前三/后三大小单双（SegmentLen>=2）走按位抽样，不算单档属性。
func isAttributeRandom(rule playRule) bool {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "dxds":
		return !isPerPosDxdsRandom(rule)
	case "daxiao", "danshuang", "zhuangxian",
		"longhu", "longhuhe", "longhubao", "teshu",
		"hezhi", "kuadu", "weishu", "budingwei", "baodan":
		return true
	}
	return false
}

// randomDrawCountMax 与编辑页 rdSingleCountMax / rdPerPosMax 对齐：按玩法宇宙定上限。
// 整注 200、组选号池=号池长度、属性=选项宇宙、包胆 1、尾数≤9、按位默认 10（一星 9、按位大小单双 4）。
func randomDrawCountMax(rule playRule) int {
	if isWholeTicketRandom(rule) {
		return 200
	}
	if isZuxuanPoolRandom(rule) {
		n := len(playNumberPool(rule))
		if n < 3 {
			return 3
		}
		return n
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if isAttributeRandom(rule) {
		switch bm {
		case "baodan":
			return 1
		case "weishu":
			// 与前端 WEISHU_MAX_BET_UNITS 一致
			n := len(attributeUniverse(rule))
			if n <= 0 {
				n = 10
			}
			if n > weishuMaxBetUnitsCap {
				return weishuMaxBetUnitsCap
			}
			return n
		case "kuadu", "hezhi":
			// 选项个数上限还受组合注数约束（三星跨度满选 10→1000>900，最多 9）
			u := attributeUniverse(rule)
			n := len(u)
			if n < 1 {
				n = len(playNumberPool(rule))
			}
			if n < 1 {
				return 1
			}
			if max := maxBetUnitsForPlay(rule); max > 0 {
				for n > 1 && !attributeCountFeasibleUnderMax(rule, n, max) {
					n--
				}
			}
			return n
		case "budingwei":
			// 一码不定位第三方最多 2 个号（超过 →「投注数字不可超过两位数」）
			need := budingweiNeedCount(rule.CatalogSubID)
			if need <= 1 {
				return 2
			}
			n := len(playNumberPool(rule))
			if n < 1 {
				return 1
			}
			return n
		default:
			u := attributeUniverse(rule)
			if len(u) > 0 {
				return len(u)
			}
			n := len(playNumberPool(rule))
			if n < 1 {
				return 1
			}
			return n
		}
	}
	if isPerPosDxdsRandom(rule) {
		return 4
	}
	// 一星定位胆：每位最多 9（与前端 YIXING_MAX_PICKS_PER_POS 对齐）
	if isDingweiTriggerPlay(rule) {
		return 9
	}
	return 10
}

// isPerPosDxdsRandom 前二/后二/前三/后三大小单双：按位随机（十\n个），非整期单档。
// 五星和值大小/单双、PC28 整期大小单双仍走 isAttributeRandom。
func isPerPosDxdsRandom(rule playRule) bool {
	if strings.ToLower(strings.TrimSpace(rule.BetMode)) != "dxds" {
		return false
	}
	if rule.SegmentLen < 2 {
		return false
	}
	if isWuxingSumDxdsRule(rule) {
		return false
	}
	if rule.PlayTemplate == "pc28_std" {
		return false
	}
	return true
}

// attributeUniverse 属性/聚合玩法的合法选项宇宙（供随机抽样）。数字池型（不定位/包胆）返回 nil，另行处理。
func attributeUniverse(rule playRule) []string {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "daxiao":
		return []string{"大", "小"}
	case "danshuang":
		return []string{"单", "双"}
	case "dxds":
		return []string{"大", "小", "单", "双"}
	case "zhuangxian":
		return []string{"庄", "闲"}
	case "longhu":
		return []string{"龙", "虎"}
	case "longhuhe":
		return []string{"龙", "虎", "和"}
	case "longhubao":
		return []string{"龙", "虎", "豹"}
	case "teshu":
		if rule.PlayTemplate == "pc28_std" {
			return []string{"豹子", "对子", "顺子", "极大", "极小"}
		}
		return []string{"豹子", "对子", "顺子"}
	case "hezhi":
		min, max := ruleNumberPool(rule)
		segLen := rule.SegmentLen
		if segLen < 1 {
			segLen = 1
		}
		lo, hi := segLen*min, segLen*max
		// 组选和值排除仅豹子（各位同码）可组成的极值和：三星 0(000)/27(999) → 1..26，前二 0/18 → 1..17。
		if rule.HezhiZuxuan && hi-lo >= 2 {
			lo, hi = lo+1, hi-1
		}
		out := make([]string, 0, hi-lo+1)
		for v := lo; v <= hi; v++ {
			out = append(out, strconv.Itoa(v))
		}
		return out
	case "kuadu":
		min, max := ruleNumberPool(rule)
		out := make([]string, 0, max-min+1)
		for v := 0; v <= max-min; v++ {
			out = append(out, strconv.Itoa(v))
		}
		return out
	case "weishu":
		// 和值尾数下的是和值的个位，与号池和区位长度无关，恒为 0-9。
		out := make([]string, 0, 10)
		for v := 0; v <= 9; v++ {
			out = append(out, strconv.Itoa(v))
		}
		return out
	}
	return nil
}

// greedyAttributeContentUnderMax 和值/跨度兜底：按单选项 wire 注数升序贪心选取。
func greedyAttributeContentUnderMax(rule playRule, k, maxUnits int) string {
	if maxUnits <= 0 {
		return ""
	}
	universe := attributeUniverse(rule)
	if len(universe) == 0 {
		return ""
	}
	if k < 1 {
		k = 1
	}
	type item struct {
		tok   string
		units int
	}
	items := make([]item, 0, len(universe))
	for _, tok := range universe {
		n := countPlayWireBetUnits(rule, tok)
		if n <= 0 {
			continue
		}
		items = append(items, item{tok: tok, units: n})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].units != items[j].units {
			return items[i].units < items[j].units
		}
		return items[i].tok < items[j].tok
	})
	picked := make([]string, 0, k)
	for _, it := range items {
		if len(picked) >= k {
			break
		}
		trial := strings.Join(append(append([]string{}, picked...), it.tok), ",")
		if countPlayWireBetUnits(rule, trial) > maxUnits {
			continue
		}
		picked = append(picked, it.tok)
	}
	if len(picked) == 0 && len(items) > 0 && items[0].units <= maxUnits {
		return items[0].tok
	}
	sort.Strings(picked)
	return strings.Join(picked, ",")
}

// randomAttributeContent 从属性/聚合玩法的选项宇宙随机抽 k 个（去重、逗号分隔）。
// 不定位/包胆为数字池型：抽 k 个不重复号码（不定位下限=选码位数）。
func randomAttributeContent(rule playRule, k int) string {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "budingwei" || bm == "baodan" {
		pool := playNumberPool(rule)
		if len(pool) == 0 {
			return ""
		}
		minK := 1
		maxK := len(pool)
		if bm == "budingwei" {
			minK = budingweiNeedCount(rule.CatalogSubID)
			// 五星二/三码至少 4；普通二码至少 2
			if mp := budingweiMinPoolForRule(rule); mp > minK {
				minK = mp
			}
			// 一码不定位第三方最多 2 个号（超过 →「投注数字不可超过两位数」）
			if isYimaBudingweiPlayRule(rule) {
				maxK = 2
			}
		}
		if bm == "baodan" {
			// 组选包胆第三方仅单胆
			maxK = 1
		}
		if k < minK {
			k = minK
		}
		if k > maxK {
			k = maxK
		}
		idx := rand.Perm(len(pool))[:k]
		sort.Ints(idx)
		out := make([]string, 0, k)
		for _, i := range idx {
			out = append(out, pool[i])
		}
		return strings.Join(out, ",")
	}
	universe := attributeUniverse(rule)
	if len(universe) == 0 {
		return ""
	}
	if k < 1 {
		k = 1
	}
	if k > len(universe) {
		k = len(universe)
	}
	idx := rand.Perm(len(universe))[:k]
	sort.Ints(idx)
	out := make([]string, 0, k)
	for _, i := range idx {
		out = append(out, universe[i])
	}
	return strings.Join(out, ",")
}

// isZuxuanPoolRandom 判定为"组选号码池型"玩法（组三/组六/组选N/组选复式）——
// 随机产号为"选 K 个号组成号码池"，非按位、也非整注单式。
// 组选包胆/组选和值属属性单选，勿因 catalog/文案含 zuxuan|组选 误入号池分支。
func isZuxuanPoolRandom(rule playRule) bool {
	if isWholeTicketRandom(rule) {
		return false
	}
	// 混合组选按位产号，勿因 catalog 含 hunhe/zuxuan 误入号池
	if isHunhePlayRule(rule) {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	switch bm {
	case "baodan", "hezhi", "kuadu", "weishu", "budingwei":
		// 组选和值等：选项个数下限 1，走 isAttributeRandom（勿被 catalog 含 zuxuan 抬成组选号池）
		return false
	// zuxuan_fs：目录 subId 常为数字（如 g004/42），不能只靠 catalog 含 zuxuan_fs 判断；
	// 漏判会走按位随机 → 位间重号（如 3,5\n5 → 3,5,5）被第三方拒「投注数字不合规」。
	case "zuxuan_fs", "zu3", "zu6", "zu24", "zu12", "zu60", "zu30", "zu120":
		return true
	}
	cat := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	if cat == "" {
		return false
	}
	if strings.Contains(cat, "baodan") || strings.Contains(cat, "_bd") {
		return false
	}
	// catalog 带 hezhi/_hz/kuadu 时勿进组选号池（即使同时含 zuxuan）
	if strings.Contains(cat, "hezhi") || strings.Contains(cat, "_hz") || strings.Contains(cat, "kuadu") {
		return false
	}
	if strings.Contains(cat, "zuxuan_fs") {
		return true
	}
	// 兼容 zu3/zu6/zuxuan/zu24… 标记出现在子玩法/目录 id 中
	for _, k := range []string{"zu3", "zu6", "zu24", "zu12", "zu60", "zu30", "zu120", "zuxuan"} {
		if strings.Contains(cat, k) {
			return true
		}
	}
	return false
}

// zuxuanPoolMinPick 组选号池随机最少选几个号。
// 组三最少 2（n*(n-1) 注）；组六最少 3；勿用 SegmentLen 把组三抬成 3。
// 包胆/和值/跨度等非组选号池玩法返回 0（勿因中三 SegmentLen=3 误报「号码池至少选择 3 个」）。
func zuxuanPoolMinPick(rule playRule) int {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	switch bm {
	case "baodan", "hezhi", "kuadu", "weishu", "budingwei", "teshu",
		"daxiao", "danshuang", "dxds", "longhu", "longhuhe", "longhubao", "zhuangxian":
		return 0
	case "zu3":
		return 2
	case "zu6":
		return 3
	case "zuxuan_fs":
		// 前二/后二 C(n,2)、三星组选复式（组三+组六）均至少 2 码
		return 2
	case "zu24", "zu12", "zu60", "zu30", "zu120":
		if n := playPositionCount(rule); n >= 2 {
			return n
		}
		return 3
	}
	// catalog 标明包胆/和值等：即使位长=3 也不套用组选号池下限
	if strings.Contains(sub, "baodan") || strings.Contains(sub, "_bd") ||
		strings.Contains(sub, "hezhi") || strings.Contains(sub, "_hz") ||
		strings.Contains(sub, "kuadu") || strings.Contains(sub, "budingwei") {
		return 0
	}
	if strings.Contains(sub, "zu3") && !strings.Contains(sub, "zu30") {
		return 2
	}
	if strings.Contains(sub, "zu6") {
		return 3
	}
	// 未识别为组三/组六/组选N 时不加硬下限（避免包胆/未知玩法被 SegmentLen 抬成 ≥2/≥3）
	return 0
}

// randomZuxuanPool 随机选 k 个不重复号码组成组选号码池（升序，逗号分隔）。
// k 下限见 zuxuanPoolMinPick，上限为号池大小。
func randomZuxuanPool(rule playRule, k int) string {
	pool := playNumberPool(rule)
	if len(pool) == 0 {
		return ""
	}
	minK := zuxuanPoolMinPick(rule)
	if k < minK {
		k = minK
	}
	if k > len(pool) {
		k = len(pool)
	}
	idx := rand.Perm(len(pool))[:k]
	sort.Ints(idx)
	out := make([]string, 0, k)
	for _, i := range idx {
		out = append(out, pool[i])
	}
	return strings.Join(out, ",")
}

// randomWholeTickets 随机抽 n 个完整组合（每位随机取一个号拼成一注），去重。
// 组选单式（zuxuan_ds）内位号升序归一，按组合去重；直选单式保留位序。
// 内容格式与 evaluateZhixuanDanshi 兼容：逗号分隔的定长 token。
func randomWholeTickets(rule playRule, n int) string {
	positions := playPositionCount(rule)
	if positions <= 0 {
		positions = 1
	}
	pool := playNumberPool(rule)
	if len(pool) == 0 {
		return ""
	}
	if n < 1 {
		n = 1
	}
	// 上限保护：最多 200 注
	const maxN = 200
	if n > maxN {
		n = maxN
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID))
	isHunhe := bm == "hunhe"
	// 组选单式 / 混合组选单式：位号升序归一（按组合去重）；混合额外排除豹子（全同号）。
	isZuxuan := isHunhe || bm == "zuxuan_ds" || sub == "zuxuan_ds"
	seen := make(map[string]struct{}, n)
	out := make([]string, 0, n)
	for attempts := 0; len(out) < n && attempts < n*100+100; attempts++ {
		digits := make([]string, positions)
		for p := 0; p < positions; p++ {
			digits[p] = pool[rand.Intn(len(pool))]
		}
		key := strings.Join(digits, "")
		if isZuxuan {
			sorted := append([]string(nil), digits...)
			sort.Strings(sorted)
			key = strings.Join(sorted, "")
			digits = sorted
		}
		if (isHunhe || isZuxuan) && allSameTokens(digits) {
			// 混合/组选单式排除豹子（二星即对子；第三方计 0 注）
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, strings.Join(digits, ""))
	}
	return strings.Join(out, ",")
}

// allSameTokens 判定所有 token 相同（豹子/全同号）。
func allSameTokens(tokens []string) bool {
	if len(tokens) <= 1 {
		return false
	}
	for i := 1; i < len(tokens); i++ {
		if tokens[i] != tokens[0] {
			return false
		}
	}
	return true
}

func playPositionCount(rule playRule) int {
	if len(rule.SegmentPos) > 0 {
		return len(rule.SegmentPos)
	}
	if rule.SegmentLen > 1 {
		return rule.SegmentLen
	}
	return 1
}

func playNumberPool(rule playRule) []string {
	min, max := ruleNumberPool(rule)
	if max-min > 98 {
		max = min + 98
	}
	pool := make([]string, 0, max-min+1)
	for v := min; v <= max; v++ {
		// 11 选 5（max==11）补零；六合/PK10/时时彩保持与开奖球/历史池一致的无补零或单位数形态
		if max == 11 {
			pool = append(pool, fmt.Sprintf("%02d", v))
		} else {
			pool = append(pool, strconv.Itoa(v))
		}
	}
	return pool
}

func randomDigits(rule playRule, count int) string {
	pool := playNumberPool(rule)
	if count < 1 {
		count = 1
	}
	if count > len(pool) {
		count = len(pool)
	}
	idx := rand.Perm(len(pool))[:count]
	sort.Ints(idx)
	out := make([]string, 0, count)
	for _, i := range idx {
		out = append(out, pool[i])
	}
	return strings.Join(out, ",")
}

// ---------- 冷热出号 ----------

func hotColdStrategy(hc *hotColdWarmCfg) string {
	if hc == nil {
		return "keep"
	}
	if hc.Strategy != "" {
		return hc.Strategy
	}
	if hc.WinRotate {
		return "after_hit"
	}
	return "keep"
}

func (w *Worker) pickHotColdWarm(
	ctx context.Context,
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	draw sqlcdb.LotteryDraw,
) pickDecision {
	if cur := strings.TrimSpace(inst.CurrentPick); cur != "" && !hotColdPickNeedsRebuild(cfg, cur) {
		cur = applyRenxuanRunPositionWrap(cfg, cur)
		return pickDecision{Content: normalizeZhixuanDanshiContent(cfg.Play, cur)}
	}
	periods := 20
	if cfg.HotCold != nil && cfg.HotCold.TotalPeriods > 0 {
		periods = cfg.HotCold.TotalPeriods
	}
	draws := w.recentDrawBalls(ctx, inst.LotteryCode, draw.IssueNo, periods)
	content := applyRenxuanRunPositionWrap(cfg, buildHotColdPickContent(cfg, draws))
	// 不再把手选号池/schemeGroups 整页回退成固定复式
	return pickDecision{Content: normalizeZhixuanDanshiContent(cfg.Play, content)}
}

// pickHotColdWarmFromDraws 供预览/单测：无 DB 时用传入开奖序列取码。
func pickHotColdWarmFromDraws(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance, draws [][]string) pickDecision {
	if cur := strings.TrimSpace(inst.CurrentPick); cur != "" && !hotColdPickNeedsRebuild(cfg, cur) {
		cur = applyRenxuanRunPositionWrap(cfg, cur)
		return pickDecision{Content: normalizeZhixuanDanshiContent(cfg.Play, cur)}
	}
	content := applyRenxuanRunPositionWrap(cfg, buildHotColdPickContent(cfg, draws))
	return pickDecision{Content: normalizeZhixuanDanshiContent(cfg.Play, content)}
}

// hotColdPickNeedsRebuild 多位面板却保了单位内容（无换行）时强制重取，避免旧引擎单号锁死。
// 直选单式展开后的整注串（如 "555" / "432,435"）无换行，属合法保号，不重取。
func hotColdPickNeedsRebuild(cfg parsedSchemeConfig, currentPick string) bool {
	// 任选选位：出号为「位名\n号码」或整体号池，勿按多位面板强制重取
	if isRenxuanNeedsPositionRule(cfg.Play) {
		return false
	}
	if playPositionCount(cfg.Play) <= 1 {
		return false
	}
	if strings.Contains(currentPick, "\n") {
		return false
	}
	seg := cfg.Play.SegmentLen
	if seg <= 0 {
		seg = playPositionCount(cfg.Play)
	}
	if seg > 1 && len(parseNumberTokens(currentPick, seg)) > 0 {
		return false
	}
	return true
}

// recentDrawBalls 取当期之前最近 N 期开奖球（不含当期）。
func (w *Worker) recentDrawBalls(ctx context.Context, lotteryCode, currentIssue string, periods int) [][]string {
	if w == nil || w.q == nil || periods <= 0 {
		return nil
	}
	if periods > 500 {
		periods = 500
	}
	rows, err := w.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    int32(periods + 8),
	})
	if err != nil || len(rows) == 0 {
		return nil
	}
	out := make([][]string, 0, periods)
	for _, r := range rows {
		// 数值期号比较，避免纯字符串比较在乱序族下误过滤
		if currentIssue != "" && compareIssueNo(r.IssueNo, currentIssue) >= 0 {
			continue
		}
		balls := sqlcdb.ParseDrawBalls(r.Balls)
		if len(balls) == 0 {
			continue
		}
		out = append(out, balls)
		if len(out) >= periods {
			break
		}
	}
	return out
}

// buildHotColdPickContent 按名次取号：
//
//	近 N 期频次「最热→最冷」排序后，按 hc.Ranks（0=最热）取当前号码。
//	无 Ranks 时由 pickTypes 展开为热/冷半区名次（空则默认热区）。
//	组选/不定位等整体型：配置了号码 pool（如 "0,1,6,7,9"）时锁定频次宇宙，不得出池外号。
//	按位型：pool 只作位启用标记，不锁死预览复式。
func buildHotColdPickContent(cfg parsedSchemeConfig, draws [][]string) string {
	hc := cfg.HotCold
	if hc == nil {
		return ""
	}
	pool := playNumberPool(cfg.Play)

	// 属性家族（大小单双/龙虎/和值等）；任选和值按投注选位计频
	if isHotColdAttributePlay(cfg.Play) {
		if !hotColdLineEnabled(hc, 0) {
			return ""
		}
		res := HotColdWarmAttributeTiersForPositions(cfg.Play, draws, hc.PositionIdxs)
		full := append(append([]string{}, res.Hot...), res.Cold...)
		picked := pickHotColdByRanks(full, resolveHotColdRanksForOrder(hc, 0, len(full)))
		return strings.Join(sortHotColdBetTokens(picked), ",")
	}
	// 号码整体频次（组选/不定位/包胆）；任选组选复式按投注选位合并计频
	if isHotColdDigitOverall(cfg.Play) {
		if !hotColdLineEnabled(hc, 0) {
			return ""
		}
		pool = hotColdLockedDigitPool(hc, pool, 0)
		hot, cold := hotColdWarmTiersOverallForPositions(draws, cfg.Play, pool, hc.PositionIdxs)
		full := append(append([]string{}, hot...), cold...)
		ranks := resolveHotColdRanksForOrder(hc, 0, len(full))
		// 组选包胆第三方仅单胆：多档名次只取最先命中的一码（勿拼成 1,3,5,6,7 →「投注数字不合规」）
		if isBaodanPlayRule(cfg.Play) && len(ranks) > 1 {
			ranks = ranks[:1]
		}
		// 一码不定位最多 2 个号（超过 →「投注数字不可超过两位数」）
		if isYimaBudingweiPlayRule(cfg.Play) && len(ranks) > 2 {
			ranks = ranks[:2]
		}
		picked := pickHotColdByRanks(full, ranks)
		if isBaodanPlayRule(cfg.Play) && len(picked) > 1 {
			picked = picked[:1]
		}
		if isYimaBudingweiPlayRule(cfg.Play) && len(picked) > 2 {
			picked = picked[:2]
		}
		return strings.Join(sortHotColdBetTokens(picked), ",")
	}
	// 任选·直选单式：按开奖选位（恰好 k 个绝对位）计频取号，再由 applyRenxuanRunPositionWrap 加投注选位前缀
	if content, ok := buildRenxuanHcwOpenPosPickContent(cfg, draws, pool); ok {
		return content
	}
	// 按位型：pool 非空仅表示该位启用（编辑预览），运行时仍用玩法全号池动态热冷区
	n := playPositionCount(cfg.Play)
	enabled := hotColdPositionEnabled(hc, n)
	lines := make([]string, n)
	filled := 0
	for i := 0; i < n; i++ {
		if !enabled[i] {
			continue
		}
		pos := hotColdPositionIdx(cfg.Play, i)
		hot, _, cold := hotColdWarmTiers(draws, pos, pool)
		full := append(append([]string{}, hot...), cold...)
		picked := pickHotColdByRanks(full, resolveHotColdRanksForOrder(hc, i, len(full)))
		lines[i] = strings.Join(sortHotColdBetTokens(picked), ",")
		if lines[i] != "" {
			filled++
		}
	}
	if filled == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// buildRenxuanHcwOpenPosPickContent 任选直选单式冷热：按 openPositionIdxs（k 个绝对位）取各列冷热号。
// 返回按位号池（行用 \n 分隔），供后续 expand + 投注选位前缀。
func buildRenxuanHcwOpenPosPickContent(cfg parsedSchemeConfig, draws [][]string, pool []string) (string, bool) {
	if !isRenxuanPerPosTriggerPlay(cfg.Play) {
		return "", false
	}
	hc := cfg.HotCold
	if hc == nil {
		return "", false
	}
	k := cfg.Play.SegmentLen
	if k <= 0 {
		k = renPickCount(cfg.Play.CatalogSubID)
	}
	if k <= 0 {
		k = 2
	}
	openIdxs := normalizeRenxuanHcwOpenPositionIdxs(hc.OpenPositionIdxs, k)
	lines := make([]string, len(openIdxs))
	filled := 0
	for i, pos := range openIdxs {
		if !hotColdLineEnabled(hc, i) {
			continue
		}
		hot, _, cold := hotColdWarmTiers(draws, pos, pool)
		full := append(append([]string{}, hot...), cold...)
		picked := pickHotColdByRanks(full, resolveHotColdRanksForOrder(hc, i, len(full)))
		lines[i] = strings.Join(sortHotColdBetTokens(picked), ",")
		if lines[i] != "" {
			filled++
		}
	}
	if filled == 0 {
		return "", true // 已识别该玩法但未选出号
	}
	return strings.Join(lines, "\n"), true
}

// hotColdLockedDigitPool 若配置了号码池则返回池内号码（与玩法宇宙求交）；否则用默认宇宙。
// 组选整体：pool[0]="0,1,6,7,9"；按位：pool[i] 为该位号串。
func hotColdLockedDigitPool(hc *hotColdWarmCfg, fallback []string, lineIdx int) []string {
	if hc == nil {
		return fallback
	}
	raw := ""
	if lineIdx >= 0 && lineIdx < len(hc.Pool) {
		raw = strings.TrimSpace(hc.Pool[lineIdx])
	}
	// 整体型常把号池写在首行；按位若本行空则不锁
	if raw == "" && lineIdx == 0 && len(hc.Pool) == 1 {
		raw = strings.TrimSpace(hc.Pool[0])
	}
	if raw == "" {
		return fallback
	}
	allow := map[string]bool{}
	for _, t := range parseDigitTokens(raw) {
		allow[t] = true
	}
	if len(allow) == 0 {
		return fallback
	}
	out := make([]string, 0, len(allow))
	for _, d := range fallback {
		if allow[d] {
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		// 池内号码不在玩法宇宙时仍按配置顺序用池
		for _, t := range parseDigitTokens(raw) {
			out = append(out, t)
		}
	}
	return out
}

// resolveHotColdRanksForOrder 解析名次；若配置名次相对全号池而当前宇宙已缩小导致全无效，
// 则回退为按 pickTypes 在当前宇宙上取热/冷半区。
func resolveHotColdRanksForOrder(hc *hotColdWarmCfg, lineIdx, orderLen int) []int {
	ranks := resolveHotColdRanks(hc, lineIdx, orderLen)
	if len(ranks) > 0 || hc == nil || orderLen <= 0 {
		return ranks
	}
	if lineIdx >= 0 && lineIdx < len(hc.Ranks) && len(hc.Ranks[lineIdx]) > 0 {
		// 有配置名次但相对当前宇宙全越界 → 按 pickTypes 重算
		types := normalizeHotColdPickTypes(hc.PickTypes)
		if len(types) == 0 {
			types = []string{"hot"}
		}
		wantHot, wantCold := hotColdWants(types)
		half := (orderLen + 1) / 2
		var out []int
		if wantHot {
			for i := 0; i < half; i++ {
				out = append(out, i)
			}
		}
		if wantCold {
			for i := half; i < orderLen; i++ {
				out = append(out, i)
			}
		}
		return out
	}
	return ranks
}

// resolveHotColdRanks 优先用配置名次；否则由 pickTypes 展开为半区名次。
func resolveHotColdRanks(hc *hotColdWarmCfg, lineIdx, orderLen int) []int {
	if hc == nil || orderLen <= 0 {
		return nil
	}
	if lineIdx >= 0 && lineIdx < len(hc.Ranks) && len(hc.Ranks[lineIdx]) > 0 {
		return normalizeHotColdRanks(hc.Ranks[lineIdx], orderLen)
	}
	types := normalizeHotColdPickTypes(hc.PickTypes)
	if len(types) == 0 {
		types = []string{"hot"}
	}
	wantHot, wantCold := hotColdWants(types)
	half := (orderLen + 1) / 2
	if half > orderLen {
		half = orderLen
	}
	var ranks []int
	if wantHot {
		for i := 0; i < half; i++ {
			ranks = append(ranks, i)
		}
	}
	if wantCold {
		for i := half; i < orderLen; i++ {
			ranks = append(ranks, i)
		}
	}
	return ranks
}

func normalizeHotColdRanks(ranks []int, orderLen int) []int {
	if orderLen <= 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(ranks))
	out := make([]int, 0, len(ranks))
	for _, r := range ranks {
		if r < 0 || r >= orderLen {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	return out
}

// pickHotColdByRanks 从「最热→最冷」全序按名次取号。
func pickHotColdByRanks(ordered []string, ranks []int) []string {
	if len(ordered) == 0 || len(ranks) == 0 {
		return nil
	}
	out := make([]string, 0, len(ranks))
	for _, r := range ranks {
		if r < 0 || r >= len(ordered) {
			continue
		}
		out = append(out, ordered[r])
	}
	return out
}

// pickHotColdZone 兼容旧调用：从全序取热/冷半区。
func pickHotColdZone(full []string, wantHot, wantCold bool) []string {
	n := len(full)
	if n == 0 {
		return nil
	}
	half := (n + 1) / 2
	if half > n {
		half = n
	}
	var ranks []int
	if wantHot {
		for i := 0; i < half; i++ {
			ranks = append(ranks, i)
		}
	}
	if wantCold {
		for i := half; i < n; i++ {
			ranks = append(ranks, i)
		}
	}
	return pickHotColdByRanks(full, ranks)
}

func hotColdHasAnyPool(pool []string) bool {
	for i := range pool {
		if hotColdManualAt(pool, i) != "" {
			return true
		}
	}
	return false
}

func hotColdLineEnabled(hc *hotColdWarmCfg, lineIdx int) bool {
	if hc == nil {
		return false
	}
	anyRanks := hotColdCfgHasRanks(hc.Ranks)
	anyPool := hotColdHasAnyPool(hc.Pool)
	if !anyRanks && !anyPool {
		return true
	}
	if anyRanks {
		return lineIdx >= 0 && lineIdx < len(hc.Ranks) && len(hc.Ranks[lineIdx]) > 0
	}
	return hotColdManualAt(hc.Pool, lineIdx) != ""
}

// hotColdPositionEnabled：有 ranks/pool 时仅启用对应非空位；否则全部启用。
func hotColdPositionEnabled(hc *hotColdWarmCfg, n int) []bool {
	out := make([]bool, n)
	for i := 0; i < n; i++ {
		out[i] = hotColdLineEnabled(hc, i)
	}
	return out
}

func sortHotColdBetTokens(tokens []string) []string {
	out := append([]string{}, tokens...)
	sort.SliceStable(out, func(i, j int) bool {
		ni, ei := strconv.Atoi(out[i])
		nj, ej := strconv.Atoi(out[j])
		if ei == nil && ej == nil {
			return ni < nj
		}
		return out[i] < out[j]
	})
	return out
}

// hotColdWants 出号类型 → 是否取热端/冷端。
func hotColdWants(types []string) (wantHot, wantCold bool) {
	for _, t := range types {
		if t == "hot" {
			wantHot = true
		}
		if t == "cold" {
			wantCold = true
		}
	}
	return wantHot, wantCold
}

// hotColdManualAt 取某位配置串；非空表示该号码位置启用（具体号码仅编辑预览）。
func hotColdManualAt(pool []string, i int) string {
	if i < 0 || i >= len(pool) {
		return ""
	}
	return strings.TrimSpace(pool[i])
}

func normalizeHotColdPickTypes(raw []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 2)
	for _, t := range raw {
		t = strings.ToLower(strings.TrimSpace(t))
		if (t == "hot" || t == "cold") && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func isHotColdDigitOverall(rule playRule) bool {
	// 混合组选：与直选复式同按位冷热（千/百/十），下注前再展成整注并排除豹子
	if isHunhePlayRule(rule) {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	switch bm {
	case "zu3", "zu6", "zu24", "zu12", "zu60", "zu30", "zu120", "budingwei", "baodan",
		"zuxuan_fs", "zuxuan_ds":
		// 组选单式勿走按位「5\n6」——guaji 计 0 注；整体号频后由 Format 展成整注
		return true
	}
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID) + " " + strings.TrimSpace(rule.CatalogSubID))
	return strings.Contains(sub, "zuxuan") || strings.Contains(sub, "zu3") || strings.Contains(sub, "zu6") ||
		strings.Contains(sub, "budingwei") || strings.Contains(sub, "baodan")
}

// isHotColdAttributePlay 该玩法下的是属性值（大小/单双/和值/尾数……）而不是球号。
//
// 名单要和 universeKindForRule 保持一致：那边认定为 attribute 的，这边就必须走属性计频。
// 漏掉一个的后果是静默的——冷热出号会掉进按位分支去统计原始球号频次，
// 于是给「后三和值尾数」选出 "180,280,380" 这种多位内容，第三方直接拒单，
// 注单永远卡在 pending。weishu 就这么漏了很久。
func isHotColdAttributePlay(rule playRule) bool {
	switch strings.ToLower(strings.TrimSpace(rule.BetMode)) {
	case "daxiao", "danshuang", "dxds", "zhuangxian",
		"longhu", "longhuhe", "longhubao", "teshu",
		"hezhi", "kuadu", "weishu":
		return true
	}
	return false
}

func hotColdPositionIdx(rule playRule, lineIdx int) int {
	if len(rule.SegmentPos) > 0 {
		if lineIdx >= 0 && lineIdx < len(rule.SegmentPos) {
			return rule.SegmentPos[lineIdx]
		}
		return rule.SegmentPos[0]
	}
	if rule.SegmentLen <= 1 {
		return rule.PositionIdx
	}
	return rule.SegmentStart + lineIdx
}

// hotColdWarmTiersOverall 跨位合并频次后二等分热/冷（组选/不定位/包胆）。
func hotColdWarmTiersOverall(draws [][]string, rule playRule, pool []string) (hot, cold []string) {
	return hotColdWarmTiersOverallForPositions(draws, rule, pool, nil)
}

// hotColdWarmTiersOverallForPositions 同 hotColdWarmTiersOverall；
// 任选组选复式等带选位玩法按 positionIdxs 合并计频（空则默认前 k 位，如任二万千）。
func hotColdWarmTiersOverallForPositions(draws [][]string, rule playRule, pool []string, positionIdxs []int) (hot, cold []string) {
	counts := make(map[string]int, len(pool))
	posList := renxuanOverallPositionIdxs(rule, positionIdxs)
	for _, balls := range draws {
		if len(posList) > 0 {
			for _, pos := range posList {
				if pos >= 0 && pos < len(balls) {
					counts[strings.TrimSpace(balls[pos])]++
				}
			}
			continue
		}
		positions := playPositionCount(rule)
		for i := 0; i < positions; i++ {
			pos := hotColdPositionIdx(rule, i)
			if pos >= 0 && pos < len(balls) {
				counts[strings.TrimSpace(balls[pos])]++
			}
		}
	}
	sorted := append([]string(nil), pool...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if counts[sorted[i]] != counts[sorted[j]] {
			return counts[sorted[i]] > counts[sorted[j]]
		}
		return sorted[i] < sorted[j]
	})
	n := len(sorted)
	half := (n + 1) / 2
	if half > n {
		half = n
	}
	return sorted[:half], sorted[half:]
}

// renxuanOverallPositionIdxs 任选整体号池冷热的投注选位；非任选选位玩法返回 nil。
func renxuanOverallPositionIdxs(rule playRule, positionIdxs []int) []int {
	if !isRenxuanNeedsPositionRule(rule) || !isHotColdDigitOverall(rule) {
		return nil
	}
	k := rule.SegmentLen
	if k <= 0 {
		k = renPickCount(rule.CatalogSubID)
	}
	if k <= 0 {
		k = renPickCount(rule.SubPlayID)
	}
	if k <= 0 {
		k = 2
	}
	return normalizeRenxuanRunPositionIdxs(positionIdxs, k)
}

// hotColdWarmTiers 按最近 N 期频次排序二等分（热/冷；对齐 v6 第三方，无温档）。
// warm 恒为空切片，保留返回值以兼容旧调用方。
func hotColdWarmTiers(draws [][]string, positionIdx int, pool []string) (hot, warm, cold []string) {
	counts := make(map[string]int, len(pool))
	for _, balls := range draws {
		if positionIdx >= 0 && positionIdx < len(balls) {
			counts[strings.TrimSpace(balls[positionIdx])]++
		}
	}
	sorted := append([]string(nil), pool...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if counts[sorted[i]] != counts[sorted[j]] {
			return counts[sorted[i]] > counts[sorted[j]]
		}
		return sorted[i] < sorted[j]
	})
	n := len(sorted)
	half := (n + 1) / 2
	if half > n {
		half = n
	}
	hot = sorted[:half]
	warm = []string{}
	cold = sorted[half:]
	return hot, warm, cold
}

// ---------- 高级开某投某 ----------

func (w *Worker) pickTriggerBet(
	ctx context.Context,
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	draw sqlcdb.LotteryDraw,
) pickDecision {
	prevBalls := w.previousDrawBalls(ctx, inst.LotteryCode, draw)
	return resolveTriggerBetDecision(cfg, prevBalls, inst.LastDirection)
}

// resolveTriggerBetDecision 高级开某投某出号。
// 定位胆多选位：每位按该位上期开奖各自查映射下注（例：上期 17232、选万/百/个 → 1,,2,,2），
// 不可把某一命中行的号码复制到所有位。
// 无启用行命中开出时本期跳过（不回退启用第 1 行）。
func resolveTriggerBetDecision(cfg parsedSchemeConfig, prevBalls []string, lastDirection string) pickDecision {
	if cfg.Trigger == nil || len(cfg.Trigger.Rows) == 0 {
		return pickDecision{Skip: true}
	}
	enabled := make([]triggerRow, 0, len(cfg.Trigger.Rows))
	for _, r := range cfg.Trigger.Rows {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	if len(enabled) == 0 {
		return pickDecision{Skip: true}
	}

	direction := nextTriggerDirection(cfg.Trigger.Mode, lastDirection)

	// 任选非直选复式：开奖选位单点查映射 + 投注选位前缀（须先于通用组选路径）
	if isRenxuanNeedsPositionTriggerPlay(cfg.Play) {
		return pickTriggerBetRenxuanNeedsPosition(cfg, enabled, prevBalls, direction)
	}

	// 组选号池 / 组选单式：开出看区位内任一位球号（前二=万或千），命中后投该行正/反内容
	if isZuxuanSegmentOpenTriggerPlay(cfg.Play) {
		return pickTriggerBetZuxuanPool(cfg, enabled, prevBalls, direction)
	}

	// 一星定位胆 / 前三直选复式等：按位独立映射（UI 已取消投注位芯片，默认段内全位）
	if triggerBetUsesPosition(cfg.Play) {
		return pickTriggerBetPerPosition(cfg, enabled, prevBalls, direction)
	}

	// 龙虎 / PC28 等：整期一个开出条件 → 一行映射；未命中则本期不投
	if len(prevBalls) == 0 {
		return pickDecision{Skip: true}
	}
	var row *triggerRow
	for i := range enabled {
		if triggerOpenMatches(cfg.Play, prevBalls, enabled[i].Open) {
			row = &enabled[i]
			break
		}
	}
	if row == nil {
		return pickDecision{Skip: true}
	}
	content, dir := triggerRowPickContent(*row, direction)
	if content == "" {
		return pickDecision{Skip: true}
	}
	return pickDecision{Content: content, Direction: dir}
}

// pickTriggerBetRenxuanNeedsPosition 任选开某投某：
// 开奖选位（单）取上期球号查启用行；直选单式按投注选位列组合号码，号池/和值用整行内容；
// 票面加投注选位（≥k）前缀。
func pickTriggerBetRenxuanNeedsPosition(
	cfg parsedSchemeConfig,
	enabled []triggerRow,
	prevBalls []string,
	direction string,
) pickDecision {
	if len(prevBalls) == 0 {
		return pickDecision{Skip: true}
	}
	k := cfg.Play.SegmentLen
	if k <= 0 {
		k = renPickCount(cfg.Play.CatalogSubID)
	}
	if k <= 0 {
		k = 2
	}
	openIdx := resolveRenxuanOpenPositionIdx(cfg)
	if openIdx < 0 || openIdx >= len(prevBalls) {
		return pickDecision{Skip: true}
	}
	var betIdxs []int
	if cfg.Trigger != nil {
		betIdxs = normalizeRenxuanRunPositionIdxs(cfg.Trigger.PositionIdxs, k)
	} else {
		betIdxs = defaultRenxuanTriggerPositionIdxs(k)
	}
	open := normalizeTriggerToken(strings.TrimSpace(prevBalls[openIdx]))
	row, ok := findEnabledTriggerRowByOpen(enabled, open)
	if !ok {
		return pickDecision{Skip: true}
	}
	content, dir := triggerRowPickContent(row, direction)
	if content == "" {
		return pickDecision{Skip: true}
	}
	if isRenxuanPerPosTriggerPlay(cfg.Play) {
		// 启用区固定 k 列（第一位…第k位），与投注选位个数无关；前缀仍用投注选位
		parts := splitTriggerFieldParts(content, k)
		exp, okExp := expandZhixuanPositionPoolToDanshi(strings.Join(parts, "\n"), k)
		if !okExp {
			return pickDecision{Skip: true}
		}
		content = exp
	} else if isRenxuanPositionDanshiRule(cfg.Play) {
		if exp, okExp := expandZhixuanPositionPoolToDanshi(content, k); okExp {
			content = exp
		}
	}
	if minK := zuxuanPoolMinPick(cfg.Play); minK >= 2 {
		n := len(uniqueStringTokens(splitContentTokens(content)))
		if n > 0 && n < minK {
			return pickDecision{Skip: true}
		}
	}
	posLine := renxuanPositionNamesCSV(betIdxs)
	if strings.TrimSpace(content) == "" {
		return pickDecision{Skip: true}
	}
	return pickDecision{Content: posLine + "\n" + content, Direction: dir}
}

// splitTriggerFieldParts 与前端 triggerFieldParts 对齐：按换行分列，缺列补空；无换行则各位同值。
func splitTriggerFieldParts(raw string, n int) []string {
	if n < 1 {
		n = 1
	}
	text := strings.ReplaceAll(raw, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	if !strings.Contains(text, "\n") {
		one := strings.TrimSpace(text)
		out := make([]string, n)
		for i := range out {
			out[i] = one
		}
		return out
	}
	parts := strings.Split(text, "\n")
	out := make([]string, n)
	for i := 0; i < n; i++ {
		if i < len(parts) {
			out[i] = strings.TrimSpace(parts[i])
		}
	}
	return out
}

// pickTriggerBetRenxuanDanshi / pickTriggerBetRenxuanPool 兼容旧测试入口。
func pickTriggerBetRenxuanDanshi(
	cfg parsedSchemeConfig,
	enabled []triggerRow,
	prevBalls []string,
	direction string,
) pickDecision {
	return pickTriggerBetRenxuanNeedsPosition(cfg, enabled, prevBalls, direction)
}

func pickTriggerBetRenxuanPool(
	cfg parsedSchemeConfig,
	enabled []triggerRow,
	prevBalls []string,
	direction string,
) pickDecision {
	return pickTriggerBetRenxuanNeedsPosition(cfg, enabled, prevBalls, direction)
}

func pickTriggerBetPerPosition(
	cfg parsedSchemeConfig,
	enabled []triggerRow,
	prevBalls []string,
	direction string,
) pickDecision {
	// 前三直选复式等：按玩法段出多行（万\n千\n百），而非五星定位胆稀疏行
	if !isDingweiTriggerPlay(cfg.Play) && cfg.Play.SegmentLen >= 2 {
		return pickTriggerBetPerSegment(cfg, enabled, prevBalls, direction)
	}
	positions := 5
	if cfg.Play.PlayTemplate == "pk10_std" {
		positions = 10
	}
	// 一星五位面板：忽略旧投注位勾选，始终万～个（或 PK10 全名次）按位出号
	var idxs []int
	if isDingweiFivePanelPlay(cfg.Play) {
		if len(cfg.Play.SegmentPos) > 0 {
			idxs = append([]int(nil), cfg.Play.SegmentPos...)
		} else {
			idxs = make([]int, positions)
			for i := range idxs {
				idxs[i] = i
			}
		}
	} else {
		idxs = cfg.Trigger.PositionIdxs
		if len(idxs) == 0 {
			if len(cfg.Play.SegmentPos) > 0 {
				idxs = append([]int(nil), cfg.Play.SegmentPos...)
			} else {
				idxs = []int{cfg.Play.PositionIdx}
			}
		}
	}
	lines := make([]string, positions)
	filled := 0
	outDir := direction
	for _, idx := range idxs {
		if idx < 0 {
			idx = 0
		}
		if idx >= positions {
			idx = positions - 1
		}
		if idx >= len(prevBalls) {
			return pickDecision{Skip: true}
		}
		open := normalizeTriggerToken(strings.TrimSpace(prevBalls[idx]))
		row, ok := findEnabledTriggerRowByOpen(enabled, open)
		if !ok {
			// 该位开出未命中任何启用行 → 本期不投
			return pickDecision{Skip: true}
		}
		// pos/neg 按「万\n千\n百\n十\n个」分位；旧单行则各位共用
		content, dir := triggerRowPickContentAt(row, direction, idx, positions)
		if content == "" {
			return pickDecision{Skip: true}
		}
		lines[idx] = content
		filled++
		outDir = dir
	}
	if filled == 0 {
		return pickDecision{Skip: true}
	}
	// 单位：仍压到选定投注位（兼容仅选一位 / 单位子玩法）
	if filled == 1 && len(idxs) == 1 {
		return pickDecision{
			Content:   layoutTriggerBetDingweiContent(cfg, lines[idxs[0]]),
			Direction: outDir,
		}
	}
	return pickDecision{Content: strings.Join(lines, "\n"), Direction: outDir}
}

// pickTriggerBetPerSegment 直选复式等：段内每位按绝对球位开奖查映射，输出 segmentLen 行。
// 任一位开出未命中启用行则本期不投（不回退启用第 1 行）。
func pickTriggerBetPerSegment(
	cfg parsedSchemeConfig,
	enabled []triggerRow,
	prevBalls []string,
	direction string,
) pickDecision {
	segStart := cfg.Play.SegmentStart
	segLen := cfg.Play.SegmentLen
	if segLen <= 0 {
		segLen = 1
	}
	lines := make([]string, segLen)
	filled := 0
	outDir := direction
	for rel := 0; rel < segLen; rel++ {
		abs := segStart + rel
		if abs >= len(prevBalls) {
			return pickDecision{Skip: true}
		}
		open := normalizeTriggerToken(strings.TrimSpace(prevBalls[abs]))
		row, ok := findEnabledTriggerRowByOpen(enabled, open)
		if !ok {
			return pickDecision{Skip: true}
		}
		content, dir := triggerRowPickContentAt(row, direction, rel, segLen)
		if content == "" {
			return pickDecision{Skip: true}
		}
		lines[rel] = content
		filled++
		outDir = dir
	}
	if filled == 0 {
		return pickDecision{Skip: true}
	}
	return pickDecision{Content: strings.Join(lines, "\n"), Direction: outDir}
}

func findEnabledTriggerRowByOpen(enabled []triggerRow, open string) (triggerRow, bool) {
	open = normalizeTriggerToken(open)
	if open == "" {
		return triggerRow{}, false
	}
	for _, r := range enabled {
		if normalizeTriggerToken(r.Open) == open {
			return r, true
		}
	}
	return triggerRow{}, false
}

// isZuxuanPoolTriggerPlay 组选号池类开某投某：开出条件看区位内任一位球号。
func isZuxuanPoolTriggerPlay(rule playRule) bool {
	return isZuxuanPoolRandom(rule)
}

// isZuxuanDanshiSegmentTriggerPlay 组选单式：正反投为整注，但「开出」按区位任一位球号（非仅万位）。
func isZuxuanDanshiSegmentTriggerPlay(rule playRule) bool {
	if rule.SegmentLen < 2 {
		return false
	}
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(strings.TrimSpace(rule.SubPlayID) + " " + strings.TrimSpace(rule.CatalogSubID))
	if bm == "zuxuan_ds" || strings.Contains(sub, "zuxuan_ds") {
		return true
	}
	// 数字 subId + 文案「组选单式」（catalog 合并 playMethod 时）
	if strings.Contains(sub, "组选单式") {
		return true
	}
	return false
}

// isZuxuanSegmentOpenTriggerPlay 组选复式号池 + 组选单式：开出均看区位任一位。
func isZuxuanSegmentOpenTriggerPlay(rule playRule) bool {
	return isZuxuanPoolTriggerPlay(rule) || isZuxuanDanshiSegmentTriggerPlay(rule)
}

// pickTriggerBetZuxuanPool 组三/组六/组选单式等：上期区位任一位开出命中启用行 → 投该行正/反内容。
// 例：前二上期万=8 千=0 且仅启用 0–4 时，命中开出 0 → 投 pos/neg（不要求必须是万位）。
// 例：中三上期 8,0,8 且仅启用 0–4 时，命中开出 0 → 投 pos/neg，而非因 8 未启用整期 Skip。
func pickTriggerBetZuxuanPool(
	cfg parsedSchemeConfig,
	enabled []triggerRow,
	prevBalls []string,
	direction string,
) pickDecision {
	if len(prevBalls) == 0 {
		return pickDecision{Skip: true}
	}
	seg := drawSegmentForRule(cfg.Play, prevBalls)
	if len(seg) == 0 {
		// 无段信息时回退整票球号
		seg = prevBalls
	}
	var row *triggerRow
	for _, d := range seg {
		if r, ok := findEnabledTriggerRowByOpen(enabled, d); ok {
			rr := r
			row = &rr
			break
		}
	}
	if row == nil {
		return pickDecision{Skip: true}
	}
	content, dir := triggerRowPickContent(*row, direction)
	if content == "" {
		return pickDecision{Skip: true}
	}
	// 组三/组六号池不足最少选号时本期 Skip（勿带着 1～2 码去撞第三方「单挑参数错误」）
	// 组选单式整注不走号池下限
	if !isZuxuanDanshiSegmentTriggerPlay(cfg.Play) {
		if minK := zuxuanPoolMinPick(cfg.Play); minK >= 2 {
			n := len(uniqueStringTokens(splitContentTokens(content)))
			if n < minK {
				return pickDecision{Skip: true}
			}
		}
	}
	return pickDecision{Content: content, Direction: dir}
}

// triggerRowPickContent 按投向取正/反投；反投为空时退回正投。
func triggerRowPickContent(row triggerRow, direction string) (content, dir string) {
	return triggerRowPickContentAt(row, direction, 0, 1)
}

// triggerRowPickContentAt 取某一段位的正/反投。
// pos/neg 以换行分位（万\n千\n百）；单行旧值则各位共用。
func triggerRowPickContentAt(row triggerRow, direction string, rel, segLen int) (content, dir string) {
	dir = direction
	pick := func(raw string) string {
		raw = strings.ReplaceAll(raw, "\r\n", "\n")
		raw = strings.ReplaceAll(raw, "\r", "\n")
		parts := strings.Split(raw, "\n")
		if segLen <= 1 || len(parts) <= 1 {
			return strings.TrimSpace(raw)
		}
		if rel >= 0 && rel < len(parts) {
			return strings.TrimSpace(parts[rel])
		}
		return ""
	}
	content = pick(row.Pos)
	if direction == "neg" {
		content = pick(row.Neg)
	}
	if content == "" {
		if alt := pick(row.Pos); alt != "" {
			return alt, "pos"
		}
		return "", direction
	}
	return content, dir
}

// nextTriggerDirection 投向状态机（Q4b：按上一局投向交替）。
func nextTriggerDirection(mode, last string) string {
	switch mode {
	case "always_neg":
		return "neg"
	case "alt_pos_first": // 前正后反：自正投起始，逐局交替
		if last == "pos" {
			return "neg"
		}
		if last == "neg" {
			return "pos"
		}
		return "pos"
	case "alt_neg_first": // 前反后正
		if last == "neg" {
			return "pos"
		}
		if last == "pos" {
			return "neg"
		}
		return "neg"
	default: // always_pos
		return "pos"
	}
}

// previousDrawBalls 取上一期开奖球（不含当期）。
// 优先取期号 = 当期-1 的精确上期；若刚好缺相邻一期则返回 nil（勿回退到更早开奖，
// 否则开某投某会按上上期映射下注，如 0356 和值 15 未入库时误用 0355 的 8）。
func (w *Worker) previousDrawBalls(ctx context.Context, lotteryCode string, draw sqlcdb.LotteryDraw) []string {
	if w == nil || w.q == nil {
		return nil
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	issue := strings.TrimSpace(draw.IssueNo)
	if lotteryCode == "" || issue == "" {
		return nil
	}
	expected := prevIssueNo(issue)
	if expected != "" && expected != issue {
		row, err := w.q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{
			LotteryCode: lotteryCode,
			IssueNo:     expected,
		})
		if err == nil {
			return sqlcdb.ParseDrawBalls(row.Balls)
		}
	}
	prev, err := w.q.GetPreviousLotteryDrawByIssue(ctx, sqlcdb.GetPreviousLotteryDrawByIssueParams{
		LotteryCode: lotteryCode,
		IssueNo:     issue,
	})
	if err != nil {
		return nil
	}
	// 相邻上期仍缺库：返回空，由出号 Skip；worker 侧会尽量推迟到开奖入库
	if hotColdAdjacentPrevMissing(expected, prev.IssueNo) {
		return nil
	}
	return sqlcdb.ParseDrawBalls(prev.Balls)
}

func isLonghuPlay(rule playRule) bool {
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	if bm == "longhubao" {
		return false
	}
	if bm == "longhu" || bm == "longhuhe" {
		return true
	}
	return strings.TrimSpace(rule.PlayTypeID) == playTypeLonghu
}

// longhuResult 计算上期龙虎结果（复用 longhuPositions 的位映射；无子玩法时取万 vs 个）。
func longhuResult(rule playRule, balls []string) string {
	p1, p2, _ := longhuPositions(rule.CatalogSubID)
	if p1 < 0 || p2 < 0 {
		p1, p2 = 0, len(balls)-1
	}
	if p1 >= len(balls) || p2 >= len(balls) || p1 < 0 || p2 < 0 {
		return ""
	}
	a, b := atoiBall(balls[p1]), atoiBall(balls[p2])
	switch {
	case a > b:
		return "龙"
	case a < b:
		return "虎"
	default:
		return "和"
	}
}

func normalizeTriggerToken(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "long", "dragon":
		return "龙"
	case "hu", "tiger":
		return "虎"
	case "he", "tie", "draw":
		return "和"
	}
	// 数字 token 按数值归一（兼容 "07" 与 "7"）
	if n, err := strconv.Atoi(s); err == nil {
		return strconv.Itoa(n)
	}
	return s
}
