package handler

import (
	"net/http"
	"regexp"
	"strings"

	"caipiao/backend/internal/apix"
)

type realtimeDiagnosticsProvider interface {
	Snapshot() map[string]any
}

var cloudRealtimeDiagnosticSections = [...]string{"bus", "publisher", "hub", "scanner"}

var (
	diagnosticURLUserInfo   = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^/\s@]+@`)
	diagnosticLabeledSecret = regexp.MustCompile(`(?i)(?:password|passwd|pwd|token|credentials?|secret)\s*[:=]\s*[^,\s;]+`)
	diagnosticProviderBody  = regexp.MustCompile(`(?i)(?:raw\s+)?(?:provider\s+)?body\s*[:=]\s*(?:\{.*\}|\[.*\]|[^,\s;]+)`)
)

func (h *Handler) AdminCloudRealtimeDiagnostics(w http.ResponseWriter, _ *http.Request) {
	apix.OK(w, h.safeCloudRealtimeSnapshot())
}

func (h *Handler) safeCloudRealtimeSnapshot() map[string]any {
	snapshot := map[string]any{"enabled": false}
	if h != nil && h.cloudRealtimeDiagnostics != nil {
		for key, value := range h.cloudRealtimeDiagnostics.Snapshot() {
			if diagnosticSecretKey(key) {
				continue
			}
			snapshot[key] = sanitizeDiagnosticValue(value)
		}
	}
	for _, section := range cloudRealtimeDiagnosticSections {
		if _, ok := snapshot[section]; !ok {
			snapshot[section] = map[string]any{}
		}
	}
	return snapshot
}

func (h *Handler) addCloudRealtimeHealth(payload map[string]any) {
	if h == nil || h.cloudRealtimeDiagnostics == nil {
		return
	}
	snapshot := h.cloudRealtimeDiagnostics.Snapshot()
	enabled, _ := snapshot["enabled"].(bool)
	if !enabled {
		return
	}
	connected := false
	if bus, ok := snapshot["bus"].(map[string]any); ok {
		connected, _ = bus["connected"].(bool)
	}
	if connected {
		payload["cloudRealtime"] = map[string]any{"status": "up"}
		return
	}
	payload["cloudRealtime"] = map[string]any{
		"status":    "degraded",
		"component": "realtime_bus",
	}
}

func sanitizeDiagnosticValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		clean := make(map[string]any, len(typed))
		for key, child := range typed {
			if diagnosticSecretKey(key) {
				continue
			}
			clean[key] = sanitizeDiagnosticValue(child)
		}
		return clean
	case []any:
		clean := make([]any, len(typed))
		for index, child := range typed {
			clean[index] = sanitizeDiagnosticValue(child)
		}
		return clean
	case string:
		return sanitizeDiagnosticString(typed)
	default:
		return value
	}
}

func sanitizeDiagnosticString(value string) string {
	value = diagnosticURLUserInfo.ReplaceAllString(value, `${1}[redacted]@`)
	value = diagnosticLabeledSecret.ReplaceAllString(value, "[redacted]")
	return diagnosticProviderBody.ReplaceAllString(value, "body=[redacted]")
}

func diagnosticSecretKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, fragment := range []string{"password", "token", "credential", "secret"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}
