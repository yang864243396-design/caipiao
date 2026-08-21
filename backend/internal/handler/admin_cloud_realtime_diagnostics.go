package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"caipiao/backend/internal/apix"
)

type realtimeDiagnosticsProvider interface {
	Snapshot() map[string]any
}

var cloudRealtimeDiagnosticSections = [...]string{"bus", "publisher", "hub", "scanner"}

var (
	diagnosticURLUserInfo   = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)[^/\s@]+@`)
	diagnosticLabeledSecret = regexp.MustCompile(`(?i)(?:password|passwd|pwd|token|credentials?|secret)\s*[:=]\s*[^,\s;]+`)
)

const diagnosticRedacted = "[redacted]"

var diagnosticBodyLabels = [...]string{
	"provider response body", "provider_response_body", "provider-response-body", "providerresponsebody",
	"provider response", "provider_response", "provider-response", "providerresponse",
	"provider body", "provider_body", "provider-body", "providerbody",
	"raw response", "raw_response", "raw-response", "rawresponse",
	"response body", "response_body", "response-body", "responsebody",
	"raw body", "raw_body", "raw-body", "rawbody",
	"response", "body",
}

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
			if diagnosticBodyKey(key) {
				clean[key] = diagnosticRedacted
				continue
			}
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
	if sanitized, ok := sanitizeDiagnosticJSON(value); ok {
		return sanitized
	}
	return sanitizeDiagnosticPlainText(value)
}

func sanitizeDiagnosticJSON(value string) (string, bool) {
	var decoded any
	if json.Unmarshal([]byte(value), &decoded) != nil {
		return "", false
	}
	sanitized, err := json.Marshal(sanitizeDiagnosticJSONValue(decoded))
	if err != nil {
		return "", false
	}
	return string(sanitized), true
}

func sanitizeDiagnosticJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		clean := make(map[string]any, len(typed))
		for key, child := range typed {
			switch {
			case diagnosticBodyKey(key), diagnosticSecretKey(key):
				clean[key] = diagnosticRedacted
			default:
				clean[key] = sanitizeDiagnosticJSONValue(child)
			}
		}
		return clean
	case []any:
		clean := make([]any, len(typed))
		for index, child := range typed {
			clean[index] = sanitizeDiagnosticJSONValue(child)
		}
		return clean
	case string:
		return sanitizeDiagnosticPlainText(typed)
	default:
		return value
	}
}

func sanitizeDiagnosticPlainText(value string) string {
	value = redactDiagnosticBodySuffix(value)
	value = diagnosticURLUserInfo.ReplaceAllString(value, `${1}[redacted]@`)
	return diagnosticLabeledSecret.ReplaceAllString(value, diagnosticRedacted)
}

func redactDiagnosticBodySuffix(value string) string {
	lower := strings.ToLower(value)
	redactAt := -1
	for _, label := range diagnosticBodyLabels {
		searchFrom := 0
		for searchFrom < len(lower) {
			relative := strings.Index(lower[searchFrom:], label)
			if relative < 0 {
				break
			}
			start := searchFrom + relative
			end := start + len(label)
			if diagnosticLabelBoundary(lower, start, end) {
				separator := end
				for separator < len(lower) && (lower[separator] == ' ' || lower[separator] == '\t') {
					separator++
				}
				if separator < len(lower) && (lower[separator] == ':' || lower[separator] == '=') {
					if redactAt < 0 || separator+1 < redactAt {
						redactAt = separator + 1
					}
					break
				}
			}
			searchFrom = start + 1
		}
	}
	if redactAt < 0 {
		return value
	}
	return value[:redactAt] + diagnosticRedacted
}

func diagnosticLabelBoundary(value string, start, end int) bool {
	if start > 0 && diagnosticWordByte(value[start-1]) {
		return false
	}
	return end >= len(value) || !diagnosticWordByte(value[end])
}

func diagnosticWordByte(value byte) bool {
	return value == '_' || unicode.IsLetter(rune(value)) || unicode.IsDigit(rune(value))
}

func diagnosticBodyKey(key string) bool {
	normalized := strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(key))
	switch normalized {
	case "body", "raw", "rawbody", "response", "rawresponse", "responsebody", "providerbody", "providerresponse", "providerresponsebody":
		return true
	default:
		return false
	}
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
