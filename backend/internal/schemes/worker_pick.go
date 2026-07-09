package schemes

import (
	"context"
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
	if cfg.Contrary || cfg.Kind != "custom" || cfg.RunTypeID == "" {
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
		return pickHotColdWarm(cfg, inst)
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
		row := currentJushuRow(cfg.Jushu, int(inst.PickIndex))
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
		// 仅中奖轮换时持久化池；未轮换保持空值，使运行中修改选号池即时生效
		if hit && cfg.HotCold != nil && cfg.HotCold.WinRotate {
			// 中奖轮换：池内号码各自轮换到下一个（同位号码池循环）
			currentPick = rotatePoolContent(dec.Content, cfg.Play)
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

// ---------- 固定号码 ----------

func pickFixedNumber(cfg parsedSchemeConfig) pickDecision {
	if len(cfg.Groups) > 0 && strings.TrimSpace(cfg.Groups[0]) != "" {
		return pickDecision{Content: cfg.Groups[0]}
	}
	return pickDecision{Content: cfg.GroupContent}
}

// ---------- 随机出号 ----------

func pickRandomDraw(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance) pickDecision {
	if strings.TrimSpace(inst.CurrentPick) != "" {
		return pickDecision{Content: inst.CurrentPick}
	}
	positions := playPositionCount(cfg.Play)
	lines := make([]string, 0, positions)
	for i := 0; i < positions; i++ {
		count := 1
		if cfg.Random != nil && i < len(cfg.Random.Counts) {
			count = cfg.Random.Counts[i]
		}
		lines = append(lines, randomDigits(cfg.Play, count))
	}
	return pickDecision{Content: strings.Join(lines, "\n")}
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
	min, max := rule.NumberPoolMin, rule.NumberPoolMax
	if max <= 0 || max < min {
		// 六合彩模板号码池 1-49，其余兜底 0-9
		if rule.PlayTemplate == "lhc_std" || isLHCTypeID(rule.PlayTypeID) {
			min, max = 1, 49
		} else {
			min, max = 0, 9
		}
	}
	if max-min > 98 {
		max = min + 98
	}
	pool := make([]string, 0, max-min+1)
	for v := min; v <= max; v++ {
		pool = append(pool, strconv.Itoa(v))
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

// ---------- 冷热温出号 ----------

func pickHotColdWarm(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance) pickDecision {
	if strings.TrimSpace(inst.CurrentPick) != "" {
		return pickDecision{Content: inst.CurrentPick}
	}
	if cfg.HotCold == nil || len(cfg.HotCold.Pool) == 0 {
		return pickDecision{Content: cfg.GroupContent}
	}
	return pickDecision{Content: strings.Join(cfg.HotCold.Pool, "\n")}
}

// rotatePoolContent 中奖轮换：池内每个号码在该位号码池内 +1 循环。
// 号码按数值归一匹配（兼容 "07" 与 "7" 两种 token 形态，保留原有补零位宽）。
func rotatePoolContent(content string, rule playRule) string {
	pool := playNumberPool(rule)
	poolIdx := make(map[int]int, len(pool))
	for i, p := range pool {
		if n, err := strconv.Atoi(p); err == nil {
			poolIdx[n] = i
		}
	}
	lines := strings.Split(content, "\n")
	outLines := make([]string, 0, len(lines))
	for _, line := range lines {
		tokens := strings.Split(strings.NewReplacer("，", ",", " ", ",").Replace(line), ",")
		seen := map[string]struct{}{}
		out := make([]string, 0, len(tokens))
		for _, t := range tokens {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			next := t
			if n, err := strconv.Atoi(t); err == nil {
				if idx, ok := poolIdx[n]; ok {
					next = pool[(idx+1)%len(pool)]
					if len(t) == 2 && t[0] == '0' && len(next) == 1 {
						next = "0" + next
					}
				}
			}
			if _, dup := seen[next]; dup {
				continue
			}
			seen[next] = struct{}{}
			out = append(out, next)
		}
		if len(out) > 0 {
			outLines = append(outLines, strings.Join(out, ","))
		}
	}
	if len(outLines) == 0 {
		return content
	}
	return strings.Join(outLines, "\n")
}

// hotColdWarmTiers 按最近 N 期频次排序三等分（热/温/冷），供编辑页统计接口复用。
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
	third := (n + 2) / 3
	cut1 := third
	if cut1 > n {
		cut1 = n
	}
	cut2 := 2 * third
	if cut2 > n {
		cut2 = n
	}
	hot = sorted[:cut1]
	warm = sorted[cut1:cut2]
	cold = sorted[cut2:]
	return hot, warm, cold
}

// ---------- 高级开某投某 ----------

func (w *Worker) pickTriggerBet(
	ctx context.Context,
	cfg parsedSchemeConfig,
	inst sqlcdb.SchemeInstance,
	draw sqlcdb.LotteryDraw,
) pickDecision {
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

	prevBalls := w.previousDrawBalls(ctx, inst.LotteryCode, draw)
	row := enabled[0] // Q4c：无匹配走启用第 1 行
	if len(prevBalls) > 0 {
		for _, r := range enabled {
			if triggerOpenMatches(cfg.Play, prevBalls, r.Open) {
				row = r
				break
			}
		}
	}

	direction := nextTriggerDirection(cfg.Trigger.Mode, inst.LastDirection)
	content := row.Pos
	if direction == "neg" {
		content = row.Neg
	}
	if strings.TrimSpace(content) == "" {
		// 该向号码未填：退回正投，再退回跳过
		if strings.TrimSpace(row.Pos) != "" {
			direction, content = "pos", row.Pos
		} else {
			return pickDecision{Skip: true}
		}
	}
	return pickDecision{Content: content, Direction: direction}
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
func (w *Worker) previousDrawBalls(ctx context.Context, lotteryCode string, draw sqlcdb.LotteryDraw) []string {
	rows, err := w.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    12,
	})
	if err != nil || len(rows) == 0 {
		return nil
	}
	for _, r := range rows {
		if r.IssueNo == draw.IssueNo {
			continue
		}
		if r.IssueNo < draw.IssueNo {
			return sqlcdb.ParseDrawBalls(r.Balls)
		}
	}
	return nil
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
