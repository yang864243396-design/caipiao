package schemeeventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"caipiao/backend/internal/schemebetting"
)

const ContiguousTargetExpanderDurable = "scheme-contiguous-target-expander"

type PeriodBoundary struct {
	LotteryCode  string    `json:"lotteryCode"`
	CurrentIssue string    `json:"currentIssue"`
	NextIssue    string    `json:"nextIssue"`
	ReceivedAt   time.Time `json:"receivedAt"`
	Generation   uint64    `json:"generation"`
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

func (bus *Bus) ConsumePeriodBoundaries(ctx context.Context, durable string, handler PeriodBoundaryHandler) error {
	if bus == nil || handler == nil || strings.TrimSpace(durable) == "" {
		return errors.New("period boundary consumer configuration is incomplete")
	}
	sub, err := bus.js.Subscribe(bus.PeriodBoundarySubject("*"), func(message *nats.Msg) {
		var event PeriodBoundary
		if err := json.Unmarshal(message.Data, &event); err != nil {
			bus.deadLetterAndTerminate(ctx, message, "invalid_json")
			return
		}
		if strings.TrimSpace(event.LotteryCode) == "" || strings.TrimSpace(event.CurrentIssue) == "" || strings.TrimSpace(event.NextIssue) == "" || event.Generation == 0 {
			bus.deadLetterAndTerminate(ctx, message, "invalid_period_boundary")
			return
		}
		if err := handler(ctx, event); err != nil {
			bus.retryOrDeadLetterAfter(ctx, message, "handler_error", 100*time.Millisecond)
			return
		}
		_ = message.Ack()
	}, nats.Durable(strings.TrimSpace(durable)), nats.ManualAck(), nats.AckExplicit(), nats.BindStream(bus.stream), nats.MaxAckPending(256))
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

func (bus *Bus) ConsumeContiguousTargetReady(ctx context.Context, shard uint32, durable string, handler ContiguousTargetReadyHandler) error {
	if bus == nil || handler == nil || strings.TrimSpace(durable) == "" {
		return errors.New("contiguous target-ready consumer configuration is incomplete")
	}
	sub, err := bus.js.QueueSubscribe(bus.ContiguousTargetReadySubject(shard), strings.TrimSpace(durable), func(message *nats.Msg) {
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
	if err != nil {
		return errors.New("contiguous target-ready consumer unavailable")
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
	return ctx.Err()
}
