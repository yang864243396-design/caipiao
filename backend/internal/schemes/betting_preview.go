package schemes

import (
	"encoding/json"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
)

const bettingPreviewRowLimit = 20

// BettingExecutionPreview 玩法详情投注 Tab 预览行（由方案内容 + 历史开奖推演，非真实投注）。
type BettingExecutionPreview struct {
	Time    string
	Scheme  string
	Numbers string
	Period  string
	Draw    string
	Win     bool
}

type simPickState struct {
	pickIndex     int32
	currentPick   string
	lastDirection string
}

// PreviewBettingExecutions 按方案配置与历史开奖推演最近 N 期投注展示。
func PreviewBettingExecutions(
	schemeName string,
	contentSeed string,
	kind string,
	configJSON []byte,
	draws []sqlcdb.ListLotteryDrawsRow,
) []BettingExecutionPreview {
	schemeName = strings.TrimSpace(schemeName)
	if schemeName == "" {
		schemeName = "方案"
	}
	if strings.TrimSpace(contentSeed) == "" {
		contentSeed = schemeName
	}
	if kind == "" {
		kind = "custom"
	}
	configJSON = ensurePreviewConfigContent(configJSON, contentSeed)
	cfg := parseSchemeConfig(kind, configJSON, 0, 0)
	if strings.TrimSpace(cfg.GroupContent) == "" && len(cfg.Groups) > 0 {
		cfg.GroupContent = cfg.Groups[0]
	}

	ordered := append([]sqlcdb.ListLotteryDrawsRow(nil), draws...)
	if len(ordered) == 0 {
		return nil
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].IssueNo < ordered[j].IssueNo
	})
	if len(ordered) > bettingPreviewRowLimit {
		ordered = ordered[len(ordered)-bettingPreviewRowLimit:]
	}

	state := simPickState{}
	rows := make([]BettingExecutionPreview, 0, len(ordered))
	var prevBalls []string
	histDraws := make([][]string, 0, len(ordered))

	for _, draw := range ordered {
		dec := resolvePickPreview(cfg, state, draw.IssueNo, prevBalls, histDraws)
		if dec.Skip {
			balls := sqlcdb.ParseDrawBalls(draw.Balls)
			if len(balls) > 0 {
				histDraws = append(histDraws, balls)
			}
			prevBalls = balls
			continue
		}
		content := strings.TrimSpace(dec.Content)
		if content == "" {
			content = cfg.GroupContent
		}
		if strings.TrimSpace(content) == "" {
			prevBalls = sqlcdb.ParseDrawBalls(draw.Balls)
			continue
		}

		balls := sqlcdb.ParseDrawBalls(draw.Balls)
		eval := evaluatePlayHit(cfg.Play, balls, content, cfg.Contrary, cfg.ContraryPlan, cfg.Play.PositionIdx)

		periodShort := thirdPartyPeriodShort(draw.IssueNo)
		nextShort := thirdPartyPeriodShort(bumpPreviewIssue(draw.IssueNo))
		timeLabel := periodShort
		if nextShort != "" && nextShort != "—" {
			timeLabel = periodShort + "-" + nextShort
		}

		rows = append(rows, BettingExecutionPreview{
			Time:    timeLabel,
			Scheme:  schemeName,
			Numbers: formatPreviewNumbers(content),
			Period:  periodShort,
			Draw:    strings.Join(balls, " "),
			Win:     eval.Hit,
		})

		pickIdx, curPick, lastDir := advancePickState(cfg, previewInstState(state), dec, eval.Hit)
		state = simPickState{pickIndex: pickIdx, currentPick: curPick, lastDirection: lastDir}
		if len(balls) > 0 {
			histDraws = append(histDraws, balls)
		}
		prevBalls = balls
	}

	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows
}

func previewInstState(s simPickState) sqlcdb.SchemeInstance {
	return sqlcdb.SchemeInstance{
		PickIndex:     s.pickIndex,
		CurrentPick:   s.currentPick,
		LastDirection: s.lastDirection,
	}
}

