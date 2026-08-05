package schemes

import (
	"fmt"
	"strconv"
	"strings"
)

// 只读对账（cmd/scheme-audit）用的导出入口。playRule 与结算细节不外泄，
// 保证审计与线上结算走同一条判定路径——否则报告里全是误报。

// RebuildHotColdPickContent 按与 worker 相同的 buildHotColdPickContent 重算出号内容。
// draws 为当期之前最近 N 期开奖球（不含当期），顺序任意（计频只看集合）。
func RebuildHotColdPickContent(kind string, config []byte, draws [][]string) (content string, totalPeriods int, ok bool) {
	if len(config) == 0 {
		return "", 0, false
	}
	cfg := parseSchemeConfig(kind, config, 0, 0)
	if cfg.RunTypeID != "" && cfg.RunTypeID != RunTypeHotColdWarm {
		// 允许未写 runTypeId 的旧配置，只要有 hotColdWarm 块
		if cfg.HotCold == nil {
			return "", 0, false
		}
	}
	if cfg.HotCold == nil {
		return "", 0, false
	}
	totalPeriods = cfg.HotCold.TotalPeriods
	if totalPeriods <= 0 {
		totalPeriods = 20
	}
	content = normalizeZhixuanDanshiContent(cfg.Play, buildHotColdPickContent(cfg, draws))
	return content, totalPeriods, true
}

// AdjudicateSchemeBet 按方案定义与开奖球号重算一注的中奖判定。
//
// 路径与 settleSimCloudBet 完全一致：同样的 parseSchemeConfig、attachOddsBase 与
// evaluatePlayHit(contrary=false)。betContent 传 cloud_bet_records.bet_content
// （已含反投展开），为空时回退方案当期内容。
//
// ok=false 表示缺少判定所需输入，调用方应跳过而不是当成不一致。
func AdjudicateSchemeBet(
	kind string, config []byte, lotteryCode, betContent string, roundIndex int, balls []string,
) (hit bool, odds float64, ok bool) {
	if len(config) == 0 || len(balls) == 0 {
		return false, 0, false
	}
	if roundIndex < 0 {
		roundIndex = 0
	}
	cfg := parseSchemeConfig(kind, config, roundIndex, roundIndex)
	cfg.Play = attachOddsBase(cfg.Play, lotteryCode)
	if strings.TrimSpace(betContent) == "" {
		betContent = cfg.GroupContent
	}
	if strings.TrimSpace(betContent) == "" {
		return false, 0, false
	}
	ev := evaluatePlayHit(cfg.Play, balls, betContent, false, "", cfg.Play.PositionIdx)
	return ev.Hit, ev.Odds, true
}

// SchemePlayDigest 方案玩法摘要，用于审计报告定位问题方案。
type SchemePlayDigest struct {
	PlayTemplate string
	PlayTypeID   string
	SubPlayID    string
	BetMode      string
	PlayLabel    string
	RunTypeID    string
	SegmentLen   int
	PoolMin      int
	PoolMax      int
	GroupContent string
}

// ValidateSchemeConfig 校验方案配置里保存的投注内容是否落在合法投注空间内。
//
// 与 cmd/scheme-audit 的 config_* 检查同一入口，保证"保存时拦下的"和"对账时报出的"
// 是同一套判据；否则两边会各自漂移，最后谁也不信谁。
// 无内容、或玩法内容形态判不出时返回 nil——判不准就不拦，误拦比漏拦更伤。
func ValidateSchemeConfig(kind string, config []byte) []Violation {
	if len(config) == 0 {
		return nil
	}
	if UniverseKindForScheme(kind, config) == "" {
		return nil
	}
	cfg := parseSchemeConfig(kind, config, 0, 0)
	// 开某投某：schemeGroups 只是样例占位，真正内容在 triggerBet.rows 的正/反投里。
	if cfg.RunTypeID == RunTypeAdvTriggerBet && cfg.Trigger != nil {
		return validateAdvTriggerBetConfig(kind, config, cfg)
	}
	// 冷热出号：schemeGroups 常为编辑预览/占位（按位号池或误拼成一行单码），
	// 真正下注明细由 hotColdWarm + 往期频次动态生成，不能按单式整注去验 schemeGroups。
	if cfg.RunTypeID == RunTypeHotColdWarm {
		return validateHotColdWarmConfig(kind, config, cfg)
	}
	// 高级定码轮换：验每一局内容（勿只验 schemeGroups[0]）
	if cfg.RunTypeID == RunTypeAdvFixedRotate {
		return validateContentList(kind, config, jushuContents(cfg), "第 %d 局：")
	}
	// 随机出号：运行时按宇宙抽样；预览占位仍须落在号池内（如跨度 0–9）。
	if cfg.RunTypeID == RunTypeRandomDraw {
		return validateContentList(kind, config, cfg.Groups, "预览第 %d 组：")
	}
	// 定码轮换 / 固定出号：校验全部分组
	return validateContentList(kind, config, cfg.Groups, "第 %d 组：")
}

