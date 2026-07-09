package schemes

import (
	"strings"

	"caipiao/backend/internal/cloudlimits"
	"caipiao/backend/internal/schemelimits"
)

// status_reason：paused 停投原因；running 运行阶段（await_next_bet / cloud_active）。
const (
	StatusReasonManual            = "manual"
	StatusReasonInsufficientFunds = "insufficient_funds"
	StatusReasonMaintenance       = "maintenance"
	StatusReasonEndTime           = "end_time"
	StatusReasonAwaitNextBet      = "await_next_bet"
	StatusReasonCloudActive       = "cloud_active"
	StatusReasonBetFailed         = "bet_failed"
)

const (
	StatusReasonSchemeStopLoss   = schemelimits.ReasonStopLoss
	StatusReasonSchemeTakeProfit = schemelimits.ReasonTakeProfit
	StatusReasonTotalStopLoss    = cloudlimits.ReasonTotalStopLoss
	StatusReasonTotalTakeProfit  = cloudlimits.ReasonTotalTakeProfit
)

func instanceStatusLabel(status, reason, betFailedDetail string) string {
	switch status {
	case "running":
		switch reason {
		case StatusReasonAwaitNextBet:
			return "将在下期开始投注"
		case StatusReasonCloudActive:
			return "正在云端挂机中"
		}
		return "运行中"
	case "soft_stopped":
		return "已封停"
	case "paused":
		switch reason {
		case StatusReasonInsufficientFunds:
			return "钱包余额不足"
		case StatusReasonMaintenance:
			return "因彩种维护关闭"
		case StatusReasonEndTime:
			return "已到结束时间"
		}
		return "已暂停"
	case "pending":
		switch reason {
		case StatusReasonInsufficientFunds:
			return "钱包余额不足"
		case StatusReasonBetFailed:
			if detail := strings.TrimSpace(betFailedDetail); detail != "" {
				return "投注失败-" + detail
			}
			return "投注失败"
		case StatusReasonSchemeStopLoss:
			return "已达方案止损"
		case StatusReasonSchemeTakeProfit:
			return "已达方案止盈"
		case StatusReasonTotalStopLoss:
			return "已达总止损"
		case StatusReasonTotalTakeProfit:
			return "已达总止盈"
		case StatusReasonMaintenance:
			return "因彩种维护关闭"
		case StatusReasonEndTime:
			return "已到结束时间"
		}
		return "等待开启"
	default:
		return "等待开启"
	}
}
