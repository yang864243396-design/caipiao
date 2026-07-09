package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/copyhall"
)

func (h *Handler) CopyHallRankings(w http.ResponseWriter, r *http.Request) {
	if h.copyHall == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	lotteryCode := r.URL.Query().Get("lotteryCode")
	board := r.URL.Query().Get("board")
	if lotteryCode == "" || board == "" {
		apix.Validation(w, "lotteryCode 与 board 不能为空")
		return
	}
	result, err := h.copyHall.Rankings(r.Context(), lotteryCode, board)
	if err != nil {
		if errors.Is(err, copyhall.ErrInvalidQuery) {
			apix.Validation(w, "lotteryCode 或 board 无效")
			return
		}
		apix.Internal(w)
		return
	}
	apix.OK(w, result)
}
