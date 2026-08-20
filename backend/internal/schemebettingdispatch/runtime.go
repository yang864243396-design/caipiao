package schemebettingdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebetting"
)

type Config struct {
	Mode          string
	Owner         string
	LotteryCodes  []string
	Shards        []int32
	Batch         int32
	Concurrency   int
	LeaseDuration time.Duration
	PollInterval  time.Duration
}

func (cfg Config) normalized() (Config, error) {
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode != "gray" && cfg.Mode != "production" {
		return Config{}, fmt.Errorf("scheme dispatcher mode %q is not formal", cfg.Mode)
	}
	cfg.Owner = strings.TrimSpace(cfg.Owner)
	if cfg.Owner == "" {
		return Config{}, errors.New("scheme dispatcher owner is required")
	}
	if len(cfg.LotteryCodes) == 0 {
		return Config{}, errors.New("scheme dispatcher requires an explicit lottery allowlist")
	}
	if len(cfg.Shards) == 0 {
		return Config{}, errors.New("scheme dispatcher requires explicit shards")
	}
	if cfg.Batch <= 0 {
		cfg.Batch = 32
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 8
	}
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = 5 * time.Second
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 100 * time.Millisecond
	}
	return cfg, nil
}

type FrozenGuajiRequest struct {
	Origin           string           `json:"origin,omitempty"`
	LocalOrderNo     string           `json:"localOrderNo,omitempty"`
	LocalBetPayload  json.RawMessage  `json:"localBetPayload,omitempty"`
	RequestID        string           `json:"requestId"`
	MemberAccount    string           `json:"memberAccount"`
	Request          guajibet.Request `json:"request"`
	SchemeName       string           `json:"schemeName"`
	LotteryLabel     string           `json:"lotteryLabel"`
	LotteryCategory  string           `json:"lotteryCategory"`
	DefinitionID     string           `json:"definitionId"`
	PlayType         string           `json:"playType"`
	RoundLabel       string           `json:"roundLabel"`
	BetContent       string           `json:"betContent"`
	BetUnits         int              `json:"betUnits"`
	RuleSnapshot     json.RawMessage  `json:"ruleSnapshot"`
	RuleVersion      int              `json:"ruleVersion"`
	RuleSnapshotHash string           `json:"ruleSnapshotHash"`
}

type Transport struct {
	Placer         guajibet.SingleAttemptPlacer
	PeriodVerifier providerPeriodVerifier
	Now            func() time.Time
}

type providerPeriodVerifier interface {
	VerifyOpenPeriodForMember(context.Context, string, string) (string, time.Time, error)
}

type definitelyNotSentError struct{ err error }

func (e definitelyNotSentError) Error() string           { return e.err.Error() }
func (e definitelyNotSentError) Unwrap() error           { return e.err }
func (e definitelyNotSentError) DefinitelyNotSent() bool { return true }

func (transport Transport) PlaceOnce(ctx context.Context, command schemebetting.LeasedCommand) (schemebetting.ProviderAcceptance, error) {
	if transport.Placer == nil || !transport.Placer.Enabled() {
		return schemebetting.ProviderAcceptance{}, definitelyNotSentError{err: errors.New("guaji single-attempt placer is disabled")}
	}
	var frozen FrozenGuajiRequest
	if err := json.Unmarshal(command.FrozenRequest, &frozen); err != nil {
		return schemebetting.ProviderAcceptance{}, definitelyNotSentError{err: fmt.Errorf("decode frozen request: %w", err)}
	}
	if strings.TrimSpace(frozen.RequestID) == "" || strings.TrimSpace(frozen.MemberAccount) == "" ||
		strings.TrimSpace(frozen.Request.LotteryCode) == "" || strings.TrimSpace(frozen.Request.IssueNo) == "" {
		return schemebetting.ProviderAcceptance{}, definitelyNotSentError{err: errors.New("frozen request identity is incomplete")}
	}
	if err := transport.verifyProviderTarget(ctx, command, frozen); err != nil {
		return schemebetting.ProviderAcceptance{}, definitelyNotSentError{err: err}
	}
	result, err := transport.Placer.PlaceRealBetOnce(ctx, frozen.MemberAccount, frozen.Request)
	if err != nil {
		if isDefinitiveProviderReject(err) {
			return schemebetting.ProviderAcceptance{}, definitelyNotSentError{err: err}
		}
		return schemebetting.ProviderAcceptance{}, err
	}
	return schemebetting.ProviderAcceptance{
		OrderID: strings.TrimSpace(result.ThirdPartyBetID), PeriodNo: strings.TrimSpace(result.Periods), Amount: result.Amount, AccountID: result.GuajiAccountID, Currency: result.Currency,
	}, nil
}

