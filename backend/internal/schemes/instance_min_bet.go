package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/guaji"
)

// 第三方单次投注最低金额（元）：USDT=0.1，其他币种=1
const (
	minSingleBetAmountUSDT  = 0.1
	minSingleBetAmountOther = 1.0
)

// ErrMinBetAmountTooLow 单次投注金额低于第三方最低限制，不可开启。
var ErrMinBetAmountTooLow = errors.New("单次投注金额低于最低限制 请提高投注单位、倍数系数、倍投倍率或注数后再开启")

func minSingleBetAmountForCurrency(currency string) float64 {
	if guaji.NormalizeCurrency(currency) == guaji.CurrencyUSDT {
		return minSingleBetAmountUSDT
	}
	return minSingleBetAmountOther
}

func errMinBetAmountTooLow(currency string) error {
	minAmt := minSingleBetAmountForCurrency(currency)
	return fmt.Errorf("%w: 单次投注金额低于%.4g 请提高投注单位、倍数系数、倍投倍率或注数后再开启", ErrMinBetAmountTooLow, minAmt)
}

// validateSchemeMinBetAmount 启动前校验：betUnit × 实例倍数系数 × 模式最低倍率 × 注数 >= 币种最低额
func validateSchemeMinBetAmount(cfgBytes []byte, kind, currency string, instanceMult pgtype.Numeric) error {
	amount := schemeMinSingleBetAmount(cfgBytes, kind, instanceMult)
	minAmt := minSingleBetAmountForCurrency(currency)
	if amount+1e-9 < minAmt {
		return errMinBetAmountTooLow(currency)
	}
	return nil
}

func schemeMinSingleBetAmount(cfgBytes []byte, kind string, instanceMult pgtype.Numeric) float64 {
	unit := baseBetUnitYuan
	minModeMult := 1.0
	betUnits := 1
	if len(cfgBytes) > 0 {
		var cfg map[string]interface{}
		if err := json.Unmarshal(cfgBytes, &cfg); err == nil {
			unit = schemeBetUnitFromConfig(cfg)
			minModeMult = schemeMinModeMultiplier(cfg)
		}
		betUnits = schemeMinBetUnits(kind, cfgBytes)
	}
	coef := instanceBaseCoef(instanceMult)
	if unit <= 0 {
		unit = baseBetUnitYuan
	}
	if coef <= 0 {
		coef = 1
	}
	if minModeMult <= 0 {
		minModeMult = 1
	}
	if betUnits <= 0 {
		betUnits = 1
	}
	return round2(unit * coef * minModeMult * float64(betUnits))
}

// schemeMinBetUnits 取方案各组中的最低注数；无有效内容时按 1。
func schemeMinBetUnits(kind string, cfgBytes []byte) int {
	if kind == "" {
		kind = "custom"
	}
	parsed := parseSchemeConfig(kind, cfgBytes, 0, 0)
	minUnits := 0
	for _, g := range parsed.Groups {
		u := planPickBetUnits(parsed, g)
		if u <= 0 {
			continue
		}
		if minUnits == 0 || u < minUnits {
			minUnits = u
		}
	}
	if minUnits <= 0 {
		u := planPickBetUnits(parsed, parsed.GroupContent)
		if u > 0 {
			minUnits = u
		}
	}
	if minUnits <= 0 {
		return 1
	}
	return minUnits
}

// schemeMinModeMultiplier 取方案模式（rounds / betMultiplier）中的最低有效倍率；无效时按 1。
func schemeMinModeMultiplier(cfg map[string]interface{}) float64 {
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
	minMult := math.Inf(1)
	for _, r := range rounds {
		m := r.Mult
		if m <= 0 {
			m = 1
		}
		if m < minMult {
			minMult = m
		}
	}
	if math.IsInf(minMult, 1) {
		return 1
	}
	return minMult
}

func (s *Service) memberPrimaryCurrency(ctx context.Context, memberID int64) string {
	if s == nil || s.pool == nil {
		return guaji.CurrencyCNY
	}
	var c string
	err := s.pool.QueryRow(ctx, `SELECT primary_currency FROM members WHERE id = $1`, memberID).Scan(&c)
	if err != nil || strings.TrimSpace(c) == "" {
		return guaji.CurrencyCNY
	}
	return guaji.NormalizeCurrency(c)
}
