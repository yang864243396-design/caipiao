package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/copyhall"
)

func (h *Handler) AdminCopyHallRankings(w http.ResponseWriter, r *http.Request) {
	if h.copyHall == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	boardKind := r.URL.Query().Get("board")
	if boardKind == "" {
		apix.Validation(w, "board 不能为空")
		return
	}
	result, err := h.copyHall.AdminRankingsBoard(r.Context(), boardKind)
	if err != nil {
		h.handleCopyHallAdminErr(w, err)
		return
	}
	apix.OK(w, result)
}

type saveCopyHallBoardRequest struct {
	Slots []copyhall.RankSlot `json:"slots"`
}

func (h *Handler) AdminCopyHallSaveBoard(w http.ResponseWriter, r *http.Request) {
	if h.copyHall == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	boardKind := r.PathValue("boardKind")
	var req saveCopyHallBoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apix.Validation(w, "请求体须为 JSON")
		return
	}
	board, err := h.copyHall.AdminSaveBoard(r.Context(), boardKind, req.Slots)
	if err != nil {
		h.handleCopyHallAdminErr(w, err)
		return
	}
	boardName := "大神榜"
	if boardKind == "contrary" {
		boardName = "反买榜"
	}
	h.writeAudit(r, fmt.Sprintf("更新跟单大厅 %s", boardName))
	apix.OK(w, board)
}

func (h *Handler) AdminCopyHallResetBoard(w http.ResponseWriter, r *http.Request) {
	if h.copyHall == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	boardKind := r.PathValue("boardKind")
	board, err := h.copyHall.AdminResetBoard(r.Context(), boardKind)
	if err != nil {
		h.handleCopyHallAdminErr(w, err)
		return
	}
	boardName := "大神榜"
	if boardKind == "contrary" {
		boardName = "反买榜"
	}
	h.writeAudit(r, fmt.Sprintf("恢复跟单大厅默认 %s", boardName))
	apix.OK(w, board)
}

func (h *Handler) AdminCopyHallResetAll(w http.ResponseWriter, r *http.Request) {
	if h.copyHall == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	state, err := h.copyHall.AdminResetAll(r.Context())
	if err != nil {
		h.handleCopyHallAdminErr(w, err)
		return
	}
	h.writeAudit(r, "恢复跟单大厅全部默认榜单")
	apix.OK(w, state)
}

func (h *Handler) handleCopyHallAdminErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, copyhall.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	case errors.Is(err, copyhall.ErrInvalidQuery):
		apix.Validation(w, "榜单参数无效")
	case errors.Is(err, copyhall.ErrInvalidBoard):
		apix.Validation(w, err.Error())
	default:
		apix.Internal(w)
	}
}
