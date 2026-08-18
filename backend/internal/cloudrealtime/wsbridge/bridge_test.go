package wsbridge

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"caipiao/backend/internal/cloudrealtime"
	"caipiao/backend/internal/realtimebus"
	"caipiao/backend/internal/schemes"
	"caipiao/backend/internal/ws"
)

func TestBridgeSubscribesExactMemberSubjectsAndEmitsSnapshotsBeforeLegacyHints(t *testing.T) {
	bus := newFakeBus()
	bridge := New(bus, "caipiao")
	var events []ws.Envelope
	cancel, err := bridge.SubscribeMember(7, func(env ws.Envelope) { events = append(events, env) })
	if err != nil {
		t.Fatal(err)
	}
	wantSubjects := []string{"caipiao.client.7.scheme", "caipiao.client.7.cloud_stats"}
	if got := bus.subjects(); !reflect.DeepEqual(got, wantSubjects) {
		t.Fatalf("subjects=%v want=%v", got, wantSubjects)
	}

	schemeMessage := cloudrealtime.SchemeSnapshotMessage{
		SchemaVersion: cloudrealtime.SchemaVersion,
		GeneratedAt:   "2026-08-18T00:00:00Z",
		Items: []schemes.Instance{
			{ID: "inst-a", Status: "running"},
			{ID: "inst-b", Status: "paused"},
		},
		RemovedIDs: []string{"inst-gone"},
	}
	bus.deliverJSON(t, wantSubjects[0], wantSubjects[0], schemeMessage)
	if len(events) != 4 {
		t.Fatalf("scheme events=%d want=4", len(events))
	}
	if events[0].Name != ws.NameSchemeInstancesSnapshot || events[0].Topic != ws.TopicClientSchemeInstance {
		t.Fatalf("snapshot envelope=%#v", events[0])
	}
	if got, ok := events[0].Payload.(cloudrealtime.SchemeSnapshotMessage); !ok || !reflect.DeepEqual(got, schemeMessage) {
		t.Fatalf("snapshot payload=%#v", events[0].Payload)
	}
	for index, event := range events[1:] {
		if event.Name != ws.NameSchemeInstanceUpdated || event.Topic != ws.TopicClientSchemeInstance {
			t.Fatalf("legacy event %d=%#v", index, event)
		}
		assertLegacyHintFields(t, event.Payload)
	}

	statsMessage := cloudrealtime.StatsSnapshotMessage{
		SchemaVersion: cloudrealtime.SchemaVersion,
		GeneratedAt:   "2026-08-18T00:00:01Z",
		Stats: schemes.CloudCenterStats{
			Formal: schemes.CloudCenterChannelStats{TotalTurnover: 12.5},
		},
	}
	bus.deliverJSON(t, wantSubjects[1], wantSubjects[1], statsMessage)
	if len(events) != 5 {
		t.Fatalf("events=%d want=5", len(events))
	}
	if events[4].Name != ws.NameCloudStatsSnapshot || events[4].Topic != ws.TopicClientCloudStats {
		t.Fatalf("stats envelope=%#v", events[4])
	}
	if got, ok := events[4].Payload.(cloudrealtime.StatsSnapshotMessage); !ok || !reflect.DeepEqual(got, statsMessage) {
		t.Fatalf("stats payload=%#v", events[4].Payload)
	}

	cancel()
	cancel()
	if got := bus.unsubscribeCount(); got != 2 {
		t.Fatalf("unsubscribes=%d", got)
	}
}

func TestBridgeRejectsMismatchedAndUnverifiableDeliveries(t *testing.T) {
	bus := newFakeBus()
	bridge := New(bus, "caipiao")
	var events []ws.Envelope
	_, err := bridge.SubscribeMember(7, func(env ws.Envelope) { events = append(events, env) })
	if err != nil {
		t.Fatal(err)
	}
	schemeSubject := "caipiao.client.7.scheme"
	statsSubject := "caipiao.client.7.cloud_stats"

	validScheme := cloudrealtime.SchemeSnapshotMessage{
		SchemaVersion: 1,
		GeneratedAt:   "2026-08-18T00:00:00Z",
		Items:         []schemes.Instance{{ID: "inst-a", Status: "running"}},
	}
	bus.deliverJSON(t, schemeSubject, "caipiao.client.8.scheme", validScheme)
	bus.deliver(schemeSubject, schemeSubject, []byte("not-json"))
	invalidVersion := validScheme
	invalidVersion.SchemaVersion = 2
	bus.deliverJSON(t, schemeSubject, schemeSubject, invalidVersion)
	invalidTime := validScheme
	invalidTime.GeneratedAt = "not-a-time"
	bus.deliverJSON(t, schemeSubject, schemeSubject, invalidTime)

	validStats := cloudrealtime.StatsSnapshotMessage{SchemaVersion: 1, GeneratedAt: "2026-08-18T00:00:00Z"}
	bus.deliverJSON(t, statsSubject, "caipiao.client.8.cloud_stats", validStats)
	bus.deliver(statsSubject, statsSubject, []byte(`{"schemaVersion":1}`))
	if len(events) != 0 {
		t.Fatalf("unverifiable deliveries emitted %d events", len(events))
	}
}

