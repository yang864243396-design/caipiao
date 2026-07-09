package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/audit"
)

func (h *Handler) AdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	if h.audit == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.audit.List(r.Context(), queryInt(r, "limit", 100))
	if err != nil {
		h.handleAuditErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) handleAuditErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, audit.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	default:
		apix.Internal(w)
	}
}
