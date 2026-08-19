package schemebettingdispatch

import (
	"encoding/json"
	"testing"
	"time"

	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebetting"
)

func TestValidateAPIBetCommandRequiresIdempotencyAndCurrency(t *testing.T) {
	command := APIBetCommand{
		RequestID: "request-1", LocalOrderNo: "BO1", MemberID: 1, MemberAccount: "member-1",
		BetPayload: json.RawMessage(`{"groupContent":"1"}`),
		Request:    guajibet.Request{LotteryCode: "lottery", IssueNo: "T", Amount: 2, Currency: "CNY"},
	}
	if err := validateAPIBetCommand(command); err != nil {
		t.Fatal(err)
	}
	command.RequestID = ""
	if err := validateAPIBetCommand(command); err == nil {
		t.Fatal("missing request id must fail")
	}
	command.RequestID = "request-1"
	command.Request.Currency = ""
	if err := validateAPIBetCommand(command); err == nil {
		t.Fatal("missing frozen currency must fail")
	}
}

func TestAPIDeadlineBudgetMatchesFastPeriodSafety(t *testing.T) {
	target := schemebetting.PeriodSnapshot{OpenAt: time.Unix(0, 0), CloseAt: time.Unix(0, 0).Add(6 * time.Second)}
	if got := apiDeadlineBudget(target).Total(); got != 1300*time.Millisecond {
		t.Fatalf("fast period budget = %s", got)
	}
}
