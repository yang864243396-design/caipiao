package handler

import (
	"errors"
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/middleware"
)

func (h *Handler) MemberProfile(w http.ResponseWriter, r *http.Request) {
	h.withMember(w, r, func(svc *member.Service, account string) {
		profile, err := svc.Profile(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		apix.OK(w, profile)
	})
}

func (h *Handler) MemberWallet(w http.ResponseWriter, r *http.Request) {
	h.withMember(w, r, func(svc *member.Service, account string) {
		wallet, err := svc.Wallet(r.Context(), account)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		apix.OK(w, wallet)
	})
}

func (h *Handler) OrdersLedger(w http.ResponseWriter, r *http.Request) {
	h.withMember(w, r, func(svc *member.Service, account string) {
		q := member.LedgerQuery{
			DateFrom: r.URL.Query().Get("dateFrom"),
			DateTo:   r.URL.Query().Get("dateTo"),
			Type:     r.URL.Query().Get("type"),
			OrderNo:  r.URL.Query().Get("orderNo"),
			Cursor:   r.URL.Query().Get("cursor"),
			Limit:    queryInt(r, "limit", 20),
		}
		result, err := svc.Ledger(r.Context(), account, q)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}

type memberFn func(svc *member.Service, account string)

func (h *Handler) withMember(w http.ResponseWriter, r *http.Request, fn memberFn) {
	if h.members == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok || claims.Subject == "" {
		apix.Unauthorized(w, "")
		return
	}
	fn(h.members, claims.Subject)
}

func (h *Handler) handleMemberErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, member.ErrNotFound):
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "会员不存在")
	case errors.Is(err, member.ErrInvalidQuery):
		apix.Validation(w, err.Error())
	case errors.Is(err, member.ErrUnavailable):
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
	default:
		apix.Internal(w)
	}
}