func resolvePickPreview(
	cfg parsedSchemeConfig,
	state simPickState,
	issueNo string,
	prevBalls []string,
	histDraws [][]string,
) pickDecision {
	inst := previewInstState(state)
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
		return pickTriggerBetPreview(cfg, inst, prevBalls)
	case RunTypeHotColdWarm:
		// 预览用「当期之前」的历史开奖取冷热码（与运行时 recentDrawBalls 一致）
		periods := 20
		if cfg.HotCold != nil && cfg.HotCold.TotalPeriods > 0 {
			periods = cfg.HotCold.TotalPeriods
		}
		hist := histDraws
		if len(hist) > periods {
			hist = hist[len(hist)-periods:]
		}
		return pickHotColdWarmFromDraws(cfg, inst, hist)
	case RunTypeRandomDraw:
		return pickRandomDrawPreview(cfg, inst, issueNo)
	case RunTypeFixedNumber:
		return pickFixedNumber(cfg)
	case RunTypeBuiltinPlan:
		return pickDecision{Skip: true}
	default:
		return pickDecision{Content: cfg.GroupContent}
	}
}

func pickTriggerBetPreview(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance, prevBalls []string) pickDecision {
	return resolveTriggerBetDecision(cfg, prevBalls, inst.LastDirection)
}

func pickRandomDrawPreview(cfg parsedSchemeConfig, inst sqlcdb.SchemeInstance, issueNo string) pickDecision {
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
		lines = append(lines, deterministicRandomDigits(cfg.Play, count, issueNo+strconv.Itoa(i)))
	}
	return pickDecision{Content: strings.Join(lines, "\n")}
}

func deterministicRandomDigits(rule playRule, count int, seed string) string {
	pool := playNumberPool(rule)
	if count < 1 {
		count = 1
	}
	if count > len(pool) {
		count = len(pool)
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	rng := h.Sum32()
	perm := make([]int, len(pool))
	for i := range perm {
		perm[i] = i
	}
	for i := len(perm) - 1; i > 0; i-- {
		rng = rng*1664525 + 1013904223
		j := int(rng % uint32(i+1))
		perm[i], perm[j] = perm[j], perm[i]
	}
	idx := perm[:count]
	sort.Ints(idx)
	out := make([]string, 0, count)
	for _, i := range idx {
		out = append(out, pool[i])
	}
	return strings.Join(out, ",")
}

func ensurePreviewConfigContent(configJSON []byte, seed string) []byte {
	cfg := map[string]interface{}{}
	if len(configJSON) > 0 {
		_ = json.Unmarshal(configJSON, &cfg)
	}
	if len(extractSchemeGroups(cfg)) > 0 {
		return configJSON
	}
	if gc, ok := cfg["groupContent"].(string); ok && strings.TrimSpace(gc) != "" {
		cfg["schemeGroups"] = []string{strings.TrimSpace(gc)}
		raw, err := json.Marshal(cfg)
		if err == nil {
			return raw
		}
		return configJSON
	}
	typeID, _ := cfg["playTypeId"].(string)
	if typeID == "" {
		typeID, _ = cfg["typeId"].(string)
	}
	subID, _ := cfg["subPlayId"].(string)
	if subID == "" {
		subID, _ = cfg["subId"].(string)
	}
	template, _ := cfg["playTemplate"].(string)
	betMode, _ := cfg["betMode"].(string)
	label, _ := cfg["playMethod"].(string)
	if strings.TrimSpace(label) == "" {
		label = resolvePlayTypeLabel(cfg)
	}
	content := DemoGroupContentForSubPlay(template, typeID, subID, betMode, label, seed)
	if strings.TrimSpace(content) == "" {
		content = "1,3,7"
	}
	cfg["schemeGroups"] = []interface{}{content}
	if _, ok := cfg["runTypeId"]; !ok {
		cfg["runTypeId"] = RunTypeFixedRotate
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return configJSON
	}
	return raw
}

func formatPreviewNumbers(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "—"
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, ",", " ")
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return "—"
	}
	return strings.Join(fields, " ")
}

// thirdPartyPeriodShort 第三方期号展示：仅取最后 3 位（与玩法详情投注 Tab 约定一致）。
func thirdPartyPeriodShort(issue string) string {
	issue = strings.TrimSpace(issue)
	if issue == "" {
		return "—"
	}
	runes := []rune(issue)
	if len(runes) <= 3 {
		return issue
	}
	return string(runes[len(runes)-3:])
}

// ThirdPartyPeriodDisplay 第三方期号展示：去掉前三位前缀（历史开奖、投注记录 Tab）。
func ThirdPartyPeriodDisplay(issue string) string {
	issue = strings.TrimSpace(issue)
	if issue == "" {
		return "—"
	}
	runes := []rune(issue)
	if len(runes) <= 3 {
		return issue
	}
	return string(runes[3:])
}

func bumpPreviewIssue(issue string) string {
	n, err := strconv.ParseInt(strings.TrimSpace(issue), 10, 64)
	if err != nil {
		return issue + "1"
	}
	return strconv.FormatInt(n+1, 10)
}
