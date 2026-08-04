package schemes

import (
	"context"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
)

// HotColdWarmTiersInput 冷热温属性分档查询入参（前端编辑页按当前玩法请求）。
type HotColdWarmTiersInput struct {
	LotteryCode     string
	PlayTypeID      string
	SubPlayID       string
	PlayTemplate    string
	BetMode         string
	CatalogSubID    string
	PlayMethodLabel string
	NumberPoolMin   int
	NumberPoolMax   int
	SegmentLen      int
	Periods         int
	// PositionIdxs 任选投注选位（0=万…4=个）；任选和值/尾数按此计频。
	PositionIdxs []int
}

// HotColdWarmTiers 拉取最近 N 期开奖，按属性选项命中频次分档（热/温/冷）。
func (s *Service) HotColdWarmTiers(ctx context.Context, in HotColdWarmTiersInput) (HotColdWarmTiersResult, error) {
	if s == nil || s.q == nil {
		return HotColdWarmTiersResult{}, ErrUnavailable
	}
	periods := in.Periods
	if periods < 20 {
		periods = 20
	}
	if periods > 500 {
		periods = 500
	}
	rows, err := s.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: in.LotteryCode,
		RowLimit:    int32(periods),
	})
	if err != nil {
		return HotColdWarmTiersResult{}, err
	}
	draws := make([][]string, 0, len(rows))
	for _, r := range rows {
		balls := sqlcdb.ParseDrawBalls(r.Balls)
		if len(balls) > 0 {
			draws = append(draws, balls)
		}
	}
	tpl := strings.TrimSpace(in.PlayTemplate)
	betMode := strings.TrimSpace(in.BetMode)
	if betMode == "" {
		betMode = inferAttributeBetModeFromLabel(in.PlayMethodLabel)
	}
	var rule playRule
	// SSC 目录 typeId（g001 前三等）须走 resolveSSCPlayRule，否则 SegmentStart 会落到默认 1 导致前三特殊号等计频错位。
	if tpl == "" || tpl == "ssc_std" || tpl == "fast_ssc_std" {
		rule = resolveSSCPlayRule(in.PlayTypeID, in.SubPlayID, betMode, in.PlayMethodLabel)
		if tpl != "" {
			rule.PlayTemplate = tpl
		}
	} else {
		cfg := map[string]interface{}{
			"playTypeId":   in.PlayTypeID,
			"subPlayId":    in.SubPlayID,
			"playTemplate": tpl,
			"betMode":      betMode,
			"catalogSubId": in.CatalogSubID,
		}
		rule = resolvePlayRule(cfg, in.PlayMethodLabel)
	}
	if strings.TrimSpace(rule.BetMode) == "" {
		rule.BetMode = betMode
	}
	// 龙虎等按 CatalogSubID 解析对比位。数字 guaji id 本身无区位，须合并玩法文案（万千）。
	rule.CatalogSubID = resolveHotColdCatalogSubID(rule, in)
	rule = applyHotColdWarmInputOverrides(rule, in)
	return HotColdWarmAttributeTiersForPositions(rule, draws, in.PositionIdxs), nil
}

// resolveHotColdCatalogSubID 优先选用能解析出龙虎位对的文案；否则保留 resolve 结果或回退 subId。
func resolveHotColdCatalogSubID(rule playRule, in HotColdWarmTiersInput) string {
	hint := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(in.PlayMethodLabel),
		strings.TrimSpace(in.CatalogSubID),
		strings.TrimSpace(in.SubPlayID),
	}, " "))
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	needPair := bm == "longhu" || bm == "longhuhe" ||
		strings.EqualFold(strings.TrimSpace(in.PlayTypeID), "longhu") ||
		strings.EqualFold(strings.TrimSpace(in.PlayTypeID), "g010")
	if needPair {
		for _, cand := range []string{hint, strings.TrimSpace(rule.CatalogSubID), strings.TrimSpace(in.CatalogSubID), strings.TrimSpace(in.SubPlayID)} {
			if cand == "" {
				continue
			}
			if p1, p2, _ := longhuPositions(cand); p1 >= 0 && p2 >= 0 {
				return cand
			}
		}
	}
	if hint != "" {
		return hint
	}
	if s := strings.TrimSpace(rule.CatalogSubID); s != "" {
		return s
	}
	if s := strings.TrimSpace(in.CatalogSubID); s != "" {
		return s
	}
	return strings.TrimSpace(in.SubPlayID)
}

// applyHotColdWarmInputOverrides 合并请求体号池/位数。SSC 和值/跨度/尾数不计前端 segmentLen
//（单档 UI 常为 1，覆盖后跨度恒 0、次数全堆在「0」）。
func applyHotColdWarmInputOverrides(rule playRule, in HotColdWarmTiersInput) playRule {
	tpl := strings.TrimSpace(in.PlayTemplate)
	bm := strings.ToLower(strings.TrimSpace(rule.BetMode))
	sscDigitAttr := (tpl == "" || tpl == "ssc_std" || tpl == "fast_ssc_std") &&
		(bm == "hezhi" || bm == "kuadu" || bm == "weishu")
	if sscDigitAttr {
		if bm != "hezhi" && in.NumberPoolMax > 0 && in.NumberPoolMax >= in.NumberPoolMin {
			rule.NumberPoolMin = in.NumberPoolMin
			rule.NumberPoolMax = in.NumberPoolMax
		}
		if bm == "hezhi" {
			text := in.PlayMethodLabel + " " + in.CatalogSubID + " " + in.SubPlayID
			rule.HezhiZuxuan = strings.Contains(text, "组选")
		}
		return rule
	}
	if in.NumberPoolMax > 0 && in.NumberPoolMax >= in.NumberPoolMin {
		rule.NumberPoolMin = in.NumberPoolMin
		rule.NumberPoolMax = in.NumberPoolMax
	}
	if in.SegmentLen > 0 && attributeUsesInputSegmentLen(rule.BetMode) {
		rule.SegmentLen = in.SegmentLen
	}
	return rule
}

// attributeUsesInputSegmentLen 是否允许用请求体 segmentLen 覆盖 resolve 结果。
func attributeUsesInputSegmentLen(betMode string) bool {
	switch strings.ToLower(strings.TrimSpace(betMode)) {
	case "hezhi", "kuadu", "weishu":
		return true
	default:
		return false
	}
}

// inferAttributeBetModeFromLabel 从玩法文案推断属性家族 betMode（冷热分档接口兜底）。
func inferAttributeBetModeFromLabel(label string) string {
	s := strings.TrimSpace(label)
	switch {
	case strings.Contains(s, "特殊号"):
		return "teshu"
	case strings.Contains(s, "龙虎豹"):
		return "longhubao"
	case strings.Contains(s, "大小单双"):
		return "dxds"
	case strings.Contains(s, "庄闲"):
		return "zhuangxian"
	case strings.Contains(s, "和值尾数") || (strings.Contains(s, "尾数") && !strings.Contains(s, "单双") && !strings.Contains(s, "大小")):
		return "weishu"
	case strings.Contains(s, "跨度"):
		return "kuadu"
	case strings.Contains(s, "和值"):
		return "hezhi"
	case strings.Contains(s, "龙虎"):
		if strings.Contains(s, "和") {
			return "longhuhe"
		}
		return "longhu"
	default:
		return ""
	}
}
