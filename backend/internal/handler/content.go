package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/content"
	"caipiao/backend/internal/member"
)

func (h *Handler) ContentAnnouncements(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	h.withMember(w, r, func(svc *member.Service, account string) {
		m, err := svc.GetByAccount(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		items, err := h.content.ListAnnouncements(r.Context(), m.ID)
		if err != nil {
			h.handleContentErr(w, err)
			return
		}
		apix.OK(w, map[string]any{"items": items})
	})
}

func (h *Handler) ContentAnnouncementDetail(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	id := r.PathValue("id")
	h.withMember(w, r, func(svc *member.Service, account string) {
		m, err := svc.GetByAccount(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		detail, err := h.content.GetAnnouncement(r.Context(), m.ID, id)
		if err != nil {
			h.handleContentErr(w, err)
			return
		}
		apix.OK(w, detail)
	})
}

func (h *Handler) ContentFaqList(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.content.ListFaq(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) ContentFaqDetail(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	id := r.PathValue("id")
	detail, err := h.content.GetFaq(r.Context(), id)
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, detail)
}

func (h *Handler) ContentHelpList(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.content.ListHelp(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

type feedbackRequest struct {
	Subject string `json:"subject"`
	Content string `json:"content"`
}

func (h *Handler) ContentFeedbackSubmit(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	h.withMember(w, r, func(svc *member.Service, account string) {
		m, err := svc.GetByAccount(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		var req feedbackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apix.Validation(w, "请求体须为 JSON，且包含 subject / content")
			return
		}
		result, err := h.content.SubmitFeedback(r.Context(), m.ID, content.FeedbackInput{
			Subject: req.Subject,
			Content: req.Content,
		})
		if err != nil {
			h.handleContentErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) PublicLobbySlots(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.content.PublicLobbySlots(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) PublicSiteBrand(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	item, err := h.content.PublicSiteBrand(r.Context())
	if err != nil {
		h.handleContentErr(w, err)
		return
	}
	apix.OK(w, item)
}

func (h *Handler) handleContentErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, content.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, content.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "内容不存在")
	case errors.Is(err, content.ErrInvalid):
		apix.Validation(w, "请求参数无效")
	case errors.Is(err, content.ErrDuplicateSlotKey):
		apix.Validation(w, "slotKey 已存在")
	case errors.Is(err, content.ErrRoleNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "记录不存在")
	case errors.Is(err, content.ErrProtectedRole):
		apix.Validation(w, "不可删除内置超级管理员")
	case errors.Is(err, content.ErrAdminUserNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "账号不存在")
	case errors.Is(err, content.ErrProtectedAdminUser):
		apix.Validation(w, "不可删除内置超级管理员账号")
	case errors.Is(err, content.ErrDuplicateAdminAcct):
		apix.Validation(w, "登录账号已存在")
	case errors.Is(err, content.ErrAdminUserSelfDelete):
		apix.Validation(w, "不可删除当前登录账号")
	default:
		apix.Internal(w)
	}
}
