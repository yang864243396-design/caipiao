package schemebetting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

var (
	ErrStaleLease       = errors.New("scheme betting lease is stale")
	ErrDispatchDeferred = errors.New("scheme betting dispatch deferred")
	ErrInvalidCommand   = errors.New("scheme betting command is incomplete")
	ErrDispatcherConfig = errors.New("scheme betting dispatcher is not configured")
)

const (
	leaseHeartbeatShutdownTimeout = 100 * time.Millisecond
	finishFailureRecordTimeout    = 2 * time.Second
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

type progressSingleAttemptTransport interface {
	PlaceOnceWithProgress(
		ctx context.Context,
		command LeasedCommand,
		report func(stage string, requestWritten, writeKnown bool),
	) (ProviderAcceptance, error)
}

type dispatchProgress struct {
	mu             sync.Mutex
	stage          string
	requestWritten bool
	writeKnown     bool
}

func (progress *dispatchProgress) report(stage string, requestWritten, writeKnown bool) {
	progress.mu.Lock()
	progress.stage = strings.TrimSpace(stage)
	progress.requestWritten = requestWritten
	progress.writeKnown = writeKnown
	progress.mu.Unlock()
}

func (progress *dispatchProgress) deadlineError(cause error) error {
	progress.mu.Lock()
	defer progress.mu.Unlock()
	return dispatchDeadlineError{
		stage: progress.stage, requestWritten: progress.requestWritten,
		writeKnown: progress.writeKnown, cause: cause,
	}
}

type dispatchDeadlineError struct {
	stage          string
	requestWritten bool
	writeKnown     bool
	cause          error
}

func (err dispatchDeadlineError) Error() string {
	stage := strings.TrimSpace(err.stage)
	if stage == "" {
		stage = "unknown"
	}
	return fmt.Sprintf("provider placement exceeded safe deadline stage=%s request_written=%t write_known=%t: %v",
		stage, err.requestWritten, err.writeKnown, err.cause)
}

func (err dispatchDeadlineError) Unwrap() error { return err.cause }

func (err dispatchDeadlineError) DefinitelyNotSent() bool {
	return err.writeKnown && !err.requestWritten
}

type placeCallResult struct {
	acceptance ProviderAcceptance
	err        error
}

type AcceptedFinalizer interface {
	FinalizeAccepted(ctx context.Context, outboxID int64) error
}

type PreSendFailureHandler interface {
	HandlePreSendFailure(ctx context.Context, outboxID int64) error
}

type FinishAttemptFailureRecorder interface {
	RecordFinishAttemptFailure(ctx context.Context, command LeasedCommand, detail string) (bool, error)
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
			return ErrDispatchDeferred
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
	stopHeartbeat := d.startLeaseHeartbeat(ctx, cancelPlace, command)
	progress := &dispatchProgress{}
	placeDone := make(chan placeCallResult, 1)
	go func() {
		var result ProviderAcceptance
		var err error
		if observed, ok := d.Transport.(progressSingleAttemptTransport); ok {
			result, err = observed.PlaceOnceWithProgress(placeCtx, command, progress.report)
		} else {
			result, err = d.Transport.PlaceOnce(placeCtx, command)
		}
		placeDone <- placeCallResult{acceptance: result, err: err}
	}()
	var providerResult ProviderAcceptance
	var placeErr error
	select {
	case call := <-placeDone:
		providerResult, placeErr = call.acceptance, call.err
	case <-placeCtx.Done():
		placeErr = progress.deadlineError(placeCtx.Err())
	}
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
		d.recordFinishAttemptFailure(command, err)
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

func (d Dispatcher) recordFinishAttemptFailure(command LeasedCommand, finishErr error) {
	recorder, ok := d.Store.(FinishAttemptFailureRecorder)
	if !ok || finishErr == nil {
		return
	}
	detail := boundedDispatchError(fmt.Errorf("finish_attempt_failed: %w", finishErr))
	ctx, cancel := context.WithTimeout(context.Background(), finishFailureRecordTimeout)
	defer cancel()
	recorded, err := recorder.RecordFinishAttemptFailure(ctx, command, detail)
	if err != nil {
		slog.Error("scheme betting finish failure evidence persistence failed", "outboxId", command.ID, "schemeId", command.SchemeID, "err", err)
		return
	}
	if !recorded {
		slog.Warn("scheme betting finish failure evidence rejected by fencing", "outboxId", command.ID, "schemeId", command.SchemeID, "owner", command.Lease.Owner, "token", command.Lease.Token)
	}
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
	done := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(d.LeaseHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				done <- nil
				return
			case <-ticker.C:
				renewed, err := renewer.RenewLease(heartbeatCtx, command, d.LeaseDuration)
				if err != nil {
					heartbeatErr := fmt.Errorf("lease heartbeat failed owner=%s token=%d: %w", command.Lease.Owner, command.Lease.Token, err)
					slog.Warn("scheme betting dispatch lease heartbeat failed", "outboxId", command.ID, "schemeId", command.SchemeID, "owner", command.Lease.Owner, "token", command.Lease.Token, "err", err)
					cancelProvider()
					done <- heartbeatErr
					return
				}
				if !renewed {
					heartbeatErr := fmt.Errorf("lease heartbeat lost owner=%s token=%d", command.Lease.Owner, command.Lease.Token)
					slog.Warn("scheme betting dispatch lease heartbeat lost", "outboxId", command.ID, "schemeId", command.SchemeID, "owner", command.Lease.Owner, "token", command.Lease.Token)
					cancelProvider()
					done <- heartbeatErr
					return
				}
			}
		}
	}()
	return func() error {
		cancel()
		timer := time.NewTimer(leaseHeartbeatShutdownTimeout)
		defer timer.Stop()
		select {
		case heartbeatErr := <-done:
			return heartbeatErr
		case <-timer.C:
			return fmt.Errorf("lease heartbeat shutdown timed out after %s owner=%s token=%d",
				leaseHeartbeatShutdownTimeout, command.Lease.Owner, command.Lease.Token)
		}
	}
}
