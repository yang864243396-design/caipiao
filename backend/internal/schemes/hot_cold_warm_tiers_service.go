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
	// 龙虎等按 CatalogSubID 解析对比位；缺失时回退 SubPlayID，避免位解析失败导致零命中。
	rule.CatalogSubID = in.CatalogSubID
	if strings.TrimSpace(rule.CatalogSubID) == "" {
		rule.CatalogSubID = in.SubPlayID
	}
	if in.NumberPoolMax > 0 && in.NumberPoolMax >= in.NumberPoolMin {
		rule.NumberPoolMin = in.NumberPoolMin
		rule.NumberPoolMax = in.NumberPoolMax
	}
	// 和值/跨度等选项宇宙需要真实数字段长（如前三和值 0..27）。
	// 特殊号/龙虎/大小单双等：前端 playConfig 常把 segmentLen 置为 1（单档选项池 UI），
	// 若覆盖 resolve 出的前三=3，形态判定会全程 0 命中。
	if in.SegmentLen > 0 && attributeUsesInputSegmentLen(rule.BetMode) {
		rule.SegmentLen = in.SegmentLen
	}
	return HotColdWarmAttributeTiers(rule, draws), nil
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
