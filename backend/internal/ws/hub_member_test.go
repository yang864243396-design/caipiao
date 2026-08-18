package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"runtime"
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
	subscriptions    []*fakeMemberSubscription
	latestByMember   map[int64]int
	subscribeEntered chan int64
	subscribeRelease <-chan struct{}
	subscribeErr     error
	cancelEntered    chan int64
	cancelRelease    <-chan struct{}
	cancelSignal     chan int64
	active           int
	maxActive        int
}

type fakeMemberSubscription struct {
	memberID int64
	emit     func(Envelope)
	active   bool
}

type synchronousMemberEventSource struct {
	mu             sync.Mutex
	emitMemberID   int64
	events         []Envelope
	subscribeCalls []int64
}

func (s *synchronousMemberEventSource) SubscribeMember(memberID int64, emit func(Envelope)) (func(), error) {
	s.mu.Lock()
	s.subscribeCalls = append(s.subscribeCalls, memberID)
	emitMemberID := s.emitMemberID
	events := append([]Envelope(nil), s.events...)
	s.mu.Unlock()
	if memberID == emitMemberID {
		for _, event := range events {
			emit(event)
		}
	}
	return func() {}, nil
}

func (s *synchronousMemberEventSource) subscribedMembers() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.subscribeCalls...)
}

func (f *fakeMemberEventSource) SubscribeMember(memberID int64, emit func(Envelope)) (func(), error) {
	f.mu.Lock()
	f.subscribeCalls = append(f.subscribeCalls, memberID)
	entered := f.subscribeEntered
	release := f.subscribeRelease
	subscribeErr := f.subscribeErr
	f.mu.Unlock()

	if entered != nil {
		entered <- memberID
	}
	if release != nil {
		<-release
	}
	if subscribeErr != nil {
		return nil, subscribeErr
	}

	f.mu.Lock()
	if f.latestByMember == nil {
		f.latestByMember = make(map[int64]int)
	}
	subscription := &fakeMemberSubscription{memberID: memberID, emit: emit, active: true}
	f.subscriptions = append(f.subscriptions, subscription)
	f.latestByMember[memberID] = len(f.subscriptions) - 1
	f.active++
	if f.active > f.maxActive {
		f.maxActive = f.active
	}
	f.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			entered := f.cancelEntered
			release := f.cancelRelease
			f.mu.Unlock()
			if entered != nil {
				entered <- memberID
			}
			if release != nil {
				<-release
			}

			f.mu.Lock()
			f.cancelCalls++
			if subscription.active {
				subscription.active = false
				f.active--
			}
			if latest, ok := f.latestByMember[memberID]; ok && f.subscriptions[latest] == subscription {
				delete(f.latestByMember, memberID)
			}
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
	index, ok := f.latestByMember[memberID]
	var subscription *fakeMemberSubscription
	active := false
	if ok {
		subscription = f.subscriptions[index]
		active = subscription.active
	}
	f.mu.Unlock()
	if subscription == nil || !active {
		return false
	}
	subscription.emit(env)
	return true
}

func (f *fakeMemberEventSource) emitSubscription(index int, env Envelope) bool {
	f.mu.Lock()
	if index < 0 || index >= len(f.subscriptions) {
		f.mu.Unlock()
		return false
	}
	emit := f.subscriptions[index].emit
	f.mu.Unlock()
	emit(env)
	return true
}

func (f *fakeMemberEventSource) activeCounts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.active, f.maxActive
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

