package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"caipiao/backend/internal/auth"
	"caipiao/backend/internal/config"
)

type fakeMemberEventSource struct {
	mu               sync.Mutex
	subscribeCalls   []int64
	cancelCalls      int
	handlers         map[int64]func(Envelope)
	subscribeEntered chan int64
	subscribeRelease <-chan struct{}
	cancelSignal     chan int64
}

func (f *fakeMemberEventSource) SubscribeMember(memberID int64, emit func(Envelope)) (func(), error) {
	f.mu.Lock()
	f.subscribeCalls = append(f.subscribeCalls, memberID)
	if f.handlers == nil {
		f.handlers = make(map[int64]func(Envelope))
	}
	f.handlers[memberID] = emit
	entered := f.subscribeEntered
	release := f.subscribeRelease
	f.mu.Unlock()

	if entered != nil {
		entered <- memberID
	}
	if release != nil {
		<-release
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			f.cancelCalls++
			delete(f.handlers, memberID)
			signal := f.cancelSignal
			f.mu.Unlock()
			if signal != nil {
				signal <- memberID
			}
		})
	}, nil
}

func (f *fakeMemberEventSource) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.subscribeCalls), f.cancelCalls
}

func (f *fakeMemberEventSource) subscribedMembers() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int64(nil), f.subscribeCalls...)
}

func (f *fakeMemberEventSource) emit(memberID int64, env Envelope) bool {
	f.mu.Lock()
	handler := f.handlers[memberID]
	f.mu.Unlock()
	if handler == nil {
		return false
	}
	handler(env)
	return true
}

func newTestClientConn(identity ClientIdentity) *Conn {
	return &Conn{
		kind:          KindClient,
		authenticated: true,
		account:       identity.Account,
		memberID:      identity.MemberID,
		topics:        make(map[string]struct{}),
		send:          make(chan Envelope, 4),
	}
}

func TestHubSubscribesOncePerMemberAndReleasesLastConnection(t *testing.T) {
	source := &fakeMemberEventSource{}
	h := NewHub()
	h.SetMemberEventSource(source)
	c1 := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	c2 := newTestClientConn(ClientIdentity{Account: "renamed-account", MemberID: 7})

	h.Register(c1)
	h.Register(c2)
	if subscribe, _ := source.counts(); subscribe != 1 {
		t.Fatalf("subscribe=%d", subscribe)
	}
	if got := source.subscribedMembers(); !reflect.DeepEqual(got, []int64{7}) {
		t.Fatalf("members=%v", got)
	}
	if got := h.Diagnostics().SubscribedMembers; got != 1 {
		t.Fatalf("subscribed members=%d", got)
	}

	h.Unregister(c1)
	if _, cancel := source.counts(); cancel != 0 {
		t.Fatalf("cancel=%d", cancel)
	}
	h.Unregister(c2)
	if _, cancel := source.counts(); cancel != 1 {
		t.Fatalf("cancel=%d", cancel)
	}
	if got := h.Diagnostics().SubscribedMembers; got != 0 {
		t.Fatalf("subscribed members=%d", got)
	}
}

func TestHubRoutesMemberEventsByNumericIdentityNotAccount(t *testing.T) {
	source := &fakeMemberEventSource{}
	h := NewHub()
	h.SetMemberEventSource(source)
	member7 := newTestClientConn(ClientIdentity{Account: "shared", MemberID: 7})
	member8 := newTestClientConn(ClientIdentity{Account: "shared", MemberID: 8})
	h.Register(member7)
	h.Register(member8)
	t.Cleanup(func() {
		h.Unregister(member7)
		h.Unregister(member8)
	})
	h.Subscribe(member7, []string{TopicClientSchemeInstance})
	h.Subscribe(member8, []string{TopicClientSchemeInstance})

	env := NewEvent(NameSchemeInstancesSnapshot, TopicClientSchemeInstance, map[string]any{"schemaVersion": 1})
	if !source.emit(7, env) {
		t.Fatal("member 7 handler missing")
	}
	select {
	case got := <-member7.send:
		if got.Name != NameSchemeInstancesSnapshot {
			t.Fatalf("member 7 event=%q", got.Name)
		}
	default:
		t.Fatal("member 7 did not receive event")
	}
	select {
	case got := <-member8.send:
		t.Fatalf("member 8 received cross-member event %#v", got)
	default:
	}
}

