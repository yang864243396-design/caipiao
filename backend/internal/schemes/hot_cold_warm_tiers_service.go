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
	cfg := map[string]interface{}{
		"playTypeId":   in.PlayTypeID,
		"subPlayId":    in.SubPlayID,
		"playTemplate": in.PlayTemplate,
		"betMode":      in.BetMode,
		"catalogSubId": in.CatalogSubID,
	}
	rule := resolvePlayRule(cfg, in.PlayMethodLabel)
	// 龙虎等按 CatalogSubID 解析对比位；缺失时回退 SubPlayID，避免位解析失败导致零命中。
	rule.CatalogSubID = in.CatalogSubID
	if strings.TrimSpace(rule.CatalogSubID) == "" {
		rule.CatalogSubID = in.SubPlayID
	}
	if in.NumberPoolMax > 0 && in.NumberPoolMax >= in.NumberPoolMin {
		rule.NumberPoolMin = in.NumberPoolMin
		rule.NumberPoolMax = in.NumberPoolMax
	}
	// 前端已知真实段长（如 K3=3、PC28=3）；用于和值等聚合玩法的选项宇宙上下界，
	// 避免 resolvePlayRule 对 hezhi/teshu 类 playTypeId 用默认段长产生不可能的和值项。
	if in.SegmentLen > 0 {
		rule.SegmentLen = in.SegmentLen
	}
	return HotColdWarmAttributeTiers(rule, draws), nil
}
