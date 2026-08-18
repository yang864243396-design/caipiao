package cloudrealtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"caipiao/backend/internal/realtimebus"
)

func TestNATSClusterIsolatesMembersAndRecoversAfterAPIReconnect(t *testing.T) {
	url := os.Getenv("NATS_TEST_URL")
	if url == "" {
		t.Skip("NATS_TEST_URL not set")
	}

	worker, err := realtimebus.NewNATS(realtimebus.NATSConfig{
		URL:           url,
		Name:          "cloud-realtime-cluster-test-worker",
		ReconnectWait: 25 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = worker.Close() })
	waitForConnected(t, worker, 5*time.Second)

	proxyURL, proxy := newClusterFaultProxy(t, url)
	api := newClusterTestAPI(t, proxyURL)
	t.Cleanup(func() { _ = api.Close() })
	prefix := fmt.Sprintf("caipiao-test-%d", time.Now().UnixNano())
	member7Subject, err := SchemeSubject(prefix, 7)
	if err != nil {
		t.Fatal(err)
	}
	member8Subject, err := SchemeSubject(prefix, 8)
	if err != nil {
		t.Fatal(err)
	}

	first := subscribeClusterMember(t, api, member7Subject)
	waitForClusterSubscription(t, worker, member7Subject, first)
	if err := worker.Publish(context.Background(), member8Subject, []byte(`{"schemaVersion":1,"member":8}`)); err != nil {
		t.Fatal(err)
	}
	assertNoClusterMessage(t, first, 150*time.Millisecond)
	assertSingleClusterMessage(t, worker, member7Subject, []byte(`{"schemaVersion":1,"member":7,"sequence":1}`), first)

	var disconnected atomic.Bool
	var reconnected atomic.Bool
	api.OnConnectionChange(func(connected bool) {
		if !connected {
			disconnected.Store(true)
			return
		}
		if disconnected.Load() {
			reconnected.Store(true)
		}
	})
	beforeReconnects := api.Diagnostics().Reconnects
	waitForCondition(t, 5*time.Second, func() bool { return proxy.ActiveConnections() > 0 }, "fault proxy did not observe the API connection")
	proxy.DropConnections()
	waitForCondition(t, 5*time.Second, disconnected.Load, "same API bus did not report disconnection")
	waitForCondition(t, 5*time.Second, func() bool {
		diagnostics := api.Diagnostics()
		return reconnected.Load() && diagnostics.Connected && diagnostics.Reconnects > beforeReconnects
	}, "same API bus did not automatically reconnect")

	waitForClusterSubscription(t, worker, member7Subject, first)
	assertSingleClusterMessage(t, worker, member7Subject, []byte(`{"schemaVersion":1,"member":7,"sequence":2}`), first)
}

type clusterFaultProxy struct {
	listener net.Listener
	target   string

	mu          sync.Mutex
	connections map[*clusterProxyConnection]struct{}
	wg          sync.WaitGroup
}

type clusterProxyConnection struct {
	client   net.Conn
	upstream net.Conn
	once     sync.Once
}

func newClusterFaultProxy(t *testing.T, rawURL string) (string, *clusterFaultProxy) {
	t.Helper()
	if strings.Contains(rawURL, ",") {
		t.Skip("automatic reconnect proxy does not support multi-URL NATS_TEST_URL")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal("NATS_TEST_URL is not a valid URL")
	}
	if parsed.Scheme != "nats" {
		t.Skipf("automatic reconnect proxy supports only plaintext nats:// URLs, got %q", parsed.Scheme)
	}
	if strings.EqualFold(parsed.Query().Get("tls_required"), "true") || strings.EqualFold(parsed.Query().Get("tls"), "true") {
		t.Skip("automatic reconnect proxy does not support TLS-enabled NATS_TEST_URL query options")
	}
	host := parsed.Hostname()
	if host == "" {
		t.Skip("automatic reconnect proxy requires a NATS_TEST_URL hostname")
	}
	port := parsed.Port()
	if port == "" {
		port = "4222"
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start NATS fault proxy: %v", err)
	}
	proxy := &clusterFaultProxy{
		listener:    listener,
		target:      net.JoinHostPort(host, port),
		connections: make(map[*clusterProxyConnection]struct{}),
	}
	proxy.wg.Add(1)
	go proxy.accept()
	t.Cleanup(proxy.Close)

	proxied := *parsed
	proxied.Host = listener.Addr().String()
	return proxied.String(), proxy
}

