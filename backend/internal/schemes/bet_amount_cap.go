package schemes

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// 第三方单次投注最高金额（对齐 guaji code=40053「最高下注限额100000.00USDT」）。
const maxSingleBetAmount = 100000.0

// ErrMaxBetAmountExceeded 单次投注金额超过第三方上限（可 errors.Is / 文案匹配）。
var ErrMaxBetAmountExceeded = errors.New("最高下注限额")

func maxBetAmountExceededMessage(currency string) string {
	cur := normalizeSchemeCurrency(currency)
	return fmt.Sprintf("%s%.2f%s", ErrMaxBetAmountExceeded.Error(), maxSingleBetAmount, cur)
}

func errMaxBetAmountExceeded(currency string) error {
	// 文案对齐第三方：最高下注限额100000.00USDT（%w 便于 errors.Is）
	return fmt.Errorf("%w%.2f%s", ErrMaxBetAmountExceeded, maxSingleBetAmount, normalizeSchemeCurrency(currency))
}

// isBetAmountExceededError 本端金额上限或第三方「最高下注限额」文案。
func isBetAmountExceededError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrMaxBetAmountExceeded) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, ErrMaxBetAmountExceeded.Error())
}

func betAmountExceedsMax(amount float64) bool {
	return amount > maxSingleBetAmount+1e-9
}

// validateSchemeMaxBetAmount 启动前：最大可能单次金额（单位×实例倍数×模式最高倍率×最大组注数）不得超过上限。
func validateSchemeMaxBetAmount(cfgBytes []byte, kind, currency string, instanceMult pgtype.Numeric) error {
	amount := schemeMaxSingleBetAmount(cfgBytes, kind, instanceMult)
	if betAmountExceedsMax(amount) {
		return errMaxBetAmountExceeded(currency)
	}
	return nil
}

func schemeMaxSingleBetAmount(cfgBytes []byte, kind string, instanceMult pgtype.Numeric) float64 {
	unit := baseBetUnitYuan
	maxModeMult := 1.0
	betUnits := 1
	if len(cfgBytes) > 0 {
		var cfg map[string]interface{}
		if err := json.Unmarshal(cfgBytes, &cfg); err == nil {
			unit = schemeBetUnitFromConfig(cfg)
			maxModeMult = schemeMaxModeMultiplier(cfg)
		}
		betUnits = schemeMaxBetUnits(kind, cfgBytes)
	}
	coef := instanceBaseCoef(instanceMult)
	if unit <= 0 {
		unit = baseBetUnitYuan
	}
	if coef <= 0 {
		coef = 1
	}
	if maxModeMult <= 0 {
		maxModeMult = 1
	}
	if betUnits <= 0 {
		betUnits = 1
	}
	return round2(unit * coef * maxModeMult * float64(betUnits))
}

// schemeMaxBetUnits 取方案各组中的最高注数；无有效内容时按 1。
func schemeMaxBetUnits(kind string, cfgBytes []byte) int {
	if kind == "" {
		kind = "custom"
	}
	parsed := parseSchemeConfig(kind, cfgBytes, 0, 0)
	maxUnits := 0
	for _, g := range parsed.Groups {
		u := planPickBetUnits(parsed, g)
		if u > maxUnits {
			maxUnits = u
		}
	}
	if u := planPickBetUnits(parsed, parsed.GroupContent); u > maxUnits {
		maxUnits = u
	}
	if parsed.HotCold != nil && len(parsed.HotCold.Pool) > 0 {
		if u := planPickBetUnits(parsed, strings.Join(parsed.HotCold.Pool, "\n")); u > maxUnits {
			maxUnits = u
		}
	}
	if maxUnits <= 0 {
		return 1
	}
	return maxUnits
}

// schemeMaxModeMultiplier 取方案模式（rounds / betMultiplier）中的最高有效倍率；无效时按 1。
func schemeMaxModeMultiplier(cfg map[string]interface{}) float64 {
	if cfg == nil {
		return 1
	}
	rounds := resolveRounds(cfg)
	if len(rounds) == 0 {
		if bm, ok := cfg["betMultiplier"].(map[string]interface{}); ok {
			compiled := compileBetMultiplierRounds(bm, cfg)
			rounds = normalizeSchemeRounds(compiled)
		}
	}
	if len(rounds) == 0 {
		return 1
	}
	maxMult := 0.0
	for _, r := range rounds {
		m := r.Mult
		if m <= 0 {
			m = 1
		}
		if m > maxMult {
			maxMult = m
		}
	}
	if maxMult <= 0 {
		return 1
	}
	return maxMult
}
