package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/member"
	ordersadmin "caipiao/backend/internal/orders/admin"
)

func (h *Handler) AdminListBetOrders(w http.ResponseWriter, r *http.Request) {
	if h.ordersAdmin == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	q := ordersadmin.BetListQuery{
		IssueNo:       r.URL.Query().Get("issueNo"),
		MemberAccount: r.URL.Query().Get("memberAccount"),
		SchemeName:    r.URL.Query().Get("schemeName"),
		LotteryCode:   r.URL.Query().Get("lotteryCode"),
		Page:          queryInt(r, "page", 1),
		PageSize:      queryInt(r, "pageSize", 10),
	}
	result, err := h.ordersAdmin.ListBets(r.Context(), q)
	if err != nil {
		if errors.Is(err, ordersadmin.ErrUnavailable) {
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
			return
		}
		apix.Internal(w)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) AdminListChaseOrders(w http.ResponseWriter, r *http.Request) {
	if h.ordersAdmin == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	q := ordersadmin.ChaseListQuery{
		ChaseNo:       r.URL.Query().Get("chaseNo"),
		MemberAccount: r.URL.Query().Get("memberAccount"),
		Status:        r.URL.Query().Get("status"),
		LotteryCode:   r.URL.Query().Get("lotteryCode"),
		Page:          queryInt(r, "page", 1),
		PageSize:      queryInt(r, "pageSize", 10),
	}
	result, err := h.ordersAdmin.ListChases(r.Context(), q)
	if err != nil {
		if errors.Is(err, ordersadmin.ErrUnavailable) {
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
			return
		}
		apix.Internal(w)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) AdminListLedgerEntries(w http.ResponseWriter, r *http.Request) {
	if h.members == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	result, err := h.members.AdminFundRecords(r.Context(), member.AdminSiteFundRecordsQuery{
		DateFrom:      r.URL.Query().Get("dateFrom"),
		DateTo:        r.URL.Query().Get("dateTo"),
		FlowType:      r.URL.Query().Get("flowType"),
		Currency:      r.URL.Query().Get("currency"),
		MemberAccount: r.URL.Query().Get("memberAccount"),
		LedgerNo:      r.URL.Query().Get("ledgerNo"),
		Page:          queryInt(r, "page", 1),
		PageSize:      queryInt(r, "pageSize", 10),
	})
	if err != nil {
		h.handleMemberErr(w, err)
		return
	}
	apix.OK(w, result)
}

func (h *Handler) adminOrdersList(
	w http.ResponseWriter,
	r *http.Request,
	fn func(ordersadmin.ListQuery) (any, error),
) {
	if h.ordersAdmin == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	q := ordersadmin.ListQuery{
		Keyword:  r.URL.Query().Get("keyword"),
		Page:     queryInt(r, "page", 1),
		PageSize: queryInt(r, "pageSize", 10),
	}
	result, err := fn(q)
	if err != nil {
		if errors.Is(err, ordersadmin.ErrUnavailable) {
			apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
			return
		}
		apix.Internal(w)
		return
	}
	apix.OK(w, result)
}
