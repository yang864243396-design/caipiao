package schemeeventbus

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestDrawMessageIDIsStableAndIncludesHash(t *testing.T) {
	event := DrawConfirmed{LotteryCode: "lottery", PeriodNo: "period", DrawHash: "hash-a", ConfirmedAt: time.Now()}
	a := drawMessageID(event)
	b := drawMessageID(event)
	if a == "" || a != b {
		t.Fatalf("message ids %q %q", a, b)
	}
	event.DrawHash = "hash-b"
	if a == drawMessageID(event) {
		t.Fatal("draw hash must participate in message id")
	}
}

func TestBetEventsContainNoFrozenRequestOrCredentialMaterial(t *testing.T) {
	ready, err := json.Marshal(BetReady{OutboxID: 7, RequestID: "request-7", ShardNo: 2, SafeDeadline: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	reconcile, err := json.Marshal(BetReconcile{OutboxID: 7, RequestID: "request-7", ShardNo: 2, State: "accepted"})
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range [][]byte{ready, reconcile} {
		for _, forbidden := range [][]byte{[]byte("frozenRequest"), []byte("token"), []byte("authorization"), []byte("password")} {
			if bytes.Contains(bytes.ToLower(payload), bytes.ToLower(forbidden)) {
				t.Fatalf("event leaked forbidden field %q: %s", forbidden, payload)
			}
		}
	}
}

func TestDeadLetterStoresDigestInsteadOfSensitivePayload(t *testing.T) {
	message := &nats.Msg{Subject: "caipiao.scheme.strategy.ready.1", Data: []byte(`{"token":"secret-value"}`)}
	event := newDeadLetter(message, "invalid_json", time.Unix(10, 0))
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if event.PayloadDigest == "" || event.DeliveryCount != 1 {
		t.Fatalf("unexpected dead letter: %+v", event)
	}
	if bytes.Contains(payload, []byte("secret-value")) || bytes.Contains(payload, message.Data) {
		t.Fatalf("dead letter leaked source payload: %s", payload)
	}
}

func TestDrawConfirmedCarriesCanonicalResultAndAllTimestamps(t *testing.T) {
	now := time.Now().UTC()
	payload, err := json.Marshal(DrawConfirmed{
		LotteryCode: "lottery", PeriodNo: "period", Balls: []string{"1", "2", "3"},
		DrawHash: "hash", Source: "ws", ProviderAt: now, ReceivedAt: now, ConfirmedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded DrawConfirmed
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Balls) != 3 || decoded.ProviderAt.IsZero() || decoded.ReceivedAt.IsZero() || decoded.ConfirmedAt.IsZero() {
		t.Fatalf("decoded draw event = %+v", decoded)
	}
}

func TestSubjectsUseFixedShardSuffix(t *testing.T) {
	bus := &Bus{prefix: "tenant"}
	if got := bus.DrawSubject(); got != "tenant.scheme.draw.confirmed" {
		t.Fatalf("draw subject=%q", got)
	}
	if got := bus.StrategySubject(17); got != "tenant.scheme.strategy.ready.17" {
		t.Fatalf("strategy subject=%q", got)
	}
}

func TestStrategyBacklogCapacityIncludesPendingAndUnacked(t *testing.T) {
	if !strategyBacklogWithinCapacity(200, 55, 256) {
		t.Fatal("255 outstanding messages should remain admissible")
	}
	if strategyBacklogWithinCapacity(200, 56, 256) {
		t.Fatal("capacity equality must reject admission")
	}
}

func TestValidateBetReadyRejectsWrongShardAndIncompleteIdentity(t *testing.T) {
	deadline := time.Now().Add(time.Second)
	for _, event := range []BetReady{
		{OutboxID: 0, RequestID: "request-7", ShardNo: 2, SafeDeadline: deadline},
		{OutboxID: 7, RequestID: "", ShardNo: 2, SafeDeadline: deadline},
		{OutboxID: 7, RequestID: "request-7", ShardNo: 3, SafeDeadline: deadline},
		{OutboxID: 7, RequestID: "request-7", ShardNo: 2},
	} {
		if err := validateBetReady(event, 2); err == nil {
			t.Fatalf("event should be rejected: %+v", event)
		}
	}
}

func TestValidateBetReadyAcceptsExactShardEvent(t *testing.T) {
	event := BetReady{OutboxID: 7, RequestID: "request-7", ShardNo: 2, SafeDeadline: time.Now().Add(time.Second)}
	if err := validateBetReady(event, 2); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBetReconcileRejectsWrongShardAndIncompleteIdentity(t *testing.T) {
	for _, event := range []BetReconcile{
		{OutboxID: 0, RequestID: "request-7", ShardNo: 2, State: "rejected", Reason: "provider_pre_send_failed"},
		{OutboxID: 7, RequestID: "", ShardNo: 2, State: "rejected", Reason: "provider_pre_send_failed"},
		{OutboxID: 7, RequestID: "request-7", ShardNo: 3, State: "rejected", Reason: "provider_pre_send_failed"},
		{OutboxID: 7, RequestID: "request-7", ShardNo: 2, State: "", Reason: "provider_pre_send_failed"},
	} {
		if err := validateBetReconcile(event, 2); err == nil {
			t.Fatalf("event should be rejected: %+v", event)
		}
	}
}

func TestValidateBetReconcileAllowsTerminalEventWithoutReason(t *testing.T) {
	event := BetReconcile{OutboxID: 7, RequestID: "request-7", ShardNo: 2, State: "accepted"}
	if err := validateBetReconcile(event, 2); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBetReconcileAcceptsExactShardEvent(t *testing.T) {
	event := BetReconcile{
		OutboxID: 7, RequestID: "request-7", ShardNo: 2,
		State: "rejected", Reason: "provider_pre_send_failed",
	}
	if err := validateBetReconcile(event, 2); err != nil {
		t.Fatal(err)
	}
}
