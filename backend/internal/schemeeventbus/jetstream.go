package schemeeventbus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"caipiao/backend/internal/schemebetting"
)

type Config struct {
	URL             string
	Name            string
	User            string
	Password        string
	Token           string
	CredentialsFile string
	SubjectPrefix   string
	StreamName      string
	Replicas        int
	MaxAge          time.Duration
}

type DrawConfirmed struct {
	LotteryCode   string    `json:"lotteryCode"`
	PeriodNo      string    `json:"periodNo"`
	Balls         []string  `json:"balls"`
	DrawHash      string    `json:"drawHash"`
	ProviderEvent string    `json:"providerEventId,omitempty"`
	Source        string    `json:"source"`
	ProviderAt    time.Time `json:"providerAt,omitempty"`
	ReceivedAt    time.Time `json:"receivedAt"`
	ConfirmedAt   time.Time `json:"confirmedAt"`
}

type StrategyReady struct {
	RecordID     int64  `json:"recordId"`
	SchemeID     string `json:"schemeId"`
	LotteryCode  string `json:"lotteryCode"`
	SourcePeriod string `json:"sourcePeriod"`
	StateVersion int64  `json:"stateVersion"`
	ShardNo      uint32 `json:"shardNo"`
}

type BetReady struct {
	OutboxID     int64     `json:"outboxId"`
	RequestID    string    `json:"requestId"`
	ShardNo      int32     `json:"shardNo"`
	SafeDeadline time.Time `json:"safeDeadline"`
}

type BetReconcile struct {
	OutboxID  int64  `json:"outboxId"`
	RequestID string `json:"requestId"`
	ShardNo   int32  `json:"shardNo"`
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
}

type DeadLetter struct {
	SourceSubject string    `json:"sourceSubject"`
	PayloadDigest string    `json:"payloadDigest"`
	DeliveryCount uint64    `json:"deliveryCount"`
	FailureClass  string    `json:"failureClass"`
	FailedAt      time.Time `json:"failedAt"`
}

type Bus struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	prefix string
	stream string
}

func New(config Config) (*Bus, error) {
	config.SubjectPrefix = strings.Trim(strings.TrimSpace(config.SubjectPrefix), ".")
	if config.SubjectPrefix == "" {
		config.SubjectPrefix = "caipiao"
	}
	if strings.TrimSpace(config.StreamName) == "" {
		config.StreamName = "SCHEME_EVENTS"
	}
	if config.Replicas <= 0 {
		config.Replicas = 1
	}
	if config.MaxAge <= 0 {
		config.MaxAge = 72 * time.Hour
	}
	options := []nats.Option{nats.MaxReconnects(-1), nats.RetryOnFailedConnect(true)}
	if config.Name != "" {
		options = append(options, nats.Name(config.Name))
	}
	switch {
	case config.CredentialsFile != "":
		options = append(options, nats.UserCredentials(config.CredentialsFile))
	case config.Token != "":
		options = append(options, nats.Token(config.Token))
	case config.User != "" || config.Password != "":
		options = append(options, nats.UserInfo(config.User, config.Password))
	}
	nc, err := nats.Connect(config.URL, options...)
	if err != nil {
		return nil, errors.New("scheme event bus connection failed")
	}
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(4096))
	if err != nil {
		nc.Close()
		return nil, errors.New("scheme event bus JetStream unavailable")
	}
	bus := &Bus{nc: nc, js: js, prefix: config.SubjectPrefix, stream: config.StreamName}
	_, err = js.AddStream(&nats.StreamConfig{
		Name: config.StreamName, Subjects: []string{bus.prefix + ".scheme.>"},
		Storage: nats.FileStorage, Retention: nats.LimitsPolicy, Replicas: config.Replicas,
		MaxAge: config.MaxAge, Discard: nats.DiscardOld,
	})
	if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		if _, infoErr := js.StreamInfo(config.StreamName); infoErr != nil {
			nc.Close()
			return nil, errors.New("scheme event stream unavailable")
		}
	}
	return bus, nil
}

func (bus *Bus) Close() {
	if bus != nil && bus.nc != nil {
		bus.nc.Close()
	}
}

func (bus *Bus) DrawSubject() string { return bus.prefix + ".scheme.draw.confirmed" }

func (bus *Bus) StrategySubject(shard uint32) string {
	return fmt.Sprintf("%s.scheme.strategy.ready.%d", bus.prefix, shard)
}

func (bus *Bus) BetReadySubject(shard int32) string {
	return fmt.Sprintf("%s.scheme.bet.ready.%d", bus.prefix, shard)
}

func (bus *Bus) BetReconcileSubject(shard int32) string {
	return fmt.Sprintf("%s.scheme.bet.reconcile.%d", bus.prefix, shard)
}

