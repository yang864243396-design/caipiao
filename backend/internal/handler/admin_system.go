package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/content"
	"caipiao/backend/internal/middleware"
)

func (h *Handler) AdminListRoles(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	items, err := svc.AdminListRoles(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) AdminSaveRole(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var body content.AdminRole
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminSaveRole(r.Context(), body)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("保存角色 %s", item.ID))
	apix.OK(w, item)
}

func (h *Handler) AdminDeleteRole(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id := r.PathValue("id")
	if err := svc.AdminDeleteRole(r.Context(), id); err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("删除角色 %s", id))
	apix.OK(w, map[string]string{"id": id})
}

func (h *Handler) AdminListUsers(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	items, err := svc.AdminListUsers(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var body content.AdminUserSaveInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminCreateUser(r.Context(), body)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("创建 Admin 账号 %s", item.Account))
	apix.OK(w, item)
}

func (h *Handler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id, err := parseAdminUserID(r.PathValue("id"))
	if err != nil || id <= 0 {
		apix.Validation(w, "无效的用户 id")
		return
	}
	var body content.AdminUserSaveInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminUpdateUser(r.Context(), id, body)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("更新 Admin 账号 %s", item.Account))
	apix.OK(w, item)
}

func (h *Handler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id, err := parseAdminUserID(r.PathValue("id"))
	if err != nil || id <= 0 {
		apix.Validation(w, "无效的用户 id")
		return
	}
	currentAccount := ""
	if claims, ok := middleware.ClaimsFromContext(r.Context()); ok {
		currentAccount = claims.Subject
	}
	if err := svc.AdminDeleteUser(r.Context(), id, currentAccount); err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("删除 Admin 账号 id=%d", id))
	apix.OK(w, map[string]int64{"id": id})
}

func parseAdminUserID(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty")
	}
	return strconv.ParseInt(raw, 10, 64)
}
