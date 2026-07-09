package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/games"
)

func (h *Handler) AdminLotteryCatalogList(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.games.AdminListCatalog(r.Context())
	if err != nil {
		h.handleLotteryCatalogErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

type patchLotteryCatalogRequest struct {
	DisplayName         string `json:"displayName"`
	OutboundLotteryCode string `json:"outboundLotteryCode"`
	SortOrder           int    `json:"sortOrder"`
	SaleStatus          string `json:"saleStatus"`
	EnterMaintenance    bool   `json:"enterMaintenance"`
}

func (h *Handler) AdminLotteryCatalogPatch(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	code := r.PathValue("code")
	var req patchLotteryCatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	row, err := h.games.AdminPatchCatalog(r.Context(), code, games.PatchCatalogInput{
		DisplayName:         req.DisplayName,
		OutboundLotteryCode: req.OutboundLotteryCode,
		SortOrder:           req.SortOrder,
		SaleStatus:          req.SaleStatus,
		EnterMaintenance:    req.EnterMaintenance,
	})
	if err != nil {
		h.handleLotteryCatalogErr(w, err)
		return
	}
	if row.SaleStatus == "on_sale" {
		h.scheduleMaintenanceResume()
	}
	apix.OK(w, row)
}

func (h *Handler) AdminPlayTemplatesList(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	items, err := h.games.AdminListPlayTemplates(r.Context())
	if err != nil {
		h.handleLotteryCatalogErr(w, err)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}

func (h *Handler) AdminPlayTree(w http.ResponseWriter, r *http.Request) {
	if h.games == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	templateCode := r.PathValue("templateCode")
	tree, err := h.games.AdminPlayTree(r.Context(), templateCode)
	if err != nil {
		h.handleLotteryCatalogErr(w, err)
		return
	}
	apix.OK(w, tree)
}

func (h *Handler) handleLotteryCatalogErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, games.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, games.ErrCatalogNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "彩种不存在")
	case errors.Is(err, games.ErrCatalogNotEditable):
		apix.Fail(w, http.StatusForbidden, apix.CodeForbidden, "仅维护态彩种可编辑；上架彩种请先设为维护")
	case errors.Is(err, games.ErrCatalogInvalidPatch):
		apix.Fail(w, http.StatusBadRequest, apix.CodeValidation, "彩种维护字段不合法")
	default:
		apix.Internal(w)
	}
}
