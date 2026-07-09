package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/orders/bets"
)

func (h *Handler) OrdersBets(w http.ResponseWriter, r *http.Request) {
	h.withMember(w, r, func(_ *member.Service, account string) {
		if h.bets == nil {
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
			return
		}
		result, err := h.bets.List(r.Context(), bets.Query{
			Account:            account,
			DateFrom:           r.URL.Query().Get("dateFrom"),
			DateTo:             r.URL.Query().Get("dateTo"),
			GameCode:           r.URL.Query().Get("gameCode"),
			SchemeDefinitionID: r.URL.Query().Get("schemeDefinitionId"),
			OrderNo:            r.URL.Query().Get("orderNo"),
			Cursor:             r.URL.Query().Get("cursor"),
			Limit:              queryInt(r, "limit", 20),
		})
		if err != nil {
			h.handleOrdersErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

func (h *Handler) handleOrdersErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
	case errors.Is(err, bets.ErrInvalidQuery):
		apix.Validation(w, err.Error())
	default:
		apix.Internal(w)
	}
}
