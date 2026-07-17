package schemes

import (
	"math"
	"strings"
	"sync/atomic"
)

// refThreeStarOdds 三星直选「1 元单注可中」参考基准。
// 对应第三方 real/rate.lott_odds=1940（2 元）÷2=970。odds 常量表以此为基准，
// 实际派彩按第三方账号赔率线（lott_odds/hash_odds）按比例缩放。
const refThreeStarOdds = 970.0

// liveGuajiOdds 缓存第三方账号级赔率线（2 元三星直选基准）。
// lott：普通彩（波场/以太/币安）；hash：哈希彩。0 表示未知（回退参考基准，缩放=1）。
var (
	liveLottOdds atomic.Uint64 // math.Float64bits
	liveHashOdds atomic.Uint64
)

// SetGuajiOdds 由后台刷新器写入第三方 real/rate 的 lott_odds/hash_odds（2 元三星直选基准）。
func SetGuajiOdds(lottOdds, hashOdds float64) {
	if lottOdds > 0 {
		liveLottOdds.Store(math.Float64bits(lottOdds))
	}
	if hashOdds > 0 {
		liveHashOdds.Store(math.Float64bits(hashOdds))
	}
}

// oddsBaseForLottery 返回该彩种「1 元三星直选」基准派彩（未知返回 0，交由 oddsScale 兜底）。
func oddsBaseForLottery(lotteryCode string) float64 {
	code := strings.ToLower(strings.TrimSpace(lotteryCode))
	var raw uint64
	if strings.HasPrefix(code, "hash_") {
		raw = liveHashOdds.Load()
	} else {
		raw = liveLottOdds.Load()
	}
	v := math.Float64frombits(raw)
	if v <= 0 {
		return 0
	}
	return v / 2
}

// attachOddsBase 将第三方账号级赔率线注入玩法规则（预估/模拟/本地估算共用）。
func attachOddsBase(rule playRule, lotteryCode string) playRule {
	rule.OddsBase = oddsBaseForLottery(lotteryCode)
	return rule
}

// oddsScale 相对参考基准的缩放系数。base<=0（未取到第三方赔率）时为 1，保持常量表原值。
func oddsScale(base float64) float64 {
	if base <= 0 {
		return 1
	}
	return base / refThreeStarOdds
}

// scaleEvalOdds 按第三方赔率线缩放单区评估的赔率与净奖金（仅缩放派彩，不动本金）。
// 仅用于「叶子内部按参考基准计算、出口统一缩放」的单区路径（如任选），
// 多区位路径须在各区叶子内缩放，勿在折算后再缩放。
func scaleEvalOdds(ev betEvaluation, base float64) betEvaluation {
	s := oddsScale(base)
	if s == 1 {
		return ev
	}
	ev.Odds *= s
	ev.PrizeNet *= s
	return ev
}