func TestBridgeRejectsInvalidMemberWithoutSubscribing(t *testing.T) {
	bus := newFakeBus()
	bridge := New(bus, "caipiao")
	if _, err := bridge.SubscribeMember(0, func(ws.Envelope) {}); err == nil {
		t.Fatal("zero member accepted")
	}
	if got := bus.subjects(); len(got) != 0 {
		t.Fatalf("subjects=%v", got)
	}
}

func TestBridgeCleansSchemeSubscriptionWhenStatsSubscriptionFails(t *testing.T) {
	bus := newFakeBus()
	bus.failSubject = "caipiao.client.7.cloud_stats"
	bridge := New(bus, "caipiao")
	if _, err := bridge.SubscribeMember(7, func(ws.Envelope) {}); err == nil {
		t.Fatal("stats subscription failure ignored")
	}
	if got := bus.unsubscribeCount(); got != 1 {
		t.Fatalf("scheme cleanup unsubscribes=%d", got)
	}
}

func assertLegacyHintFields(t *testing.T, payload any) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 3 {
		t.Fatalf("legacy fields=%v", fields)
	}
	if fields["instanceId"] == "" || fields["status"] == "" || fields["hint"] != "refresh_running_list" {
		t.Fatalf("legacy payload=%v", fields)
	}
}

type fakeBus struct {
	mu            sync.Mutex
	subscriptions []fakeBusSubscriptionRecord
	unsubscribes  int
	failSubject   string
}

type fakeBusSubscriptionRecord struct {
	subject string
	handler realtimebus.Handler
}

func newFakeBus() *fakeBus {
	return &fakeBus{}
}

func (b *fakeBus) Publish(context.Context, string, []byte) error { return nil }

func (b *fakeBus) Subscribe(subject string, handler realtimebus.Handler) (realtimebus.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if subject == b.failSubject {
		return nil, errors.New("subscribe failed")
	}
	b.subscriptions = append(b.subscriptions, fakeBusSubscriptionRecord{subject: subject, handler: handler})
	return &fakeSubscription{bus: b}, nil
}

func (b *fakeBus) OnConnectionChange(func(bool)) {}

func (b *fakeBus) Diagnostics() realtimebus.Diagnostics {
	return realtimebus.Diagnostics{Kind: "fake", Connected: true}
}

func (b *fakeBus) Close() error { return nil }

func (b *fakeBus) subjects() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	subjects := make([]string, 0, len(b.subscriptions))
	for _, subscription := range b.subscriptions {
		subjects = append(subjects, subscription.subject)
	}
	return subjects
}

func (b *fakeBus) unsubscribeCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.unsubscribes
}

func (b *fakeBus) deliverJSON(t *testing.T, registeredSubject, deliveredSubject string, payload any) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	b.deliver(registeredSubject, deliveredSubject, encoded)
}

func (b *fakeBus) deliver(registeredSubject, deliveredSubject string, payload []byte) {
	b.mu.Lock()
	var handler realtimebus.Handler
	for _, subscription := range b.subscriptions {
		if subscription.subject == registeredSubject {
			handler = subscription.handler
			break
		}
	}
	b.mu.Unlock()
	if handler != nil {
		handler(deliveredSubject, payload)
	}
}

type fakeSubscription struct {
	bus  *fakeBus
	once sync.Once
}

func (s *fakeSubscription) Unsubscribe() error {
	s.once.Do(func() {
		s.bus.mu.Lock()
		s.bus.unsubscribes++
		s.bus.mu.Unlock()
	})
	return nil
}

var _ realtimebus.Bus = (*fakeBus)(nil)
