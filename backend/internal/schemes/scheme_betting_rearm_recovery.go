package schemes

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeeventbus"
)

type automaticRearmSource interface {
	GetAutomaticRearmCandidate(context.Context, int64) (sqlcdb.AutomaticRearmCandidate, bool, error)
}

type automaticRearmRecoverySource interface {
	automaticRearmSource
	ListAutomaticRearmCandidates(context.Context, []string, []int32, int32) ([]sqlcdb.AutomaticRearmCandidate, error)
}

func safeAutomaticRearmOutcome(state, reason string) bool {
	state = strings.TrimSpace(state)
	reason = strings.TrimSpace(reason)
	return (state == "rejected" && reason == "provider_pre_send_failed") ||
		(state == "expired" && (reason == "safe_deadline_elapsed" || reason == "dispatcher_lost_before_start_deadline_elapsed"))
}

func handleAutomaticRearmEvent(
	ctx context.Context,
	event schemeeventbus.BetReconcile,
	source automaticRearmSource,
	enabler formalEventEnabler,
) error {
	if source == nil || enabler == nil || !safeAutomaticRearmOutcome(event.State, event.Reason) {
		return nil
	}
	candidate, found, err := source.GetAutomaticRearmCandidate(ctx, event.OutboxID)
	if err != nil || !found {
		return err
	}
	if candidate.OutboxID != event.OutboxID || candidate.RequestID != event.RequestID ||
		candidate.ShardNo != event.ShardNo || candidate.State != event.State || candidate.Reason != event.Reason ||
		!safeAutomaticRearmOutcome(candidate.State, candidate.Reason) {
		return nil
	}
	err = enabler.RearmEventScheme(ctx, candidate.SchemeID, "system", "automatic recovery of proven unsent event chain")
	if errors.Is(err, errEventSchemeNotBlocked) {
		return nil
	}
	return err
}

func (w *Worker) HandleBetReconcile(ctx context.Context, event schemeeventbus.BetReconcile) error {
	if w == nil || w.q == nil {
		return errors.New("scheme automatic rearm worker unavailable")
	}
	return handleAutomaticRearmEvent(ctx, event, w.q, w)
}

func runAutomaticRearmBatch(
	ctx context.Context,
	source automaticRearmRecoverySource,
	enabler formalEventEnabler,
	lotteryCodes []string,
	shards []int32,
	batch, concurrency int,
) (int, error) {
	if source == nil || enabler == nil || len(lotteryCodes) == 0 || len(shards) == 0 {
		return 0, nil
	}
	if batch <= 0 {
		batch = 32
	}
	if concurrency <= 0 || concurrency > 8 {
		concurrency = 8
	}
	candidates, err := source.ListAutomaticRearmCandidates(ctx, lotteryCodes, shards, int32(batch))
	if err != nil {
		return 0, err
	}
	sem := make(chan struct{}, concurrency)
	var workers sync.WaitGroup
	var processed int
	var resultMu sync.Mutex
	for _, candidate := range candidates {
		select {
		case <-ctx.Done():
			workers.Wait()
			return processed, ctx.Err()
		case sem <- struct{}{}:
		}
		workers.Add(1)
		go func(candidate sqlcdb.AutomaticRearmCandidate) {
			defer workers.Done()
			defer func() { <-sem }()
			event := schemeeventbus.BetReconcile{
				OutboxID: candidate.OutboxID, RequestID: candidate.RequestID, ShardNo: candidate.ShardNo,
				State: candidate.State, Reason: candidate.Reason,
			}
			if err := handleAutomaticRearmEvent(ctx, event, source, enabler); err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.Warn("scheme automatic rearm deferred", "schemeId", candidate.SchemeID, "outboxId", candidate.OutboxID, "err", err)
				}
				return
			}
			resultMu.Lock()
			processed++
			resultMu.Unlock()
		}(candidate)
	}
	workers.Wait()
	return processed, nil
}

// RunAutomaticRearmRecovery is a bounded PostgreSQL safety net for lost or
// expired JetStream wakeups. It scans only a partial index of blocked schemes;
// normal recovery remains event-driven.
func (w *Worker) RunAutomaticRearmRecovery(
	ctx context.Context, lotteryCodes []string, shards []int32, interval time.Duration, batch, concurrency int,
) {
	if w == nil || w.q == nil || len(lotteryCodes) == 0 || len(shards) == 0 {
		return
	}
	if interval < time.Second {
		interval = time.Second
	}
	if batch <= 0 {
		batch = 32
	}
	if concurrency <= 0 || concurrency > 8 {
		concurrency = 8
	}
	run := func() {
		_, err := runAutomaticRearmBatch(ctx, w.q, w, lotteryCodes, shards, batch, concurrency)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.Warn("scheme automatic rearm recovery scan failed", "err", err)
			}
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
