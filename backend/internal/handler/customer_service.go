package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/content"
)

func (h *Handler) ClientCustomerServiceAgents(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	items, err := svc.ListEnabledCustomerServiceAgents(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) AdminListCustomerServiceAgents(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	items, err := svc.ListCustomerServiceAgents(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) AdminSaveCustomerServiceAgent(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var body content.CustomerServiceAgent
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.SaveCustomerServiceAgent(r.Context(), body)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("编辑客服 %s", item.ID))
	apix.OK(w, item)
}

func (h *Handler) AdminDeleteCustomerServiceAgent(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id := r.PathValue("id")
	if err := svc.DeleteCustomerServiceAgent(r.Context(), id); err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("删除客服 %s", id))
	apix.OK(w, map[string]string{"id": id})
}