func TestHubDeliversSynchronousAcquisitionCallbacksExactlyOnce(t *testing.T) {
	scheme := NewEvent(NameSchemeInstancesSnapshot, TopicClientSchemeInstance, map[string]any{"schemaVersion": 1})
	stats := NewEvent(NameCloudStatsSnapshot, TopicClientCloudStats, map[string]any{"schemaVersion": 1})
	source := &synchronousMemberEventSource{emitMemberID: 7, events: []Envelope{scheme, stats}}
	h := NewHub()
	h.SetMemberEventSource(source)
	member7 := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	otherMember := newTestClientConn(ClientIdentity{Account: "b", MemberID: 8})
	wantTopics := []string{TopicClientSchemeInstance, TopicClientCloudStats}
	for _, c := range []*Conn{member7, otherMember} {
		if got := h.Subscribe(c, wantTopics); !reflect.DeepEqual(got, wantTopics) {
			t.Fatalf("topics=%v want=%v", got, wantTopics)
		}
	}
	if !h.Register(otherMember) {
		t.Fatal("other member route acquisition was not ready")
	}
	if !h.Register(member7) {
		t.Fatal("member route acquisition was not ready")
	}
	if got := source.subscribedMembers(); !reflect.DeepEqual(got, []int64{8, 7}) {
		t.Fatalf("source subscribed members=%v want=[8 7]", got)
	}
	t.Cleanup(func() {
		h.Unregister(member7)
		h.Unregister(otherMember)
	})

	if got := len(member7.send); got != 2 {
		t.Fatalf("synchronous acquisition callbacks delivered=%d want=2", got)
	}
	for _, want := range []Envelope{scheme, stats} {
		got := <-member7.send
		if got.EventID != want.EventID || got.Name != want.Name || got.Topic != want.Topic {
			t.Fatalf("event=%#v want=%#v", got, want)
		}
	}
	if got := len(member7.send); got != 0 {
		t.Fatalf("duplicate synchronous acquisition callbacks=%d", got)
	}
	if got := len(otherMember.send); got != 0 {
		t.Fatalf("other member synchronous callbacks=%d want=0", got)
	}
}

