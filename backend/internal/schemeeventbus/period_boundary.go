package schemeeventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemebetting"
)

const ContiguousTargetExpanderDurable = "scheme-contiguous-target-expander"

const contiguousTargetBoundaryExpansionPageSize int32 = 32

type PeriodBoundary struct {
	LotteryCode  string    `json:"lotteryCode"`
	CurrentIssue string    `json:"currentIssue"`
	NextIssue    string    `json:"nextIssue"`
	ReceivedAt   time.Time `json:"receivedAt"`
	// Generation is a deterministic boundary idempotency token. Consumers
	// must not infer ordering from its numeric value.
	Generation uint64 `json:"generation"`
}

func (event PeriodBoundary) MessageID() string {
	return fmt.Sprintf(
		"period-boundary:%s:%s:%s:%d",
		strings.TrimSpace(event.LotteryCode), strings.TrimSpace(event.CurrentIssue), strings.TrimSpace(event.NextIssue), event.Generation,
	)
}

type ContiguousTargetReady struct {
	DecisionID         int64  `json:"decisionId"`
	SchemeID           string `json:"schemeId"`
	LotteryCode        string `json:"lotteryCode"`
	SourcePeriod       string `json:"sourcePeriod"`
	BoundaryGeneration uint64 `json:"boundaryGeneration"`
}

func (event ContiguousTargetReady) MessageID() string {
	return fmt.Sprintf("contiguous-target:%d:%d", event.DecisionID, event.BoundaryGeneration)
}

func (event ContiguousTargetReady) Shard(shardCount uint32) uint32 {
	return schemebetting.ShardForScheme(event.SchemeID, shardCount)
}

func (bus *Bus) PeriodBoundarySubject(lotteryCode string) string {
	return fmt.Sprintf("%s.period.boundary.%s", bus.prefix, strings.TrimSpace(lotteryCode))
}

func (bus *Bus) ContiguousTargetReadySubject(shard uint32) string {
	return fmt.Sprintf("%s.target.ready.%d", bus.prefix, shard)
}

