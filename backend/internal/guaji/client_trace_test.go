package guaji

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptrace"
	"testing"
	"time"
)

type traceRoundTripFunc func(*http.Request) (*http.Response, error)

type traceDefinitelyNotSent interface {
	DefinitelyNotSent() bool
}

func (fn traceRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newTraceTestClient(transport http.RoundTripper) *Client {
	const baseURL = "https://guaji.test"
	return &Client{
		cfg: Config{
			Enabled: true, HTTPBase: baseURL, AuthBase: baseURL, WSBase: "wss://guaji.test", HTTPTimeout: time.Second,
		},
		http: &http.Client{Transport: transport, Timeout: time.Second},
	}
}

func TestDoJSONRawWroteRequestCallbackErrorIsDefinitelyNotSent(t *testing.T) {
	transportErr := errors.New("tls prewrite failure")
	client := newTraceTestClient(traceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		trace := httptrace.ContextClientTrace(req.Context())
		if trace == nil || trace.WroteRequest == nil {
			t.Fatal("request write trace is not installed")
		}
		trace.WroteRequest(httptrace.WroteRequestInfo{Err: transportErr})
		return nil, transportErr
	}))

	_, _, err := client.doJSONRaw(context.Background(), http.MethodPost, client.cfg.HTTPBase, "/api/web_bets/lott", "token", map[string]int{"game_id": 74})
	var marker traceDefinitelyNotSent
	if !errors.As(err, &marker) || !marker.DefinitelyNotSent() {
		t.Fatalf("expected definitely-not-sent marker, got %T %v", err, err)
	}
	if !errors.Is(err, transportErr) {
		t.Fatalf("transport error identity lost: %v", err)
	}
	var detail *requestTransportError
	if !errors.As(err, &detail) {
		t.Fatalf("structured transport detail missing: %T %v", err, err)
	}
	if detail.Operation != "POST /api/web_bets/lott" || detail.Phase != "write" || detail.RequestWritten {
		t.Fatalf("transport detail=%+v", detail)
	}
}

func TestDoJSONRawSuccessfulWriteCallbackRemainsAmbiguous(t *testing.T) {
	transportErr := errors.New("response timeout after request write")
	client := newTraceTestClient(traceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		trace := httptrace.ContextClientTrace(req.Context())
		if trace == nil || trace.WroteRequest == nil {
			t.Fatal("request write trace is not installed")
		}
		trace.WroteRequest(httptrace.WroteRequestInfo{})
		return nil, transportErr
	}))

	_, _, err := client.doJSONRaw(context.Background(), http.MethodPost, client.cfg.HTTPBase, "/api/web_bets/lott", "token", map[string]int{"game_id": 74})
	var marker traceDefinitelyNotSent
	if errors.As(err, &marker) && marker.DefinitelyNotSent() {
		t.Fatalf("written request was classified definitely not sent: %T %v", err, err)
	}
	if !errors.Is(err, transportErr) {
		t.Fatalf("transport error identity lost: %v", err)
	}
	var detail *requestTransportError
	if !errors.As(err, &detail) {
		t.Fatalf("structured transport detail missing: %T %v", err, err)
	}
	if detail.Operation != "POST /api/web_bets/lott" || detail.Phase != "response" || !detail.RequestWritten {
		t.Fatalf("transport detail=%+v", detail)
	}
}
