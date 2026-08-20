package schemebetting

import (
	"strings"
	"time"
)

const (
	OutboxAcceptedWrongPeriod       OutboxState = "accepted_wrong_period"
	OutboxExternalAcceptanceUnknown OutboxState = "external_acceptance_unknown"
)

type ProviderAcceptance struct {
	OrderID   string
	PeriodNo  string
	Amount    float64
	AccountID int64
	Currency  string
}

type DispatchObservation struct {
	RequestStarted    bool
	DefinitelyNotSent bool
	Result            *ProviderAcceptance
	Err               error
}

type DispatchResolution struct {
	State       OutboxState
	Reason      string
	BlocksChain bool
	Retryable   bool
}

func ResolveDispatchOutcome(targetPeriod string, observation DispatchObservation) DispatchResolution {
	targetPeriod = strings.TrimSpace(targetPeriod)
	if observation.Result != nil && observation.Err == nil {
		orderID := strings.TrimSpace(observation.Result.OrderID)
		periodNo := strings.TrimSpace(observation.Result.PeriodNo)
		if orderID == "" || periodNo == "" {
			return DispatchResolution{State: OutboxSentUnknown, Reason: "provider_acceptance_missing_identity", BlocksChain: true}
		}
		if observation.Result.Amount <= 0 || observation.Result.AccountID <= 0 || strings.TrimSpace(observation.Result.Currency) == "" {
			return DispatchResolution{State: OutboxSentUnknown, Reason: "provider_acceptance_missing_financial_identity", BlocksChain: true}
		}
		if periodNo != targetPeriod {
			return DispatchResolution{State: OutboxAcceptedWrongPeriod, Reason: "accepted_wrong_period", BlocksChain: true}
		}
		return DispatchResolution{State: OutboxAccepted, Reason: "accepted"}
	}
	if observation.DefinitelyNotSent {
		return DispatchResolution{State: OutboxRejected, Reason: "provider_pre_send_failed"}
	}
	return DispatchResolution{State: OutboxSentUnknown, Reason: "provider_acceptance_pending_reconciliation", BlocksChain: true}
}

type LeaseFence struct {
	Owner string
	Token int64
	Until time.Time
}

func (lease LeaseFence) CanCommit(owner string, token int64, now time.Time) bool {
	return strings.TrimSpace(owner) != "" && owner == lease.Owner && token == lease.Token && now.UTC().Before(lease.Until.UTC())
}

type ChainState string

const (
	ChainStateIdle                 ChainState = "idle"
	ChainStateActive               ChainState = "active"
	ChainStateBlockedRequiresRearm ChainState = "blocked_requires_rearm"
)

func ApplyDispatchToChain(current ChainState, outcome OutboxState) ChainState {
	if current == ChainStateBlockedRequiresRearm {
		return current
	}
	switch outcome {
	case OutboxSentUnknown, OutboxRejected, OutboxExpired, OutboxCancelled, OutboxAcceptedWrongPeriod, OutboxExternalAcceptanceUnknown:
		return ChainStateBlockedRequiresRearm
	default:
		return current
	}
}

func RearmChain(ChainState) ChainState {
	return ChainStateActive
}
