package schemelimits

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

const (
	ReasonStopLoss   = "scheme_stop_loss"
	ReasonTakeProfit = "scheme_take_profit"
)

type Limits struct {
	StopLossYuan   float64
	TakeProfitYuan float64
}

func Parse(config []byte) Limits {
	out := Limits{}
	if len(config) == 0 {
		return out
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return out
	}
	out.StopLossYuan = parseAmount(cfg["stopLoss"])
	out.TakeProfitYuan = parseAmount(cfg["takeProfit"])
	return out
}

func parseAmount(v interface{}) float64 {
	switch n := v.(type) {
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || f <= 0 {
			return 0
		}
		return round2(f)
	case float64:
		if n <= 0 {
			return 0
		}
		return round2(n)
	case json.Number:
		f, err := n.Float64()
		if err != nil || f <= 0 {
			return 0
		}
		return round2(f)
	case int:
		if n <= 0 {
			return 0
		}
		return round2(float64(n))
	default:
		return 0
	}
}

func Evaluate(sessionPnl float64, config []byte) (reason string, hit bool) {
	limits := Parse(config)
	sessionPnl = round2(sessionPnl)
	if limits.StopLossYuan > 0 && sessionPnl <= -limits.StopLossYuan {
		return ReasonStopLoss, true
	}
	if limits.TakeProfitYuan > 0 && sessionPnl >= limits.TakeProfitYuan {
		return ReasonTakeProfit, true
	}
	return "", false
}

func Detail(reason string, sessionPnl float64, limits Limits) string {
	switch reason {
	case ReasonStopLoss:
		return formatDetail("止损", limits.StopLossYuan, sessionPnl)
	case ReasonTakeProfit:
		return formatDetail("止盈", limits.TakeProfitYuan, sessionPnl)
	default:
		return ""
	}
}

func formatDetail(kind string, threshold, sessionPnl float64) string {
	if threshold <= 0 {
		return ""
	}
	return strings.TrimSpace(
		kind + " " + trimFloat(threshold) + " 元，本次盈亏 " + trimFloat(sessionPnl) + " 元",
	)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func trimFloat(v float64) string {
	if math.Abs(v-math.Round(v)) < 0.005 {
		return strconv.FormatInt(int64(math.Round(v)), 10)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}