func (bus *Bus) DeadLetterSubject() string { return bus.prefix + ".scheme.deadletter" }

func strategyBacklogWithinCapacity(pending, ackPending, limit uint64) bool {
	return limit > 0 && pending+ackPending < limit
}

func (bus *Bus) CheckSchemeBettingBacklog(_ context.Context, shardNo int32) error {
	if bus == nil || shardNo < 0 {
		return errors.New("strategy event backlog is unavailable")
	}
	durable := fmt.Sprintf("scheme-strategy-shard-%d", shardNo)
	info, err := bus.js.ConsumerInfo(bus.stream, durable)
	if err != nil {
		return errors.New("strategy event consumer is unavailable")
	}
	const admissionLimit = uint64(256)
	if !strategyBacklogWithinCapacity(info.NumPending, uint64(info.NumAckPending), admissionLimit) {
		return fmt.Errorf("capacity_strategy_backlog:%d/%d", info.NumPending+uint64(info.NumAckPending), admissionLimit)
	}
	return nil
}

func (bus *Bus) PublishDraw(ctx context.Context, event DrawConfirmed) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return bus.publish(ctx, bus.DrawSubject(), drawMessageID(event), payload)
}

func (bus *Bus) PublishStrategyReady(ctx context.Context, event StrategyReady, shardCount uint32) error {
	event.ShardNo = schemebetting.ShardForScheme(event.SchemeID, shardCount)
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	id := schemebetting.CommandIdentity(event.SchemeID, event.SourcePeriod, event.LotteryCode, event.StateVersion)
	return bus.publish(ctx, bus.StrategySubject(event.ShardNo), id, payload)
}