func (bus *Bus) PublishPeriodBoundary(ctx context.Context, event PeriodBoundary) error {
	if bus == nil || strings.TrimSpace(event.LotteryCode) == "" || strings.TrimSpace(event.CurrentIssue) == "" || strings.TrimSpace(event.NextIssue) == "" || event.Generation == 0 {
		return errors.New("period boundary event identity is incomplete")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return bus.publish(ctx, bus.PeriodBoundarySubject(event.LotteryCode), event.MessageID(), payload)
}

type PeriodBoundaryHandler func(context.Context, PeriodBoundary) error

type periodBoundaryDisposition uint8

const (
	periodBoundaryAck periodBoundaryDisposition = iota + 1
	periodBoundaryRetry
	periodBoundaryDeadLetter
)

func periodBoundaryDelivery(ctx context.Context, payload []byte, handler PeriodBoundaryHandler) (periodBoundaryDisposition, string) {
	var event PeriodBoundary
	if err := json.Unmarshal(payload, &event); err != nil {
		return periodBoundaryDeadLetter, "invalid_json"
	}
	if strings.TrimSpace(event.LotteryCode) == "" || strings.TrimSpace(event.CurrentIssue) == "" || strings.TrimSpace(event.NextIssue) == "" || event.Generation == 0 {
		return periodBoundaryDeadLetter, "invalid_period_boundary"
	}
	if err := handler(ctx, event); err != nil {
		return periodBoundaryRetry, "handler_error"
	}
	return periodBoundaryAck, ""
}

func (bus *Bus) ConsumePeriodBoundaries(ctx context.Context, durable string, handler PeriodBoundaryHandler) error {
	return bus.ConsumePeriodBoundariesReady(ctx, durable, handler, nil)
}

func establishReadySubscription(subscribe func() (*nats.Subscription, error), ready func()) (*nats.Subscription, error) {
	sub, err := subscribe()
	if err != nil {
		return nil, err
	}
	if ready != nil {
		ready()
	}
	return sub, nil
}

// ConsumePeriodBoundariesReady preserves ConsumePeriodBoundaries semantics and
// signals ready synchronously after JetStream has established the durable.
func (bus *Bus) ConsumePeriodBoundariesReady(ctx context.Context, durable string, handler PeriodBoundaryHandler, ready func()) error {
	if bus == nil || handler == nil || strings.TrimSpace(durable) == "" {
		return errors.New("period boundary consumer configuration is incomplete")
	}
	sub, err := establishReadySubscription(func() (*nats.Subscription, error) {
		return bus.js.Subscribe(bus.PeriodBoundarySubject("*"), func(message *nats.Msg) {
			disposition, failureClass := periodBoundaryDelivery(ctx, message.Data, handler)
			switch disposition {
			case periodBoundaryAck:
				_ = message.Ack()
			case periodBoundaryRetry:
				bus.retryOrDeadLetterAfter(ctx, message, failureClass, 100*time.Millisecond)
			case periodBoundaryDeadLetter:
				bus.deadLetterAndTerminate(ctx, message, failureClass)
			}
		}, nats.Durable(strings.TrimSpace(durable)), nats.ManualAck(), nats.AckExplicit(), nats.BindStream(bus.stream), nats.MaxAckPending(256))
	}, ready)
	if err != nil {
		return errors.New("period boundary consumer unavailable")
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
	return ctx.Err()
}

func (bus *Bus) PublishContiguousTargetReady(ctx context.Context, event ContiguousTargetReady, shardCount uint32) error {
	if bus == nil || event.DecisionID <= 0 || strings.TrimSpace(event.SchemeID) == "" || strings.TrimSpace(event.LotteryCode) == "" || strings.TrimSpace(event.SourcePeriod) == "" || event.BoundaryGeneration == 0 {
		return errors.New("contiguous target-ready event identity is incomplete")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return bus.publish(ctx, bus.ContiguousTargetReadySubject(event.Shard(shardCount)), event.MessageID(), payload)
}

type ContiguousTargetReadyHandler func(context.Context, ContiguousTargetReady) error

// AwaitingContiguousTargetSource is the bounded persisted-wait lookup used by
// the boundary expander. A boundary can arrive before phase one commits; a
// later durable delivery or recovery is therefore allowed to query again.
type AwaitingContiguousTargetSource interface {
	ListAwaitingContiguousTargets(context.Context, []string, []int32, int64, int32) ([]sqlcdb.AwaitingContiguousTargetRow, error)
}

// ContiguousTargetReadyPublisher isolates boundary expansion from the event
// transport while preserving the exact durable target-ready message identity.
type ContiguousTargetReadyPublisher interface {
	PublishContiguousTargetReady(context.Context, ContiguousTargetReady, uint32) error
}

// ExpandContiguousTargetBoundary turns one accepted period boundary into one
// bounded page of target-ready wakeups. It intentionally performs no target
// lookup or provider operation: the shard consumer owns resolution.
func ExpandContiguousTargetBoundary(
	ctx context.Context,
	event PeriodBoundary,
	source AwaitingContiguousTargetSource,
	publisher ContiguousTargetReadyPublisher,
	shards []int32,
	shardCount uint32,
) error {
	if source == nil || publisher == nil || strings.TrimSpace(event.LotteryCode) == "" || event.Generation == 0 || shardCount == 0 || len(shards) == 0 {
		return errors.New("contiguous target boundary expander configuration is incomplete")
	}
	rows, err := source.ListAwaitingContiguousTargets(ctx, []string{event.LotteryCode}, shards, 0, contiguousTargetBoundaryExpansionPageSize)
	if err != nil {
		return err
	}
	for _, row := range rows {
		ready := ContiguousTargetReady{
			DecisionID: row.DecisionID, SchemeID: row.SchemeID, LotteryCode: row.LotteryCode,
			SourcePeriod: row.SourcePeriodNo, BoundaryGeneration: event.Generation,
		}
		if err := publisher.PublishContiguousTargetReady(ctx, ready, shardCount); err != nil {
			return err
		}
	}
	return nil
}

func (bus *Bus) ConsumeContiguousTargetReady(ctx context.Context, shard uint32, durable string, handler ContiguousTargetReadyHandler) error {
	return bus.ConsumeContiguousTargetReadyWithReady(ctx, shard, durable, handler, nil)
}

// ConsumeContiguousTargetReadyWithReady preserves the existing consumer API
// and exposes only the successful-subscription readiness edge.
func (bus *Bus) ConsumeContiguousTargetReadyWithReady(ctx context.Context, shard uint32, durable string, handler ContiguousTargetReadyHandler, ready func()) error {
	if bus == nil || handler == nil || strings.TrimSpace(durable) == "" {
		return errors.New("contiguous target-ready consumer configuration is incomplete")
	}
	sub, err := establishReadySubscription(func() (*nats.Subscription, error) {
		return bus.js.QueueSubscribe(bus.ContiguousTargetReadySubject(shard), strings.TrimSpace(durable), func(message *nats.Msg) {
			var event ContiguousTargetReady
			if err := json.Unmarshal(message.Data, &event); err != nil {
				bus.deadLetterAndTerminate(ctx, message, "invalid_json")
				return
			}
			if event.DecisionID <= 0 || strings.TrimSpace(event.SchemeID) == "" || strings.TrimSpace(event.LotteryCode) == "" || strings.TrimSpace(event.SourcePeriod) == "" || event.BoundaryGeneration == 0 {
				bus.deadLetterAndTerminate(ctx, message, "invalid_contiguous_target_ready")
				return
			}
			if err := handler(ctx, event); err != nil {
				bus.retryOrDeadLetter(ctx, message, "handler_error")
				return
			}
			_ = message.Ack()
		}, nats.Durable(strings.TrimSpace(durable)), nats.ManualAck(), nats.AckExplicit(), nats.BindStream(bus.stream), nats.MaxAckPending(256))
	}, ready)
	if err != nil {
		return errors.New("contiguous target-ready consumer unavailable")
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
	return ctx.Err()
}
