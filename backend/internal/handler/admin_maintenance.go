package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/maintenance"
	"caipiao/backend/internal/ws"
)

func (h *Handler) AdminGetMaintenance(w http.ResponseWriter, r *http.Request) {
	if h.maintenance == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	state, err := h.maintenance.AdminGet(r.Context())
	if err != nil {
		h.handleMaintenanceErr(w, err)
		return
	}
	apix.OK(w, state)
}

func (h *Handler) AdminSaveMaintenance(w http.ResponseWriter, r *http.Request) {
	if h.maintenance == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	var body maintenance.AdminState
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	prev, _ := h.maintenance.AdminGet(r.Context())
	state, err := h.maintenance.AdminSave(r.Context(), body)
	if err != nil {
		h.handleMaintenanceErr(w, err)
		return
	}
	if prev.Enabled && !state.Enabled {
		h.scheduleMaintenanceResume()
	}
	if h.wsHub != nil {
		if pub, pubErr := h.maintenance.PublicGet(r.Context()); pubErr == nil {
			payload := ws.MaintenanceChangedPayload{
				Enabled:             pub.Enabled,
				Title:               pub.Title,
				Message:             pub.Message,
				PopupAnnouncementID: pub.PopupAnnouncementID,
			}
			if pub.PopupAnnouncement != nil {
				payload.PopupAnnouncement = pub.PopupAnnouncement
			}
			ws.PublishMaintenance(h.wsHub, payload)
		}
	}
	h.writeAudit(r, "更新系统维护配置")
	apix.OK(w, state)
}

func (h *Handler) handleMaintenanceErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, maintenance.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, maintenance.ErrInvalid):
		apix.Validation(w, "请求参数无效")
	default:
		apix.Internal(w)
	}
}
