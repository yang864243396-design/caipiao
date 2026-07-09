package schemes

import (
	"errors"
	"testing"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guajibet"
)

func TestInstanceStatusLabel(t *testing.T) {
	cases := []struct {
		status, reason, detail, want string
	}{
		{"pending", StatusReasonInsufficientFunds, "", "钱包余额不足"},
		{"pending", StatusReasonBetFailed, "", "投注失败"},
		{"pending", StatusReasonBetFailed, "余额不足", "投注失败-余额不足"},
		{"pending", StatusReasonMaintenance, "", "因彩种维护关闭"},
		{"pending", StatusReasonEndTime, "", "已到结束时间"},
		{"pending", StatusReasonSchemeStopLoss, "", "已达方案止损"},
		{"pending", StatusReasonSchemeTakeProfit, "", "已达方案止盈"},
		{"pending", StatusReasonTotalStopLoss, "", "已达总止损"},
		{"pending", StatusReasonTotalTakeProfit, "", "已达总止盈"},
		{"pending", "", "", "等待开启"},
		{"running", "", "", "运行中"},
		{"running", StatusReasonAwaitNextBet, "", "将在下期开始投注"},
		{"running", StatusReasonCloudActive, "", "正在云端挂机中"},
		{"paused", StatusReasonManual, "", "已暂停"},
		{"paused", StatusReasonInsufficientFunds, "", "钱包余额不足"},
		{"paused", StatusReasonMaintenance, "", "因彩种维护关闭"},
		{"paused", StatusReasonEndTime, "", "已到结束时间"},
		{"soft_stopped", "", "", "已封停"},
	}
	for _, c := range cases {
		if got := instanceStatusLabel(c.status, c.reason, c.detail); got != c.want {
			t.Fatalf("%s/%s/%s => %q want %q", c.status, c.reason, c.detail, got, c.want)
		}
	}
}

func TestGuajiBetFailedDetail_apiMessage(t *testing.T) {
	err := errors.Join(guajibet.ErrPlaceRejected, &guaji.APIError{Code: 400, Message: "当前期已封盘"})
	got := guajiBetFailedDetail(err)
	if got != "当前期已封盘" {
		t.Fatalf("got %q", got)
	}
}

func TestGuajiBetFailedDetail_tokenInvalid(t *testing.T) {
	got := guajiBetFailedDetail(guajibet.ErrTokenInvalid)
	if got != "授权已失效，请重新授权" {
		t.Fatalf("got %q", got)
	}
}
