package cloudlimits

import (
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	ReasonTotalStopLoss   = "total_stop_loss"
	ReasonTotalTakeProfit = "total_take_profit"
)

type Limits struct {
	StopLossYuan   float64
	TakeProfitYuan float64
}

func LimitsFromSettings(totalStopLoss, totalTakeProfit pgtype.Numeric) Limits {
	return Limits{
		StopLossYuan:   numericToFloat(totalStopLoss),
		TakeProfitYuan: numericToFloat(totalTakeProfit),
	}
}

func numericToFloat(n pgtype.Numeric) float64 {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return round2(f.Float64)
}

func Evaluate(totalPnl float64, limits Limits) (reason string, hit bool) {
	totalPnl = round2(totalPnl)
	if limits.StopLossYuan > 0 && totalPnl <= -limits.StopLossYuan {
		return ReasonTotalStopLoss, true
	}
	if limits.TakeProfitYuan > 0 && totalPnl >= limits.TakeProfitYuan {
		return ReasonTotalTakeProfit, true
	}
	return "", false
}

func Detail(reason string, totalPnl float64, limits Limits) string {
	switch reason {
	case ReasonTotalStopLoss:
		return formatDetail("总止损", limits.StopLossYuan, totalPnl)
	case ReasonTotalTakeProfit:
		return formatDetail("总止盈", limits.TakeProfitYuan, totalPnl)
	default:
		return ""
	}
}

func formatDetail(kind string, threshold, totalPnl float64) string {
	if threshold <= 0 {
		return ""
	}
	return strings.TrimSpace(
		kind + " " + trimFloat(threshold) + " 元，正式盘合计盈亏 " + trimFloat(totalPnl) + " 元",
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
