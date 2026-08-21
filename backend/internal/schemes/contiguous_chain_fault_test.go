package schemes

import (
	"testing"
	"time"

	"caipiao/backend/internal/guaji"
)

func TestContiguousChainFaultStrategyAndBoundaryRedeliveryIsPersistentlyIdempotent(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.deliverBoundaryBeforePhaseOne()
	f.deliverStrategyReady(3)
	f.deliverBoundaryReady(3)
	f.assertOneStrategyAdvance()
	f.assertOneDecisionForSource()
	f.assertOneOutboxWithStableRequestID()
}

func TestContiguousChainFaultRestartUnexpiredWaitingResolvesOnce(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.deliverStrategyReady(1)
	f.restartAndRecoverWaiting(2)
	f.assertOneDecisionForSource()
	f.assertOneOutboxWithStableRequestID()
}

func TestContiguousChainFaultRestartExpiredWaitingIsTerminal(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.deliverStrategyReady(1)
	f.expireWaitingDecision()
	f.restartAndRecoverWaiting(2)
	f.assertMissedWithoutOutbox()
}

func TestContiguousChainFaultResolverAndExpiryRaceHasOneDatabaseWinner(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.deliverStrategyReady(1)
	f.publishExactBoundary()
	f.raceResolverAndExpiry()
	f.assertOneTerminalWinnerAndAtMostOneOutbox()
}

func TestContiguousChainFaultConnectedWSCanIdentifyOneStaleLottery(t *testing.T) {
	base := time.Date(2026, time.August, 21, 0, 0, 0, 0, time.UTC)
	health := guaji.NewBoundaryHealth([]string{"tron_ffc_3s", "tron_ffc_6s"})
	health.Observe("tron_ffc_3s", "100", "101", base, 3*time.Second)
	health.Observe("tron_ffc_6s", "200", "201", base.Add(3*time.Second), 6*time.Second)

	stale := health.Stale(base.Add(4 * time.Second))
	if len(stale) != 1 || stale[0].LotteryCode != "tron_ffc_3s" {
		t.Fatalf("stale lotteries=%+v want only tron_ffc_3s", stale)
	}
	if snapshot := health.SnapshotAt("tron_ffc_6s", base.Add(4*time.Second)); snapshot.Stale {
		t.Fatalf("shared WS fresh lottery marked stale: %+v", snapshot)
	}
}

func TestContiguousChainFaultMissedNPlusOneNeverDispatchesNPlusTwo(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.deliverStrategyReady(1)
	f.publishBoundaryPastSource()
	f.deliverBoundaryReady(3)
	f.publishLaterBoundary()
	f.restartAndRecoverWaiting(2)
	f.dispatchBetReady(2)
	f.assertMissedWithoutOutbox()
	f.assertNoProviderSubmission()
}

func TestContiguousChainFaultManualRearmStartsNewRoundOneChain(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.makeMissedChain()
	f.manualRearm()
	f.assertNewChainStartsAtRoundOne()
}

func TestContiguousChainFaultProviderWrongPeriodBlocksWithoutResend(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.completePhaseOneAndResolve()
	f.dispatchWrongProviderPeriod(3)
	f.assertProviderFaultBlocked("accepted_wrong_period")
	f.assertOneProviderSubmission()
}

func TestContiguousChainFaultUnknownFingerprintStaysPendingWithoutResend(t *testing.T) {
	f := newProductionContiguousChainE2EFixture(t)
	f.completePhaseOneAndResolve()
	f.dispatchUnknownProviderResult(3)
	f.reconcileUnknownFingerprint(2)
	f.assertProviderFaultBlocked("provider_acceptance_pending_reconciliation")
	f.assertOneProviderSubmission()
}