func jushuContents(cfg parsedSchemeConfig) []string {
	if len(cfg.Jushu) == 0 {
		return nil
	}
	out := make([]string, 0, len(cfg.Jushu))
	for _, row := range cfg.Jushu {
		out = append(out, row.Content)
	}
	return out
}

// validateContentList 逐条校验非空内容；prefixFmt 含一个 %d（1-based 序号）。
func validateContentList(kind string, config []byte, contents []string, prefixFmt string) []Violation {
	var out []Violation
	for i, raw := range contents {
		c := strings.TrimSpace(raw)
		if c == "" {
			continue
		}
		for _, v := range ValidateSchemeBetContent(kind, config, c, 0) {
			detail := v.Detail
			if prefixFmt != "" {
				detail = fmt.Sprintf(prefixFmt, i+1) + detail
			}
			out = append(out, Violation{Code: v.Code, Detail: detail})
		}
	}
	return out
}

// validateHotColdWarmConfig 只校验启用位上的号池 token（不把按位号池当直选单式票）。
func validateHotColdWarmConfig(kind string, config []byte, cfg parsedSchemeConfig) []Violation {
	if cfg.HotCold == nil {
		return nil
	}
	u, ok := UniverseForScheme(kind, config)
	if !ok {
		return nil
	}
	// 属性宇宙：按选项校验每一组非空号池
	if u.Kind == UniverseAttribute {
		for i, line := range cfg.HotCold.Pool {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			for _, v := range ValidateSchemeBetContent(kind, config, line, 0) {
				return []Violation{{
					Code:   v.Code,
					Detail: "冷热号池第 " + strconv.Itoa(i+1) + " 组：" + v.Detail,
				}}
			}
		}
		return nil
	}
	// 组三/组六等整体号池：保存时强制最低选号（组三≥2、组六≥3）
	if isHotColdDigitOverall(cfg.Play) {
		if v := validateZuxuanPoolMinPick(cfg.Play, cfg.HotCold); len(v) > 0 {
			return v
		}
	}
	// 号码 / 单式 / 按位：pool 只是「该位是否启用 + 编辑预览」，运行时按热冷区重算。
	// 此处只拦越界单码，绝不把「5+5+5 个单码」误报成「15 个单式组合不合法」。
	min, max := ruleNumberPool(cfg.Play)
	var bad []string
	for _, line := range cfg.HotCold.Pool {
		for _, tok := range splitContentTokens(line) {
			if _, ok := normalizePoolToken(tok, min, max); !ok {
				bad = append(bad, tok)
			}
		}
	}
	if len(bad) == 0 {
		return nil
	}
	pool := make([]string, 0, max-min+1)
	for v := min; v <= max; v++ {
		pool = append(pool, strconv.Itoa(v))
	}
	return outOfPoolViolation(dedupStrings(bad), pool)
}

// validateZuxuanPoolMinPick 冷热整体号池最少选号（以 ranks 为准，无 ranks 时看 pool）。
func validateZuxuanPoolMinPick(rule playRule, hcw *hotColdWarmCfg) []Violation {
	if hcw == nil {
		return nil
	}
	minPick := zuxuanPoolMinPick(rule)
	if minPick < 2 {
		return nil
	}
	pickN := 0
	if hotColdCfgHasRanks(hcw.Ranks) {
		if len(hcw.Ranks) > 0 {
			pickN = len(hcw.Ranks[0])
		}
	} else {
		seen := map[string]struct{}{}
		for _, line := range hcw.Pool {
			for _, tok := range splitContentTokens(line) {
				tok = strings.TrimSpace(tok)
				if tok == "" {
					continue
				}
				seen[tok] = struct{}{}
			}
		}
		pickN = len(seen)
	}
	if pickN >= minPick {
		return nil
	}
	label := "号码池"
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sub := strings.ToLower(rule.SubPlayID + " " + rule.CatalogSubID)
	switch {
	case isSixingZu6PlayRule(rule):
		label = "组选6"
	case bm == "zu6" || (strings.Contains(sub, "zu6") && !strings.Contains(sub, "zu60") && !strings.Contains(sub, "zu120")):
		label = "组六"
	case bm == "zu3" || (strings.Contains(sub, "zu3") && !strings.Contains(sub, "zu30")):
		label = "组三"
	}
	return []Violation{{
		Code:   ViolationZeroUnits,
		Detail: fmt.Sprintf("%s至少选择 %d 个号码", label, minPick),
	}}
}

