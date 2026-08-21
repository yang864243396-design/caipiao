package schemeeventbus

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemebetting"
)

func TestPeriodBoundaryMessageIDIncludesLotteryAndGeneration(t *testing.T) {
	event := PeriodBoundary{LotteryCode: "tron_ffc_6s", CurrentIssue: "100", NextIssue: "101", Generation: 7}
	if got := event.MessageID(); got != "period-boundary:tron_ffc_6s:100:101:7" {
		t.Fatalf("message ID = %q", got)
	}
}

func TestContiguousTargetReadyRoutesBySchemeShard(t *testing.T) {
	event := ContiguousTargetReady{
		DecisionID: 9, SchemeID: "inst-9", LotteryCode: "tron_ffc_6s", SourcePeriod: "100", BoundaryGeneration: 7,
	}
	if got, want := event.Shard(64), schemebetting.ShardForScheme("inst-9", 64); got != want {
		t.Fatalf("shard = %d, want %d", got, want)
	}
	if got := event.MessageID(); got != "contiguous-target:9:7" {
		t.Fatalf("message ID = %q", got)
	}
}

func TestPeriodBoundaryDeliveryDispositionAcksOnlyHandlerSuccess(t *testing.T) {
	valid, err := json.Marshal(PeriodBoundary{
		LotteryCode: "tron_ffc_6s", CurrentIssue: "100", NextIssue: "101",
		ReceivedAt: time.Unix(100, 0).UTC(), Generation: 7,
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		payload      []byte
		handler      PeriodBoundaryHandler
		want         periodBoundaryDisposition
		failureClass string
	}{
		{
			name: "handler success", payload: valid,
			handler: func(context.Context, PeriodBoundary) error { return nil },
			want:    periodBoundaryAck,
		},
		{
			name: "handler failure", payload: valid,
			handler: func(context.Context, PeriodBoundary) error { return errors.New("temporary") },
			want:    periodBoundaryRetry, failureClass: "handler_error",
		},
		{
			name: "malformed", payload: []byte("{"),
			handler: func(context.Context, PeriodBoundary) error {
				t.Fatal("handler called for malformed event")
				return nil
			},
			want: periodBoundaryDeadLetter, failureClass: "invalid_json",
		},
		{
			name: "incomplete", payload: []byte(`{"lotteryCode":"tron_ffc_6s","generation":7}`),
			handler: func(context.Context, PeriodBoundary) error {
				t.Fatal("handler called for incomplete event")
				return nil
			},
			want: periodBoundaryDeadLetter, failureClass: "invalid_period_boundary",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, failureClass := periodBoundaryDelivery(context.Background(), test.payload, test.handler)
			if got != test.want || failureClass != test.failureClass {
				t.Fatalf("disposition = %v/%q, want %v/%q", got, failureClass, test.want, test.failureClass)
			}
		})
	}
}

func TestEstablishReadySubscriptionSignalsOnlyAfterSuccessfulSubscribe(t *testing.T) {
	var order []string
	ready := func() { order = append(order, "ready") }
	subscribe := func() (*nats.Subscription, error) {
		order = append(order, "subscribed")
		return &nats.Subscription{}, nil
	}

	if _, err := establishReadySubscription(subscribe, ready); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "subscribed" || order[1] != "ready" {
		t.Fatalf("subscription order=%v, want subscribed then ready", order)
	}

	called := false
	if _, err := establishReadySubscription(func() (*nats.Subscription, error) {
		return nil, errors.New("subscribe failed")
	}, func() { called = true }); err == nil {
		t.Fatal("failed subscription returned nil error")
	}
	if called {
		t.Fatal("failed subscription signaled readiness")
	}
}

// Removing the production boundary expander must fail this test: a boundary
// which arrives before phase one has no target-ready row to publish, while a
// later durable delivery must expand the newly committed wait with the same
// stable boundary identity.
func TestExpandContiguousTargetBoundaryDefersUntilAwaitingDecisionExists(t *testing.T) {
	source := &boundaryExpansionSource{}
	publisher := &boundaryExpansionPublisher{}
	event := PeriodBoundary{LotteryCode: "tron_ffc_6s", CurrentIssue: "100", NextIssue: "101", Generation: 9}

	if err := ExpandContiguousTargetBoundary(context.Background(), event, source, publisher, []int32{3}, 64); err != nil {
		t.Fatal(err)
	}
	if got := publisher.Count(); got != 0 {
		t.Fatalf("pre-phase boundary target-ready events=%d want 0", got)
	}

	source.SetRows([]sqlcdb.AwaitingContiguousTargetRow{{
		DecisionID: 41, SchemeID: "inst-41", LotteryCode: "tron_ffc_6s", SourcePeriodNo: "100", ShardNo: 3,
	}})
	if err := ExpandContiguousTargetBoundary(context.Background(), event, source, publisher, []int32{3}, 64); err != nil {
		t.Fatal(err)
	}
	if err := ExpandContiguousTargetBoundary(context.Background(), event, source, publisher, []int32{3}, 64); err != nil {
		t.Fatal(err)
	}
	events := publisher.Events()
	if len(events) != 2 || events[0].DecisionID != 41 || events[1].DecisionID != 41 || events[0].MessageID() != events[1].MessageID() {
		t.Fatalf("post-commit target-ready redelivery=%+v", events)
	}
}

type boundaryExpansionSource struct {
	mu   sync.Mutex
	rows []sqlcdb.AwaitingContiguousTargetRow
}

func (source *boundaryExpansionSource) SetRows(rows []sqlcdb.AwaitingContiguousTargetRow) {
	source.mu.Lock()
	defer source.mu.Unlock()
	source.rows = append([]sqlcdb.AwaitingContiguousTargetRow(nil), rows...)
}

func (source *boundaryExpansionSource) ListAwaitingContiguousTargets(
	_ context.Context, _ []string, _ []int32, _ int64, _ int32,
) ([]sqlcdb.AwaitingContiguousTargetRow, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	return append([]sqlcdb.AwaitingContiguousTargetRow(nil), source.rows...), nil
}

type boundaryExpansionPublisher struct {
	mu     sync.Mutex
	events []ContiguousTargetReady
}

func (publisher *boundaryExpansionPublisher) PublishContiguousTargetReady(
	_ context.Context, event ContiguousTargetReady, _ uint32,
) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	publisher.events = append(publisher.events, event)
	return nil
}

func (publisher *boundaryExpansionPublisher) Count() int {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	return len(publisher.events)
}

func (publisher *boundaryExpansionPublisher) Events() []ContiguousTargetReady {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	return append([]ContiguousTargetReady(nil), publisher.events...)
}
