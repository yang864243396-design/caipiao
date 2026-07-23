package guajibet

import (
	"context"
	"errors"

	"caipiao/backend/internal/db/sqlcdb"
)

// T4：第三方真实下单网关契约（games / schemes / accountsvc 共用，避免 import cycle）。

var (
	ErrNoActiveAuth  = errors.New("无启用中的授权账号")
	ErrTokenInvalid  = errors.New("授权已失效，请重新授权")
	ErrUpstream      = errors.New("第三方服务暂时不可用，请稍后重试")
	ErrInsufficient  = errors.New("第三方可用余额不足")
	ErrPeriodClosed  = errors.New("当前期已封盘")
	ErrPlaceRejected = errors.New("第三方接单失败")
	// ErrZeroBets 直选复式豹子/对子等第三方计 0 注、无法下注的选号。
	ErrZeroBets = errors.New("投注注数为0（直选复式不含豹子/对子）")
)

// Request 由 outbound 解析 + 投注参数组装。
type Request struct {
	LotteryCode string // 平台彩种 code；PlaceRealBet 优先据此查 lottery_catalog.outbound_lottery_code
	GameID     string
	RuleID     string
	IssueNo    string
	Content    string
	PlayMethod string
	Amount     float64
	Multiplier int
	BetsNums   int
	AmountUnit float64
	// Currency 方案币种（USDT/TRX/CNY）；空则回退会员主币种。
	Currency string
	// RuleMeta 来自 sub_plays.segment_rule；非空 PlayTemplate 时用于 bets_nums / solo 计算。
	RuleMeta RuleMeta
}

// Result 第三方接单成功返回。
type Result struct {
	GuajiAccountID  int64
	ThirdPartyBetID string
	Periods         string // 第三方 periods 期号（防重与校验依据）
	Currency        string
}

// Placer 抽象第三方接单：取启用授权 Token、校验主币种余额、调用 web_bets/lott。
type Placer interface {
	Enabled() bool
	PlaceRealBet(ctx context.Context, memberAccount string, req Request) (Result, error)
	MirrorBetDebitLedger(ctx context.Context, qtx *sqlcdb.Queries, memberID int64, orderNo string, stake float64, guajiAccountID int64, currency string) error
}
