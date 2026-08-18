package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

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

func TestCloudRealtimeDisabledDeliversLegacySchemeRefreshOverWebSocket(t *testing.T) {
	cfg := cloudRealtimeTestConfig(t)
	cfg.CloudRealtimeEnabled = false
	cfg.WSEnabled = true
	cfg.ClientDemoAccount = "disabled-realtime-client"
	cfg.ClientDemoPass = "disabled-realtime-password"

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New with rollout disabled: %v", err)
	}
	t.Cleanup(srv.Close)
	conn := openServerClientWebSocket(t, srv, 73)
	readWebSocketEnvelope(t, conn, ws.NameSubscribed)

	if srv.realtimeMarker == nil {
		t.Fatal("disabled rollout did not install a legacy marker")
	}
	srv.realtimeMarker.MarkScheme(73, "inst-disabled")
	envelope := readWebSocketEnvelope(t, conn, ws.NameSchemeInstanceUpdated)
	if envelope.Topic != ws.TopicClientSchemeInstance {
		t.Fatalf("topic=%q", envelope.Topic)
	}
	payload, ok := envelope.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload=%#v", envelope.Payload)
	}
	if payload["instanceId"] != "inst-disabled" || payload["hint"] != "refresh_running_list" {
		t.Fatalf("payload=%#v", payload)
	}
}

func TestLegacyRealtimeMarkerScopesRefreshByNumericMemberID(t *testing.T) {
	marker := newLegacyRealtimeMarker()
	member73 := make(chan ws.Envelope, 1)
	member74 := make(chan ws.Envelope, 1)
	cancel73, err := marker.SubscribeMember(73, func(envelope ws.Envelope) { member73 <- envelope })
	if err != nil {
		t.Fatalf("subscribe member 73: %v", err)
	}
	t.Cleanup(cancel73)
	cancel74, err := marker.SubscribeMember(74, func(envelope ws.Envelope) { member74 <- envelope })
	if err != nil {
		t.Fatalf("subscribe member 74: %v", err)
	}
	t.Cleanup(cancel74)

	marker.MarkScheme(73, "inst-member-73")
	select {
	case envelope := <-member73:
		if envelope.Name != ws.NameSchemeInstanceUpdated || envelope.Topic != ws.TopicClientSchemeInstance {
			t.Fatalf("member 73 envelope=%+v", envelope)
		}
	case <-time.After(time.Second):
		t.Fatal("member 73 did not receive its refresh")
	}
	select {
	case envelope := <-member74:
		t.Fatalf("member 74 received member 73 refresh: %+v", envelope)
	default:
	}
}

func openServerClientWebSocket(t *testing.T, srv *Server, memberID int64) *websocket.Conn {
	t.Helper()
	srv.wsServer.ResolveClientIdentity = func(_ context.Context, account string) (ws.ClientIdentity, error) {
		return ws.ClientIdentity{Account: account, MemberID: memberID}, nil
	}
	token, err := srv.authSvc.LoginClient(srv.cfg.ClientDemoAccount, srv.cfg.ClientDemoPass)
	if err != nil {
		t.Fatalf("issue client token: %v", err)
	}
	httpServer := httptest.NewServer(srv.Handler())
	t.Cleanup(httpServer.Close)
	endpoint := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/v1/ws/client?token=" + url.QueryEscape(token.AccessToken)
	conn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		t.Fatalf("dial client websocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readWebSocketEnvelope(t *testing.T, conn *websocket.Conn, wantName string) ws.Envelope {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set websocket read deadline: %v", err)
	}
	for attempts := 0; attempts < 8; attempts++ {
		var envelope ws.Envelope
		if err := conn.ReadJSON(&envelope); err != nil {
			t.Fatalf("read websocket envelope %q: %v", wantName, err)
		}
		if envelope.Name == wantName {
			return envelope
		}
	}
	t.Fatalf("websocket envelope %q not received", wantName)
	return ws.Envelope{}
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

