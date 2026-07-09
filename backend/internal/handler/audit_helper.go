package handler

import (
	"net"
	"net/http"
	"strings"

	"caipiao/backend/internal/middleware"
)

func (h *Handler) writeAudit(r *http.Request, action string) {
	if h.audit == nil {
		return
	}
	actor := "admin"
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok && claims.Subject != "" {
		actor = claims.Subject
	}
	_ = h.audit.Append(r.Context(), actor, action, clientIP(r))
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		return r.RemoteAddr
	}
	return host
}
