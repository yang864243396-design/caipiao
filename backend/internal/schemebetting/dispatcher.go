package schemebetting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

var (
	ErrStaleLease       = errors.New("scheme betting lease is stale")
	ErrInvalidCommand   = errors.New("scheme betting command is incomplete")
	ErrDispatcherConfig = errors.New("scheme betting dispatcher is not configured")
)

type LeasedCommand struct {
	ID                int64
	SchemeID          string
	TargetPeriod      string
	FrozenRequest     []byte
	FrozenRequestHash string
	CloseAt           time.Time
	SafeDeadline      time.Time
	Lease             LeaseFence
}

type FinishDispatch struct {
	CommandID         int64
	SchemeID          string
	LeaseOwner        string
	FencingToken      int64
	State             OutboxState
	Reason            string
	ProviderOrderID   string
	AcceptedPeriod    string
	ProviderAmount    float64
	ProviderAccountID int64
	ProviderCurrency  string
	ErrorDetail       string
	FinishedAt        time.Time
	BlocksChain       bool
}

type AttemptStart struct {
	Started    bool
	SafeWindow time.Duration
}

type DispatchStore interface {
	StartAttempt(ctx context.Context, command LeasedCommand, leaseDuration time.Duration) (AttemptStart, error)
	FinishAttempt(ctx context.Context, finish FinishDispatch) (bool, error)
}

type LeaseReleaseStore interface {
	ReleaseLease(ctx context.Context, command LeasedCommand, reason string, releasedAt time.Time) (bool, error)
}

type LeaseRenewStore interface {
	RenewLease(ctx context.Context, command LeasedCommand, leaseDuration time.Duration) (bool, error)
}

type DispatchLimiter interface {
	Allow(ctx context.Context, command LeasedCommand, now time.Time) (bool, error)
}

type SingleAttemptTransport interface {
	PlaceOnce(ctx context.Context, command LeasedCommand) (ProviderAcceptance, error)
}

type AcceptedFinalizer interface {
	FinalizeAccepted(ctx context.Context, outboxID int64) error
}

type PreSendFailureHandler interface {
	HandlePreSendFailure(ctx context.Context, outboxID int64) error
}

type definitelyNotSent interface {
	DefinitelyNotSent() bool
}

type Dispatcher struct {
	Store                  DispatchStore
	Transport              SingleAttemptTransport
	Finalizer              AcceptedFinalizer
	Limiter                DispatchLimiter
	Now                    func() time.Time
	LeaseDuration          time.Duration
	LeaseHeartbeatInterval time.Duration
	PreSendFailureHandler  PreSendFailureHandler
}

