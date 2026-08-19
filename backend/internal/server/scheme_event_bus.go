package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeeventbus"
	"time"
)

func newSchemeEventBus(cfg config.Config) (*schemeeventbus.Bus, error) {
	if !cfg.SchemeEventBusEnabled {
		return nil, nil
	}
	return schemeeventbus.New(schemeeventbus.Config{
		URL: cfg.NATSURL, Name: "scheme-event-bus", User: cfg.NATSUser, Password: cfg.NATSPassword,
		Token: cfg.NATSToken, CredentialsFile: cfg.NATSCredentialsFile, SubjectPrefix: cfg.NATSSubjectPrefix,
		StreamName: cfg.SchemeEventStream, Replicas: cfg.SchemeEventReplicas, MaxAge: cfg.SchemeEventMaxAge,
	})
}

type drawEventPublisher struct {
	pool       *db.Pool
	bus        *schemeeventbus.Bus
	leaseOwner string
	leaseFor   time.Duration
}

func (publisher *drawEventPublisher) NotifyStrategyDraw(ctx context.Context, lotteryCode, periodNo string) {
	if publisher == nil || publisher.pool == nil || publisher.bus == nil {
		return
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	periodNo = strings.TrimSpace(periodNo)
	var event schemeeventbus.DrawConfirmed
	var balls []byte
	var providerAt time.Time
	event.LotteryCode = lotteryCode
	event.PeriodNo = periodNo
	err := publisher.pool.QueryRow(ctx, `
SELECT balls, drawn_at, COALESCE(draw_hash, ''), COALESCE(provider_event_id, ''), source,
       received_at, COALESCE(confirmed_at, received_at)
FROM lottery_draws
WHERE lottery_code = $1 AND issue_no = $2`, lotteryCode, periodNo).Scan(
		&balls, &providerAt, &event.DrawHash, &event.ProviderEvent, &event.Source, &event.ReceivedAt, &event.ConfirmedAt,
	)
	if err != nil {
		slog.Error("load persisted draw for JetStream publish failed", "lottery", lotteryCode, "period", periodNo, "err", err)
		return
	}
	if err := json.Unmarshal(balls, &event.Balls); err != nil || len(event.Balls) == 0 {
		slog.Error("load persisted draw balls for JetStream publish failed", "lottery", lotteryCode, "period", periodNo, "err", err)
		return
	}
	event.ProviderAt = providerAt.UTC()
	now := time.Now().UTC()
	leaseFor := publisher.leaseFor
	if leaseFor <= 0 {
		leaseFor = 5 * time.Second
	}
	_, leader, err := sqlcdb.New(publisher.pool).AcquireSchemeBettingDrawLease(
		ctx, lotteryCode, publisher.leaseOwner, now, now.Add(leaseFor),
	)
	if err != nil {
		slog.Error("acquire draw publish leader lease failed", "lottery", lotteryCode, "period", periodNo, "err", err)
		return
	}
	if !leader {
		slog.Debug("draw publish skipped by non-leader", "lottery", lotteryCode, "period", periodNo)
		return
	}
	if err := publisher.bus.PublishDraw(ctx, event); err != nil {
		slog.Error("publish draw.confirmed failed; database recovery remains active", "lottery", lotteryCode, "period", periodNo, "err", err)
	}
}

func runSchemeDrawConsumer(ctx context.Context, bus *schemeeventbus.Bus, worker interface {
	ProcessDraw(context.Context, string, string) error
}) error {
	if bus == nil || worker == nil {
		return nil
	}
	err := bus.ConsumeDraws(ctx, "scheme-strategy-expander", func(messageContext context.Context, event schemeeventbus.DrawConfirmed) error {
		return worker.ProcessDraw(messageContext, event.LotteryCode, event.PeriodNo)
	})
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func closeSchemeEventBusAfterWorkers(bus *schemeeventbus.Bus) func() {
	return func() {
		if bus != nil {
			bus.Close()
		}
	}
}