func (transport Transport) verifyProviderTarget(ctx context.Context, command schemebetting.LeasedCommand, frozen FrozenGuajiRequest) error {
	if transport.PeriodVerifier == nil {
		return errors.New("provider period verifier is not configured")
	}
	now := time.Now().UTC()
	if transport.Now != nil {
		now = transport.Now().UTC()
	}
	target := strings.TrimSpace(command.TargetPeriod)
	if target == "" || target != strings.TrimSpace(frozen.Request.IssueNo) {
		return errors.New("frozen target period does not match dispatch command")
	}
	safetyMargin := command.CloseAt.UTC().Sub(command.SafeDeadline.UTC())
	if command.CloseAt.IsZero() || command.SafeDeadline.IsZero() || safetyMargin <= 0 || !now.Before(command.SafeDeadline.UTC()) {
		return errors.New("dispatch target has no safe provider verification window")
	}
	verifyCtx, cancel := context.WithTimeout(ctx, command.SafeDeadline.UTC().Sub(now))
	defer cancel()
	period, closeAt, err := transport.PeriodVerifier.VerifyOpenPeriodForMember(
		verifyCtx, frozen.Request.LotteryCode, frozen.MemberAccount,
	)
	if err != nil {
		return fmt.Errorf("refresh provider period before dispatch: %w", err)
	}
	if strings.TrimSpace(period) != target {
		return fmt.Errorf("provider open period changed from %s to %s", target, strings.TrimSpace(period))
	}
	now = time.Now().UTC()
	if transport.Now != nil {
		now = transport.Now().UTC()
	}
	if closeAt.IsZero() || !closeAt.UTC().After(now.Add(safetyMargin)) || !now.Before(command.SafeDeadline.UTC()) {
		return errors.New("provider period is inside the dispatch safety margin")
	}
	return nil
}

func isDefinitiveProviderReject(err error) bool {
	return errors.Is(err, guajibet.ErrNoActiveAuth) ||
		errors.Is(err, guajibet.ErrTokenInvalid) ||
		errors.Is(err, guajibet.ErrInsufficient) ||
		errors.Is(err, guajibet.ErrPeriodClosed) ||
		errors.Is(err, guajibet.ErrZeroBets)
}

type Runtime struct {
	q          *sqlcdb.Queries
	dispatcher schemebetting.Dispatcher
	cfg        Config
	finalizer  acceptedRecovery
	resolver   unknownAcceptanceResolver
	events     betEventPublisher
}

type acceptedRecovery interface {
	RecoverAccepted(context.Context, int32) error
}

type unknownAcceptanceResolver interface {
	ResolveUnknown(context.Context, int64, string, string, schemebetting.UnknownResolution) error
}

type betEventPublisher interface {
	PublishBetReady(context.Context, int64, string, int32, time.Time) error
	PublishBetReconcile(context.Context, int64, string, int32, string, string) error
}

func New(q *sqlcdb.Queries, placer guajibet.SingleAttemptPlacer, cfg Config) (*Runtime, error) {
	if q == nil {
		return nil, errors.New("scheme dispatcher database is required")
	}
	normalized, err := cfg.normalized()
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{q: q, cfg: normalized}
	runtime.dispatcher = schemebetting.Dispatcher{Store: q, Transport: Transport{Placer: placer}}
	return runtime, nil
}

func (runtime *Runtime) SetAcceptanceFinalizer(finalizer *AcceptanceFinalizer) {
	if runtime == nil {
		return
	}
	runtime.finalizer = finalizer
	runtime.resolver = finalizer
	runtime.dispatcher.Finalizer = finalizer
}

func (runtime *Runtime) ResolveUnknownEventBet(ctx context.Context, outboxID int64, actor, reason string, resolution schemebetting.UnknownResolution) error {
	if runtime == nil || runtime.resolver == nil {
		return errors.New("scheme betting reconciliation is unavailable")
	}
	return runtime.resolver.ResolveUnknown(ctx, outboxID, actor, reason, resolution)
}

func (runtime *Runtime) SetPeriodVerifier(verifier providerPeriodVerifier) {
	if runtime == nil {
		return
	}
	transport, ok := runtime.dispatcher.Transport.(Transport)
	if !ok {
		return
	}
	transport.PeriodVerifier = verifier
	runtime.dispatcher.Transport = transport
}

func (runtime *Runtime) SetDispatchLimiter(limiter schemebetting.DispatchLimiter) {
	if runtime == nil {
		return
	}
	runtime.dispatcher.Limiter = limiter
}

