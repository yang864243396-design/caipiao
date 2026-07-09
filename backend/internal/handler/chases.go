package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/orders/chases"
)

func (h *Handler) OrdersChases(w http.ResponseWriter, r *http.Request) {
	if h.chases == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	h.withMember(w, r, func(_ *member.Service, account string) {
		result, err := h.chases.List(r.Context(), chases.Query{
			Account:  account,
			DateFrom: r.URL.Query().Get("dateFrom"),
			DateTo:   r.URL.Query().Get("dateTo"),
			GameCode: r.URL.Query().Get("gameCode"),
			Cursor:   r.URL.Query().Get("cursor"),
			Limit:    queryInt(r, "limit", 20),
		})
		if err != nil {
			h.handleChaseErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) handleChaseErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chases.ErrInvalidQuery):
		apix.Validation(w, err.Error())
	default:
		apix.Internal(w)
	}
}