func TestCloudRealtimeInitiallyUnreachableNATSClosesClientBeforeReadiness(t *testing.T) {
	cfg := cloudRealtimeTestConfig(t)
	cfg.CloudRealtimeEnabled = true
	cfg.CloudRealtimeBus = "nats"
	cfg.NATSURL = "nats://127.0.0.1:1"
	cfg.WSEnabled = true
	cfg.ClientDemoAccount = "unreachable-realtime-client"
	cfg.ClientDemoPass = "unreachable-realtime-password"

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New with unreachable NATS: %v", err)
	}
	t.Cleanup(srv.Close)
	conn := openServerClientWebSocket(t, srv, 91)
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set websocket read deadline: %v", err)
	}
	for {
		var envelope ws.Envelope
		err := conn.ReadJSON(&envelope)
		if err == nil {
			if envelope.Name == ws.NameSubscribed {
				t.Fatal("client reached websocket readiness while realtime bus was initially unavailable")
			}
			continue
		}
		var closeError *websocket.CloseError
		if !errors.As(err, &closeError) {
			t.Fatalf("read websocket close: %v", err)
		}
		if closeError.Code != websocket.CloseServiceRestart {
			t.Fatalf("close code=%d reason=%q", closeError.Code, closeError.Text)
		}
		return
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
	mu        sync.Mutex
	callback  func(bool)
	events    *[]string
	closed    int
	connected bool
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
	return realtimebus.Diagnostics{Kind: "fake", Connected: b.connected}
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
	bus := &callbackBus{connected: true}
	guard := newRealtimeCallbackGuard()
	type closeCall struct {
		code   int
		reason string
	}
	calls := make(chan closeCall, 2)
	registerRealtimeDisconnect(bus, guard, func(code int, reason string) {
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

func TestRegisterRealtimeDisconnectClosesExistingClientsWhenAlreadyDisconnected(t *testing.T) {
	bus := &callbackBus{connected: false}
	guard := newRealtimeCallbackGuard()
	closed := make(chan struct{}, 1)
	registerRealtimeDisconnect(bus, guard, func(code int, reason string) {
		if code != websocket.CloseServiceRestart || reason != "realtime_bus_unavailable" {
			t.Errorf("close code=%d reason=%q", code, reason)
		}
		closed <- struct{}{}
	})

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("already-disconnected bus did not close existing clients at callback registration")
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
		workerCancel:      func() { events = append(events, "cancel") },
		workerWait:        func() { events = append(events, "workers-stopped") },
		realtimeCallbacks: newRealtimeCallbackGuard(),
		realtimeBus:       bus,
		dbCloser:          database,
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

type signalingDBCloser struct {
	once   sync.Once
	closed chan struct{}
}

func (c *signalingDBCloser) Close() {
	c.once.Do(func() { close(c.closed) })
}

func TestServerCloseWaitsForTrackedWorkerBeforeDatabase(t *testing.T) {
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	workerStarted := make(chan struct{})
	releaseWorker := make(chan struct{})
	workerDBWork := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		close(workerStarted)
		<-workerCtx.Done()
		<-releaseWorker
		close(workerDBWork)
	}()
	<-workerStarted

	database := &signalingDBCloser{closed: make(chan struct{})}
	srv := &Server{
		workerCancel: cancelWorkers,
		workerWait:   workers.Wait,
		dbCloser:     database,
	}
	closeDone := make(chan struct{})
	go func() {
		srv.Close()
		close(closeDone)
	}()

	select {
	case <-workerCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("Close did not cancel workers")
	}
	select {
	case <-database.closed:
		t.Fatal("database closed while tracked worker could still use it")
	default:
	}
	close(releaseWorker)
	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("Close did not finish after tracked worker exited")
	}
	select {
	case <-workerDBWork:
	default:
		t.Fatal("tracked worker did not finish its final database work")
	}
	select {
	case <-database.closed:
	default:
		t.Fatal("database was not closed after tracked worker exited")
	}
}

func TestCleanupFailedServerStartWaitsBeforeClosingResources(t *testing.T) {
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	workerStarted := make(chan struct{})
	releaseWorker := make(chan struct{})
	workerFinished := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		close(workerStarted)
		<-workerCtx.Done()
		<-releaseWorker
		close(workerFinished)
	}()
	<-workerStarted

	busClosed := make(chan struct{})
	databaseClosed := make(chan struct{})
	cleanupDone := make(chan struct{})
	go func() {
		cleanupFailedServerStart(cancelWorkers, workers.Wait, func() { close(busClosed) }, func() { close(databaseClosed) })
		close(cleanupDone)
	}()
	select {
	case <-workerCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("constructor cleanup did not cancel workers")
	}
	select {
	case <-busClosed:
		t.Fatal("constructor cleanup closed bus while worker was running")
	default:
	}
	select {
	case <-databaseClosed:
		t.Fatal("constructor cleanup closed database while worker was running")
	default:
	}

	close(releaseWorker)
	select {
	case <-cleanupDone:
	case <-time.After(time.Second):
		t.Fatal("constructor cleanup did not finish after worker exited")
	}
	select {
	case <-workerFinished:
	default:
		t.Fatal("worker did not finish before constructor cleanup returned")
	}
	select {
	case <-busClosed:
	default:
		t.Fatal("constructor cleanup did not close bus")
	}
	select {
	case <-databaseClosed:
	default:
		t.Fatal("constructor cleanup did not close database")
	}
}

func TestServerCloseQuiescesRealtimeCallbacks(t *testing.T) {
	bus := &callbackBus{connected: true}
	guard := newRealtimeCallbackGuard()
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	callbackEffects := make(chan struct{}, 2)
	registerRealtimeDisconnect(bus, guard, func(int, string) {
		close(callbackStarted)
		<-releaseCallback
		callbackEffects <- struct{}{}
	})

	database := &signalingDBCloser{closed: make(chan struct{})}
	srv := &Server{
		workerCancel:      func() {},
		realtimeCallbacks: guard,
		realtimeBus:       bus,
		dbCloser:          database,
	}
	go bus.emit(false)
	select {
	case <-callbackStarted:
	case <-time.After(time.Second):
		t.Fatal("disconnect callback did not start")
	}
	closeDone := make(chan struct{})
	go func() {
		srv.Close()
		close(closeDone)
	}()
	select {
	case <-database.closed:
		t.Fatal("database closed while realtime callback was running")
	default:
	}
	close(releaseCallback)
	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("Close did not wait for realtime callback")
	}
	select {
	case <-callbackEffects:
	default:
		t.Fatal("in-flight realtime callback did not finish")
	}

	bus.emit(false)
	select {
	case <-callbackEffects:
		t.Fatal("realtime callback produced a side effect after Close")
	default:
	}
}