func (bus *Bus) PublishBetReady(ctx context.Context, outboxID int64, requestID string, shardNo int32, safeDeadline time.Time) error {
	if bus == nil {
		return errors.New("scheme event bus is unavailable")
	}
	event := BetReady{
		OutboxID: outboxID, RequestID: strings.TrimSpace(requestID), ShardNo: shardNo,
		SafeDeadline: safeDeadline.UTC(),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return bus.publish(ctx, bus.BetReadySubject(shardNo), fmt.Sprintf("bet_ready_%d", outboxID), payload)
}

func (bus *Bus) PublishBetReconcile(ctx context.Context, outboxID int64, requestID string, shardNo int32, state, reason string) error {
	if bus == nil {
		return errors.New("scheme event bus is unavailable")
	}
	event := BetReconcile{
		OutboxID: outboxID, RequestID: strings.TrimSpace(requestID), ShardNo: shardNo,
		State: strings.TrimSpace(state), Reason: strings.TrimSpace(reason),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	messageID := fmt.Sprintf("bet_reconcile_%d_%s", outboxID, event.State)
	return bus.publish(ctx, bus.BetReconcileSubject(shardNo), messageID, payload)
}

func (bus *Bus) publish(ctx context.Context, subject, messageID string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	message := nats.NewMsg(subject)
	message.Data = payload
	message.Header.Set(nats.MsgIdHdr, messageID)
	_, err := bus.js.PublishMsg(message, nats.Context(ctx))
	if err != nil {
		return errors.New("scheme event publish failed")
	}
	return nil
}

type DrawHandler func(context.Context, DrawConfirmed) error

func (bus *Bus) ConsumeDraws(ctx context.Context, durable string, handler DrawHandler) error {
	if bus == nil || handler == nil || strings.TrimSpace(durable) == "" {
		return errors.New("scheme draw consumer configuration is incomplete")
	}
	sub, err := bus.js.Subscribe(bus.DrawSubject(), func(message *nats.Msg) {
		var event DrawConfirmed
		if err := json.Unmarshal(message.Data, &event); err != nil {
			bus.deadLetterAndTerminate(ctx, message, "invalid_json")
			return
		}
		if err := handler(ctx, event); err != nil {
			bus.retryOrDeadLetterAfter(ctx, message, "handler_error", 100*time.Millisecond)
			return
		}
		_ = message.Ack()
	}, nats.Durable(durable), nats.ManualAck(), nats.AckExplicit(), nats.BindStream(bus.stream), nats.MaxAckPending(1024))
	if err != nil {
		return errors.New("scheme draw consumer unavailable")
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
	return ctx.Err()
}

func drawMessageID(event DrawConfirmed) string {
	raw := strings.Join([]string{strings.TrimSpace(event.LotteryCode), strings.TrimSpace(event.PeriodNo), strings.TrimSpace(event.DrawHash)}, "\x00")
	sum := sha256.Sum256([]byte(raw))
	return "draw_" + hex.EncodeToString(sum[:16])
}

type StrategyHandler func(context.Context, StrategyReady) error

func (bus *Bus) ConsumeStrategyReady(ctx context.Context, shard uint32, durable string, handler StrategyHandler) error {
	if bus == nil || handler == nil || strings.TrimSpace(durable) == "" {
		return errors.New("scheme strategy consumer configuration is incomplete")
	}
	sub, err := bus.js.Subscribe(bus.StrategySubject(shard), func(message *nats.Msg) {
		var event StrategyReady
		if err := json.Unmarshal(message.Data, &event); err != nil {
			bus.deadLetterAndTerminate(ctx, message, "invalid_json")
			return
		}
		if event.ShardNo != shard {
			bus.deadLetterAndTerminate(ctx, message, "wrong_shard")
			return
		}
		if err := handler(ctx, event); err != nil {
			bus.retryOrDeadLetter(ctx, message, "handler_error")
			return
		}
		_ = message.Ack()
	}, nats.Durable(durable), nats.ManualAck(), nats.AckExplicit(), nats.BindStream(bus.stream), nats.MaxAckPending(256))
	if err != nil {
		return errors.New("scheme strategy consumer unavailable")
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
	return ctx.Err()
}

type BetReadyHandler func(context.Context, BetReady) error

func validateBetReady(event BetReady, shard int32) error {
	if event.OutboxID <= 0 || strings.TrimSpace(event.RequestID) == "" || event.SafeDeadline.IsZero() {
		return errors.New("scheme bet-ready event identity is incomplete")
	}
	if event.ShardNo != shard {
		return errors.New("scheme bet-ready event has wrong shard")
	}
	return nil
}

// ConsumeBetReady shares one durable queue per shard across dispatcher nodes.
// The event contains identity only; the consumer must load and fence the
// authoritative command from PostgreSQL before it can place a bet.
func (bus *Bus) ConsumeBetReady(ctx context.Context, shard int32, durable string, handler BetReadyHandler) error {
	if bus == nil || handler == nil || shard < 0 || strings.TrimSpace(durable) == "" {
		return errors.New("scheme bet-ready consumer configuration is incomplete")
	}
	durable = strings.TrimSpace(durable)
	sub, err := bus.js.QueueSubscribe(bus.BetReadySubject(shard), durable, func(message *nats.Msg) {
		var event BetReady
		if err := json.Unmarshal(message.Data, &event); err != nil {
			bus.deadLetterAndTerminate(ctx, message, "invalid_json")
			return
		}
		if err := validateBetReady(event, shard); err != nil {
			bus.deadLetterAndTerminate(ctx, message, "invalid_bet_ready")
			return
		}
		if err := handler(ctx, event); err != nil {
			bus.retryOrDeadLetter(ctx, message, "handler_error")
			return
		}
		_ = message.Ack()
	}, nats.Durable(durable), nats.ManualAck(), nats.AckExplicit(), nats.BindStream(bus.stream), nats.MaxAckPending(256))
	if err != nil {
		return errors.New("scheme bet-ready consumer unavailable")
	}
	defer sub.Unsubscribe()
	<-ctx.Done()
	return ctx.Err()
}

const maxConsumerDeliveries = uint64(10)

func newDeadLetter(message *nats.Msg, failureClass string, failedAt time.Time) DeadLetter {
	digest := sha256.Sum256(message.Data)
	deliveryCount := uint64(1)
	if metadata, err := message.Metadata(); err == nil && metadata != nil && metadata.NumDelivered > 0 {
		deliveryCount = metadata.NumDelivered
	}
	return DeadLetter{
		SourceSubject: message.Subject,
		PayloadDigest: hex.EncodeToString(digest[:]),
		DeliveryCount: deliveryCount,
		FailureClass:  strings.TrimSpace(failureClass),
		FailedAt:      failedAt.UTC(),
	}
}

func (bus *Bus) retryOrDeadLetter(ctx context.Context, message *nats.Msg, failureClass string) {
	bus.retryOrDeadLetterAfter(ctx, message, failureClass, 0)
}

func (bus *Bus) retryOrDeadLetterAfter(ctx context.Context, message *nats.Msg, failureClass string, delay time.Duration) {
	metadata, err := message.Metadata()
	if err == nil && metadata != nil && metadata.NumDelivered >= maxConsumerDeliveries {
		bus.deadLetterAndTerminate(ctx, message, failureClass)
		return
	}
	if delay > 0 {
		_ = message.NakWithDelay(delay)
		return
	}
	_ = message.Nak()
}

func (bus *Bus) deadLetterAndTerminate(ctx context.Context, message *nats.Msg, failureClass string) {
	event := newDeadLetter(message, failureClass, time.Now())
	payload, err := json.Marshal(event)
	if err == nil {
		messageID := fmt.Sprintf("dead_%s_%d", event.PayloadDigest[:32], event.DeliveryCount)
		_ = bus.publish(ctx, bus.DeadLetterSubject(), messageID, payload)
	}
	_ = message.Term()
}