func TestHubReconnectWaitsForCancellationAndRejectsSupersededCallbacks(t *testing.T) {
	cancelEntered := make(chan int64, 2)
	cancelRelease := make(chan struct{})
	subscribeEntered := make(chan int64, 2)
	source := &fakeMemberEventSource{
		subscribeEntered: subscribeEntered,
		cancelEntered:    cancelEntered,
		cancelRelease:    cancelRelease,
	}
	h := NewHub()
	h.SetMemberEventSource(source)
	first := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	h.Register(first)
	if got := waitInt64(t, subscribeEntered, "first member subscription"); got != 7 {
		t.Fatalf("first subscribed member=%d", got)
	}
	h.Subscribe(first, []string{TopicClientSchemeInstance})
	h.mu.RLock()
	firstRoute := h.members[7]
	h.mu.RUnlock()

	unregisterDone := make(chan struct{})
	go func() {
		h.Unregister(first)
		close(unregisterDone)
	}()
	if got := waitInt64(t, cancelEntered, "first member cancellation"); got != 7 {
		t.Fatalf("canceling member=%d", got)
	}

	reconnected := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
	h.Subscribe(reconnected, []string{TopicClientSchemeInstance})
	reconnectDone := make(chan bool, 1)
	go func() {
		reconnectDone <- h.Register(reconnected)
	}()
	waitForCondition(t, "reconnect attached to route", func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.memberByConn[reconnected] == 7
	})
	h.mu.RLock()
	reconnectRoute := h.members[7]
	h.mu.RUnlock()
	if reconnectRoute != firstRoute {
		t.Fatal("reconnect replaced route before cancellation completed")
	}

	stale := NewEvent(NameSchemeInstancesSnapshot, TopicClientSchemeInstance, map[string]any{"generation": "old"})
	if !source.emitSubscription(0, stale) {
		t.Fatal("old subscription callback missing")
	}
	select {
	case got := <-reconnected.send:
		t.Fatalf("superseded callback reached reconnect %#v", got)
	default:
	}
	if active, maxActive := source.activeCounts(); active != 1 || maxActive != 1 {
		t.Fatalf("during cancellation active=%d maxActive=%d", active, maxActive)
	}
	if subscribe, _ := source.counts(); subscribe != 1 {
		t.Fatalf("subscriptions before cancellation release=%d", subscribe)
	}
	select {
	case <-reconnectDone:
		t.Fatal("reconnect returned before cancellation completed")
	default:
	}

	close(cancelRelease)
	waitClosed(t, unregisterDone, "first unregister completion")
	if got := waitInt64(t, subscribeEntered, "replacement member subscription"); got != 7 {
		t.Fatalf("replacement subscribed member=%d", got)
	}
	if !waitBool(t, reconnectDone, "reconnect completion after cancellation") {
		t.Fatal("reconnect was not ready after replacement subscription acquisition")
	}
	if active, maxActive := source.activeCounts(); active != 1 || maxActive != 1 {
		t.Fatalf("after reconnect active=%d maxActive=%d", active, maxActive)
	}

	if !source.emitSubscription(0, stale) {
		t.Fatal("old callback unavailable after reconnect")
	}
	select {
	case got := <-reconnected.send:
		t.Fatalf("old generation reached reconnect %#v", got)
	default:
	}
	fresh := NewEvent(NameSchemeInstancesSnapshot, TopicClientSchemeInstance, map[string]any{"generation": "new"})
	if !source.emitSubscription(1, fresh) {
		t.Fatal("replacement callback missing")
	}
	select {
	case got := <-reconnected.send:
		if got.EventID != fresh.EventID {
			t.Fatalf("event=%q want=%q", got.EventID, fresh.EventID)
		}
	default:
		t.Fatal("replacement callback was not delivered")
	}
	h.Unregister(reconnected)
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
	firstDone := make(chan bool, 1)
	go func() {
		firstDone <- h.Register(c1)
	}()
	if got := waitInt64(t, entered, "source subscription"); got != 7 {
		t.Fatalf("member=%d", got)
	}

	secondDone := make(chan bool, 1)
	go func() {
		secondDone <- h.Register(c2)
	}()
	waitForCondition(t, "second connection joined in-flight route", func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.memberByConn[c2] == 7
	})
	select {
	case <-secondDone:
		t.Fatal("second registration reported readiness while source acquisition was blocked")
	default:
	}
	h.Unregister(c1)
	h.Unregister(c2)
	close(release)
	if waitBool(t, firstDone, "first registration completion") {
		t.Fatal("orphaned first registration reported readiness")
	}
	if got := waitInt64(t, canceled, "orphaned subscription cancellation"); got != 7 {
		t.Fatalf("canceled member=%d", got)
	}
	if waitBool(t, secondDone, "second registration completion") {
		t.Fatal("orphaned second registration reported readiness")
	}
	if subscribe, cancel := source.counts(); subscribe != 1 || cancel != 1 {
		t.Fatalf("subscribe=%d cancel=%d", subscribe, cancel)
	}
}

func TestHubAcquisitionFailureClosesAllWaitingConnectionsBeforeRegistrationReturns(t *testing.T) {
	entered := make(chan int64)
	release := make(chan struct{})
	source := &fakeMemberEventSource{
		subscribeEntered: entered,
		subscribeRelease: release,
		subscribeErr:     errors.New("nats subscription unavailable"),
	}
	h := NewHub()
	h.SetMemberEventSource(source)
	closed := make(chan int, 2)
	newConnection := func(id int) *Conn {
		c := newTestClientConn(ClientIdentity{Account: "a", MemberID: 7})
		c.closeFn = func(code int, reason string) {
			if code != websocket.CloseServiceRestart || reason != "realtime_route_unavailable" {
				t.Errorf("connection %d close=%d/%q", id, code, reason)
			}
			closed <- id
		}
		return c
	}
	c1 := newConnection(1)
	c2 := newConnection(2)
	firstReady := make(chan bool, 1)
	go func() { firstReady <- h.Register(c1) }()
	if got := waitInt64(t, entered, "failed source acquisition"); got != 7 {
		t.Fatalf("member=%d", got)
	}
	secondReady := make(chan bool, 1)
	go func() { secondReady <- h.Register(c2) }()
	waitForCondition(t, "second connection joined failed in-flight route", func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.memberByConn[c2] == 7
	})
	select {
	case <-secondReady:
		t.Fatal("waiting connection reported readiness before acquisition result")
	default:
	}

	close(release)
	closedIDs := map[int]bool{
		waitInt(t, closed, "first failed-route close"):  true,
		waitInt(t, closed, "second failed-route close"): true,
	}
	if len(closedIDs) != 2 || !closedIDs[1] || !closedIDs[2] {
		t.Fatalf("closed connections=%v", closedIDs)
	}
	if waitBool(t, firstReady, "first failed registration") {
		t.Fatal("first failed registration reported readiness")
	}
	if waitBool(t, secondReady, "second failed registration") {
		t.Fatal("second failed registration reported readiness")
	}
	if subscribe, _ := source.counts(); subscribe != 1 {
		t.Fatalf("shared failed acquisition subscriptions=%d", subscribe)
	}
	h.Unregister(c1)
	h.Unregister(c2)
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