func TestHubBindClientIdentityAcquiresAfterRegistrationOnce(t *testing.T) {
	source := &fakeMemberEventSource{}
	h := NewHub()
	h.SetMemberEventSource(source)
	c := &Conn{kind: KindClient, topics: make(map[string]struct{}), send: make(chan Envelope, 4)}
	h.Register(c)

	identity := ClientIdentity{Account: "verified", MemberID: 7}
	if !h.BindClientIdentity(c, identity) {
		t.Fatal("first bind rejected")
	}
	if !h.BindClientIdentity(c, identity) {
		t.Fatal("idempotent bind rejected")
	}
	if subscribe, _ := source.counts(); subscribe != 1 {
		t.Fatalf("subscribe=%d", subscribe)
	}
	if h.BindClientIdentity(c, ClientIdentity{Account: "other", MemberID: 8}) {
		t.Fatal("identity rebind accepted")
	}

	h.Unregister(c)
	if _, cancel := source.counts(); cancel != 1 {
		t.Fatalf("cancel=%d", cancel)
	}
}

func TestHubCancelsSubscriptionAcquiredAfterLastConnectionLeaves(t *testing.T) {
	entered := make(chan int64)
	release := make(chan struct{})
	canceled := make(chan int64, 1)
	source := &fakeMemberEventSource{
		subscribeEntered: entered,
		subscribeRelease: release,
		cancelSignal:     canceled,
	}
	h := NewHub()
	h.SetMemberEventSource(source)
	c1 := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	c2 := newTestClientConn(ClientIdentity{Account: "b", MemberID: 7})
	firstDone := make(chan struct{})
	go func() {
		h.Register(c1)
		close(firstDone)
	}()
	if got := waitInt64(t, entered, "source subscription"); got != 7 {
		t.Fatalf("member=%d", got)
	}

	secondDone := make(chan struct{})
	go func() {
		h.Register(c2)
		close(secondDone)
	}()
	waitClosed(t, secondDone, "second registration while source is blocked")
	h.Unregister(c1)
	h.Unregister(c2)
	close(release)
	waitClosed(t, firstDone, "first registration completion")
	if got := waitInt64(t, canceled, "orphaned subscription cancellation"); got != 7 {
		t.Fatalf("canceled member=%d", got)
	}
	if subscribe, cancel := source.counts(); subscribe != 1 || cancel != 1 {
		t.Fatalf("subscribe=%d cancel=%d", subscribe, cancel)
	}
}

