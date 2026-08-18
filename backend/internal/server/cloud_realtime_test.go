package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/realtimebus"
	"caipiao/backend/internal/schemeevents"
	"caipiao/backend/internal/ws"
)

func cloudRealtimeTestConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		Port:                         "0",
		Env:                          "test",
		JWTSecret:                    "cloud-realtime-test-secret",
		TokenTTL:                     time.Hour,
		DatabaseURL:                  "",
		DBRequired:                   false,
		SchemeWorkerEnabled:          false,
		WSEnabled:                    false,
		CMSUploadDir:                 t.TempDir(),
		NATSSubjectPrefix:            "caipiao",
		CloudRealtimeCoalesce:        time.Millisecond,
		CloudStatsCoalesce:           time.Millisecond,
		CloudReconcileInterval:       time.Millisecond,
		CloudReconcileBatch:          10,
		SchemeWorkerConcurrency:      1,
		SchemeWorkerPlaceConcurrency: 1,
	}
}

func TestCloudRealtimeDisabledPreservesLegacyStartupAndHealth(t *testing.T) {
	cfg := cloudRealtimeTestConfig(t)
	cfg.CloudRealtimeEnabled = false
	cfg.CloudRealtimeBus = "nats"
	cfg.NATSURL = "nats://127.0.0.1:1"

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New with rollout disabled: %v", err)
	}
	t.Cleanup(srv.Close)
	if srv.realtimeBus != nil || srv.realtimePublisher != nil || srv.realtimeReconciler != nil {
		t.Fatalf("disabled rollout constructed realtime runtime: bus=%T publisher=%p reconciler=%p", srv.realtimeBus, srv.realtimePublisher, srv.realtimeReconciler)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	srv.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "cloudRealtime") {
		t.Fatalf("legacy health changed: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCloudRealtimeUnreachableNATSStartsBoundedAndDegraded(t *testing.T) {
	cfg := cloudRealtimeTestConfig(t)
	cfg.CloudRealtimeEnabled = true
	cfg.CloudRealtimeBus = "nats"
	cfg.NATSURL = "nats://127.0.0.1:1"

	started := time.Now()
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New with unreachable NATS: %v", err)
	}
	t.Cleanup(srv.Close)
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("unreachable NATS blocked startup for %s", elapsed)
	}
	if srv.realtimeBus == nil || srv.realtimeBus.Diagnostics().Connected {
		t.Fatalf("bus diagnostics=%+v, want disconnected NATS runtime", srv.realtimeBus.Diagnostics())
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	srv.Handler().ServeHTTP(w, r)
	var response struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if response.Data["status"] != "ok" {
		t.Fatalf("application health=%v, want ok", response.Data["status"])
	}
	realtime, ok := response.Data["cloudRealtime"].(map[string]any)
	if !ok || realtime["status"] != "degraded" {
		t.Fatalf("cloudRealtime=%#v, want degraded", response.Data["cloudRealtime"])
	}
}

func TestCloudRealtimeMemoryRuntimeWiresAllComponents(t *testing.T) {
	cfg := cloudRealtimeTestConfig(t)
	cfg.CloudRealtimeEnabled = true
	cfg.CloudRealtimeBus = "memory"

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New with memory realtime: %v", err)
	}
	t.Cleanup(srv.Close)
	if srv.realtimeBus == nil || srv.realtimeBus.Diagnostics().Kind != "memory" {
		t.Fatalf("bus=%T diagnostics=%+v", srv.realtimeBus, srv.realtimeBus.Diagnostics())
	}
	if srv.realtimePublisher == nil || srv.realtimeReconciler == nil {
		t.Fatalf("publisher=%p reconciler=%p", srv.realtimePublisher, srv.realtimeReconciler)
	}
	if srv.wsServer == nil || srv.wsServer.ResolveClientIdentity == nil {
		t.Fatal("numeric websocket identity resolver was not installed")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/diagnostics/cloud-realtime", nil)
	srv.handler.AdminCloudRealtimeDiagnostics(w, r)
	for _, section := range []string{"bus", "publisher", "hub", "scanner"} {
		if !strings.Contains(w.Body.String(), `"`+section+`"`) {
			t.Fatalf("missing %q diagnostics in %s", section, w.Body.String())
		}
	}
}

type fakeIdentitySource struct {
	wantAccount string
	ref         member.MemberRef
	err         error
}

func (s *fakeIdentitySource) GetByAccount(_ context.Context, account string) (member.MemberRef, error) {
	if account != s.wantAccount {
		return member.MemberRef{}, fmt.Errorf("account=%q want %q", account, s.wantAccount)
	}
	return s.ref, s.err
}