func TestServerQueryTokenMemberSourceFailureClosesBeforeReadiness(t *testing.T) {
	authService, token := testClientAuth(t)
	source := &fakeMemberEventSource{
		subscribeEntered: make(chan int64, 1),
		subscribeErr:     errors.New("nats subscription unavailable"),
	}
	h := NewHub()
	h.SetMemberEventSource(source)
	server := &Server{
		Hub:  h,
		Auth: authService,
		ResolveClientIdentity: func(context.Context, string) (ClientIdentity, error) {
			return ClientIdentity{Account: "client-a", MemberID: 7}, nil
		},
	}
	httpServer := httptest.NewServer(http.HandlerFunc(server.HandleClient))
	defer httpServer.Close()

	conn, _, err := websocket.DefaultDialer.Dial(websocketURL(httpServer.URL)+"?token="+url.QueryEscape(token), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if got := waitInt64(t, source.subscribeEntered, "query-token failed member subscription"); got != 7 {
		t.Fatalf("member=%d", got)
	}
	assertMemberRouteFailureBeforeReadiness(t, conn)
}

func TestServerCommandAuthMemberSourceFailureClosesBeforeReadiness(t *testing.T) {
	authService, token := testClientAuth(t)
	source := &fakeMemberEventSource{
		subscribeEntered: make(chan int64, 1),
		subscribeErr:     errors.New("nats subscription unavailable"),
	}
	h := NewHub()
	h.SetMemberEventSource(source)
	server := &Server{
		Hub:  h,
		Auth: authService,
		ResolveClientIdentity: func(context.Context, string) (ClientIdentity, error) {
			return ClientIdentity{Account: "client-a", MemberID: 9}, nil
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
	if got := waitInt64(t, source.subscribeEntered, "command-auth failed member subscription"); got != 9 {
		t.Fatalf("member=%d", got)
	}
	assertMemberRouteFailureBeforeReadiness(t, conn)
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

func assertMemberRouteFailureBeforeReadiness(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		var env Envelope
		err := conn.ReadJSON(&env)
		if err != nil {
			var closeErr *websocket.CloseError
			if !errors.As(err, &closeErr) {
				t.Fatalf("read before member-route close: %v", err)
			}
			if closeErr.Code != websocket.CloseServiceRestart || closeErr.Text != "realtime_route_unavailable" {
				t.Fatalf("close=%d/%q", closeErr.Code, closeErr.Text)
			}
			return
		}
		if env.Name == NameAuthOK || env.Name == NameSubscribed {
			t.Fatalf("received readiness event %q before member-route failure close", env.Name)
		}
	}
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

func waitInt(t *testing.T, ch <-chan int, name string) int {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
		return 0
	}
}

func waitBool(t *testing.T, ch <-chan bool, name string) bool {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
		return false
	}
}

func waitForCondition(t *testing.T, name string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", name)
		}
		runtime.Gosched()
	}
}
