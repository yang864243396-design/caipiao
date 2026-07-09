package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/content"
)

func (h *Handler) requireContent(w http.ResponseWriter) *content.Service {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return nil
	}
	return h.content
}

func (h *Handler) AdminListAnnouncements(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	items, err := svc.AdminListAnnouncements(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) AdminSaveAnnouncement(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var body content.AdminAnnouncement
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminSaveAnnouncement(r.Context(), body)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("编辑公告 %s", item.ID))
	apix.OK(w, item)
}

func (h *Handler) AdminDeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id := r.PathValue("id")
	if err := svc.AdminDeleteAnnouncement(r.Context(), id); err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("删除公告 %s", id))
	apix.OK(w, map[string]string{"id": id})
}

func (h *Handler) AdminSetAnnouncementPinned(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id := r.PathValue("id")
	var body struct {
		Pinned bool `json:"pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminSetAnnouncementPinned(r.Context(), id, body.Pinned)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	action := "取消置顶"
	if body.Pinned {
		action = "置顶"
	}
	h.writeAudit(r, fmt.Sprintf("%s公告 %s", action, item.ID))
	apix.OK(w, item)
}

func (h *Handler) AdminListFaqArticles(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	items, err := svc.AdminListFaqArticles(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) AdminSaveFaqArticle(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var body content.AdminFaqArticle
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminSaveFaqArticle(r.Context(), body)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, item)
}

func (h *Handler) AdminDeleteFaqArticle(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id := r.PathValue("id")
	if err := svc.AdminDeleteFaqArticle(r.Context(), id); err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]string{"id": id})
}

func (h *Handler) AdminListHelpArticles(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	items, err := svc.AdminListHelpArticles(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) AdminSaveHelpArticle(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var body content.AdminHelpArticle
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminSaveHelpArticle(r.Context(), body)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, item)
}

func (h *Handler) AdminDeleteHelpArticle(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	id := r.PathValue("id")
	if err := svc.AdminDeleteHelpArticle(r.Context(), id); err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]string{"id": id})
}

func (h *Handler) AdminContentBundle(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	if r.Method != http.MethodGet {
		apix.Fail(w, http.StatusMethodNotAllowed, apix.CodeValidation, "不支持的请求方法")
		return
	}
	ctx := r.Context()
	announcements, err := svc.AdminListAnnouncements(ctx)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	faqArticles, err := svc.AdminListFaqArticles(ctx)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	helpArticles, err := svc.AdminListHelpArticles(ctx)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	lobbySlots, err := svc.AdminListLobbySlots(ctx)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{
		"announcements": announcements,
		"faqArticles":   faqArticles,
		"helpArticles":  helpArticles,
		"lobbySlots":    lobbySlots,
	})
}

func (h *Handler) AdminSaveLobbySlot(w http.ResponseWriter, r *http.Request) {
	svc := h.requireContent(w)
	if svc == nil {
		return
	}
	var body content.AdminLobbySlot
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := svc.AdminSaveLobbySlot(r.Context(), body)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("编辑大厅运营位 %s", item.SlotKey))
	apix.OK(w, item)
}

