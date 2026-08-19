package schemebettingdispatch

import (
	"testing"

	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebetting"
)

func TestValidateUnknownResolutionRequiresProviderEvidence(t *testing.T) {
	frozen := FrozenGuajiRequest{Request: guajibet.Request{IssueNo: "20260819001", Amount: 12.5, Currency: "CNY"}}
	resolution := schemebetting.UnknownResolution{
		Outcome: "accepted", ProviderOrderID: "provider-1", AcceptedPeriod: "20260819001",
		ProviderAmount: 12.5, ProviderCurrency: "CNY", ProviderAccountID: 9,
	}
	if _, err := validateUnknownResolution(resolution, frozen, "20260819001"); err == nil {
		t.Fatal("accepted resolution without evidence must fail")
	}
	resolution.Evidence = "provider order detail screenshot checked"
	if state, err := validateUnknownResolution(resolution, frozen, "20260819001"); err != nil || state != schemebetting.OutboxAccepted {
		t.Fatalf("valid accepted resolution state=%s err=%v", state, err)
	}
}

func TestValidateUnknownResolutionRejectsFinancialMismatchAndMarksWrongPeriod(t *testing.T) {
	frozen := FrozenGuajiRequest{Request: guajibet.Request{IssueNo: "20260819001", Amount: 12.5, Currency: "CNY"}}
	resolution := schemebetting.UnknownResolution{
		Outcome: "accepted", Evidence: "provider order detail checked", ProviderOrderID: "provider-1",
		AcceptedPeriod: "20260819001", ProviderAmount: 13, ProviderCurrency: "CNY", ProviderAccountID: 9,
	}
	if _, err := validateUnknownResolution(resolution, frozen, "20260819001"); err == nil {
		t.Fatal("provider amount mismatch must fail")
	}
	resolution.ProviderAmount = 12.5
	resolution.AcceptedPeriod = "20260819002"
	if state, err := validateUnknownResolution(resolution, frozen, "20260819001"); err != nil || state != schemebetting.OutboxAcceptedWrongPeriod {
		t.Fatalf("wrong-period resolution state=%s err=%v", state, err)
	}
}

func TestValidateUnknownResolutionAllowsEvidenceBackedRejection(t *testing.T) {
	frozen := FrozenGuajiRequest{Request: guajibet.Request{IssueNo: "20260819001", Amount: 12.5, Currency: "CNY"}}
	resolution := schemebetting.UnknownResolution{Outcome: "rejected", Evidence: "provider account history checked"}
	if state, err := validateUnknownResolution(resolution, frozen, "20260819001"); err != nil || state != schemebetting.OutboxRejected {
		t.Fatalf("rejected resolution state=%s err=%v", state, err)
	}
}
