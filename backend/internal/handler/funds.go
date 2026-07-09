package handler

import (
	"net/http"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/member"
)

func (h *Handler) FundRecords(w http.ResponseWriter, r *http.Request) {
	h.withMember(w, r, func(svc *member.Service, account string) {
		q := member.FundRecordsQuery{
			DateFrom: r.URL.Query().Get("dateFrom"),
			DateTo:   r.URL.Query().Get("dateTo"),
			FlowType: r.URL.Query().Get("flowType"),
			Currency: r.URL.Query().Get("currency"),
			Cursor:   r.URL.Query().Get("cursor"),
			Limit:    queryInt(r, "limit", 20),
		}
		result, err := svc.FundRecords(r.Context(), account, q)
		if err != nil {
			h.handleMemberErr(w, err)
			return
		}
		apix.OK(w, result)
	})
}
