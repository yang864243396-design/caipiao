package games

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
)

var ErrSubPlayNotFound = errors.New("sub play not found")

// OutboundBet 是本平台彩种/玩法解析为第三方 web_bets/lott 下单参数的结果（T2）。
//
// 映射口径（C3 / C7 / C46）：
//   - GameID = lottery_catalog.outbound_lottery_code（运营维护态填真实第三方彩种码；空回退对内 code）
//   - RuleID = 第三方数字 rule.id（sub_plays.outbound_play_code 或 segment_rule.guajiRuleId）
//
// 本平台不维护独立第三方对照表（C7）；outbound_* 即对接码来源。
type OutboundBet struct {
	LotteryCode         string `json:"lotteryCode"`
	OutboundLotteryCode string `json:"outboundLotteryCode"`
	GameID              string `json:"gameId"`
	PlayTemplate        string `json:"playTemplate"`
	TypeID              string `json:"typeId"`
	SubID               string `json:"subId"`
	OutboundPlayCode    string `json:"outboundPlayCode"`
	RuleID              string `json:"ruleId"`
}

// ResolveOutbound 将彩种 code + 玩法（typeId/subId）解析为第三方下单参数。
// 维护态彩种不可下单（ErrLotteryMaintenance）；下线/不存在彩种 ErrLotteryNotFound。
func (s *Service) ResolveOutbound(ctx context.Context, lotteryCode, typeID, subID string) (OutboundBet, error) {
	if s == nil || s.q == nil {
		return OutboundBet{}, ErrUnavailable
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	typeID = strings.TrimSpace(typeID)
	subID = strings.TrimSpace(subID)
	if lotteryCode == "" || typeID == "" || subID == "" {
		return OutboundBet{}, fmt.Errorf("%w: lotteryCode/typeId/subId 不能为空", ErrSubPlayNotFound)
	}

	cat, err := s.q.GetLotteryCatalogByCode(ctx, lotteryCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OutboundBet{}, ErrLotteryNotFound
		}
		return OutboundBet{}, err
	}
	switch cat.SaleStatus {
	case "maintenance":
		return OutboundBet{}, ErrLotteryMaintenance
	case "on_sale":
		// ok
	default:
		return OutboundBet{}, ErrLotteryNotFound
	}

	template := textVal(cat.PlayTemplate)
	if template == "" {
		return OutboundBet{}, ErrLotteryNotFound
	}

	sub, err := s.q.GetSubPlay(ctx, sqlcdb.GetSubPlayParams{
		TemplateCode: template,
		TypeID:       typeID,
		SubID:        subID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OutboundBet{}, ErrSubPlayNotFound
		}
		return OutboundBet{}, err
	}

	outboundLottery := textVal(cat.OutboundLotteryCode)
	if outboundLottery == "" {
		outboundLottery = cat.Code
	}
	outboundPlay := textVal(sub.OutboundPlayCode)
	if outboundPlay == "" {
		outboundPlay = fmt.Sprintf("%s:%s:%s", template, typeID, subID)
	}
	ruleID := guajibet.ExtractGuajiRuleID(textVal(sub.OutboundPlayCode), sub.SegmentRule, sub.SubID)
	if ruleID == "" {
		return OutboundBet{}, fmt.Errorf("第三方 rule_id 未配置，请执行 guaji-rules-sync: %s/%s/%s outbound=%q", template, typeID, subID, outboundPlay)
	}

	return OutboundBet{
		LotteryCode:         cat.Code,
		OutboundLotteryCode: outboundLottery,
		GameID:              outboundLottery,
		PlayTemplate:        template,
		TypeID:              typeID,
		SubID:               subID,
		OutboundPlayCode:    outboundPlay,
		RuleID:              ruleID,
	}, nil
}
