package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/middleware"
	"caipiao/backend/internal/schemebetting"
)

type SchemeBettingActionService interface {
	EnableEventScheme(context.Context, string, string, string) error
	RearmEventScheme(context.Context, string, string, string) error
	CancelEventBet(context.Context, int64, string, string) error
	ResolveUnknownEventBet(context.Context, int64, string, string, schemebetting.UnknownResolution) error
}

type schemeBettingActionRequest struct {
	Reason string `json:"reason"`
}

type schemeBettingResolveRequest struct {
	Reason string `json:"reason"`
	schemebetting.UnknownResolution
}

func (h *Handler) AdminSchemeBettingEnable(w http.ResponseWriter, r *http.Request) {
	if h.schemeBettingActions == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u65b9\u6848\u6295\u6ce8\u52a8\u4f5c\u670d\u52a1\u672a\u5c31\u7eea")
		return
	}
	request, actor, ok := decodeSchemeBettingAction(w, r)
	if !ok {
		return
	}
	if err := h.schemeBettingActions.EnableEventScheme(r.Context(), strings.TrimSpace(r.PathValue("schemeId")), actor, request.Reason); err != nil {
		writeSchemeBettingActionError(w, err)
		return
	}
	h.writeAudit(r, "scheme_betting_enable_event")
	apix.OK(w, map[string]any{"enabled": true})
}

func (h *Handler) AdminSchemeBettingRearm(w http.ResponseWriter, r *http.Request) {
	if h.schemeBettingActions == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u65b9\u6848\u6295\u6ce8\u52a8\u4f5c\u670d\u52a1\u672a\u5c31\u7eea")
		return
	}
	request, actor, ok := decodeSchemeBettingAction(w, r)
	if !ok {
		return
	}
	if err := h.schemeBettingActions.RearmEventScheme(r.Context(), strings.TrimSpace(r.PathValue("schemeId")), actor, request.Reason); err != nil {
		writeSchemeBettingActionError(w, err)
		return
	}
	h.writeAudit(r, "scheme_betting_rearm")
	apix.OK(w, map[string]any{"rearmed": true})
}

func (h *Handler) AdminSchemeBettingCancel(w http.ResponseWriter, r *http.Request) {
	if h.schemeBettingActions == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u65b9\u6848\u6295\u6ce8\u52a8\u4f5c\u670d\u52a1\u672a\u5c31\u7eea")
		return
	}
	outboxID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("outboxId")), 10, 64)
	if err != nil || outboxID <= 0 {
		apix.Validation(w, "\u65e0\u6548\u7684 outbox ID")
		return
	}
	request, actor, ok := decodeSchemeBettingAction(w, r)
	if !ok {
		return
	}
	if err := h.schemeBettingActions.CancelEventBet(r.Context(), outboxID, actor, request.Reason); err != nil {
		writeSchemeBettingActionError(w, err)
		return
	}
	h.writeAudit(r, "scheme_betting_cancel")
	apix.OK(w, map[string]any{"cancelled": true})
}

func (h *Handler) AdminSchemeBettingResolveUnknown(w http.ResponseWriter, r *http.Request) {
	if h.schemeBettingActions == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u65b9\u6848\u6295\u6ce8\u52a8\u4f5c\u670d\u52a1\u672a\u5c31\u7eea")
		return
	}
	outboxID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("outboxId")), 10, 64)
	if err != nil || outboxID <= 0 {
		apix.Validation(w, "\u65e0\u6548\u7684 outbox ID")
		return
	}
	var request schemeBettingResolveRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		apix.Validation(w, "\u8bf7\u63d0\u4f9b\u5b8c\u6574\u7684\u5bf9\u8d26\u7ed3\u679c")
		return
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if len([]rune(request.Reason)) < 4 {
		apix.Validation(w, "\u64cd\u4f5c\u7406\u7531\u81f3\u5c11 4 \u4e2a\u5b57\u7b26")
		return
	}
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		apix.Unauthorized(w, "")
		return
	}
	if strings.TrimSpace(claims.AdminRoleID) != "r_super" {
		apix.Fail(w, http.StatusForbidden, apix.CodeForbidden, "\u4ec5\u8d85\u7ea7\u7ba1\u7406\u5458\u53ef\u5904\u7406\u6295\u6ce8\u5bf9\u8d26")
		return
	}
	if err := h.schemeBettingActions.ResolveUnknownEventBet(
		r.Context(), outboxID, strings.TrimSpace(claims.Subject), request.Reason, request.UnknownResolution,
	); err != nil {
		writeSchemeBettingActionError(w, err)
		return
	}
	h.writeAudit(r, "scheme_betting_resolve_unknown")
	apix.OK(w, map[string]any{"resolved": true})
}

func decodeSchemeBettingAction(w http.ResponseWriter, r *http.Request) (schemeBettingActionRequest, string, bool) {
	var request schemeBettingActionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		apix.Validation(w, "\u8bf7\u63d0\u4f9b\u64cd\u4f5c\u7406\u7531")
		return request, "", false
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if len([]rune(request.Reason)) < 4 {
		apix.Validation(w, "\u64cd\u4f5c\u7406\u7531\u81f3\u5c11 4 \u4e2a\u5b57\u7b26")
		return request, "", false
	}
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		apix.Unauthorized(w, "")
		return request, "", false
	}
	if strings.TrimSpace(claims.AdminRoleID) != "r_super" {
		apix.Fail(w, http.StatusForbidden, apix.CodeForbidden, "\u4ec5\u8d85\u7ea7\u7ba1\u7406\u5458\u53ef\u6267\u884c\u65b9\u6848\u6295\u6ce8\u64cd\u4f5c")
		return request, "", false
	}
	return request, strings.TrimSpace(claims.Subject), true
}

func writeSchemeBettingActionError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		apix.Internal(w)
		return
	}
	if strings.Contains(message, "required") || strings.Contains(message, "requires") ||
		strings.Contains(message, "reason") || strings.Contains(message, "evidence") ||
		strings.Contains(message, "must") || strings.Contains(message, "differs") {
		apix.Validation(w, message)
		return
	}
	apix.Fail(w, http.StatusConflict, apix.CodeValidation, message)
}
