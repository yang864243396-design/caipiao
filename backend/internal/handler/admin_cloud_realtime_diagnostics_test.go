package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeRealtimeDiagnostics struct {
	snapshot map[string]any
}

func (f fakeRealtimeDiagnostics) Snapshot() map[string]any {
	return f.snapshot
}

func TestAdminCloudRealtimeDiagnosticsReturnsSafeReadOnlySnapshot(t *testing.T) {
	snapshot := map[string]any{
		"enabled": true,
		"bus": map[string]any{
			"connected": false,
			"lastError": "dial nats://diag-user:diag-pass@127.0.0.1 token=raw-token",
			"password":  "must-not-leak",
		},
		"publisher": map[string]any{"schemeQueueSize": 2, "token": "must-not-leak"},
		"hub":       map[string]any{"connections": 3},
		"scanner":   map[string]any{"leader": false, "credentials": "must-not-leak"},
	}
	h := &Handler{cloudRealtimeDiagnostics: fakeRealtimeDiagnostics{snapshot: snapshot}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/admin/diagnostics/cloud-realtime", nil)

	h.AdminCloudRealtimeDiagnostics(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, section := range []string{"bus", "publisher", "hub", "scanner"} {
		if !strings.Contains(body, `"`+section+`"`) {
			t.Fatalf("missing %q section in %s", section, body)
		}
	}
	lowerBody := strings.ToLower(body)
	for _, secret := range []string{"must-not-leak", "password", "token", "credentials", "diag-user", "diag-pass", "raw-token"} {
		if strings.Contains(lowerBody, secret) {
			t.Fatalf("diagnostic secret %q leaked in %s", secret, body)
		}
	}
	if snapshot["bus"].(map[string]any)["password"] != "must-not-leak" {
		t.Fatal("handler mutated the provider snapshot")
	}
}

func TestHealthReportsCloudRealtimeDegradedWithoutFailingApplication(t *testing.T) {
	h := &Handler{cloudRealtimeDiagnostics: fakeRealtimeDiagnostics{snapshot: map[string]any{
		"enabled": true,
		"bus":     map[string]any{"kind": "nats", "connected": false},
	}}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Health(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if response.Data["status"] != "ok" {
		t.Fatalf("application status=%v, want ok", response.Data["status"])
	}
	realtime, ok := response.Data["cloudRealtime"].(map[string]any)
	if !ok || realtime["status"] != "degraded" || realtime["component"] != "realtime_bus" {
		t.Fatalf("cloudRealtime=%#v, want degraded realtime_bus", response.Data["cloudRealtime"])
	}
}

func TestHealthOmitsCloudRealtimeWhenRolloutDisabled(t *testing.T) {
	h := &Handler{cloudRealtimeDiagnostics: fakeRealtimeDiagnostics{snapshot: map[string]any{
		"enabled": false,
		"bus":     map[string]any{"connected": false},
	}}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)

	h.Health(w, r)

	if strings.Contains(w.Body.String(), "cloudRealtime") {
		t.Fatalf("disabled rollout changed legacy health payload: %s", w.Body.String())
	}
}
