package guaji_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"time"

	"caipiao/backend/internal/guaji"
)

func TestLoginSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 200,
			"data": map[string]any{
				"token":         "tok-abc",
				"refresh_token": "ref-xyz",
				"username":      "testcq01",
				"token_type":    "Bearer",
			},
		})
	}))
	defer srv.Close()

	c := guaji.NewClient(guaji.Config{
		Enabled:     true,
		HTTPBase:    srv.URL,
		AuthBase:    srv.URL,
		WSBase:      "wss://example.test",
		Origin:      srv.URL,
		Referer:     srv.URL + "/",
		IsAI:        true,
		HTTPTimeout: 5 * time.Second,
	})

	res, err := c.Login(context.Background(), "testcq01", "secret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.Token != "tok-abc" || res.Username != "testcq01" {
		t.Fatalf("unexpected login result: %+v", res)
	}
}

func TestLoginMFARequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    guaji.CodeMFARequired,
			"message": "need mfa",
			"extra": map[string]any{
				"login_key": "lk-1",
			},
		})
	}))
	defer srv.Close()

	c := guaji.NewClient(guaji.Config{
		Enabled:     true,
		HTTPBase:    srv.URL,
		AuthBase:    srv.URL,
		WSBase:      "wss://example.test",
		HTTPTimeout: 5 * time.Second,
	})

	_, err := c.Login(context.Background(), "u", "p")
	if err == nil {
		t.Fatal("expected mfa error")
	}
	mfa, ok := err.(*guaji.MFARequiredError)
	if !ok {
		t.Fatalf("expected MFARequiredError, got %T: %v", err, err)
	}
	if mfa.LoginKey != "lk-1" {
		t.Fatalf("login_key=%q", mfa.LoginKey)
	}
}

func TestUserInfoCNYBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/i/info" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok-abc" {
			t.Fatalf("authorization=%q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 201,
			"data": map[string]any{
				"id":       1,
				"username": "no8899",
				"account": map[string]any{
					"balance_cny": 123.45,
				},
			},
		})
	}))
	defer srv.Close()

	c := guaji.NewClient(guaji.Config{
		Enabled:     true,
		HTTPBase:    srv.URL,
		AuthBase:    srv.URL,
		WSBase:      "wss://example.test",
		HTTPTimeout: 5 * time.Second,
	})

	info, err := c.UserInfo(context.Background(), "tok-abc")
	if err != nil {
		t.Fatalf("user info: %v", err)
	}
	if info.CNYBalance() != 123.45 {
		t.Fatalf("balance=%v", info.CNYBalance())
	}
}

func TestDefaultAuthBase(t *testing.T) {
	t.Setenv("GUAJI_AUTH_BASE", "")
	t.Setenv("GUAJI_HTTP_BASE", "https://www.v6hs1.com")
	cfg := guaji.LoadConfigFromEnv()
	if cfg.AuthBase != "https://www.v6hs1.com" {
		t.Fatalf("auth base=%q", cfg.AuthBase)
	}
}

func TestPingAuthEndpointReachableOnAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    40001,
			"message": "invalid username",
		})
	}))
	defer srv.Close()

	c := guaji.NewClient(guaji.Config{
		Enabled:     true,
		HTTPBase:    srv.URL,
		AuthBase:    srv.URL,
		WSBase:      "wss://example.test",
		HTTPTimeout: 5 * time.Second,
	})
	if err := c.PingAuthEndpoint(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestConfigValidRequiresAuthBaseOnProd(t *testing.T) {
	cfg := guaji.Config{
		Enabled:  true,
		HTTPBase: "https://s9-xia.5rf9q.com",
		AuthBase: "",
		WSBase:   "wss://s9-ws.5rf9q.com",
	}
	if err := cfg.Valid(); err == nil {
		t.Fatal("expected misconfigured error")
	}
}