func (runtime *Runtime) SetBetEventPublisher(publisher betEventPublisher) {
	if runtime != nil {
		runtime.events = publisher
	}
}

func (runtime *Runtime) Run(ctx context.Context) {
	if runtime == nil {
		return
	}
	ticker := time.NewTicker(runtime.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := runtime.runOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("scheme betting dispatcher cycle failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (runtime *Runtime) runOnce(ctx context.Context) error {
	now := time.Now().UTC()
	if err := runtime.publishPendingBetEvents(ctx, now); err != nil {
		slog.Warn("scheme betting event recovery publish failed", "err", err)
	}
	if _, err := runtime.q.RecoverExpiredUnstartedFormalOutbox(ctx, now, runtime.cfg.Batch); err != nil {
		return err
	}
	_, sweepErr := runtime.q.MarkAbandonedStartedDispatchUnknown(ctx, now, runtime.cfg.Batch)
	if err := runAcceptanceRecovery(ctx, sweepErr, runtime.finalizer, runtime.cfg.Batch); err != nil {
		return err
	}
	if _, err := runtime.q.ExpireDueFormalOutbox(ctx, now, runtime.cfg.Batch); err != nil {
		return err
	}
	sem := make(chan struct{}, runtime.cfg.Concurrency)
	var workers sync.WaitGroup
	for _, shard := range runtime.cfg.Shards {
		_, acquired, err := runtime.q.AcquireSchemeBettingShardLease(
			ctx, "dispatcher", shard, runtime.cfg.Owner, now, now.Add(2*runtime.cfg.LeaseDuration),
		)
		if err != nil {
			return err
		}
		if !acquired {
			continue
		}
		commands, err := runtime.q.LeaseFormalSchemeBetOutbox(ctx, sqlcdb.LeaseFormalOutboxParams{
			Mode: runtime.cfg.Mode, LeaseOwner: runtime.cfg.Owner, LotteryCodes: runtime.cfg.LotteryCodes, ShardNo: shard, Limit: runtime.cfg.Batch,
			Now: now, LeaseUntil: now.Add(runtime.cfg.LeaseDuration),
		})
		if err != nil {
			return err
		}
		for _, command := range commands {
			select {
			case <-ctx.Done():
				workers.Wait()
				return ctx.Err()
			case sem <- struct{}{}:
			}
			workers.Add(1)
			go func(command schemebetting.LeasedCommand) {
				defer workers.Done()
				defer func() { <-sem }()
				if err := runtime.dispatcher.Dispatch(ctx, command); err != nil && !errors.Is(err, schemebetting.ErrStaleLease) {
					slog.Error("scheme betting dispatch failed", "outboxId", command.ID, "schemeId", command.SchemeID, "err", err)
				}
			}(command)
		}
	}
	workers.Wait()
	if err := runtime.publishPendingBetEvents(ctx, time.Now().UTC()); err != nil {
		slog.Warn("scheme betting reconcile publish failed", "err", err)
	}
	return nil
}

func (runtime *Runtime) publishPendingBetEvents(ctx context.Context, now time.Time) error {
	if runtime == nil || runtime.events == nil {
		return nil
	}
	ready, err := runtime.q.ListUnpublishedBetReady(ctx, runtime.cfg.Batch)
	if err != nil {
		return err
	}
	for _, event := range ready {
		if err := runtime.events.PublishBetReady(ctx, event.OutboxID, event.RequestID, event.ShardNo, event.SafeDeadline); err != nil {
			_ = runtime.q.MarkBetReadyPublishFailed(ctx, event.OutboxID)
			continue
		}
		if err := runtime.q.MarkBetReadyPublished(ctx, event.OutboxID, now); err != nil {
			return err
		}
	}
	reconciliations, err := runtime.q.ListUnpublishedBetReconcile(ctx, runtime.cfg.Batch)
	if err != nil {
		return err
	}
	for _, event := range reconciliations {
		if err := runtime.events.PublishBetReconcile(ctx, event.OutboxID, event.RequestID, event.ShardNo, event.State, event.Reason); err != nil {
			_ = runtime.q.MarkBetReconcilePublishFailed(ctx, event.OutboxID, event.State)
			continue
		}
		if err := runtime.q.MarkBetReconcilePublished(ctx, event.OutboxID, event.State, now); err != nil {
			return err
		}
	}
	return nil
}

func runAcceptanceRecovery(ctx context.Context, sweepErr error, recovery acceptedRecovery, limit int32) error {
	if sweepErr != nil {
		return sweepErr
	}
	if recovery == nil {
		return nil
	}
	return recovery.RecoverAccepted(ctx, limit)
}
