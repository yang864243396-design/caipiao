package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
)

func (h *Handler) AdminListSchemeTemplates(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "pageSize", 10)
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	result, err := h.schemes.AdminListTemplatesPaged(r.Context(), schemes.AdminTemplateListQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     name,
	})
	if err != nil {
		h.handleSchemeTemplateErr(w, err)
		return
	}
	apix.OK(w, result)
}

type saveSchemeTemplateRequest struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	LotteryCode string          `json:"lotteryCode"`
	Brief       string          `json:"brief"`
	SortOrder   int             `json:"sortOrder"`
	Enabled     bool            `json:"enabled"`
	Rounds      json.RawMessage `json:"rounds"`
}

func (h *Handler) AdminSaveSchemeTemplate(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	var req saveSchemeTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	item, err := h.schemes.AdminSaveTemplate(r.Context(), schemes.SaveTemplateInput{
		ID:          req.ID,
		Name:        req.Name,
		LotteryCode: req.LotteryCode,
		Brief:       req.Brief,
		SortOrder:   req.SortOrder,
		Enabled:     req.Enabled,
		Rounds:      req.Rounds,
	})
	if err != nil {
		h.handleSchemeTemplateErr(w, err)
		return
	}
	action := "创建"
	if req.ID != "" {
		action = "更新"
	}
	h.writeAudit(r, fmt.Sprintf("%s方案模板 %s", action, item.ID))
	apix.OK(w, item)
}

func (h *Handler) AdminGetSchemeTemplate(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		apix.Validation(w, "id 不能为空")
		return
	}
	item, err := h.schemes.AdminGetTemplate(r.Context(), id)
	if err != nil {
		h.handleSchemeTemplateErr(w, err)
		return
	}
	apix.OK(w, item)
}

func (h *Handler) AdminDeleteSchemeTemplate(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	id := r.PathValue("id")
	if err := h.schemes.AdminDeleteTemplate(r.Context(), id); err != nil {
		h.handleSchemeTemplateErr(w, err)
		return
	}
	h.writeAudit(r, fmt.Sprintf("删除方案模板 %s", id))
	apix.OK(w, map[string]string{"id": id})
}

func (h *Handler) AdminResetSchemeTemplates(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.schemes.AdminResetTemplates(r.Context())
	if err != nil {
		h.handleSchemeTemplateErr(w, err)
		return
	}
	h.writeAudit(r, "恢复方案模板库默认")
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) ClientSchemeTemplates(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := strings.TrimSpace(r.PathValue("definitionId"))
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		items, err := h.schemes.ClientListTemplates(r.Context(), account, definitionID)
		if err != nil {
			h.handleSchemeTemplateErr(w, err)
			return
		}
		apix.OK(w, map[string]any{"items": items})
	})
}

type clientSaveSchemeTemplateRequest struct {
	Name         string          `json:"name"`
	DefinitionID string          `json:"definitionId"`
	Brief        string          `json:"brief"`
	Rounds       json.RawMessage `json:"rounds"`
}

func (h *Handler) ClientGetSchemeTemplate(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	id := r.PathValue("templateId")
	if id == "" {
		apix.Validation(w, "templateId 不能为空")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		item, err := h.schemes.ClientGetTemplate(r.Context(), account, id)
		if err != nil {
			h.handleSchemeTemplateErr(w, err)
			return
		}
		apix.OK(w, item)
	})
}

func (h *Handler) ClientCreateSchemeTemplate(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	definitionID := strings.TrimSpace(r.PathValue("definitionId"))
	if definitionID == "" {
		apix.Validation(w, "definitionId 不能为空")
		return
	}
	var req clientSaveSchemeTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		item, err := h.schemes.ClientCreateMemberTemplate(r.Context(), account, schemes.ClientSaveMemberTemplateInput{
			Name:         req.Name,
			DefinitionID: definitionID,
			Brief:        req.Brief,
			Rounds:       req.Rounds,
		})
		if err != nil {
			h.handleSchemeTemplateErr(w, err)
			return
		}
		apix.OK(w, item)
	})
}

func (h *Handler) ClientUpdateSchemeTemplate(w http.ResponseWriter, r *http.Request) {
	if h.schemes == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	id := r.PathValue("templateId")
	if id == "" {
		apix.Validation(w, "templateId 不能为空")
		return
	}
	var req clientSaveSchemeTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		item, err := h.schemes.ClientUpdateMemberTemplate(r.Context(), account, id, schemes.ClientUpdateMemberTemplateInput{
			Name:   req.Name,
			Brief:  req.Brief,
			Rounds: req.Rounds,
		})
		if err != nil {
			h.handleSchemeTemplateErr(w, err)
			return
		}
		apix.OK(w, item)
	})
}

func (h *Handler) handleSchemeTemplateErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, schemes.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, schemes.ErrTemplateNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "模板不存在")
	case errors.Is(err, schemes.ErrTemplateForbidden):
		apix.Fail(w, http.StatusForbidden, apix.CodeForbidden, "无权修改该模板")
	case errors.Is(err, schemes.ErrInvalidTemplate):
		apix.Validation(w, "模板参数无效")
	case errors.Is(err, schemes.ErrDefinitionNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "方案不存在")
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
	default:
		apix.Internal(w)
	}
}