func TestClientIdentityResolverUsesJWTSubjectAndNumericMemberID(t *testing.T) {
	resolve := clientIdentityResolver(&fakeIdentitySource{
		wantAccount: "member-account",
		ref:         member.MemberRef{ID: 73, Account: "member-account"},
	})

	identity, err := resolve(context.Background(), "member-account")
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if identity != (ws.ClientIdentity{Account: "member-account", MemberID: 73}) {
		t.Fatalf("identity=%+v", identity)
	}
}

func TestClientIdentityResolverRejectsUnavailableMemberService(t *testing.T) {
	resolve := clientIdentityResolver(nil)
	if _, err := resolve(context.Background(), "member-account"); err == nil {
		t.Fatal("nil member service resolved a client identity")
	}
}

type callbackBus struct {
	mu       sync.Mutex
	callback func(bool)
	events   *[]string
	closed   int
}

func (b *callbackBus) Publish(context.Context, string, []byte) error { return nil }
func (b *callbackBus) Subscribe(string, realtimebus.Handler) (realtimebus.Subscription, error) {
	return nil, errors.New("not used")
}
func (b *callbackBus) OnConnectionChange(callback func(bool)) {
	b.mu.Lock()
	b.callback = callback
	b.mu.Unlock()
}
func (b *callbackBus) Diagnostics() realtimebus.Diagnostics {
	return realtimebus.Diagnostics{Kind: "fake", Connected: true}
}
func (b *callbackBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed++
	if b.events != nil {
		*b.events = append(*b.events, "bus")
	}
	return nil
}
func (b *callbackBus) emit(connected bool) {
	b.mu.Lock()
	callback := b.callback
	b.mu.Unlock()
	callback(connected)
}

func TestRealtimeDisconnectClosesOnlyOnUnavailableTransition(t *testing.T) {
	bus := &callbackBus{}
	type closeCall struct {
		code   int
		reason string
	}
	calls := make(chan closeCall, 2)
	registerRealtimeDisconnect(bus, func(code int, reason string) {
		calls <- closeCall{code: code, reason: reason}
	})

	bus.emit(true)
	select {
	case call := <-calls:
		t.Fatalf("connected transition closed clients: %+v", call)
	default:
	}
	bus.emit(false)
	select {
	case call := <-calls:
		if call.code != 1012 || call.reason != "realtime_bus_unavailable" {
			t.Fatalf("close=%+v", call)
		}
	case <-time.After(time.Second):
		t.Fatal("disconnect did not close client sockets")
	}
}

type recordingMarkerSetter struct {
	marker schemeevents.Marker
}

func (s *recordingMarkerSetter) SetRealtimeMarker(marker schemeevents.Marker) {
	s.marker = marker
}

type noopMarker struct{}

func (noopMarker) MarkScheme(int64, string) {}

func TestInjectRealtimeMarkerTouchesEveryConstructedTarget(t *testing.T) {
	marker := noopMarker{}
	targets := []*recordingMarkerSetter{{}, {}, {}, {}, {}}
	setters := make([]realtimeMarkerSetter, 0, len(targets))
	for _, target := range targets {
		setters = append(setters, target)
	}
	injectRealtimeMarker(marker, setters...)
	for index, target := range targets {
		if target.marker != marker {
			t.Fatalf("target %d did not receive publisher marker", index)
		}
	}
}

type recordingDBCloser struct {
	events *[]string
	closed int
}

func (c *recordingDBCloser) Close() {
	c.closed++
	*c.events = append(*c.events, "db")
}

func TestServerCloseOrdersWorkersRealtimeAndDatabaseExactlyOnce(t *testing.T) {
	events := make([]string, 0, 4)
	bus := &callbackBus{events: &events}
	database := &recordingDBCloser{events: &events}
	srv := &Server{
		workerCancel: func() { events = append(events, "cancel") },
		realtimeWait: func() { events = append(events, "workers-stopped") },
		realtimeBus:  bus,
		dbCloser:     database,
	}

	srv.Close()
	srv.Close()

	want := []string{"cancel", "workers-stopped", "bus", "db"}
	if fmt.Sprint(events) != fmt.Sprint(want) {
		t.Fatalf("close order=%v want %v", events, want)
	}
	if bus.closed != 1 || database.closed != 1 {
		t.Fatalf("bus closes=%d db closes=%d", bus.closed, database.closed)
	}
}