// validateAdvTriggerBetConfig 校验启用行的正投/反投（按位号池会在 ValidateSchemeBetContent 内展开）。
// 启动要求：每个启用号码的正投、反投都必须填写（和值/组选和值等单档与按位分列均适用）。
func validateAdvTriggerBetConfig(kind string, config []byte, cfg parsedSchemeConfig) []Violation {
	var out []Violation
	anyEnabled := false
	usesPos := triggerBetUsesPosition(cfg.Play)
	segLen := cfg.Play.SegmentLen
	if segLen < 1 {
		segLen = 1
	}
	if !usesPos {
		segLen = 1
	}
	for _, row := range cfg.Trigger.Rows {
		if !row.Enabled {
			continue
		}
		anyEnabled = true
		open := strings.TrimSpace(row.Open)
		posParts := triggerFieldPartsForValidate(row.Pos, segLen)
		negParts := triggerFieldPartsForValidate(row.Neg, segLen)
		for i := 0; i < segLen; i++ {
			if strings.TrimSpace(posParts[i]) == "" {
				detail := "开出 " + open + " 的正投未填写"
				if usesPos && segLen > 1 {
					detail = "开出 " + open + " 的正投第 " + strconv.Itoa(i+1) + " 位未填写"
				}
				out = append(out, Violation{Code: ViolationEmptyContent, Detail: detail})
			}
			if strings.TrimSpace(negParts[i]) == "" {
				detail := "开出 " + open + " 的反投未填写"
				if usesPos && segLen > 1 {
					detail = "开出 " + open + " 的反投第 " + strconv.Itoa(i+1) + " 位未填写"
				}
				out = append(out, Violation{Code: ViolationEmptyContent, Detail: detail})
			}
		}
		for _, cell := range []struct {
			name string
			raw  string
		}{
			{"正投", row.Pos},
			{"反投", row.Neg},
		} {
			c := strings.TrimSpace(cell.raw)
			if c == "" {
				continue
			}
			// 任选开某投某：格子无位名前缀，校验前补投注选位（与前端 / 出票 wrap 一致）
			if isRenxuanNeedsPositionRule(cfg.Play) && cfg.Trigger != nil {
				c = wrapRenxuanNeedsPositionContent(cfg.Play, c, cfg.Trigger.PositionIdxs)
			}
			for _, v := range ValidateSchemeBetContent(kind, config, c, 0) {
				out = append(out, Violation{
					Code:   v.Code,
					Detail: "开出 " + open + " 的" + cell.name + "：" + v.Detail,
				})
			}
		}
	}
	if !anyEnabled {
		out = append(out, Violation{Code: ViolationEmptyContent, Detail: "请至少启用一行开某投某映射"})
	}
	return out
}

// triggerFieldPartsForValidate 与前端 triggerFieldParts 对齐：无换行时各位共用；有换行按位切。
func triggerFieldPartsForValidate(raw string, n int) []string {
	if n < 1 {
		n = 1
	}
	text := strings.ReplaceAll(raw, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
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

// DigestScheme 提取方案玩法摘要。
func DigestScheme(kind string, config []byte) (SchemePlayDigest, bool) {
	if len(config) == 0 {
		return SchemePlayDigest{}, false
	}
	cfg := parseSchemeConfig(kind, config, 0, 0)
	min, max := ruleNumberPool(cfg.Play)
	return SchemePlayDigest{
		PlayTemplate: cfg.Play.PlayTemplate,
		PlayTypeID:   cfg.Play.PlayTypeID,
		SubPlayID:    cfg.Play.SubPlayID,
		BetMode:      cfg.Play.BetMode,
		PlayLabel:    cloudPlayTypeLabel(cfg.PlayTypeLabel, cfg.SubPlayLabel),
		RunTypeID:    cfg.RunTypeID,
		SegmentLen:   cfg.Play.SegmentLen,
		PoolMin:      min,
		PoolMax:      max,
		GroupContent: cfg.GroupContent,
	}, true
}