func TestHubCloseClientConnectionsDoesNotHoldLockDuringNetworkClose(t *testing.T) {
	h := NewHub()
	closeEntered := make(chan struct{})
	releaseClose := make(chan struct{})
	client := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	client.closeFn = func(code int, reason string) {
		if code != 1012 || reason != "realtime_bus_unavailable" {
			t.Errorf("close=%d/%q", code, reason)
		}
		close(closeEntered)
		<-releaseClose
	}
	adminClosed := make(chan struct{}, 1)
	admin := &Conn{
		kind:          KindAdmin,
		authenticated: true,
		account:       "admin",
		topics:        make(map[string]struct{}),
		send:          make(chan Envelope, 1),
		closeFn:       func(int, string) { adminClosed <- struct{}{} },
	}
	h.Register(client)
	h.Register(admin)
	t.Cleanup(func() {
		h.Unregister(client)
		h.Unregister(admin)
	})

	closeDone := make(chan struct{})
	go func() {
		h.CloseClientConnections(1012, "realtime_bus_unavailable")
		close(closeDone)
	}()
	waitClosed(t, closeEntered, "client close entry")
	countDone := make(chan int, 1)
	go func() { countDone <- h.ConnCount() }()
	select {
	case got := <-countDone:
		if got != 2 {
			t.Fatalf("connections=%d", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ConnCount blocked behind network close")
	}
	close(releaseClose)
	waitClosed(t, closeDone, "bulk close completion")
	select {
	case <-adminClosed:
		t.Fatal("admin connection was closed")
	default:
	}
}

func TestClientDefaultSubscriptionAcknowledgesSchemeStatsAndWallet(t *testing.T) {
	h := NewHub()
	c := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	c.hub = h
	h.Register(c)
	t.Cleanup(func() { h.Unregister(c) })

	c.subscribeClientTopics()
	env := <-c.send
	if env.Name != NameSubscribed {
		t.Fatalf("event=%q", env.Name)
	}
	payload, err := json.Marshal(env.Payload)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Topics []string `json:"topics"`
	}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	want := []string{TopicClientSchemeInstance, TopicClientCloudStats, TopicClientWallet}
	if !reflect.DeepEqual(got.Topics, want) {
		t.Fatalf("topics=%v want=%v", got.Topics, want)
	}
}

func TestCommandAuthBindsVerifiedNumericIdentity(t *testing.T) {
	source := &fakeMemberEventSource{}
	h := NewHub()
	h.SetMemberEventSource(source)
	c := &Conn{hub: h, kind: KindClient, topics: make(map[string]struct{}), send: make(chan Envelope, 4)}
	h.Register(c)
	t.Cleanup(func() { h.Unregister(c) })

	c.handleMessage([]byte(`{"type":"command","name":"auth","payload":{"accessToken":"good"}}`), func(token string) (ClientIdentity, bool) {
		return ClientIdentity{Account: "verified", MemberID: 7}, token == "good"
	})
	if got := source.subscribedMembers(); !reflect.DeepEqual(got, []int64{7}) {
		t.Fatalf("members=%v", got)
	}
	first, second := <-c.send, <-c.send
	if first.Name != NameAuthOK || second.Name != NameSubscribed {
		t.Fatalf("events=%q,%q", first.Name, second.Name)
	}
}

func TestAdminCommandAuthKeepsExistingTopicsWithoutMemberSubscription(t *testing.T) {
	source := &fakeMemberEventSource{}
	h := NewHub()
	h.SetMemberEventSource(source)
	c := &Conn{hub: h, kind: KindAdmin, topics: make(map[string]struct{}), send: make(chan Envelope, 4)}
	h.Register(c)
	t.Cleanup(func() { h.Unregister(c) })

	c.handleMessage([]byte(`{"type":"command","name":"auth","payload":{"accessToken":"good"}}`), func(token string) (ClientIdentity, bool) {
		return ClientIdentity{Account: "admin"}, token == "good"
	})
	if got := source.subscribedMembers(); len(got) != 0 {
		t.Fatalf("admin acquired member subscriptions=%v", got)
	}
	first, second := <-c.send, <-c.send
	if first.Name != NameAuthOK || second.Name != NameSubscribed {
		t.Fatalf("events=%q,%q", first.Name, second.Name)
	}
	payload, err := json.Marshal(second.Payload)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Topics []string `json:"topics"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatal(err)
	}
	want := []string{TopicAdminWithdrawQueue, TopicAdminSchemeMonitor, TopicAdminDashboardKpi}
	if !reflect.DeepEqual(body.Topics, want) {
		t.Fatalf("topics=%v want=%v", body.Topics, want)
	}
}

func TestLegacyWalletPublishRemainsAccountScoped(t *testing.T) {
	h := NewHub()
	member7 := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	member8 := newTestClientConn(ClientIdentity{Account: "b", MemberID: 8})
	h.Register(member7)
	h.Register(member8)
	t.Cleanup(func() {
		h.Unregister(member7)
		h.Unregister(member8)
	})
	h.Subscribe(member7, []string{TopicClientWallet})
	h.Subscribe(member8, []string{TopicClientWallet})

	PublishWallet(h, "a", WalletUpdatedPayload{Available: 10, Currency: "USDT"})
	select {
	case got := <-member7.send:
		if got.Name != NameWalletUpdated {
			t.Fatalf("member 7 event=%q", got.Name)
		}
	default:
		t.Fatal("member 7 did not receive wallet event")
	}
	select {
	case got := <-member8.send:
		t.Fatalf("member 8 received cross-account wallet event %#v", got)
	default:
	}
}

func TestServerQueryTokenBindsResolvedNumericIdentity(t *testing.T) {
	authService, token := testClientAuth(t)
	source := &fakeMemberEventSource{subscribeEntered: make(chan int64, 1)}
	h := NewHub()
	h.SetMemberEventSource(source)
	resolved := make(chan string, 1)
	server := &Server{
		Hub:  h,
		Auth: authService,
		ResolveClientIdentity: func(ctx context.Context, account string) (ClientIdentity, error) {
			if err := ctx.Err(); err != nil {
				return ClientIdentity{}, err
			}
			resolved <- account
			return ClientIdentity{Account: account, MemberID: 7}, nil
		},
	}
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleClient))
	defer httpServer.Close()

	conn, _, err := websocket.DefaultDialer.Dial(websocketURL(httpServer.URL)+"?token="+url.QueryEscape(token), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if got := waitInt64(t, source.subscribeEntered, "query-token member subscription"); got != 7 {
		t.Fatalf("member=%d", got)
	}
	select {
	case got := <-resolved:
		if got != "client-a" {
			t.Fatalf("resolved account=%q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("identity resolver not called")
	}
	assertSubscribedFrame(t, conn)
}

func TestServerCommandAuthBindsResolvedNumericIdentity(t *testing.T) {
	authService, token := testClientAuth(t)
	source := &fakeMemberEventSource{subscribeEntered: make(chan int64, 1)}
	h := NewHub()
	h.SetMemberEventSource(source)
	server := &Server{
		Hub:  h,
		Auth: authService,
		ResolveClientIdentity: func(ctx context.Context, account string) (ClientIdentity, error) {
			if err := ctx.Err(); err != nil {
				return ClientIdentity{}, err
			}
			return ClientIdentity{Account: account, MemberID: 9}, nil
		},
	}
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleClient))
	defer httpServer.Close()

	conn, _, err := websocket.DefaultDialer.Dial(websocketURL(httpServer.URL), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connected Envelope
	if err := conn.ReadJSON(&connected); err != nil {
		t.Fatal(err)
	}
	if connected.Name != NameConnected {
		t.Fatalf("first event=%q", connected.Name)
	}
	if err := conn.WriteJSON(map[string]any{
		"type":    FrameTypeCommand,
		"name":    "auth",
		"payload": map[string]any{"accessToken": token},
	}); err != nil {
		t.Fatal(err)
	}
	if got := waitInt64(t, source.subscribeEntered, "command-auth member subscription"); got != 9 {
		t.Fatalf("member=%d", got)
	}
	assertSubscribedFrame(t, conn)
}

func testClientAuth(t *testing.T) (*auth.Service, string) {
	t.Helper()
	svc := auth.NewService(config.Config{
		JWTSecret:         "task-5-test-secret",
		ClientDemoAccount: "client-a",
		ClientDemoPass:    "password",
		TokenTTL:          time.Hour,
	}, nil)
	result, err := svc.LoginClient("client-a", "password")
	if err != nil {
		t.Fatal(err)
	}
	return svc, result.AccessToken
}

func websocketURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func assertSubscribedFrame(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 4; i++ {
		var env Envelope
		if err := conn.ReadJSON(&env); err != nil {
			t.Fatal(err)
		}
		if env.Name != NameSubscribed {
			continue
		}
		payload, err := json.Marshal(env.Payload)
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Topics []string `json:"topics"`
		}
		if err := json.Unmarshal(payload, &body); err != nil {
			t.Fatal(err)
		}
		want := []string{TopicClientSchemeInstance, TopicClientCloudStats, TopicClientWallet}
		if !reflect.DeepEqual(body.Topics, want) {
			t.Fatalf("topics=%v want=%v", body.Topics, want)
		}
		return
	}
	t.Fatal("system.subscribed frame not received")
}

func waitClosed(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitInt64(t *testing.T, ch <-chan int64, name string) int64 {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
		return 0
	}
}