func (d Dispatcher) Dispatch(ctx context.Context, command LeasedCommand) error {
	if d.Store == nil || d.Transport == nil {
		return ErrDispatcherConfig
	}
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	if command.ID <= 0 || strings.TrimSpace(command.TargetPeriod) == "" || len(command.FrozenRequest) == 0 || command.FrozenRequestHash == "" || CanonicalJSONPayloadHash(command.FrozenRequest) != command.FrozenRequestHash {
		return ErrInvalidCommand
	}
	if strings.TrimSpace(command.Lease.Owner) == "" || command.Lease.Token <= 0 {
		return ErrStaleLease
	}
	if d.Limiter != nil {
		allowed, err := d.Limiter.Allow(ctx, command, now)
		if err != nil {
			return err
		}
		if !allowed {
			releaser, ok := d.Store.(LeaseReleaseStore)
			if !ok {
				return ErrDispatcherConfig
			}
			released, err := releaser.ReleaseLease(ctx, command, "dispatch_rate_limited", now)
			if err != nil {
				return err
			}
			if !released {
				return ErrStaleLease
			}
			return nil
		}
	}
	start, err := d.Store.StartAttempt(ctx, command, d.LeaseDuration)
	if err != nil {
		return err
	}
	if !start.Started || start.SafeWindow <= 0 {
		return ErrStaleLease
	}

	placeCtx, cancelPlace := context.WithTimeout(ctx, start.SafeWindow)
	defer cancelPlace()
	stopHeartbeat := d.startLeaseHeartbeat(placeCtx, cancelPlace, command)
	providerResult, placeErr := d.Transport.PlaceOnce(placeCtx, command)
	heartbeatErr := stopHeartbeat()
	observation := DispatchObservation{RequestStarted: true, Err: placeErr}
	if placeErr == nil {
		observation.Result = &providerResult
	} else {
		var safe definitelyNotSent
		if errors.As(placeErr, &safe) && safe.DefinitelyNotSent() {
			observation.DefinitelyNotSent = true
		}
	}
	resolution := ResolveDispatchOutcome(command.TargetPeriod, observation)
	if resolution.Reason == "provider_pre_send_failed" && d.PreSendFailureHandler == nil {
		resolution.BlocksChain = true
	}
	finishedAt := time.Now().UTC()
	if d.Now != nil {
		finishedAt = d.Now().UTC()
	}
	finished, err := d.Store.FinishAttempt(ctx, FinishDispatch{
		CommandID: command.ID, SchemeID: command.SchemeID, LeaseOwner: command.Lease.Owner,
		FencingToken: command.Lease.Token, State: resolution.State, Reason: resolution.Reason,
		ProviderOrderID: strings.TrimSpace(providerResult.OrderID), AcceptedPeriod: strings.TrimSpace(providerResult.PeriodNo),
		ProviderAmount: providerResult.Amount, ProviderAccountID: providerResult.AccountID, ProviderCurrency: providerResult.Currency,
		ErrorDetail: boundedDispatchErrors(placeErr, heartbeatErr), FinishedAt: finishedAt, BlocksChain: resolution.BlocksChain,
	})
	if err != nil {
		return err
	}
	if !finished {
		return ErrStaleLease
	}
	if resolution.State == OutboxAccepted && d.Finalizer != nil {
		if err := d.Finalizer.FinalizeAccepted(ctx, command.ID); err != nil {
			return err
		}
	}
	if resolution.Reason == "provider_pre_send_failed" && d.PreSendFailureHandler != nil {
		if err := d.PreSendFailureHandler.HandlePreSendFailure(ctx, command.ID); err != nil {
			return err
		}
	}
	return nil
}

func boundedDispatchError(err error) string {
	if err == nil {
		return ""
	}
	const maxRunes = 2000
	detail := strings.TrimSpace(err.Error())
	runes := []rune(detail)
	if len(runes) > maxRunes {
		detail = string(runes[:maxRunes])
	}
	return detail
}

func boundedDispatchErrors(errs ...error) string {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			parts = append(parts, strings.TrimSpace(err.Error()))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return boundedDispatchError(errors.New(strings.Join(parts, "; ")))
}

func (d Dispatcher) startLeaseHeartbeat(ctx context.Context, cancelProvider context.CancelFunc, command LeasedCommand) func() error {
	renewer, ok := d.Store.(LeaseRenewStore)
	if !ok || d.LeaseDuration <= 0 || d.LeaseHeartbeatInterval <= 0 {
		return func() error { return nil }
	}
	heartbeatCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	var heartbeatErr error
	go func() {
		defer close(done)
		ticker := time.NewTicker(d.LeaseHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				renewed, err := renewer.RenewLease(heartbeatCtx, command, d.LeaseDuration)
				if err != nil {
					heartbeatErr = fmt.Errorf("lease heartbeat failed owner=%s token=%d: %w", command.Lease.Owner, command.Lease.Token, err)
					slog.Warn("scheme betting dispatch lease heartbeat failed", "outboxId", command.ID, "schemeId", command.SchemeID, "owner", command.Lease.Owner, "token", command.Lease.Token, "err", err)
					cancelProvider()
					return
				}
				if !renewed {
					heartbeatErr = fmt.Errorf("lease heartbeat lost owner=%s token=%d", command.Lease.Owner, command.Lease.Token)
					slog.Warn("scheme betting dispatch lease heartbeat lost", "outboxId", command.ID, "schemeId", command.SchemeID, "owner", command.Lease.Owner, "token", command.Lease.Token)
					cancelProvider()
					return
				}
			}
		}
	}()
	return func() error {
		cancel()
		<-done
		return heartbeatErr
	}
}
