package schemebetting

import (
	"context"
	"errors"
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

type DispatchStore interface {
	StartAttempt(ctx context.Context, command LeasedCommand, startedAt time.Time) (bool, error)
	FinishAttempt(ctx context.Context, finish FinishDispatch) (bool, error)
}

type LeaseReleaseStore interface {
	ReleaseLease(ctx context.Context, command LeasedCommand, reason string, releasedAt time.Time) (bool, error)
}

type LeaseRenewStore interface {
	RenewLease(ctx context.Context, command LeasedCommand, renewedAt, leaseUntil time.Time) (bool, error)
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
}

func (d Dispatcher) Dispatch(ctx context.Context, command LeasedCommand) error {
	if d.Store == nil || d.Transport == nil {
		return ErrDispatcherConfig
	}
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	if !now.Before(command.SafeDeadline.UTC()) {
		_, err := d.Store.FinishAttempt(ctx, FinishDispatch{
			CommandID: command.ID, SchemeID: command.SchemeID, LeaseOwner: command.Lease.Owner,
			FencingToken: command.Lease.Token, State: OutboxExpired, Reason: "safe_deadline_elapsed",
			FinishedAt: now, BlocksChain: true,
		})
		return err
	}
	if !command.Lease.CanCommit(command.Lease.Owner, command.Lease.Token, now) {
		return ErrStaleLease
	}
	if command.ID <= 0 || strings.TrimSpace(command.TargetPeriod) == "" || len(command.FrozenRequest) == 0 || command.FrozenRequestHash == "" || CanonicalJSONPayloadHash(command.FrozenRequest) != command.FrozenRequestHash {
		return ErrInvalidCommand
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
	started, err := d.Store.StartAttempt(ctx, command, now)
	if err != nil {
		return err
	}
	if !started {
		return ErrStaleLease
	}

	stopHeartbeat := d.startLeaseHeartbeat(ctx, command)
	providerResult, placeErr := d.Transport.PlaceOnce(ctx, command)
	stopHeartbeat()
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
	finishedAt := time.Now().UTC()
	if d.Now != nil {
		finishedAt = d.Now().UTC()
	}
	finished, err := d.Store.FinishAttempt(ctx, FinishDispatch{
		CommandID: command.ID, SchemeID: command.SchemeID, LeaseOwner: command.Lease.Owner,
		FencingToken: command.Lease.Token, State: resolution.State, Reason: resolution.Reason,
		ProviderOrderID: strings.TrimSpace(providerResult.OrderID), AcceptedPeriod: strings.TrimSpace(providerResult.PeriodNo),
		ProviderAmount: providerResult.Amount, ProviderAccountID: providerResult.AccountID, ProviderCurrency: providerResult.Currency,
		ErrorDetail: boundedDispatchError(placeErr), FinishedAt: finishedAt, BlocksChain: resolution.BlocksChain,
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

func (d Dispatcher) startLeaseHeartbeat(ctx context.Context, command LeasedCommand) func() {
	renewer, ok := d.Store.(LeaseRenewStore)
	if !ok || d.LeaseDuration <= 0 || d.LeaseHeartbeatInterval <= 0 {
		return func() {}
	}
	heartbeatCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(d.LeaseHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case renewedAt := <-ticker.C:
				renewedAt = renewedAt.UTC()
				renewed, err := renewer.RenewLease(heartbeatCtx, command, renewedAt, renewedAt.Add(d.LeaseDuration))
				if err != nil {
					slog.Warn("scheme betting dispatch lease heartbeat failed", "outboxId", command.ID, "schemeId", command.SchemeID, "err", err)
					continue
				}
				if !renewed {
					slog.Warn("scheme betting dispatch lease heartbeat lost", "outboxId", command.ID, "schemeId", command.SchemeID)
					return
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}