func (p *clusterFaultProxy) accept() {
	defer p.wg.Done()
	for {
		client, err := p.listener.Accept()
		if err != nil {
			return
		}
		upstream, err := net.DialTimeout("tcp", p.target, 5*time.Second)
		if err != nil {
			_ = client.Close()
			continue
		}
		connection := &clusterProxyConnection{client: client, upstream: upstream}
		p.mu.Lock()
		p.connections[connection] = struct{}{}
		p.mu.Unlock()
		p.wg.Add(1)
		go p.forward(connection)
	}
}

func (p *clusterFaultProxy) forward(connection *clusterProxyConnection) {
	defer p.wg.Done()
	done := make(chan struct{}, 2)
	copyDirection := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go copyDirection(connection.upstream, connection.client)
	go copyDirection(connection.client, connection.upstream)
	<-done
	connection.Close()
	<-done
	p.mu.Lock()
	delete(p.connections, connection)
	p.mu.Unlock()
}

func (p *clusterFaultProxy) ActiveConnections() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.connections)
}

func (p *clusterFaultProxy) DropConnections() {
	p.mu.Lock()
	connections := make([]*clusterProxyConnection, 0, len(p.connections))
	for connection := range p.connections {
		connections = append(connections, connection)
	}
	p.mu.Unlock()
	for _, connection := range connections {
		connection.Close()
	}
}

func (p *clusterFaultProxy) Close() {
	_ = p.listener.Close()
	p.DropConnections()
	p.wg.Wait()
}

func (c *clusterProxyConnection) Close() {
	c.once.Do(func() {
		_ = c.client.Close()
		_ = c.upstream.Close()
	})
}

func newClusterTestAPI(t *testing.T, url string) *realtimebus.NATS {
	t.Helper()
	bus, err := realtimebus.NewNATS(realtimebus.NATSConfig{
		URL:           url,
		Name:          fmt.Sprintf("cloud-realtime-cluster-test-api-%d", time.Now().UnixNano()),
		ReconnectWait: 25 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForConnected(t, bus, 5*time.Second)
	return bus
}

func subscribeClusterMember(t *testing.T, bus realtimebus.Bus, subject string) <-chan string {
	t.Helper()
	received := make(chan string, 4)
	subscription, err := bus.Subscribe(subject, func(_ string, payload []byte) {
		received <- string(payload)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = subscription.Unsubscribe() })
	return received
}

func waitForClusterSubscription(t *testing.T, publisher realtimebus.Bus, subject string, received <-chan string) {
	t.Helper()
	payload := []byte(`{"schemaVersion":1,"probe":true}`)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := publisher.Publish(context.Background(), subject, payload); err != nil {
			t.Fatal(err)
		}
		select {
		case got := <-received:
			if got != string(payload) {
				t.Fatalf("got %q want %q", got, payload)
			}
			drainClusterReadinessProbes(t, received, payload)
			return
		case <-time.After(25 * time.Millisecond):
		}
	}
	t.Fatal("timed out waiting for member subscription readiness")

}

func drainClusterReadinessProbes(t *testing.T, received <-chan string, payload []byte) {
	t.Helper()
	timer := time.NewTimer(50 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case got := <-received:
			if got != string(payload) {
				t.Fatalf("got %q while draining readiness probes, want %q", got, payload)
			}
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(50 * time.Millisecond)
		case <-timer.C:
			return
		}
	}
}

func assertSingleClusterMessage(t *testing.T, publisher realtimebus.Bus, subject string, payload []byte, received <-chan string) {
	t.Helper()
	if err := publisher.Publish(context.Background(), subject, payload); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-received:
		if got != string(payload) {
			t.Fatalf("got %q want %q", got, payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for member-scoped snapshot")
	}
	assertNoClusterMessage(t, received, 150*time.Millisecond)
}

func assertNoClusterMessage(t *testing.T, received <-chan string, duration time.Duration) {
	t.Helper()
	select {
	case got := <-received:
		t.Fatalf("unexpected extra snapshot: %s", got)
	case <-time.After(duration):
	}
}

func waitForConnected(t *testing.T, bus realtimebus.Bus, timeout time.Duration) {
	t.Helper()
	waitForCondition(t, timeout, func() bool { return bus.Diagnostics().Connected }, "NATS client did not connect")
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(message)
}
