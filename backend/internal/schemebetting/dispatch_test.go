package schemebetting

import (
	"errors"
	"testing"
	"time"
)

func TestResolveDispatchOutcomeAcceptsOnlyMatchingUniqueOrder(t *testing.T) {
	got := ResolveDispatchOutcome("period-T", DispatchObservation{
		RequestStarted: true,
		Result:         &ProviderAcceptance{OrderID: "order-1", PeriodNo: "period-T", Amount: 2, AccountID: 8, Currency: "CNY"},
	})
	if got.State != OutboxAccepted || got.BlocksChain {
		t.Fatalf("matching result = %+v", got)
	}

	wrong := ResolveDispatchOutcome("period-T", DispatchObservation{
		RequestStarted: true,
		Result:         &ProviderAcceptance{OrderID: "order-2", PeriodNo: "period-U", Amount: 2, AccountID: 8, Currency: "CNY"},
	})
	if wrong.State != OutboxAcceptedWrongPeriod || !wrong.BlocksChain {
		t.Fatalf("wrong-period result = %+v", wrong)
	}
}

func TestResolveDispatchOutcomeRequiresProviderFinancialIdentity(t *testing.T) {
	for _, result := range []ProviderAcceptance{
		{OrderID: "order-1", PeriodNo: "period-T", AccountID: 8, Currency: "CNY"},
		{OrderID: "order-1", PeriodNo: "period-T", Amount: 2, Currency: "CNY"},
		{OrderID: "order-1", PeriodNo: "period-T", Amount: 2, AccountID: 8},
	} {
		got := ResolveDispatchOutcome("period-T", DispatchObservation{RequestStarted: true, Result: &result})
		if got.State != OutboxSentUnknown || !got.BlocksChain {
			t.Fatalf("incomplete provider financial identity %+v resolved to %+v", result, got)
		}
	}
}

func TestResolveDispatchOutcomeNeverRetriesAmbiguousSend(t *testing.T) {
	tests := []DispatchObservation{
		{RequestStarted: true, Err: errors.New("timeout")},
		{RequestStarted: true, Result: &ProviderAcceptance{PeriodNo: "period-T"}},
		{RequestStarted: true, Result: &ProviderAcceptance{OrderID: "order-1"}},
	}
	for _, observation := range tests {
		got := ResolveDispatchOutcome("period-T", observation)
		if got.State != OutboxSentUnknown || !got.BlocksChain || got.Retryable {
			t.Fatalf("ambiguous observation %+v resolved to %+v", observation, got)
		}
	}
}

func TestResolveDispatchOutcomeAllowsOnlyProvenPreSendFailure(t *testing.T) {
	got := ResolveDispatchOutcome("period-T", DispatchObservation{
		Err:               errors.New("local validation failed"),
		DefinitelyNotSent: true,
	})
	if got.State != OutboxRejected || got.Reason != "provider_pre_send_failed" || got.BlocksChain || got.Retryable {
		t.Fatalf("pre-send failure = %+v", got)
	}
}

func TestLeaseFenceRejectsStaleOwnerAndExpiredLease(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	lease := LeaseFence{Owner: "dispatcher-b", Token: 8, Until: now.Add(time.Second)}
	if !lease.CanCommit("dispatcher-b", 8, now) {
		t.Fatal("current lease should commit")
	}
	if lease.CanCommit("dispatcher-a", 8, now) || lease.CanCommit("dispatcher-b", 7, now) {
		t.Fatal("stale owner or token must not commit")
	}
	if lease.CanCommit("dispatcher-b", 8, lease.Until) {
		t.Fatal("lease expiry equality must reject commit")
	}
}

func TestTerminalFailuresBlockStrictChainUntilExplicitRearm(t *testing.T) {
	chain := ChainStateActive
	chain = ApplyDispatchToChain(chain, OutboxExpired)
	if chain != ChainStateBlockedRequiresRearm {
		t.Fatalf("expired chain = %s", chain)
	}
	if ApplyDispatchToChain(chain, OutboxAccepted) != ChainStateBlockedRequiresRearm {
		t.Fatal("later evidence must not silently rearm a blocked chain")
	}
	if RearmChain(chain) != ChainStateActive {
		t.Fatal("explicit rearm must create a new active chain")
	}
}

func TestSentUnknownBlocksStrictChain(t *testing.T) {
	if got := ApplyDispatchToChain(ChainStateActive, OutboxSentUnknown); got != ChainStateBlockedRequiresRearm {
		t.Fatalf("sent unknown chain = %s", got)
	}
}
